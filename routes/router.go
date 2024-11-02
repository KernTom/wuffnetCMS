package routes

import (
	"database/sql"
	"net/http"
	"wuffnetCMS/controllers"
)

func SetupRoutes(db *sql.DB) {

	// Statikdateien unter /static verfügbar machen
	fs := http.FileServer(http.Dir("web/templates"))
	http.Handle("/web/templates/", http.StripPrefix("/web/templates/", fs))

	// Statikdateien unter /static verfügbar machen
	fs = http.FileServer(http.Dir("web/static"))
	http.Handle("/web/static/", http.StripPrefix("/web/static/", fs))

	// Route für die Hauptseite
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/templates/layout.html")
	})
	// Routen definieren
	http.HandleFunc("/api/tables", func(w http.ResponseWriter, r *http.Request) {
		controllers.GetTables(db, w, r)
	})
	http.HandleFunc("/api/table-content", func(w http.ResponseWriter, r *http.Request) {
		controllers.GetTableContent(db, w, r)
	})
	http.HandleFunc("/api/table-fields", func(w http.ResponseWriter, r *http.Request) {
		controllers.GetTableFields(db, w, r)
	})
	// Neue Route zum Speichern von Daten
	http.HandleFunc("/api/save-record", func(w http.ResponseWriter, r *http.Request) {
		controllers.SaveRecord(db, w, r)
	})
	http.HandleFunc("/api/delete-record", func(w http.ResponseWriter, r *http.Request) {
		controllers.DeleteRecord(db, w, r)
	})
}
