package main

import (
	"log"
	"net/http"

	"cyclone/internal/bot"
	"cyclone/internal/config"
)

func main() {
	// Load configuration (returns both app config and review config)
	cfg, reviewCfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create bot with both configurations
	cycloneBot, err := bot.New(cfg, reviewCfg)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	// Setup routes and start server
	cycloneBot.SetupRoutes()
	log.Printf("Starting server on port %s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, nil))
}
