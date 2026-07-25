# Motor de consultas SQL en memoria

Proyecto para el Taller de Programacion en Go. Implementa un subconjunto de SQL que consulta tablas cargadas desde archivos CSV y mantenidas en memoria.

## Estado

Hitos 1 a 5 completados. El motor admite:

```sql
[EXPLAIN]
SELECT [DISTINCT] <expresion> [AS alias], ...
  FROM tabla [INNER JOIN tabla ON condicion]...
 [WHERE condicion]
 [GROUP BY expresion, ...]
 [HAVING condicion]
 [ORDER BY expresion [ASC|DESC], ...]
 [LIMIT n] [OFFSET n]
```

Los elementos del `SELECT` son expresiones: admiten aritmetica (`+ - * /`),
signo unario, alias con `AS` y funciones de agregacion (`COUNT`, `SUM`, `AVG`,
`MIN`, `MAX`) mezcladas en cualquier orden. El punto y coma final es opcional.
La gramatica completa y las **limitaciones conocidas** estan en
[docs/grammar.md](docs/grammar.md).

## Estructura

Cada paquete interno cubre una etapa del recorrido de una consulta:

```text
cmd/sqlmem/        Punto de entrada del CLI.
internal/lexer/    SQL -> tokens.
internal/parser/   Tokens -> AST.
internal/storage/  Datos en memoria: tipos, valores, filas, tablas y carga de CSV.
internal/catalog/  Registro de tablas por nombre.
internal/functions/ Registro de funciones de agregacion.
internal/planner/  AST -> plan logico.
internal/executor/ Plan logico -> operadores (modelo Volcano).
internal/engine/   Fachada que encadena las etapas.
internal/cli/      Subcomandos, REPL y formato de salida.
tests/             Pruebas de integracion de SQL a resultado.
data/              CSV de ejemplo: sedes, areas y empleados relacionados.
docs/              Arquitectura, gramatica y decisiones de diseno.
```

El detalle del flujo y del grafo de dependencias esta en
[docs/architecture.md](docs/architecture.md).

## Requisitos

- Go 1.24 o superior.

## Comandos de desarrollo

Antes de entregar el proyecto, ejecutar los siguientes comandos desde la carpeta raiz del repositorio.

### 1. Formatear todos los archivos Go

En PowerShell:

```powershell
Get-ChildItem -Recurse -Filter *.go | ForEach-Object { gofmt -w $_.FullName }
```

Este comando busca todos los archivos con extension `.go` y aplica el formato estandar de Go. Debe ejecutarse primero porque puede corregir automaticamente sangrias, espacios y saltos de linea.

> No usar `gofmt -w .`: `gofmt` recibe archivos, no directorios.

### 2. Compilar todos los paquetes

```bash
go build ./...
```

Compila el ejecutable y todos los paquetes internos. Si aparece un error, no se debe entregar hasta corregirlo.

### 3. Ejecutar las pruebas

```bash
go test ./...
```

Ejecuta las pruebas unitarias de cada paquete y las de integracion de `tests/`, que recorren el flujo completo de SQL a resultado. Cada paquete debe terminar con `ok`.

### 4. Revisar problemas comunes

```bash
go vet ./...
```

Busca usos sospechosos del lenguaje que pueden compilar, pero provocar errores de comportamiento. El comando debe terminar sin mensajes.

### Resultado esperado

Los cuatro pasos deben finalizar sin errores. Si se corrigio codigo despues de ejecutar las pruebas, repetir desde el paso 1.

## Datos de ejemplo

`data/` contiene tres tablas relacionadas, pensadas para que los `JOIN` y los
`GROUP BY` den resultados con sentido:

```text
sedes(id, ciudad, pais)                                    6 filas
areas(id, nombre, sede_id, presupuesto)                   12 filas
empleados(id, nombre, area_id, sede_id, edad, salario,   300 filas
          activo, fecha_ingreso)
```

Cada empleado pertenece a un area y cada area a una sede, de modo que
`empleados → areas → sedes` es un join de tres tablas real.

Los datos incluyen a proposito casos que ejercitan el motor: valores `NULL` en
`area_id` (clave de union nula), `edad`, `salario` y `fecha_ingreso`; acentos y
enes en nombres y ciudades; una coma dentro de un campo entrecomillado
(`"Bogotá, D.C."`); un presupuesto sin asignar y otro de diez cifras.

## Ejecucion actual

```bash
go run ./cmd/sqlmem cargar empleados data/empleados.csv
```

Salida esperada:

```text
Tabla "empleados" cargada: 300 filas
- id: entero
- nombre: texto
- area_id: entero
- sede_id: entero
- edad: entero
- salario: decimal
- activo: booleano
- fecha_ingreso: texto
```

Consultar los datos cargados desde un CSV:

```bash
go run ./cmd/sqlmem consultar empleados data/empleados.csv "SELECT nombre, salario FROM empleados WHERE activo = true AND edad >= 25 LIMIT 10"
```

Ordenar y limitar resultados:

```bash
go run ./cmd/sqlmem consultar empleados data/empleados.csv "SELECT nombre, salario FROM empleados ORDER BY salario DESC LIMIT 5"
```

Agrupar resultados y usar agregados:

```bash
go run ./cmd/sqlmem consultar empleados data/empleados.csv "SELECT activo, COUNT(*), AVG(salario) FROM empleados GROUP BY activo ORDER BY activo"
```

Expresiones y alias en el `SELECT`:

```bash
go run ./cmd/sqlmem consultar empleados data/empleados.csv "SELECT nombre, salario * 12 AS anual FROM empleados WHERE activo = true ORDER BY anual DESC LIMIT 5"
```

Contar los nulos de cada columna, que es donde se ve la diferencia entre
`COUNT(*)` y `COUNT(columna)`:

```bash
go run ./cmd/sqlmem consultar empleados data/empleados.csv "SELECT COUNT(*), COUNT(area_id), COUNT(edad), COUNT(salario) FROM empleados"
```

```text
COUNT(*)  COUNT(area_id)  COUNT(edad)  COUNT(salario)
300       294             292          295
```

### Sesion interactiva (REPL)

```bash
go run ./cmd/sqlmem repl empleados=data/empleados.csv areas=data/areas.csv sedes=data/sedes.csv
```

Se ejecuta una consulta por linea y un error no cierra la sesion:

```text
sqlmem> SELECT nombre, salario FROM empleados WHERE activo = true LIMIT 2;
nombre            salario
Martín Castro     1337.29
Lucía Pérez       1474.58
sqlmem> .tablas
areas (12 filas, 4 columnas)
empleados (300 filas, 8 columnas)
sedes (6 filas, 3 columnas)
sqlmem> .esquema sedes
Tabla "sedes": 6 filas
- id: entero
- ciudad: texto
- pais: texto
sqlmem> .salir
```

Comandos: `.tablas`, `.esquema <tabla>`, `.ayuda` y `.salir` (o `.exit`).

### Ver el plan de ejecucion

```bash
go run ./cmd/sqlmem consultar empleados data/empleados.csv "EXPLAIN SELECT nombre, salario FROM empleados WHERE edad >= 18"
```

```text
plan
Project(nombre, salario)
└── Filter(edad >= 18)
    └── Scan(empleados)
```

Filtrar grupos con `HAVING` y paginar con `OFFSET`:

```bash
go run ./cmd/sqlmem consultar empleados data/empleados.csv "SELECT area_id, COUNT(*) FROM empleados GROUP BY area_id HAVING COUNT(*) > 30 ORDER BY COUNT(*) DESC"

go run ./cmd/sqlmem consultar empleados data/empleados.csv "SELECT nombre FROM empleados ORDER BY salario DESC LIMIT 5 OFFSET 10"
```

## Alcance previsto

1. Carga de CSV como tablas en memoria con catalogo, esquemas y tipos.
2. Lexer, parser y AST para `SELECT ... FROM ... WHERE ...`.
3. Operadores `Scan`, `Filter` y `Project` mediante el modelo Volcano. Completado.
4. `ORDER BY`, `LIMIT`, `GROUP BY` y agregados. Completado.
5. `INNER JOIN` con nested-loop y hash join. Completado.
6. Expresiones y alias en el `SELECT`, `HAVING`, `OFFSET` y varios `JOIN`. Completado.
7. REPL, `EXPLAIN`, `DISTINCT` y eleccion de estrategia de join. Completado.

## Consultas con varias tablas

Se indican las fuentes como `tabla=archivo.csv`, luego `--`, luego el SQL.

```bash
go run ./cmd/sqlmem consultar empleados=data/empleados.csv areas=data/areas.csv -- "SELECT empleados.nombre, areas.nombre FROM empleados INNER JOIN areas ON empleados.area_id = areas.id ORDER BY empleados.nombre LIMIT 10"
```

Join de tres tablas con agregados y `HAVING`:

```bash
go run ./cmd/sqlmem consultar empleados=data/empleados.csv areas=data/areas.csv sedes=data/sedes.csv -- "SELECT sedes.ciudad, COUNT(*), AVG(empleados.salario) FROM empleados INNER JOIN areas ON empleados.area_id = areas.id INNER JOIN sedes ON areas.sede_id = sedes.id GROUP BY sedes.ciudad HAVING COUNT(*) > 30 ORDER BY COUNT(*) DESC"
```

```text
sedes.ciudad  COUNT(*)  AVG(empleados.salario)
Lima          146       5422.090273972603
Arequipa      40        5441.495
Bogotá, D.C.  39        5390.29076923077
```

Los 6 empleados sin `area_id` no aparecen en el join: una clave `NULL` no casa
con nada, ni siquiera con otra `NULL`.
