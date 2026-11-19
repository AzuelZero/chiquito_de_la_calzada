<div align="center">
  <img src="assets/logo.png" width="120" alt="Perro con gafas de sol fumando un cigarrillo, logo de Chiquito de la Calzada" />
  <h1>Chiquito de la Calzada</h1>
  <p><strong>Calculadora de fechas</strong> · diferencias, sumas y restas · escritorio y CLI</p>
</div>

![Ventana principal de Chiquito de la Calzada](assets/captura.png)

Calculadora de fechas escrita en **Go**. Calcula la diferencia entre dos
momentos, la desglosa en años, meses, días, horas, minutos y segundos, y permite
sumar o restar duraciones. Incluye una aplicación de escritorio (Wails +
WebView2) y una herramienta de línea de comandos. Todo el cálculo ocurre en Go:
la interfaz es una página estática embebida en el ejecutable, sin Node, npm ni
conexión a internet.

## Qué puedes hacer

- **Diferencia entre fechas** con fecha y hora, botón «Ahora», intercambio de
  fechas y opción de contar ambos extremos.
- **Días naturales** en grande, con el desglose exacto y la relación entre las
  fechas.
- **La misma diferencia en** años, meses, semanas, días de 24 h, horas, minutos,
  segundos y milisegundos.
- **Sumar o restar** una duración a una fecha base, con resultado legible e
  ISO 8601.
- **Zonas horarias IANA** con offset y horario de verano, y avisos cuando una
  hora local no existe o se repite por un cambio de hora.
- **Salida JSON** en la CLI para integrarla en otros scripts.

## La aplicación de escritorio

Pensada para Windows 10/11. Descarga `chiquito_de_la_calzada.exe` desde
[Releases](https://github.com/AzuelZero/chiquito_de_la_calzada/releases) y ábrelo;
no necesita instalación. WebView2 suele venir con Windows y, si falta, Windows lo
ofrece al arrancar la aplicación.

- El selector superior cambia la zona horaria de todos los cálculos. La pastilla
  de al lado muestra la abreviatura y el offset actuales (`CET · UTC+01:00`,
  `CEST · UTC+02:00`…).
- Al abrir se calculan los últimos 30 días como ejemplo.
- La fecha y la hora se interpretan en la zona elegida, no en la del navegador.
- El botón «Ahora» usa la hora real de esa zona.
- Al editar una fecha, el resultado se recalcula solo; si algo falla, el
  resultado anterior no desaparece de la pantalla.
- El panel de unidades muestra las 4 relevantes para la magnitud calculada;
  «Ver todas las unidades» despliega las demás.
- En «Sumar o restar», Enter repite la última operación usada y el resultado
  también se puede copiar.
- Al intercambiar las dos fechas… **¡JARL!** Si además dejas un clip propio
  como `gui/frontend/dist/assets/jarl.mp3`, el grito suena (el fichero de
  audio no se distribuye con el repositorio).
- Los avisos de horario de verano aparecen bajo el formulario correspondiente:
  - **Hora inexistente** al adelantar el reloj: se indica cómo la interpreta Go
    (por ejemplo, `02:30` se convierte en `03:30`).
  - **Hora repetida** al atrasar el reloj: se indica cuál de las dos
    ocurrencias se ha tomado.

La ventana mide 1000 × 720 y se puede redimensionar. El diseño usa una paleta
azul-violeta, la tipografía **Fraunces** para el título y la cifra principal, y
Segoe UI para el resto.

## Zonas horarias y horario de verano

La base de datos IANA viaja dentro del ejecutable mediante `time/tzdata`, así
que las zonas funcionan aunque Windows no tenga esos datos instalados. El
horario de verano se aplica automáticamente: la misma hora de pared puede tener
un offset distinto según la fecha.

Ese cambio también afecta a la aritmética. Entre el 29 de marzo de 2026 a las
01:30 y las 04:30 en Madrid solo hay **2 horas**, no 3, porque de 02:00 a 03:00
no existe:

```text
$ chiquito_de_la_calzada_cli 2026-03-29T01:30 2026-03-29T04:30 --tz Europe/Madrid
Zona: Europe/Madrid (CET, UTC+01:00)

Fecha 1: 29/03/2026 01:30:00 (domingo)
Fecha 2: 29/03/2026 04:30:00 (domingo)

La fecha 2 es posterior a la fecha 1.
Diferencia: 2 horas
Días naturales: 0 (29/03/2026 → 29/03/2026)
```

Por eso los **días naturales** y los **días de 24 h** pueden no coincidir: los
primeros cuentan días de calendario y los segundos, tiempo real transcurrido.

En la CLI, `--tz` elige la zona y `--utc` fuerza UTC. No se pueden usar juntos.
La GUI avisa de las horas ambiguas o inexistentes; la CLI aplica las reglas de
Go para la zona indicada.

## Línea de comandos

El binario de la CLI se publica como `chiquito_de_la_calzada_cli.exe`. También
puedes compilarlo con las instrucciones del final.

```text
chiquito_de_la_calzada_cli                        Modo interactivo
chiquito_de_la_calzada_cli <fecha1>               Diferencia con hoy
chiquito_de_la_calzada_cli <fecha1> <fecha2>      Diferencia entre fechas
chiquito_de_la_calzada_cli diff <fecha1> <fecha2> Igual que arriba
chiquito_de_la_calzada_cli add <fecha> <duración> Sumar una duración
chiquito_de_la_calzada_cli sub <fecha> <duración> Restar una duración

Opciones:
  -u, --utc         Interpretar las fechas en UTC
  -z, --tz <zona>   Zona IANA, p. ej. Europe/Madrid o America/Bogota
  -j, --json        Salida en JSON
  -i, --inclusive   Contar ambos extremos en los días naturales
  -h, --help        Mostrar la ayuda
```

### Ejemplos

```sh
# Diferencia entre dos fechas en Madrid
chiquito_de_la_calzada_cli 2026-01-01 2026-03-15 --tz Europe/Madrid

# La misma diferencia en JSON
chiquito_de_la_calzada_cli diff 2026-01-01 2026-03-15 --json

# Sumar y restar duraciones
chiquito_de_la_calzada_cli add 2026-01-31 "1 mes"        # 28/02/2026
chiquito_de_la_calzada_cli sub "2026-03-15 18:00" 45min

# Modo interactivo
chiquito_de_la_calzada_cli
```

Salida de ejemplo:

```text
Zona: Europe/Madrid (CET, UTC+01:00)

Fecha 1: 01/01/2026 00:00:00 (jueves)
Fecha 2: 15/03/2026 00:00:00 (domingo)

La fecha 2 es posterior a la fecha 1.
Diferencia: 2 meses y 14 días
Días naturales: 73 (01/01/2026 → 15/03/2026)

Totales:
  Años (365,2425 días)                    0,2
  Meses (30,436875 días)                  2,4
  Semanas                               10,43
  Días (24 h)                              73
  Horas                                 1.752
  Minutos                             105.120
  Segundos                          6.307.200
  Milisegundos                  6.307.200.000
```

### Formatos aceptados

- **Fechas:** `2026-11-18`, `18/11/2026`, `18-11-2026`,
  `2026-11-18 14:30`, `18/11/2026 14:30:05`, RFC 3339 o `hoy` / `ahora`.
- **Duraciones:** `1a 2mes 3sem 4d 5h 6min 7s`. También se aceptan `1y 2mo 3w
  4d 5h 6m 7s`, un número suelto (días) y signos negativos: `-2d`, `-1mes`.
- Al sumar meses, el día se ajusta al último del mes de destino: el 31 de enero
  más un mes es el 28 de febrero (o el 29 en un año bisiesto).

## Compilar

Requisitos: **Go 1.26 o superior**. Para la GUI, Windows 10/11 y el runtime de
WebView2.

```sh
git clone https://github.com/AzuelZero/chiquito_de_la_calzada.git
cd chiquito_de_la_calzada

# CLI
go build -o chiquito_de_la_calzada_cli.exe .

# Aplicación de escritorio
go build -tags "gui desktop production" -ldflags "-H windowsgui -s -w" -o chiquito_de_la_calzada.exe ./gui
```

El icono del ejecutable se enlaza desde `gui/rsrc.syso`. Si cambias el logo,
regenera el `.ico` y el recurso:

```sh
go run tools/icogen/main.go assets/logo.png gui/app.ico
go run github.com/akavel/rsrc@latest -ico gui/app.ico -o gui/rsrc.syso
```

## Pruebas

```sh
go test ./...                                  # CLI y lógica de fechas
go test -tags "gui desktop production" ./gui   # backend de la GUI y DST
```

La GUI vive detrás del tag `gui` para que el resto del proyecto se pueda probar
sin compilar Wails.

## Estructura

```text
.
├── datecore/            Lógica compartida: fechas, duraciones, totales y DST
├── gui/                 Aplicación Wails
│   ├── frontend/dist/   HTML, CSS, JS, logo y fuente Fraunces
│   ├── app.ico          Icono multi-resolución
│   └── rsrc.syso        Recurso de icono para el .exe
├── tools/icogen/        Generador Go del .ico
├── assets/              Logo y captura para la documentación
├── main.go              CLI
├── main_test.go         Pruebas de la CLI
└── README.md
```

## Créditos

- **Logo:** fotografía propia. Las propuestas de dibujo anteriores viven en
  `assets/propuestas/`.
- **Fraunces:** tipografía de Undercase Type, bajo SIL Open Font License 1.1.
  La licencia está en
  [`gui/frontend/dist/assets/fonts/OFL.txt`](gui/frontend/dist/assets/fonts/OFL.txt).
- **Wails:** framework de escritorio para Go.
- **Nombre:** homenaje a Chiquito de la Calzada. Proyecto independiente, sin
  relación con el humorista ni sus herederos.

## Licencia

[MIT](LICENSE) © 2026 Azuel.
