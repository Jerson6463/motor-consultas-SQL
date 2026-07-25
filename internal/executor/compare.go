package executor

import (
	"fmt"
	"strings"

	"motor-consultas-sql/internal/storage"
)

// rowKey construye una clave de igualdad para una fila. Incluye el tipo y la
// marca de nulo, de modo que dos valores solo coinciden si son el mismo valor
// del mismo tipo. La comparten la agrupacion y la eliminacion de duplicados.
func rowKey(row storage.Row) string {
	parts := make([]string, len(row))
	for index, value := range row {
		parts[index] = valueKey(value)
	}
	return strings.Join(parts, "|")
}

func valueKey(value storage.Value) string {
	return fmt.Sprintf("%d:%v:%t", value.Type, value.Data, value.Null)
}

// compareForOrder aplica las reglas de ORDER BY: los NULL van al final en ASC.
func compareForOrder(left, right storage.Value) (int, error) {
	if left.Null && right.Null {
		return 0, nil
	}
	if left.Null {
		return 1, nil
	}
	if right.Null {
		return -1, nil
	}
	comparison, err := storage.Compare(left, right)
	if err != nil {
		return 0, fmt.Errorf("ordenar: %w", err)
	}
	return comparison, nil
}
