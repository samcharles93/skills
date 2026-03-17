package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"example.com/myapi/internal/api"
	"example.com/myapi/internal/config"

	"github.com/danielgtaylor/huma/v2/humacli"
)

type Options struct {
	Port int    `help:"Port to listen on" short:"p" default:"8080"`
	DB   string `help:"Database connection string" default:"postgres://localhost/mydb"`
}

func main() {
	cli := humacli.New(func(hooks humacli.Hooks, opts *Options) {
		// Load configuration
		cfg, err := config.Load()
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}

		// Initialize API
		apiServer, err := api.New(cfg)
		if err != nil {
			log.Fatalf("Failed to create API: %v", err)
		}

		// Create HTTP server
		server := &http.Server{
			Addr:    fmt.Sprintf(":%d", opts.Port),
			Handler: apiServer.Router(),
		}

		// Start server
		hooks.OnStart(func() {
			fmt.Printf("Starting server on :%d...\n", opts.Port)
			fmt.Printf("API docs available at http://localhost:%d/docs\n", opts.Port)
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("Server error: %v", err)
			}
		})

		// Graceful shutdown
		hooks.OnStop(func() {
			fmt.Println("Shutting down server...")
			ctx := context.Background()
			server.Shutdown(ctx)
			apiServer.Shutdown(ctx)
		})
	})

	cli.Run()
}
