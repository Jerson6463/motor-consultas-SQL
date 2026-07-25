package executor

import (
	"fmt"
	"io"

	"motor-consultas-sql/internal/parser"
	"motor-consultas-sql/internal/storage"
)

// HashJoin indexa las filas derechas para joins de igualdad. Las columnas ya
// vienen calificadas desde el Scan, asi que aqui solo se concatenan los
// esquemas: eso permite encadenar varios joins sin acumular prefijos.
type HashJoin struct {
	left       Operator
	right      Operator
	rows       map[string][]storage.Row
	leftIndex  int
	rightIndex int
	current    storage.Row
	matches    []storage.Row
	columns    []storage.Column
}

func NewHashJoin(left, right Operator, condition parser.Expression) (*HashJoin, error) {
	expression, ok := condition.(parser.BinaryExpression)
	if !ok || expression.Operator != parser.OpEqual {
		return nil, fmt.Errorf("hash join requiere una condicion de igualdad")
	}
	leftIdentifier, ok := expression.Left.(parser.Identifier)
	if !ok {
		return nil, fmt.Errorf("hash join requiere columnas")
	}
	rightIdentifier, ok := expression.Right.(parser.Identifier)
	if !ok {
		return nil, fmt.Errorf("hash join requiere columnas")
	}

	leftColumns := left.Columns()
	rightColumns := right.Columns()
	leftIndex, _, err := findColumn(leftColumns, leftIdentifier.Name)
	if err != nil {
		return nil, err
	}
	rightIndex, _, err := findColumn(rightColumns, rightIdentifier.Name)
	if err != nil {
		return nil, err
	}

	index := map[string][]storage.Row{}
	for {
		row, err := right.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if !row[rightIndex].Null {
			key := hashKey(row[rightIndex])
			index[key] = append(index[key], row)
		}
	}

	columns := make([]storage.Column, 0, len(leftColumns)+len(rightColumns))
	columns = append(columns, leftColumns...)
	columns = append(columns, rightColumns...)
	return &HashJoin{left: left, right: right, rows: index, leftIndex: leftIndex, columns: columns}, nil
}

func (j *HashJoin) Next() (storage.Row, error) {
	for {
		if j.current == nil {
			row, err := j.left.Next()
			if err != nil {
				return nil, err
			}
			j.current = row
			j.rightIndex = 0
			j.matches = nil
			if !row[j.leftIndex].Null {
				j.matches = j.rows[hashKey(row[j.leftIndex])]
			}
		}
		if j.rightIndex < len(j.matches) {
			right := j.matches[j.rightIndex]
			j.rightIndex++
			return append(append(storage.Row{}, j.current...), right...), nil
		}
		j.current = nil
	}
}

func (j *HashJoin) Columns() []storage.Column { return j.columns }

// Close cierra las dos entradas. La derecha ya se drenó al construir el indice,
// pero puede tener recursos que liberar igualmente.
func (j *HashJoin) Close() error { return closeBoth(j.left, j.right) }

func hashKey(value storage.Value) string {
	return fmt.Sprintf("%d:%v", value.Type, value.Data)
}
