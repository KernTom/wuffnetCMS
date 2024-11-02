package models

type Table struct {
	Name    string   `json:"name"`
	Columns []Column `json:"columns"`
}

type Column struct {
	Name       string `json:"name"`
	DataType   string `json:"data_type"`
	IsNullable bool   `json:"is_nullable"`
}

type Schema struct {
	Name   string   `json:"schema"`
	Tables []string `json:"tables"`
}
