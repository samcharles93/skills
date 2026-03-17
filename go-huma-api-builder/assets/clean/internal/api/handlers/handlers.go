package handlers

import (
	"context"
	"fmt"

	"github.com/danielgtaylor/huma/v2"
)

type HealthOutput struct {
	Body struct {
		Status string `json:"status" example:"ok"`
	}
}

func healthHandler(ctx context.Context, input *struct{}) (*HealthOutput, error) {
	resp := &HealthOutput{}
	resp.Body.Status = "ok"
	return resp, nil
}

type GreetingInput struct {
	Name string `path:"name" maxLength:"30" example:"world"`
}

type GreetingOutput struct {
	Body struct {
		Message string `json:"message" example:"Hello, world!"`
	}
}

func greetingHandler(ctx context.Context, input *GreetingInput) (*GreetingOutput, error) {
	resp := &GreetingOutput{}
	resp.Body.Message = fmt.Sprintf("Hello, %s!", input.Name)
	return resp, nil
}

func RegisterRoutes(api huma.API) {
	// Health check
	huma.Get(api, "/health", healthHandler)

	// Greeting endpoint
	huma.Register(api, huma.Operation{
		OperationID: "get-greeting",
		Summary:     "Get a greeting",
		Method:      "GET",
		Path:        "/greeting/{name}",
		Tags:        []string{"Greetings"},
	}, greetingHandler)
}
