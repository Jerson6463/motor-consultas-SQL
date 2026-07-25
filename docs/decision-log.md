# Bitacora de decisiones

## Plantilla de entrada

- Hito y fecha:
- Decision tomada:
- Alternativas evaluadas:
- Justificacion tecnica:

## Hito inicial - estructura del repositorio

- Hito y fecha: Preparacion inicial, 2026-07-24.
- Decision tomada: Separar el punto de entrada en `cmd/sqlmem`, la implementacion en `internal`, los datos de ejemplo en `data` y la documentacion en `docs`.
- Alternativas evaluadas: Mantener todos los archivos en la raiz del proyecto.
- Justificacion tecnica: La separacion facilita el crecimiento del motor sin mezclar el ejecutable, la logica interna, los datos y los documentos requeridos.

## Hito 1 - carga de CSV y tipos

- Hito y fecha: Hito 1, 2026-07-24.
- Decision tomada: Representar una fila como una lista de valores tipados y conservar explicitamente si un valor es `NULL`.
- Alternativas evaluadas: Guardar todas las celdas como texto y convertirlas durante cada consulta.
- Justificacion tecnica: Convertir los valores al cargar el CSV valida los datos una sola vez y permite que los futuros operadores comparen valores segun su tipo real.

## Hito 2 - analisis de consultas

- Hito y fecha: Hito 2, 2026-07-24.
- Decision tomada: Separar lexer, parser y AST en el paquete `internal/query`.
- Alternativas evaluadas: Interpretar la consulta directamente mientras se recorre la cadena.
- Justificacion tecnica: El AST conserva la estructura y precedencia de la consulta, por lo que los operadores de ejecucion podran construirse despues sin mezclar analisis sintactico y acceso a datos.

## Hito 3 - operadores de ejecucion

- Hito y fecha: Hito 3, 2026-07-24.
- Decision tomada: Usar una interfaz `Operator` con los metodos `Next`, `Columns` y `Close`.
- Alternativas evaluadas: Ejecutar cada consulta construyendo slices intermedios en una unica funcion.
- Justificacion tecnica: Los operadores encadenados conservan la evaluacion perezosa. `Filter` solicita filas a `Scan` y `Project` solicita filas a `Filter`, sin conocer como se implementa el operador inferior.

## Hito 4 - ordenamiento y limite

- Hito y fecha: Hito 4, 2026-07-24.
- Decision tomada: Materializar las filas solo en el operador `Order` y aplicar `Limit` despues de ordenar.
- Alternativas evaluadas: Ordenar directamente dentro de `Scan` o limitar antes del ordenamiento.
- Justificacion tecnica: Ordenar requiere conocer todas las filas candidatas; aplicar primero `LIMIT` produciria resultados incorrectos cuando la consulta incluye `ORDER BY`.
- Comando de prueba: `go run ./cmd/sqlmem consultar empleados data/empleados.csv "SELECT nombre, salario FROM empleados ORDER BY salario DESC LIMIT 2"`.
- Extension: `GROUP BY` y los agregados `COUNT`, `SUM`, `AVG`, `MIN` y `MAX` materializan los grupos solo cuando es necesario.
- Comando de prueba: `go run ./cmd/sqlmem consultar empleados data/empleados.csv "SELECT activo, COUNT(*), AVG(salario) FROM empleados GROUP BY activo ORDER BY activo"`.

## Hito 5 - INNER JOIN

- Hito y fecha: Hito 5, 2026-07-24.
- Decision tomada: Implementar nested-loop join como referencia y usar hash join para condiciones de igualdad.
- Alternativas evaluadas: Usar solo nested-loop para todas las consultas.
- Justificacion tecnica: Nested-loop compara cada fila izquierda con cada fila derecha; hash join crea un indice de la tabla derecha y evita esas comparaciones repetidas cuando la condicion es `=`.
- Verificacion: las pruebas ejecutan un JOIN entre empleados y areas y comprueban las filas resultantes.
- Comando de prueba: `go run ./cmd/sqlmem consultar empleados=data/empleados.csv areas=data/areas.csv -- "SELECT empleados.nombre, areas.nombre FROM empleados INNER JOIN areas ON empleados.area_id = areas.id ORDER BY empleados.nombre"`.

## Reorganizacion por etapas

- Hito y fecha: Reorganizacion arquitectonica, 2026-07-24.
- Decision tomada: Repartir los tres paquetes internos en una etapa por paquete
  (`lexer`, `parser`, `storage`, `catalog`, `planner`, `executor`, `engine`,
  `cli`), de forma que el flujo `SQL -> lexer -> parser -> AST -> planner ->
  plan logico -> executor -> storage -> resultado` se lea en la estructura. El
  alcance SQL no cambia: no se anadio ni se quito ninguna clausula, operador ni
  funcion.
- Alternativas evaluadas:
  1. Mantener `internal/query` con lexer, parser y AST juntos. Se descarto
     porque el AST usaba `TokenKind`, un tipo del lexer, y eso obligaba al
     executor a depender del vocabulario lexico para comparar operadores y para
     construir los nombres de las columnas agregadas.
  2. Dejar `Build` como una unica funcion que decide el orden de los operadores
     y ademas los instancia. Se descarto porque no habia forma de comprobar la
     forma del plan sin ejecutar la consulta.
  3. Anadir un paquete `repl`. Se descarto: un interprete interactivo seria una
     caracteristica nueva, fuera del alcance de la reorganizacion.
- Justificacion tecnica: El AST define ahora sus propios `BinaryOp`,
  `LiteralKind` y `AggregateFunc`, y el parser traduce los tokens en un unico
  punto; su metodo `String()` conserva el texto exacto de antes porque alimenta
  los mensajes de error y los nombres de columna como `SUM(salario)`. El
  `planner` resuelve las tablas y produce un plan logico, y el `executor` se
  limita a instanciar operadores, con lo que deja de depender de `catalog`. La
  resolucion de nombres de columna se quedo en el executor, que es quien conoce
  los esquemas, para no duplicar `findColumn` y `qualify` en dos paquetes.
- Cambio de comportamiento aceptado: el CLI ahora termina con codigo de salida 1
  cuando falla, en vez de 0. La salida por pantalla es identica.
- Efecto colateral conocido: en una consulta con dos errores a la vez (por
  ejemplo, condicion de JOIN que no es de igualdad y `GROUP BY` incompleto) se
  reporta primero el del planner. Antes se reportaba primero el del join.
- Verificacion: se comparo la salida de trece invocaciones del CLI antes y
  despues de la reorganizacion; es identica byte a byte.

## Extensibilidad - expresiones en el SELECT y registro de funciones

- Hito y fecha: Extensibilidad del motor, 2026-07-24.
- Decision tomada: Sustituir las dos listas paralelas del AST (`Columns []string`
  y `Aggregates []Aggregate`) por una unica lista ordenada de expresiones con
  alias, convertir las llamadas a funcion en un nodo de expresion resuelto
  contra un registro, y pasar el `FROM` de un `string` a un arbol. Con esa base
  se anadieron `AS`, aritmetica en el SELECT, `HAVING`, `OFFSET` y N joins.
- Alternativas evaluadas:
  1. Anadir cada clausula sobre las estructuras existentes. Se descarto porque
     un elemento del SELECT era un `string`: no habia donde poner una expresion
     ni un alias, y el orden relativo entre columnas y agregados se perdia.
  2. Mantener las funciones como palabras clave del lexer con su enum en el AST.
     Se descarto porque cada funcion nueva costaba cambios en tres paquetes.
  3. Dejar el `FROM` como tabla mas lista de joins. Se descarto porque cierra la
     puerta a las subconsultas, que necesitan que una fuente sea recursiva.
- Justificacion tecnica: Con los agregados dentro de las expresiones, el planner
  los separa reescribiendo el SELECT, el HAVING y el ORDER BY: sustituye cada
  llamada y cada clave de grupo por una referencia a la columna que produce la
  agregacion. De esa reescritura sale gratis la validacion correcta de
  `GROUP BY`, que antes solo comparaba cantidades. `internal/functions` es un
  paquete propio porque el planner necesita distinguir los agregados y
  `executor` ya importa `planner`.
- Cambios de comportamiento aceptados:
  1. `SELECT nombre, COUNT(*) ... GROUP BY activo` pasa a dar error en vez de
     emitir en silencio la columna agrupada.
  2. Las columnas de salida siguen el orden del `SELECT` tambien con agregados.
  3. `ORDER BY` puede usar columnas que no se seleccionan, porque el orden se
     aplica antes de la proyeccion.
  4. `MIN` y `MAX` declaran el tipo de su argumento en vez de declarar decimal
     mientras devolvian el valor original.
- Verificacion: se comparo la salida de las diez invocaciones del CLI que no
  debian cambiar contra la linea base guardada; es identica byte a byte. Los
  cuatro cambios anteriores tienen prueba propia en `tests/integration_test.go`.
- Anadido posterior: el operador unario (`-saldo`, `-100`, `+x`) como nodo
  propio `UnaryExpression`, en vez de traducirlo a `0 - x`, para que el nombre
  por defecto de la columna sea `-saldo` y no `0 - saldo`. Liga mas fuerte que
  la multiplicacion y se puede encadenar. Anadir un nodo de expresion obligo a
  tocar los ocho recorridos de expresiones que hay repartidos entre `parser`,
  `planner` y `executor`; conviene tenerlo en cuenta antes de anadir el
  siguiente (por ejemplo `NOT`), porque un recorrido olvidado no rompe la
  compilacion, solo deja un agujero silencioso.

## Correcciones tras la auditoria tecnica

- Hito y fecha: Auditoria y correccion, 2026-07-25.
- Contexto: se audito el motor ejecutandolo (no solo leyendolo) contra los
  requisitos H1 a H5. El resultado fue solido, pero con cuatro huecos reales.
- Decisiones tomadas:
  1. **La estrategia de join la elige el planner.** `NestedLoopJoin` estaba
     escrito pero no lo invocaba nadie (cobertura 0.0%), asi que un join que no
     fuera de igualdad fallaba aunque el algoritmo capaz de resolverlo
     estuviera en el repositorio. Ahora `planner.Join` lleva un campo
     `Strategy` y el executor solo instancia lo que el plan le pide. La
     alternativa era que el executor probase hash y cayera a nested-loop si
     fallaba, y se descarto: el executor no debe tomar decisiones de plan.
  2. **REPL en `internal/cli`, no en `internal/repl`.** El plan de la auditoria
     proponia un paquete propio, pero `cli` ya era la capa de terminal y el
     REPL necesita el mismo `render`; separarlos habria creado un ciclo o
     habria obligado a exportar la presentacion desde un paquete mal llamado.
     La cohesion manda sobre el parecido con la estructura de referencia.
  3. **`EXPLAIN` devuelve filas, no un camino aparte.** El plan dibujado se
     entrega como un resultado normal de una columna a traves del operador
     `Values`, de modo que el CLI y el REPL lo imprimen con el mismo codigo que
     cualquier consulta. La alternativa, un metodo `Explain` en el motor y una
     rama en cada consumidor, duplicaba la presentacion.
  4. **Los joins cierran sus dos entradas.** `Close` solo cerraba la izquierda;
     la derecha se drenaba al construir y se olvidaba. Era inocuo porque
     `Scan.Close` no hace nada, pero era una fuga latente.
- Ademas: `DISTINCT`, punto y coma final opcional, posicion en los errores
  semanticos, seccion de limitaciones conocidas, y se elimino
  `storage.Table.ColumnIndex`, que no tenia ningun llamador y duplicaba
  parcialmente a `executor.findColumn`.
- Verificacion: `go build`, `go test`, `go vet` limpios; la cobertura de
  `internal/executor/join.go` paso de 0.0% a mas del 90%; la salida de las
  consultas que ya existian no cambio.

## Datos de ejemplo con volumen y modelo relacional

- Hito y fecha: Poblado de datos, 2026-07-25.
- Decision tomada: Sustituir los CSV de juguete (3 y 2 filas) por tres tablas
  relacionadas con volumen: `sedes` (6), `areas` (12) y `empleados` (300).
  `empleados` gana `area_id` y `sede_id`, y `areas` gana `sede_id` y
  `presupuesto`.
- Alternativas evaluadas:
  1. Anadir solo mas filas al esquema anterior. Se descarto porque el ejemplo
     de JOIN de toda la documentacion unia `empleados.id = areas.id`, que
     funcionaba por casualidad (los identificadores coincidian) pero no
     significa nada. Con `area_id` el join expresa una relacion real y permite
     encadenar una tercera tabla.
  2. Un generador en Go (`internal/dataset` mas `cmd/gendata`) que produjera
     los CSV y alimentara unos benchmarks. Se descarto: para las 300 filas del
     repositorio el fichero escrito directamente es mas simple y mas legible, y
     el generador solo se justificaba por unas pruebas de rendimiento que no
     forman parte del encargo. Los ficheros se produjeron con un comando de una
     sola vez; en el repositorio solo queda el CSV.
- Justificacion tecnica: los datos incluyen a proposito los casos que ejercitan
  el motor, en vez de ser filas planas: nulos en cuatro columnas (incluida la
  clave de union, que permite comprobar que un `NULL` no casa con nada),
  acentos y enes, una coma dentro de un campo entrecomillado, un presupuesto sin
  asignar y otro de diez cifras. Con 300 filas y una distribucion desigual de
  areas, los `GROUP BY` producen grupos de tamanos distintos y los `HAVING` con
  umbrales reales tienen sentido.
- Efecto colateral corregido: el decimal de diez cifras revelo que la salida
  usaba `%v` sobre `float64`, que pasa a notacion cientifica
  (`9.87654321099e+09`). Nunca se habia visto porque el valor mas grande era
  4100.75. Ahora `internal/cli/render.go` formatea los decimales en notacion
  posicional.
- Verificacion: `tests/data_test.go` carga los CSV del repositorio y fija su
  esquema, tipos inferidos y numero de filas, de modo que un cambio en los
  datos avisa antes de que los ejemplos de la documentacion dejen de funcionar.
