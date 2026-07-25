package storage

// Table es una tabla cargada por completo en memoria.
type Table struct {
	Name    string
	Columns []Column
	Rows    []Row
}
