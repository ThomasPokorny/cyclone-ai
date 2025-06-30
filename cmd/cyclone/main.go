package main

import (
	"cyclone/internal/bot"
	"cyclone/internal/config"
	"log"
	"net/http"
)

func main() {
	// Load configuration (returns both app config and review config)
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create configuration provider using the config
	configProvider, err := config.NewSupabaseProvider(cfg)
	if err != nil {
		log.Fatalf("Failed to create configuration provider: %v", err)
	}

	// Create bot with both configurations
	cycloneBot, err := bot.New(cfg, configProvider)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	// Setup routes and start server
	cycloneBot.SetupRoutes()
	log.Printf("Starting server on port %s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, nil))
}
