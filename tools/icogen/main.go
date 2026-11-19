//go:build ignore

// icogen convierte el logo PNG en un icono .ico multi-resolución para el
// ejecutable de Windows. Uso:
//
//	go run tools/icogen/main.go assets/logo.png gui/app.ico
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
)

var icoSizes = []int{16, 24, 32, 48, 64, 128, 256}

type entry struct {
	size int
	data []byte
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "uso: icogen <origen.png> <destino.ico>")
		os.Exit(2)
	}

	src, err := loadPNG(os.Args[1])
	if err != nil {
		fatalf("%v", err)
	}

	entries := make([]entry, 0, len(icoSizes))
	for _, size := range icoSizes {
		thumb := resize(src, size)
		data, err := encode(size, thumb)
		if err != nil {
			fatalf("tamaño %d: %v", size, err)
		}
		entries = append(entries, entry{size: size, data: data})
	}

	if err := writeICO(os.Args[2], entries); err != nil {
		fatalf("%v", err)
	}

	fmt.Printf("icono escrito: %s (%d tamaños)\n", os.Args[2], len(entries))
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "icogen: "+format+"\n", args...)
	os.Exit(1)
}

// loadPNG decodifica la imagen y la convierte a NRGBA para conservar el alfa.
func loadPNG(path string) (*image.NRGBA, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decodificar %s: %w", path, err)
	}

	b := img.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			out.Set(x, y, img.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return out, nil
}

// resize reduce la imagen promediando el área de origen de cada píxel destino.
// Trabaja con alfa premultiplicado para que los bordes no se oscurezcan.
func resize(src *image.NRGBA, size int) *image.NRGBA {
	sw, sh := src.Bounds().Dx(), src.Bounds().Dy()
	out := image.NewNRGBA(image.Rect(0, 0, size, size))

	for dy := 0; dy < size; dy++ {
		y0 := dy * sh / size
		y1 := (dy + 1) * sh / size
		if y1 <= y0 {
			y1 = y0 + 1
		}
		for dx := 0; dx < size; dx++ {
			x0 := dx * sw / size
			x1 := (dx + 1) * sw / size
			if x1 <= x0 {
				x1 = x0 + 1
			}

			var sr, sg, sb, sa, n float64
			for y := y0; y < y1; y++ {
				for x := x0; x < x1; x++ {
					c := src.NRGBAAt(x, y)
					a := float64(c.A) / 255
					sr += float64(c.R) * a
					sg += float64(c.G) * a
					sb += float64(c.B) * a
					sa += float64(c.A)
					n++
				}
			}
			if n == 0 {
				continue
			}

			alpha := sa / n
			var r, g, b float64
			if alpha > 0 {
				r = sr / n / (alpha / 255)
				g = sg / n / (alpha / 255)
				b = sb / n / (alpha / 255)
			}
			out.SetNRGBA(dx, dy, color.NRGBA{
				R: clamp(r),
				G: clamp(g),
				B: clamp(b),
				A: clamp(alpha),
			})
		}
	}
	return out
}

func clamp(v float64) uint8 {
	switch {
	case v <= 0:
		return 0
	case v >= 255:
		return 255
	default:
		return uint8(v + 0.5)
	}
}

// encode genera la entrada del icono: PNG para 256 px (más compacto) y BMP
// de 32 bits con máscara AND para el resto, que es lo que espera Windows.
func encode(size int, img *image.NRGBA) ([]byte, error) {
	if size >= 256 {
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}

	rowBytes := ((size + 31) / 32) * 4
	maskBytes := rowBytes * size
	header := 40
	pixels := size * size * 4

	var buf bytes.Buffer
	buf.Grow(header + pixels + maskBytes)

	le := binary.LittleEndian
	writeU32 := func(v uint32) { _ = binary.Write(&buf, le, v) }

	// BITMAPINFOHEADER
	writeU32(40)                           // tamaño del header
	writeU32(uint32(size))                 // ancho
	writeU32(uint32(size * 2))             // alto (imagen + máscara)
	_ = binary.Write(&buf, le, uint16(1))  // planos
	_ = binary.Write(&buf, le, uint16(32)) // bits por píxel
	writeU32(0)                            // sin compresión
	writeU32(uint32(pixels))               // tamaño de la imagen
	writeU32(0)                            // resolución X
	writeU32(0)                            // resolución Y
	writeU32(0)                            // colores usados
	writeU32(0)                            // colores importantes

	// Píxeles BGRA de abajo arriba.
	for y := size - 1; y >= 0; y-- {
		for x := 0; x < size; x++ {
			c := img.NRGBAAt(x, y)
			buf.WriteByte(c.B)
			buf.WriteByte(c.G)
			buf.WriteByte(c.R)
			buf.WriteByte(c.A)
		}
	}

	// Máscara AND: 1 = transparente. Windows usa el canal alfa, pero algunos
	// shells antiguos miran la máscara.
	mask := make([]byte, maskBytes)
	for y := 0; y < size; y++ {
		row := y * rowBytes
		for x := 0; x < size; x++ {
			if img.NRGBAAt(x, y).A < 128 {
				mask[row+x/8] |= 0x80 >> (x % 8)
			}
		}
	}
	buf.Write(mask)

	return buf.Bytes(), nil
}

func writeICO(path string, entries []entry) error {
	var buf bytes.Buffer
	le := binary.LittleEndian

	_ = binary.Write(&buf, le, uint16(0)) // reservado
	_ = binary.Write(&buf, le, uint16(1)) // tipo icono
	_ = binary.Write(&buf, le, uint16(len(entries)))

	offset := 6 + 16*len(entries)
	for _, e := range entries {
		dim := byte(e.size)
		if e.size >= 256 {
			dim = 0
		}
		buf.WriteByte(dim)                     // ancho
		buf.WriteByte(dim)                     // alto
		buf.WriteByte(0)                       // colores de paleta
		buf.WriteByte(0)                       // reservado
		_ = binary.Write(&buf, le, uint16(1))  // planos
		_ = binary.Write(&buf, le, uint16(32)) // bits por píxel
		_ = binary.Write(&buf, le, uint32(len(e.data)))
		_ = binary.Write(&buf, le, uint32(offset))
		offset += len(e.data)
	}

	for _, e := range entries {
		buf.Write(e.data)
	}

	return os.WriteFile(path, buf.Bytes(), 0o644)
}
