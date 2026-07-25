package tests

import (
	"path/filepath"
	"testing"

	"motor-consultas-sql/internal/engine"
	"motor-consultas-sql/internal/storage"
)

// motorConDatosReales carga los CSV del repositorio, los mismos que usan los
// ejemplos de la documentacion.
func motorConDatosReales(t *testing.T) *engine.Engine {
	t.Helper()
	motor := engine.New()
	for _, nombre := range []string{"empleados", "areas", "sedes"} {
		path := filepath.Join("..", "data", nombre+".csv")
		if _, err := motor.LoadCSVFile(nombre, path); err != nil {
			t.Fatalf("LoadCSVFile(%q) devolvio error: %v", path, err)
		}
	}
	return motor
}

// TestDatosDelRepositorio fija el esquema y el tamano de los CSV de ejemplo. Si
// alguien cambia los datos, este test avisa antes de que los ejemplos del
// README dejen de funcionar.
func TestDatosDelRepositorio(t *testing.T) {
	motor := motorConDatosReales(t)

	tests := []struct {
		tabla    string
		filas    int
		columnas []string
		tipos    []storage.DataType
	}{
		{
			tabla:    "sedes",
			filas:    6,
			columnas: []string{"id", "ciudad", "pais"},
			tipos:    []storage.DataType{storage.Integer, storage.Text, storage.Text},
		},
		{
			tabla:    "areas",
			filas:    12,
			columnas: []string{"id", "nombre", "sede_id", "presupuesto"},
			tipos:    []storage.DataType{storage.Integer, storage.Text, storage.Integer, storage.Decimal},
		},
		{
			tabla:    "empleados",
			filas:    300,
			columnas: []string{"id", "nombre", "area_id", "sede_id", "edad", "salario", "activo", "fecha_ingreso"},
			tipos: []storage.DataType{
				storage.Integer, storage.Text, storage.Integer, storage.Integer,
				storage.Integer, storage.Decimal, storage.Boolean, storage.Text,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.tabla, func(t *testing.T) {
			table, ok := motor.Table(test.tabla)
			if !ok {
				t.Fatalf("la tabla %q no se cargo", test.tabla)
			}
			if len(table.Rows) != test.filas {
				t.Errorf("filas = %d; se esperaban %d", len(table.Rows), test.filas)
			}
			if len(table.Columns) != len(test.columnas) {
				t.Fatalf("columnas = %d; se esperaban %d", len(table.Columns), len(test.columnas))
			}
			for index, nombre := range test.columnas {
				if table.Columns[index].Name != nombre {
					t.Errorf("columna %d = %q; se esperaba %q", index, table.Columns[index].Name, nombre)
				}
				if table.Columns[index].Type != test.tipos[index] {
					t.Errorf("columna %q: tipo = %s; se esperaba %s",
						nombre, table.Columns[index].Type, test.tipos[index])
				}
			}
		})
	}
}

// TestDatosContienenCasosLimite comprueba que los CSV de ejemplo siguen
// ejercitando el motor: nulos, acentos, comas entrecomilladas y extremos.
func TestDatosContienenCasosLimite(t *testing.T) {
	motor := motorConDatosReales(t)

	t.Run("nulos en varias columnas", func(t *testing.T) {
		rows := consultar(t, motor,
			"SELECT COUNT(*), COUNT(area_id), COUNT(edad), COUNT(salario), COUNT(fecha_ingreso) FROM empleados")
		total := rows[0][0].Data.(int64)
		for index, nombre := range []string{"area_id", "edad", "salario", "fecha_ingreso"} {
			if rows[0][index+1].Data.(int64) >= total {
				t.Errorf("la columna %q no tiene ningun NULL", nombre)
			}
		}
	})

	t.Run("coma dentro de un campo entrecomillado", func(t *testing.T) {
		rows := consultar(t, motor, "SELECT ciudad FROM sedes WHERE pais = 'Colombia'")
		if len(rows) != 1 || rows[0][0].Data != "Bogotá, D.C." {
			t.Errorf("ciudad = %#v; se esperaba \"Bogotá, D.C.\"", rows[0][0].Data)
		}
	})

	t.Run("acentos y enes", func(t *testing.T) {
		rows := consultar(t, motor, "SELECT COUNT(*) FROM sedes WHERE pais = 'España'")
		if rows[0][0].Data != int64(1) {
			t.Errorf("no se encontro España: %#v", rows[0][0].Data)
		}
	})

	t.Run("presupuesto nulo y decimal grande", func(t *testing.T) {
		rows := consultar(t, motor, "SELECT nombre, presupuesto FROM areas WHERE id = 9")
		if !rows[0][1].Null {
			t.Errorf("el area Calidad deberia tener presupuesto NULL: %#v", rows[0][1])
		}
		rows = consultar(t, motor, "SELECT presupuesto FROM areas WHERE id = 12")
		if rows[0][0].Data != 9876543210.99 {
			t.Errorf("presupuesto = %#v; se esperaba 9876543210.99", rows[0][0].Data)
		}
	})
}

// TestConsultaDeTresTablasSobreDatosReales es la consulta que ilustra el
// modelo: empleados -> areas -> sedes.
func TestConsultaDeTresTablasSobreDatosReales(t *testing.T) {
	motor := motorConDatosReales(t)

	sql := "SELECT sedes.ciudad, COUNT(*) FROM empleados " +
		"INNER JOIN areas ON empleados.area_id = areas.id " +
		"INNER JOIN sedes ON areas.sede_id = sedes.id " +
		"GROUP BY sedes.ciudad ORDER BY sedes.ciudad"

	rows := consultar(t, motor, sql)
	if len(rows) == 0 {
		t.Fatal("el join de tres tablas no devolvio filas")
	}

	// Los empleados sin area no aparecen: su clave de union es NULL.
	var total int64
	for _, row := range rows {
		total += row[1].Data.(int64)
	}
	conArea := consultar(t, motor, "SELECT COUNT(area_id) FROM empleados")[0][0].Data.(int64)
	if total != conArea {
		t.Errorf("el join sumo %d empleados; hay %d con area asignada", total, conArea)
	}
}
