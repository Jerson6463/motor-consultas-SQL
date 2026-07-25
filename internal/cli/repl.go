package cli

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"motor-consultas-sql/internal/engine"
)

// repl abre una sesion interactiva: lee una consulta por linea, la ejecuta y
// vuelve a pedir otra. Un error se informa pero no termina la sesion.
func repl(arguments []string, stdin io.Reader, stdout, stderr io.Writer) int {
	motor, ok := loadSources(arguments, stderr)
	if !ok {
		return 1
	}

	fmt.Fprintln(stdout, "Motor de consultas SQL en memoria.")
	fmt.Fprintln(stdout, "Escriba .ayuda para ver los comandos y .salir para terminar.")

	scanner := bufio.NewScanner(stdin)
	for {
		fmt.Fprint(stdout, "sqlmem> ")
		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, ".") {
			if quit := runCommand(motor, line, stdout, stderr); quit {
				return 0
			}
			continue
		}

		// Un error de la consulta se informa y la sesion continua.
		if err := runLine(motor, line, stdout); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
		}
	}

	if err := scanner.Err(); err != nil {
		return reportError(stderr, err)
	}
	fmt.Fprintln(stdout)
	return 0
}

// runLine ejecuta una consulta y escribe su resultado.
func runLine(motor *engine.Engine, sql string, stdout io.Writer) error {
	result, err := motor.Query(sql)
	if err != nil {
		return err
	}
	defer result.Close()
	return render(stdout, result)
}

// runCommand atiende los comandos que empiezan por punto. Devuelve true cuando
// hay que terminar la sesion.
func runCommand(motor *engine.Engine, line string, stdout, stderr io.Writer) bool {
	fields := strings.Fields(line)
	switch fields[0] {
	case ".salir", ".exit":
		fmt.Fprintln(stdout, "Hasta luego.")
		return true
	case ".ayuda":
		printREPLHelp(stdout)
	case ".tablas":
		listTables(motor, stdout)
	case ".esquema":
		if len(fields) != 2 {
			fmt.Fprintln(stderr, "Uso: .esquema <tabla>")
			break
		}
		describeTable(motor, fields[1], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "Comando desconocido %q; escriba .ayuda\n", fields[0])
	}
	return false
}

func listTables(motor *engine.Engine, stdout io.Writer) {
	tables := motor.Tables()
	if len(tables) == 0 {
		fmt.Fprintln(stdout, "No hay tablas cargadas.")
		return
	}
	for _, table := range tables {
		fmt.Fprintf(stdout, "%s (%d filas, %d columnas)\n", table.Name, len(table.Rows), len(table.Columns))
	}
}

func describeTable(motor *engine.Engine, name string, stdout, stderr io.Writer) {
	table, ok := motor.Table(name)
	if !ok {
		fmt.Fprintf(stderr, "Error: la tabla %q no existe\n", name)
		return
	}
	fmt.Fprintf(stdout, "Tabla %q: %d filas\n", table.Name, len(table.Rows))
	for _, column := range table.Columns {
		fmt.Fprintf(stdout, "- %s: %s\n", column.Name, column.Type)
	}
}

func printREPLHelp(stdout io.Writer) {
	fmt.Fprintln(stdout, "Comandos:")
	fmt.Fprintln(stdout, "  .tablas           lista las tablas cargadas")
	fmt.Fprintln(stdout, "  .esquema <tabla>  muestra las columnas y sus tipos")
	fmt.Fprintln(stdout, "  .ayuda            muestra esta ayuda")
	fmt.Fprintln(stdout, "  .salir            termina la sesion")
	fmt.Fprintln(stdout, "Cualquier otra linea se ejecuta como SQL; el punto y coma final es opcional.")
}
