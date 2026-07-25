package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sesion arranca el REPL con una tabla cargada y las lineas indicadas.
func sesion(t *testing.T, lineas ...string) (int, string, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "empleados.csv")
	contenido := "nombre,edad,activo\nAna,30,true\nBeto,15,true\nCarla,40,false\n"
	if err := os.WriteFile(path, []byte(contenido), 0o600); err != nil {
		t.Fatalf("no se pudo escribir el CSV: %v", err)
	}
	return ejecutarCon(strings.Join(lineas, "\n")+"\n", "repl", "empleados="+path)
}

// TestReplEjecutaVariasConsultas es el requisito central del REPL.
func TestReplEjecutaVariasConsultas(t *testing.T) {
	code, stdout, stderr := sesion(t,
		"SELECT nombre FROM empleados WHERE edad >= 18",
		"SELECT COUNT(*) FROM empleados",
		".salir",
	)

	if code != 0 {
		t.Fatalf("codigo = %d; stderr = %q", code, stderr)
	}
	for _, want := range []string{"Ana", "Carla", "COUNT(*)", "3", "Hasta luego."} {
		if !strings.Contains(stdout, want) {
			t.Errorf("la salida no incluye %q:\n%s", want, stdout)
		}
	}
}

// TestReplNoSeCierraConUnError es el otro requisito central: un error informa y
// la sesion sigue viva.
func TestReplNoSeCierraConUnError(t *testing.T) {
	code, stdout, stderr := sesion(t,
		"SELECT nombre FROM empleados WHERE edad >= 18",
		"SELECT nombre empleados",
		"SELECT sueldo FROM empleados",
		"SELECT COUNT(*) FROM empleados",
		".salir",
	)

	if code != 0 {
		t.Fatalf("codigo = %d; se esperaba 0 pese a los errores", code)
	}
	if !strings.Contains(stderr, "posicion") {
		t.Errorf("no se informo del error de sintaxis: %q", stderr)
	}
	if !strings.Contains(stderr, "no existe") {
		t.Errorf("no se informo del error de columna: %q", stderr)
	}
	// La consulta posterior a los dos errores debe haberse ejecutado.
	if !strings.Contains(stdout, "COUNT(*)") {
		t.Errorf("la sesion no continuo tras los errores:\n%s", stdout)
	}
}

func TestReplAceptaPuntoYComa(t *testing.T) {
	_, stdout, stderr := sesion(t, "SELECT nombre FROM empleados LIMIT 1;", ".salir")
	if strings.Contains(stderr, "Error") {
		t.Fatalf("el punto y coma produjo un error: %q", stderr)
	}
	if !strings.Contains(stdout, "Ana") {
		t.Errorf("no se ejecuto la consulta:\n%s", stdout)
	}
}

func TestReplComandoSalir(t *testing.T) {
	for _, comando := range []string{".salir", ".exit"} {
		t.Run(comando, func(t *testing.T) {
			code, stdout, _ := sesion(t, comando, "SELECT COUNT(*) FROM empleados")
			if code != 0 {
				t.Errorf("codigo = %d", code)
			}
			if !strings.Contains(stdout, "Hasta luego.") {
				t.Errorf("no se despidio: %q", stdout)
			}
			// Lo que va despues de .salir no debe ejecutarse.
			if strings.Contains(stdout, "COUNT(*)") {
				t.Errorf("se ejecuto una consulta despues de salir:\n%s", stdout)
			}
		})
	}
}

func TestReplComandoTablas(t *testing.T) {
	_, stdout, _ := sesion(t, ".tablas", ".salir")
	if !strings.Contains(stdout, "empleados (3 filas, 3 columnas)") {
		t.Errorf("la lista de tablas es incorrecta:\n%s", stdout)
	}
}

func TestReplComandoEsquema(t *testing.T) {
	_, stdout, stderr := sesion(t, ".esquema empleados", ".esquema otra", ".esquema", ".salir")

	for _, want := range []string{"- nombre: texto", "- edad: entero", "- activo: booleano"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("el esquema no incluye %q:\n%s", want, stdout)
		}
	}
	if !strings.Contains(stderr, `la tabla "otra" no existe`) {
		t.Errorf("no se informo de la tabla inexistente: %q", stderr)
	}
	if !strings.Contains(stderr, "Uso: .esquema <tabla>") {
		t.Errorf("no se informo del uso: %q", stderr)
	}
}

func TestReplComandoAyudaYDesconocido(t *testing.T) {
	_, stdout, stderr := sesion(t, ".ayuda", ".loquesea", ".salir")
	if !strings.Contains(stdout, ".tablas") || !strings.Contains(stdout, ".salir") {
		t.Errorf("la ayuda no lista los comandos:\n%s", stdout)
	}
	if !strings.Contains(stderr, "Comando desconocido") {
		t.Errorf("no se informo del comando desconocido: %q", stderr)
	}
}

func TestReplIgnoraLineasVacias(t *testing.T) {
	_, stdout, stderr := sesion(t, "", "   ", "SELECT COUNT(*) FROM empleados", ".salir")
	if strings.Contains(stderr, "Error") {
		t.Errorf("una linea vacia produjo un error: %q", stderr)
	}
	if !strings.Contains(stdout, "COUNT(*)") {
		t.Errorf("no se ejecuto la consulta:\n%s", stdout)
	}
}

// TestReplTerminaSinSalir cubre el fin de la entrada (Ctrl+D o tuberia).
func TestReplTerminaSinSalir(t *testing.T) {
	code, stdout, _ := sesion(t, "SELECT COUNT(*) FROM empleados")
	if code != 0 {
		t.Errorf("codigo = %d; se esperaba 0 al agotarse la entrada", code)
	}
	if !strings.Contains(stdout, "COUNT(*)") {
		t.Errorf("no se ejecuto la consulta:\n%s", stdout)
	}
}

func TestReplFuenteInvalida(t *testing.T) {
	code, _, stderr := ejecutarCon(".salir\n", "repl", "basura")
	if code == 0 {
		t.Error("se esperaba un codigo distinto de cero")
	}
	if !strings.Contains(stderr, "fuente invalida") {
		t.Errorf("stderr = %q", stderr)
	}
}

func TestReplSinTablas(t *testing.T) {
	code, stdout, _ := ejecutarCon(".tablas\n.salir\n", "repl")
	if code != 0 {
		t.Errorf("codigo = %d", code)
	}
	if !strings.Contains(stdout, "No hay tablas cargadas.") {
		t.Errorf("stdout = %q", stdout)
	}
}
