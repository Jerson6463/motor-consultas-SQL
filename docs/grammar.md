# Gramatica soportada

```ebnf
query          = [ "EXPLAIN" ] ,
                 "SELECT" , [ "DISTINCT" ] , select_list ,
                 "FROM" , from_item ,
                 [ "WHERE" , expression ] ,
                 [ group_by ] ,
                 [ "HAVING" , expression ] ,
                 [ order_by ] ,
                 [ limit ] , [ offset ] , [ ";" ] , EOF ;

select_list    = select_item , { "," , select_item } ;
select_item    = "*" | expression , [ "AS" , identifier ] ;

from_item      = identifier , { inner_join } ;
inner_join     = "INNER" , "JOIN" , identifier , "ON" , expression ;

group_by       = "GROUP" , "BY" , expression , { "," , expression } ;
order_by       = "ORDER" , "BY" , order_term , { "," , order_term } ;
order_term     = expression , [ "ASC" | "DESC" ] ;
limit          = "LIMIT" , integer ;
offset         = "OFFSET" , integer ;

(* Precedencia, de menor a mayor *)
expression     = and_expression , { "OR" , and_expression } ;
and_expression = comparison , { "AND" , comparison } ;
comparison     = additive , [ comparison_operator , additive ] ;
additive       = multiplicative , { ( "+" | "-" ) , multiplicative } ;
multiplicative = unary , { ( "*" | "/" ) , unary } ;
unary          = ( "+" | "-" ) , unary | primary ;
primary        = "(" , expression , ")"
               | function_call
               | identifier
               | number | string | boolean | "NULL" ;

function_call  = identifier , "(" , ( "*" | expression , { "," , expression } | ) , ")" ;
comparison_operator = "=" | "<>" | "<" | ">" | "<=" | ">=" ;
```

## Notas

- Las palabras clave no distinguen mayusculas de minusculas. Los textos se
  escriben entre comillas simples y una comilla simple interna se escapa
  duplicandola, por ejemplo: `'O''Brien'`.
- **El asterisco tiene tres significados** que distingue el contexto: comodin al
  inicio de un elemento de la lista de seleccion (cuando le sigue una coma o
  `FROM`), argumento de `COUNT(*)`, y operador de multiplicacion en cualquier
  otra posicion.
- **El parser no conoce los nombres de las funciones.** Cualquier identificador
  seguido de `(` se analiza como llamada; que la funcion exista lo decide el
  registro de `internal/functions`, y el planner lo comprueba.
- El signo unario (`-saldo`, `-100`, `+x`) liga mas fuerte que la
  multiplicacion, de modo que `-a * b` es `(-a) * b`, y se puede encadenar
  (`- -a`). Exige un operando numerico y propaga `NULL`.
- Se admite **un** punto y coma final, opcional.
- `EXPLAIN` no ejecuta la consulta: devuelve el plan dibujado como un resultado
  de una sola columna llamada `plan`.

## Semantica de la lista de seleccion

Las columnas de salida siguen el orden del `SELECT`, tambien en consultas
agregadas. Una columna sin `AS` toma como nombre la forma canonica de su
expresion: `SUM(salario)`, `salario * 12`, `a + (b * c)`.

Cuando la consulta agrupa, toda columna del `SELECT`, del `HAVING` y del
`ORDER BY` debe ser una clave de `GROUP BY` o el argumento de una funcion de
agregacion; en caso contrario la consulta se rechaza.

`ORDER BY` se evalua antes de la proyeccion, asi que puede ordenar por columnas
que no aparecen en el `SELECT`, y ademas admite citar un alias definido en el.

## Estrategias de JOIN

El planner elige la estrategia mirando la condicion:

- `HashJoin` cuando la condicion es una igualdad entre dos columnas. Indexa la
  tabla derecha por su clave de union.
- `NestedLoopJoin` en cualquier otro caso (`>`, `<`, `<>`, comparaciones contra
  un literal, condiciones compuestas). Compara todos los pares de filas.

Ambas producen el mismo resultado para una igualdad; hay una prueba que lo
comprueba. `EXPLAIN` indica cual se ha elegido.

Los `INNER JOIN` encadenados se pliegan hacia la izquierda, de modo que
`a JOIN b JOIN c` se planifica como `(a JOIN b) JOIN c`. Cuando hay algun join,
cada tabla califica sus columnas como `tabla.columna` una sola vez.

Las claves `NULL` no casan, ni siquiera consigo mismas, y las claves repetidas
producen todas las combinaciones del grupo.

## Limitaciones conocidas

Lo que el motor **no** soporta hoy:

- **Subconsultas.** El `FROM` ya es un arbol, pero falta el nodo que las
  represente y la llamada recursiva en el planner.
- **`LEFT`, `RIGHT` y `FULL JOIN`.** Solo hay `INNER JOIN`.
- **Alias de tabla** (`FROM empleados e`). El plan ya sabe calificar columnas
  con un alias, pero el parser no lo lee.
- **`IS NULL` / `IS NOT NULL`.** Para buscar nulos no hay operador; `x = NULL`
  devuelve `NULL` y por tanto no selecciona ninguna fila, que es lo correcto en
  SQL pero deja el caso sin forma de expresarse.
- **Funciones escalares** (`UPPER`, `ROUND`, `COALESCE`...). El registro de
  `internal/functions` solo contiene agregados.
- **`IN`, `BETWEEN`, `LIKE`, `CASE`.**
- **Operador unario `NOT`.**
- **`INSERT`, `UPDATE`, `DELETE`, `CREATE`.** El motor es de solo lectura.
- **Varias consultas en una sola cadena.** Se admite un punto y coma final, pero
  no separar dos consultas con el; para encadenarlas esta el REPL.
- **Indices y optimizacion basada en costes.** Todo `SELECT` recorre la tabla
  entera; el planner elige la estrategia de join por la forma de la condicion,
  no por el tamano de los datos.
- **Concurrencia.** El motor es de un solo hilo; el catalogo no es seguro para
  usarse desde varias goroutines.
- **Ordenacion de texto por bytes.** `ORDER BY` sobre una columna de texto
  compara byte a byte en UTF-8, no con las reglas del idioma. Por eso los
  nombres acentuados quedan despues de los que no lo estan: `Ángela` va detras
  de `Zoe`, porque `Á` ocupa dos bytes que empiezan por `0xC3`.
- **Sin tipo fecha.** `fecha_ingreso` se carga como texto. Las fechas en formato
  ISO (`2024-03-15`) se ordenan y comparan correctamente porque el orden
  alfabetico coincide con el cronologico, pero no hay aritmetica de fechas.
- **Tamano de los datos.** Todo el CSV se carga en memoria; no hay volcado a
  disco ni procesamiento por bloques.

Nota para quien extienda el motor: anadir un **nodo de expresion** nuevo obliga
a tratarlo en los ocho recorridos de expresiones repartidos entre `parser`,
`planner` y `executor`. Olvidar uno no rompe la compilacion, porque el `default`
del `switch` lo absorbe; deja un agujero silencioso. Conviene extraer antes un
recorrido generico.

## Ejemplos

```bash
go run ./cmd/sqlmem consultar empleados data/empleados.csv "SELECT nombre, salario * 12 AS anual FROM empleados WHERE activo = true ORDER BY anual DESC LIMIT 5"

go run ./cmd/sqlmem consultar empleados data/empleados.csv "SELECT area_id, COUNT(*), AVG(salario) FROM empleados GROUP BY area_id HAVING COUNT(*) > 30 ORDER BY COUNT(*) DESC"

go run ./cmd/sqlmem consultar empleados data/empleados.csv "SELECT nombre FROM empleados ORDER BY salario DESC LIMIT 5 OFFSET 10"

go run ./cmd/sqlmem consultar empleados=data/empleados.csv areas=data/areas.csv sedes=data/sedes.csv -- "SELECT sedes.ciudad, areas.nombre, COUNT(*) FROM empleados INNER JOIN areas ON empleados.area_id = areas.id INNER JOIN sedes ON areas.sede_id = sedes.id GROUP BY sedes.ciudad, areas.nombre ORDER BY COUNT(*) DESC LIMIT 10"
```
