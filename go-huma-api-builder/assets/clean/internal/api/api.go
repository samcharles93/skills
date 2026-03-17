package api

import (
	"context"
	"net/http"

	"example.com/myapi/internal/api/handlers"
	"example.com/myapi/internal/api/middleware"
	"example.com/myapi/internal/config"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

type Server struct {
	router http.Handler
	api    huma.API
}

func New(cfg *config.Config) (*Server, error) {
	// Create router
	router := http.NewServeMux()

	// Create API config
	apiConfig := huma.DefaultConfig(cfg.APIName, cfg.APIVersion)
	apiConfig.Info.Description = cfg.APIDescription

	// Create HUMA API
	api := humago.New(router, apiConfig)

	// Add middleware
	api.UseMiddleware(middleware.Logging())
	api.UseMiddleware(middleware.Recovery(api))

	// Register routes
	handlers.RegisterRoutes(api)

	return &Server{
		router: router,
		api:    api,
	}, nil
}

func (s *Server) Router() http.Handler {
	return s.router
}

func (s *Server) Shutdown(ctx context.Context) error {
	// Cleanup resources
	return nil
}
