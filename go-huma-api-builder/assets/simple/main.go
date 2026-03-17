package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"github.com/danielgtaylor/huma/v2/humacli"
)

// GreetingInput defines the input for the greeting endpoint
type GreetingInput struct {
	Name string `path:"name" maxLength:"30" example:"world" doc:"Name to greet"`
}

// GreetingOutput defines the output for the greeting endpoint
type GreetingOutput struct {
	Body struct {
		Message string `json:"message" example:"Hello, world!" doc:"Greeting message"`
	}
}

// greetingHandler handles the greeting endpoint
func greetingHandler(ctx context.Context, input *GreetingInput) (*GreetingOutput, error) {
	resp := &GreetingOutput{}
	resp.Body.Message = fmt.Sprintf("Hello, %s!", input.Name)
	return resp, nil
}

// Options defines CLI options
type Options struct {
	Port int `help:"Port to listen on" short:"p" default:"8080"`
}

func main() {
	cli := humacli.New(func(hooks humacli.Hooks, opts *Options) {
		// Create router
		router := http.NewServeMux()

		// Create API
		config := huma.DefaultConfig("My API", "1.0.0")
		config.Info.Description = "A simple HUMA API example"
		api := humago.New(router, config)

		// Register endpoints
		huma.Register(api, huma.Operation{
			OperationID: "get-greeting",
			Summary:     "Get a greeting",
			Description: "Returns a personalized greeting message",
			Method:      http.MethodGet,
			Path:        "/greeting/{name}",
			Tags:        []string{"Greetings"},
		}, greetingHandler)

		// Create server
		server := &http.Server{
			Addr:    fmt.Sprintf(":%d", opts.Port),
			Handler: router,
		}

		// Start server
		hooks.OnStart(func() {
			fmt.Printf("Starting server on :%d...\n", opts.Port)
			fmt.Printf("API docs available at http://localhost:%d/docs\n", opts.Port)
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				fmt.Printf("Server error: %v\n", err)
			}
		})

		// Graceful shutdown
		hooks.OnStop(func() {
			fmt.Println("Shutting down server...")
			server.Shutdown(context.Background())
		})
	})

	cli.Run()
}
