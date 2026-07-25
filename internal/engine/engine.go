// Package engine es la fachada del motor: encadena las etapas de una consulta
// sin contener logica propia de analisis, planificacion ni ejecucion.
package engine

import (
	"io"
	"sort"
	"strings"

	"motor-consultas-sql/internal/catalog"
	"motor-consultas-sql/internal/executor"
	"motor-consultas-sql/internal/parser"
	"motor-consultas-sql/internal/planner"
	"motor-consultas-sql/internal/storage"
)

// Engine agrupa las tablas disponibles y ejecuta consultas contra ellas.
type Engine struct {
	catalog *catalog.Catalog
}

// New crea un motor sin tablas cargadas.
func New() *Engine {
	return &Engine{catalog: catalog.New()}
}

// LoadCSV carga una tabla desde un CSV y la registra en el catalogo.
func (e *Engine) LoadCSV(name string, input io.Reader) (*storage.Table, error) {
	table, err := storage.LoadCSV(name, input)
	if err != nil {
		return nil, err
	}
	if err := e.catalog.Add(table); err != nil {
		return nil, err
	}
	return table, nil
}

// LoadCSVFile carga una tabla desde un archivo CSV y la registra en el catalogo.
func (e *Engine) LoadCSVFile(name, path string) (*storage.Table, error) {
	table, err := storage.LoadCSVFile(name, path)
	if err != nil {
		return nil, err
	}
	if err := e.catalog.Add(table); err != nil {
		return nil, err
	}
	return table, nil
}

// Query recorre las etapas del motor: SQL -> AST -> plan logico -> operadores.
// Las filas no se calculan aqui; se obtienen al recorrer el resultado.
//
// Con EXPLAIN el plan no se ejecuta: se devuelve dibujado como un resultado de
// una sola columna, para que quien lo consuma no necesite un camino aparte.
func (e *Engine) Query(sql string) (*Result, error) {
	statement, err := parser.Parse(sql)
	if err != nil {
		return nil, err
	}
	plan, err := planner.Plan(e.catalog, statement)
	if err != nil {
		return nil, err
	}
	if statement.Explain {
		return explainResult(plan), nil
	}
	operator, err := executor.Build(plan)
	if err != nil {
		return nil, err
	}
	return &Result{operator: operator}, nil
}

// explainResult convierte el dibujo del plan en filas de una columna.
func explainResult(plan planner.Node) *Result {
	columns := []storage.Column{{Name: "plan", Type: storage.Text}}
	rows := []storage.Row{}
	for _, line := range strings.Split(strings.TrimRight(planner.Explain(plan), "\n"), "\n") {
		rows = append(rows, storage.Row{{Type: storage.Text, Data: line}})
	}
	return &Result{operator: executor.NewValues(columns, rows)}
}

// Tables devuelve las tablas registradas, ordenadas por nombre.
func (e *Engine) Tables() []*storage.Table {
	tables := e.catalog.Tables()
	sort.Slice(tables, func(i, j int) bool { return tables[i].Name < tables[j].Name })
	return tables
}

// Table busca una tabla por nombre.
func (e *Engine) Table(name string) (*storage.Table, bool) {
	return e.catalog.Table(name)
}

// Result recorre las filas de una consulta de forma perezosa.
type Result struct {
	operator executor.Operator
}

// Columns devuelve el esquema del resultado.
func (r *Result) Columns() []storage.Column { return r.operator.Columns() }

// Next entrega la siguiente fila y devuelve io.EOF cuando no quedan mas.
func (r *Result) Next() (storage.Row, error) { return r.operator.Next() }

// Close libera los recursos de la consulta.
func (r *Result) Close() error { return r.operator.Close() }
