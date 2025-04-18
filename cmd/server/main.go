package main

import (
	"log"
	"notes-app/internal/app"
	"notes-app/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Config error: %v", err)
	}

	application, err := app.New(cfg)
	if err != nil {
		log.Fatalf("App initialization error: %v", err)
	}

	if err := application.Run(); err != nil {
		log.Fatal(err)
	}
}
