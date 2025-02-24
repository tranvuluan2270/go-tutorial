package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"go-tutorial/config"
	"go-tutorial/database"
	"go-tutorial/handlers"
	"go-tutorial/router"
)

type App struct {
	// Embed the App struct from handlers package
	handlers.Handler
}

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Initialize MongoDB connection
	client, err := database.Connect(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(context.TODO())

	// Initialize application
	app := &App{}
	app.DB = client
	app.Database = cfg.Database

	// Setup router
	app.Router = router.SetupRoutes(&app.Handler)

	// Start server
	fmt.Println("Connected to MongoDB!")
	fmt.Printf("Server running at http://localhost%s\n", cfg.Port)

	log.Fatal(http.ListenAndServe(cfg.Port, app.Router))
}
