package cli

import (
	"fmt"
	"io"
	"strconv"
	"text/tabwriter"

	"motor-consultas-sql/internal/engine"
	"motor-consultas-sql/internal/storage"
)

// render recorre el resultado e imprime las filas en columnas alineadas.
// Se consume de forma perezosa: nunca se materializa el resultado completo.
func render(output io.Writer, result *engine.Result) error {
	writer := tabwriter.NewWriter(output, 0, 4, 2, ' ', 0)
	for _, column := range result.Columns() {
		fmt.Fprintf(writer, "%s\t", column.Name)
	}
	fmt.Fprintln(writer)

	for {
		row, err := result.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		for _, value := range row {
			fmt.Fprintf(writer, "%s\t", formatValue(value))
		}
		fmt.Fprintln(writer)
	}
	return writer.Flush()
}

// formatValue convierte un valor en el texto que se muestra. Los decimales se
// escriben siempre en notacion posicional: con %v, Go pasa a notacion
// cientifica en cuanto el numero es grande o muy pequeno, y un presupuesto no
// se lee bien como 9.87654321099e+09.
func formatValue(value storage.Value) string {
	if value.Null {
		return "NULL"
	}
	if decimal, ok := value.Data.(float64); ok {
		return strconv.FormatFloat(decimal, 'f', -1, 64)
	}
	return fmt.Sprintf("%v", value.Data)
}
