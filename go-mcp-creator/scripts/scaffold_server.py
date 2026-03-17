#!/usr/bin/env python3
import os
import sys
import argparse
import subprocess

MAIN_TEMPLATE = """package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	// 1. Create a new MCP server
	server := mcp.NewServer(
		mcp.Implementation{
			Name:    "{{NAME}}",
			Version: "1.0.0",
		},
	)

	// 2. Add a sample tool
	server.AddTool(mcp.Tool{
		Name:        "echo",
		Description: "Echoes back the input message",
		InputSchema: mcp.JSonschema{
			Type: "object",
			Properties: map[string]interface{}{
				"message": map[string]interface{}{
					"type":        "string",
					"description": "The message to echo",
				},
			},
			Required: []string{"message"},
		},
	}, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(request.Arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}

		return mcp.NewToolResultText(fmt.Sprintf("Echo: %s", args.Message)), nil
	})

	// 3. Start the server using stdio transport
	transport := mcp.NewStdioTransport()
	if err := server.Serve(transport); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
"""


def main():
    parser = argparse.ArgumentParser(
        description="Scaffold a new Go MCP server project."
    )
    parser.add_argument("name", help="Name of the server/project")
    parser.add_argument(
        "--path", default=".", help="Directory to create the project in"
    )
    args = parser.parse_args()

    project_dir = os.path.join(args.path, args.name)
    if os.path.exists(project_dir):
        print(f"Error: Directory {project_dir} already exists.")
        sys.exit(1)

    os.makedirs(project_dir)

    # Create main.go
    with open(os.path.join(project_dir, "main.go"), "w") as f:
        f.write(MAIN_TEMPLATE.replace("{{NAME}}", args.name))

    # Initialize go module
    print(f"Initializing Go module in {project_dir}...")
    subprocess.run(["go", "mod", "init", args.name], cwd=project_dir, check=True)

    # Add dependencies
    print("Adding Go SDK dependency...")
    subprocess.run(
        ["go", "get", "github.com/modelcontextprotocol/go-sdk/mcp"],
        cwd=project_dir,
        check=True,
    )

    # Tidy up
    subprocess.run(["go", "mod", "tidy"], cwd=project_dir, check=True)

    print(f"\n[OK] Project {args.name} scaffolded at {project_dir}")
    print(f"To run: cd {project_dir} && go run main.go")

if __name__ == "__main__":
    main()
