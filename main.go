package main

import (
	"fmt"
	"log"
	"net/http"
	"wuffnetCMS/config"
	"wuffnetCMS/routes"

	"github.com/joho/godotenv"
)

func init() {
	// Lädt die .env Datei
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Fehler beim Laden der .env Datei: %v", err)
	}

}

func main() {

	db, err := config.ConnectDB()
	if err != nil {
		log.Fatalf("Could not connect to the database: %v", err)
	}
	defer db.Close()

	routes.SetupRoutes(db)

	fmt.Println("Server running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
