package executor

import "motor-consultas-sql/internal/storage"

// Distinct descarta las filas repetidas conservando la primera aparicion de
// cada una, de modo que respeta el orden que traiga la entrada.
type Distinct struct {
	input Operator
	seen  map[string]bool
}

func NewDistinct(input Operator) *Distinct {
	return &Distinct{input: input, seen: map[string]bool{}}
}

func (d *Distinct) Next() (storage.Row, error) {
	for {
		row, err := d.input.Next()
		if err != nil {
			return nil, err
		}
		key := rowKey(row)
		if d.seen[key] {
			continue
		}
		d.seen[key] = true
		return row, nil
	}
}

func (d *Distinct) Columns() []storage.Column { return d.input.Columns() }
func (d *Distinct) Close() error              { return d.input.Close() }
