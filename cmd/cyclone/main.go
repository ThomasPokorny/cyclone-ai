package main

import (
	"log"
	"net/http"

	"github.com/ThomasPokorny/cyclone-ai/internal/bot"
	"github.com/ThomasPokorny/cyclone-ai/internal/config"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create bot
	cycloneBot, err := bot.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	// Setup routes and start server
	cycloneBot.SetupRoutes()
	log.Printf("Starting server on port %s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, nil))
}
