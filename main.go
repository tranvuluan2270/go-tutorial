package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"go-tutorial/cache"
	"go-tutorial/config"
	"go-tutorial/database"
	"go-tutorial/handlers"
	"go-tutorial/router"
	"go-tutorial/utils"
)

type App struct {
	// Embed the App struct from handlers package
	handlers.Handler
}

func main() {
	// Load MongoDB configuration
	cfg := config.LoadConfig()

	// Initialize MongoDB connection
	client, err := database.Connect(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(context.TODO())

	// Initialize Redis
	redisConfig := cache.RedisConfig{
		Host:     "localhost", // Change this to your Redis host
		Port:     "6379",      // Change this to your Redis port
		Password: "",          // Add password if required
		DB:       0,
	}

	if err := cache.InitRedis(redisConfig); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Initialize and start Redis update job
	redisUpdateJob := utils.NewRedisUpdateJob(client, cfg.Database)
	redisUpdateJob.Start()

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
