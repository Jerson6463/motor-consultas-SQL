package executor

import (
	"io"

	"motor-consultas-sql/internal/storage"
)

// Values entrega filas ya calculadas. Lo usa EXPLAIN para devolver el plan como
// un resultado normal, de modo que quien lo consuma no necesite un camino
// aparte para mostrarlo.
type Values struct {
	columns []storage.Column
	rows    []storage.Row
	index   int
}

func NewValues(columns []storage.Column, rows []storage.Row) *Values {
	return &Values{columns: columns, rows: rows}
}

func (v *Values) Next() (storage.Row, error) {
	if v.index >= len(v.rows) {
		return nil, io.EOF
	}
	row := v.rows[v.index]
	v.index++
	return row, nil
}

func (v *Values) Columns() []storage.Column { return v.columns }
func (v *Values) Close() error              { return nil }
