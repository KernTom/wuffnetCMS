package controllers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
)

type Option struct {
	Value interface{} `json:"value"`
	Label string      `json:"label"`
}

type ColumnInfo struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Options  []Option `json:"options,omitempty"` // Optional für Foreign Keys
	Readonly bool     `json:"readonly"`
}

func GetTables(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT 
			t.table_schema, 
			t.table_name,
			kcu.column_name AS primary_key_column
		FROM information_schema.tables AS t
		LEFT JOIN information_schema.table_constraints AS tc 
			ON t.table_schema = tc.table_schema 
			AND t.table_name = tc.table_name 
			AND tc.constraint_type = 'PRIMARY KEY'
		LEFT JOIN information_schema.key_column_usage AS kcu 
			ON tc.constraint_name = kcu.constraint_name 
			AND tc.table_schema = kcu.table_schema 
			AND tc.table_name = kcu.table_name
		WHERE t.table_type = 'BASE TABLE' 
			AND t.table_schema NOT IN ('pg_catalog', 'information_schema')
		ORDER BY t.table_schema, t.table_name;
	`

	rows, err := db.Query(query)
	if err != nil {
		http.Error(w, "Error fetching tables", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	schemaTablesMap := make(map[string][]map[string]interface{})
	for rows.Next() {
		var schema, tableName string
		var primaryKeyColumn sql.NullString

		if err := rows.Scan(&schema, &tableName, &primaryKeyColumn); err != nil {
			http.Error(w, "Error scanning tables", http.StatusInternalServerError)
			return
		}

		// Bereite die Primärschlüsselspalte für JSON vor
		var primaryKey string
		if primaryKeyColumn.Valid {
			primaryKey = primaryKeyColumn.String
		}

		// Struktur für Tabellen- und Primary-Key-Info anlegen
		tableInfo := map[string]interface{}{
			"tableName":        tableName,
			"primaryKeyColumn": primaryKey,
		}

		// Füge Tabelle zur Schema-Gruppe hinzu
		schemaTablesMap[schema] = append(schemaTablesMap[schema], tableInfo)
	}

	// Konvertiere das Mapping in das gewünschte JSON-Format
	var result []map[string]interface{}
	for schema, tables := range schemaTablesMap {
		result = append(result, map[string]interface{}{
			"schema": schema,
			"tables": tables,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func GetTableContent(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	schema := r.URL.Query().Get("schema")
	table := r.URL.Query().Get("table")
	search := r.URL.Query().Get("filter")
	sortBy := r.URL.Query().Get("sort_by")
	order := r.URL.Query().Get("order")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	// Standardlimit und Offset setzen
	if limitStr == "" {
		limitStr = "100"
	}
	if offsetStr == "" {
		offsetStr = "0"
	}

	// Konvertiere limit und offset in Ganzzahlen
	limitInt, err := strconv.Atoi(limitStr)
	if err != nil {
		http.Error(w, "Invalid limit parameter", http.StatusBadRequest)
		return
	}
	offsetInt, err := strconv.Atoi(offsetStr)
	if err != nil {
		http.Error(w, "Invalid offset parameter", http.StatusBadRequest)
		return
	}

	// Fehlerprüfung auf fehlende Werte
	if schema == "" || table == "" {
		http.Error(w, "Schema or table name missing", http.StatusBadRequest)
		return
	}

	// Grundlegende SQL-Queries für Abfrage und Zählen
	baseQuery := fmt.Sprintf("FROM %s.%s", pq.QuoteIdentifier(schema), pq.QuoteIdentifier(table))
	query := "SELECT * " + baseQuery
	countQuery := "SELECT COUNT(*) " + baseQuery
	args := []interface{}{}
	countArgs := []interface{}{}
	filterClause := ""

	// Filter hinzufügen, wenn Suchparameter vorhanden sind
	if search != "" {
		colQuery := fmt.Sprintf("SELECT column_name FROM information_schema.columns WHERE table_schema=$1 AND table_name=$2")
		colRows, err := db.Query(colQuery, schema, table)
		if err != nil {
			http.Error(w, "Error fetching columns for filter", http.StatusInternalServerError)
			return
		}
		defer colRows.Close()

		orConditions := []string{}
		for colRows.Next() {
			var colName string
			if err := colRows.Scan(&colName); err != nil {
				http.Error(w, "Error scanning columns", http.StatusInternalServerError)
				return
			}
			orConditions = append(orConditions, fmt.Sprintf("CAST(%s AS TEXT) ILIKE $%d", pq.QuoteIdentifier(colName), len(args)+1))
			args = append(args, "%"+search+"%")
			countArgs = append(countArgs, "%"+search+"%")
		}
		if len(orConditions) > 0 {
			filterClause = " WHERE " + strings.Join(orConditions, " OR ")
		}
		query += filterClause
		countQuery += filterClause
	}

	// Sortieroption hinzufügen, falls vorhanden
	if sortBy != "" {
		if order != "asc" && order != "desc" {
			order = "asc"
		}
		query += fmt.Sprintf(" ORDER BY %s %s", pq.QuoteIdentifier(sortBy), order)
	}

	// Limit und Offset hinzufügen nur für query, nicht für countQuery
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, limitInt, offsetInt)

	// Gesamtanzahl der gefilterten Datensätze abfragen
	var totalCount int
	err = db.QueryRow(countQuery, countArgs...).Scan(&totalCount)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error counting rows: %v", err), http.StatusInternalServerError)
		return
	}

	// Query ausführen und Fehler protokollieren
	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch table content: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Spaltennamen und -typen abrufen
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching column types: %v", err), http.StatusInternalServerError)
		return
	}

	// Ergebnisse sammeln und in JSON-Format umwandeln
	content := []map[string]interface{}{}
	for rows.Next() {
		columnValues := make([]interface{}, len(columnTypes))
		columnPointers := make([]interface{}, len(columnTypes))
		for i := range columnValues {
			columnPointers[i] = &columnValues[i]
		}

		if err := rows.Scan(columnPointers...); err != nil {
			http.Error(w, fmt.Sprintf("Error scanning row: %v", err), http.StatusInternalServerError)
			return
		}

		rowMap := map[string]interface{}{}
		for i, colType := range columnTypes {
			colName := colType.Name()
			switch colType.DatabaseTypeName() {
			case "NUMERIC", "DECIMAL":
				if columnValues[i] != nil {
					switch v := columnValues[i].(type) {
					case string:
						val, err := strconv.ParseFloat(v, 64)
						if err == nil {
							rowMap[colName] = val
						} else {
							rowMap[colName] = nil
						}
					case []uint8:
						// Falls als Byte-Slice zurückgegeben, konvertiere zu String und dann zu float64
						strVal := string(v)
						val, err := strconv.ParseFloat(strVal, 64)
						if err == nil {
							rowMap[colName] = val
						} else {
							rowMap[colName] = nil
						}
					case float64:
						// Falls bereits in float64, direkt zuweisen
						rowMap[colName] = v
					default:
						rowMap[colName] = nil
					}
				} else {
					rowMap[colName] = nil
				}
			case "TIMESTAMP", "TIMESTAMPTZ":
				if columnValues[i] != nil {
					if timeVal, ok := columnValues[i].(time.Time); ok {
						rowMap[colName] = timeVal.Format(time.RFC3339)
					} else {
						rowMap[colName] = fmt.Sprintf("%v", columnValues[i])
					}
				} else {
					rowMap[colName] = nil
				}
			case "TIME":
				if columnValues[i] != nil {
					// Formatieren für `HH:mm` ohne Sekunden
					if timeVal, ok := columnValues[i].(time.Time); ok {
						rowMap[colName] = timeVal.Format("15:04")
					} else {
						rowMap[colName] = fmt.Sprintf("%v", columnValues[i])
					}
				} else {
					rowMap[colName] = nil
				}
			default:
				// Keine zusätzliche Modifikation für Text, HTML und andere Typen
				rowMap[colName] = columnValues[i]
			}
		}
		content = append(content, rowMap)
	}

	// Paging-Informationen hinzufügen
	response := map[string]interface{}{
		"data":        content,
		"totalCount":  totalCount,
		"hasNextPage": totalCount > (offsetInt + limitInt),
	}

	// JSON-Daten zurücksenden
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error encoding JSON response: %v", err), http.StatusInternalServerError)
	}
}

func GetTableFields(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	schema := r.URL.Query().Get("schema")
	table := r.URL.Query().Get("table")

	// Fehlerprüfung für fehlende Parameter
	if schema == "" || table == "" {
		http.Error(w, "Schema or table name missing", http.StatusBadRequest)
		return
	}

	// SQL-Query zum Abrufen der Spaltennamen und Datentypen
	query := `
		SELECT 
			col.column_name, 
			col.data_type,
			referenced_table_info.table_schema AS referenced_schema,
			referenced_table_info.table_name AS referenced_table,
			referenced_table_info.column_name AS referenced_column,
			pk.constraint_type is not null as readonly
		FROM 
			information_schema.columns AS col
		LEFT JOIN 
			information_schema.key_column_usage AS kcu ON 
				col.table_schema = kcu.table_schema AND 
				col.table_name = kcu.table_name AND 
				col.column_name = kcu.column_name
		LEFT JOIN 
				information_schema.table_constraints AS tc ON 
				kcu.constraint_name = tc.constraint_name AND 
				kcu.table_schema = tc.table_schema AND
				tc.constraint_type = 'FOREIGN KEY'
		LEFT JOIN 
				information_schema.table_constraints AS pk ON 
				kcu.constraint_name = pk.constraint_name AND 
				kcu.table_schema = pk.table_schema AND
				pk.constraint_type = 'PRIMARY KEY'	
		LEFT JOIN 
			information_schema.constraint_column_usage AS referenced_table_info ON 
				tc.constraint_name = referenced_table_info.constraint_name AND
				tc.table_schema = referenced_table_info.table_schema
		WHERE 
            col.table_schema = $1 AND 
            col.table_name = $2;
	`

	rows, err := db.Query(query, schema, table)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error querying columns: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var columns []ColumnInfo

	// Ergebnisse iterieren und in die Struktur einfügen
	for rows.Next() {
		var col ColumnInfo
		var referencedSchema, referencedTable, referencedColumn sql.NullString
		if err := rows.Scan(&col.Name, &col.Type, &referencedSchema, &referencedTable, &referencedColumn, &col.Readonly); err != nil {
			http.Error(w, fmt.Sprintf("Error scanning column data: %v", err), http.StatusInternalServerError)
			return
		}
		// Typanpassung für PostgreSQL-Datentypen zu allgemeinen Typen
		col.Type = normalizeDataType(col.Type)

		// Wenn Foreign Key, dann Optionen abfragen und als `select` setzen
		if referencedSchema.Valid && referencedTable.Valid && referencedColumn.Valid {
			col.Type = "select" // Setze auf Dropdown
			optionsQuery := fmt.Sprintf(`
				SELECT %s AS value, CONCAT_WS(', ', %s) AS label 
				FROM %s.%s`,
				referencedColumn.String, // Die ID-Spalte (Primary Key) als `value`
				getConcatenatedTextColumns(db, referencedSchema.String, referencedTable.String, referencedColumn.String), referencedSchema.String,
				referencedTable.String,
			)

			optionsRows, err := db.Query(optionsQuery)
			if err != nil {
				log.Printf("Error querying options for foreign key column %s: %v", col.Name, err)
				continue
			}
			defer optionsRows.Close()

			var options []Option
			for optionsRows.Next() {
				var option Option
				if err := optionsRows.Scan(&option.Value, &option.Label); err != nil {
					log.Printf("Error scanning foreign key options for column %s: %v", col.Name, err)
					continue
				}
				options = append(options, option)
			}
			col.Options = options
		}

		columns = append(columns, col)
	}

	// Antwort im JSON-Format senden
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(columns); err != nil {
		http.Error(w, fmt.Sprintf("Error encoding JSON response: %v", err), http.StatusInternalServerError)
	}
}

// GetColumnTypes ruft die Spaltentypen der Tabelle ab
func GetColumnTypes(db *sql.DB, schema, table string) (map[string]string, error) {
	query := "SELECT column_name, data_type FROM information_schema.columns WHERE table_schema = $1 AND table_name = $2"
	rows, err := db.Query(query, schema, table)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch column types: %v", err)
	}
	defer rows.Close()

	columnTypes := make(map[string]string)
	for rows.Next() {
		var columnName, dataType string
		if err := rows.Scan(&columnName, &dataType); err != nil {
			return nil, err
		}
		columnTypes[columnName] = dataType
	}
	return columnTypes, nil
}

// Hilfsfunktion zur Normalisierung von PostgreSQL-Datentypen
func normalizeDataType(dataType string) string {
	switch strings.ToLower(dataType) {
	case "integer", "bigint", "smallint":
		return "integer"
	case "real", "double precision", "numeric":
		return "float"
	case "boolean":
		return "boolean"
	case "text", "character varying", "character":
		return "text"
	case "date":
		return "date"
	case "timestamp without time zone":
		return "timestamp"
	case "time without time zone":
		return "time"
	default:
		return "text" // Fallback auf text
	}
}
func getConcatenatedTextColumns(db *sql.DB, schema, table, fallbackColumn string) string {
	var columns []string
	query := `
        SELECT column_name 
        FROM information_schema.columns 
        WHERE table_schema = $1 
          AND table_name = $2 
          AND data_type IN ('character varying', 'text', 'char')
    `

	rows, err := db.Query(query, schema, table)
	if err != nil {
		log.Printf("Fehler beim Abrufen der Textspalten: %v", err)
		return fmt.Sprintf("CAST(%s AS TEXT)", pq.QuoteIdentifier(fallbackColumn)) // Standard: Fallback-Spalte als Text
	}
	defer rows.Close()

	for rows.Next() {
		var columnName string
		if err := rows.Scan(&columnName); err == nil {
			columns = append(columns, columnName)
		}
	}

	if len(columns) == 0 {
		// Falls keine Textspalten gefunden, Fallback-Spalte verwenden
		return fmt.Sprintf("CAST(%s AS TEXT)", pq.QuoteIdentifier(fallbackColumn))
	}

	return "CONCAT_WS(' ', " + strings.Join(columns, ", ") + ")"
}

func SaveRecord(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Anfrage-Daten parsen
	var data struct {
		Schema     string `json:"schema"`
		Table      string `json:"table"`
		PrimaryKey string `json:"primaryKey"`
		Columns    []struct {
			Name  string      `json:"name"`
			Value interface{} `json:"value"`
		} `json:"columns"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	columnTypes, err := GetColumnTypes(db, data.Schema, data.Table)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to retrieve column types: %v", err), http.StatusInternalServerError)
		return
	}

	// Variablen für die SQL-Anweisung vorbereiten
	columns := []string{}
	values := []interface{}{}
	var primaryKeyValue interface{}
	isUpdate := false

	for _, column := range data.Columns {
		colType, ok := columnTypes[column.Name]
		if !ok {
			http.Error(w, fmt.Sprintf("Unknown column: %s", column.Name), http.StatusBadRequest)
			return
		}

		// Überprüfung auf Primary Key und Setzen für Update
		if column.Name == data.PrimaryKey {
			primaryKeyValue = column.Value
			if primaryKeyValue != nil && primaryKeyValue != "" {
				isUpdate = true
			}
			continue
		}

		// Konvertierung basierend auf Typ
		var convertedValue interface{}
		switch colType {
		case "integer":
			convertedValue = column.Value
		case "boolean":
			convertedValue = column.Value == "true" || column.Value == true
		case "timestamp with time zone", "timestamp without time zone":
			if val, ok := column.Value.(string); ok {
				convertedValue, err = time.Parse(time.RFC3339, val)
				if err != nil {
					http.Error(w, fmt.Sprintf("Invalid timestamp format for %s: %v", column.Name, err), http.StatusBadRequest)
					return
				}
			}
		case "time":
			if val, ok := column.Value.(string); ok {
				convertedValue, err = time.Parse("15:04:05", val)
				if err != nil {
					http.Error(w, fmt.Sprintf("Invalid time format for %s: %v", column.Name, err), http.StatusBadRequest)
					return
				}
			}
		case "numeric", "float":
			if val, ok := column.Value.(string); ok {
				convertedValue, err = strconv.ParseFloat(strings.Replace(val, ",", ".", 1), 64)
				if err != nil {
					http.Error(w, fmt.Sprintf("Invalid number format for %s: %v", column.Name, err), http.StatusBadRequest)
					return
				}
			}
		default:
			convertedValue = column.Value
		}

		columns = append(columns, pq.QuoteIdentifier(column.Name))
		values = append(values, convertedValue)
	}

	if isUpdate {
		// UPDATE Query
		query := fmt.Sprintf("UPDATE %s.%s SET ", pq.QuoteIdentifier(data.Schema), pq.QuoteIdentifier(data.Table))

		setClauses := make([]string, len(columns))
		for i, col := range columns {
			setClauses[i] = fmt.Sprintf("%s = $%d", col, i+1)
		}

		query += strings.Join(setClauses, ", ")
		query += fmt.Sprintf(" WHERE %s = $%d", pq.QuoteIdentifier(data.PrimaryKey), len(columns)+1)
		values = append(values, primaryKeyValue)

		if _, err := db.Exec(query, values...); err != nil {
			http.Error(w, fmt.Sprintf("Failed to update record: %v", err), http.StatusInternalServerError)
			return
		}
	} else {
		// INSERT Query
		query := fmt.Sprintf("INSERT INTO %s.%s (", pq.QuoteIdentifier(data.Schema), pq.QuoteIdentifier(data.Table))
		query += strings.Join(columns, ", ")
		query += ") VALUES ("

		valuePlaceholders := make([]string, len(columns))
		for i := range valuePlaceholders {
			valuePlaceholders[i] = fmt.Sprintf("$%d", i+1)
		}
		query += strings.Join(valuePlaceholders, ", ")
		query += ") RETURNING " + pq.QuoteIdentifier(data.PrimaryKey)

		if err := db.QueryRow(query, values...).Scan(&primaryKeyValue); err != nil {
			http.Error(w, fmt.Sprintf("Failed to insert record: %v", err), http.StatusInternalServerError)
			return
		}
		data.Columns = append(data.Columns, struct {
			Name  string      `json:"name"`
			Value interface{} `json:"value"`
		}{data.PrimaryKey, primaryKeyValue})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// DeleteRecord löscht einen Datensatz basierend auf dem Primary Key
func DeleteRecord(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var data struct {
		Schema          string      `json:"schema"`
		Table           string      `json:"table"`
		PrimaryKey      string      `json:"primaryKey"`
		PrimaryKeyValue interface{} `json:"primaryKeyValue"`
	}

	// Daten aus der Anfrage parsen
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	// Überprüfen, ob die erforderlichen Felder vorhanden sind
	if data.Schema == "" || data.Table == "" || data.PrimaryKey == "" || data.PrimaryKeyValue == nil {
		http.Error(w, "Fehlende Parameter für das Löschen", http.StatusBadRequest)
		return
	}

	// SQL-Anweisung vorbereiten
	query := fmt.Sprintf("DELETE FROM %s.%s WHERE %s = $1",
		pq.QuoteIdentifier(data.Schema),
		pq.QuoteIdentifier(data.Table),
		pq.QuoteIdentifier(data.PrimaryKey))

	// Ausführen der SQL-Anweisung
	if _, err := db.Exec(query, data.PrimaryKeyValue); err != nil {
		http.Error(w, fmt.Sprintf("Fehler beim Löschen des Datensatzes: %v", err), http.StatusInternalServerError)
		return
	}

	// Erfolgsantwort senden
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Datensatz erfolgreich gelöscht"))
}
