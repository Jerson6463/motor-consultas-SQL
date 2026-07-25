package executor

import (
	"fmt"
	"strings"

	"motor-consultas-sql/internal/storage"
)

// findColumn localiza una columna en un esquema. Acepta el nombre calificado
// tabla.columna y tambien el nombre corto, siempre que no sea ambiguo.
func findColumn(columns []storage.Column, name string) (int, storage.Column, error) {
	match := -1
	for index, column := range columns {
		if equalName(column.Name, name) {
			return index, column, nil
		}
		if !strings.Contains(name, ".") && strings.HasSuffix(strings.ToLower(column.Name), "."+strings.ToLower(name)) {
			if match >= 0 {
				return 0, storage.Column{}, fmt.Errorf("la columna %q es ambigua", name)
			}
			match = index
		}
	}
	if match >= 0 {
		return match, columns[match], nil
	}
	return 0, storage.Column{}, fmt.Errorf("la columna %q no existe", name)
}

func equalName(left, right string) bool {
	return strings.EqualFold(strings.TrimSpace(left), strings.TrimSpace(right))
}

// withPosition añade a un error la posicion del texto SQL que lo provoco, para
// que los errores de resolucion de columnas se citen igual que los del parser.
func withPosition(err error, position int) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w (posicion %d)", err, position)
}
