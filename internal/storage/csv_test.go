package storage

import (
	"strings"
	"testing"
)

func TestLoadCSV(t *testing.T) {
	table, err := LoadCSV("empleados", strings.NewReader("id,nombre,activo,salario\n1,Ana,true,2500.50\n2,Beto,false,\n"))
	if err != nil {
		t.Fatalf("LoadCSV devolvio error: %v", err)
	}

	if len(table.Rows) != 2 {
		t.Fatalf("filas = %d; se esperaban 2", len(table.Rows))
	}

	gotTypes := []DataType{table.Columns[0].Type, table.Columns[1].Type, table.Columns[2].Type, table.Columns[3].Type}
	wantTypes := []DataType{Integer, Text, Boolean, Decimal}
	for index := range wantTypes {
		if gotTypes[index] != wantTypes[index] {
			t.Errorf("columna %d: tipo = %s; se esperaba %s", index, gotTypes[index], wantTypes[index])
		}
	}

	if !table.Rows[1][3].Null {
		t.Error("el salario vacio debia cargarse como NULL")
	}
}

func TestLoadCSVRejectsInvalidHeaders(t *testing.T) {
	tests := []struct {
		name string
		csv  string
	}{
		{name: "columna vacia", csv: "id,\n1,Ana\n"},
		{name: "columna repetida", csv: "id,ID\n1,2\n"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := LoadCSV("prueba", strings.NewReader(test.csv))
			if err == nil {
				t.Fatal("LoadCSV no devolvio error")
			}
		})
	}
}

func TestLoadCSVInfersDecimalForMixedIntegersAndDecimals(t *testing.T) {
	table, err := LoadCSV("ventas", strings.NewReader("monto\n20\n10.50\n"))
	if err != nil {
		t.Fatalf("LoadCSV devolvio error: %v", err)
	}

	if table.Columns[0].Type != Decimal {
		t.Errorf("tipo = %s; se esperaba decimal", table.Columns[0].Type)
	}
}

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "Nombre", want: "nombre"},
		{input: "  EDAD  ", want: "edad"},
		{input: "empleados.Salario", want: "empleados.salario"},
	}
	for _, test := range tests {
		if got := NormalizeName(test.input); got != test.want {
			t.Errorf("NormalizeName(%q) = %q; se esperaba %q", test.input, got, test.want)
		}
	}
}
