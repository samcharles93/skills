#!/usr/bin/env python3
import os
import sys
import argparse
import subprocess

PACKAGE_TEMPLATE = """package {{PACKAGE_NAME}}

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Start initializes and runs the MCP server.
func Start() error {
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
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}
"""

def main():
    parser = argparse.ArgumentParser(
        description="Scaffold a Go MCP server as a package in an existing project."
    )
    parser.add_argument("name", help="Name of the MCP server implementation")
    parser.add_argument(
        "--package-path", required=True, help="Directory to create the package in (e.g., internal/mcp)"
    )
    args = parser.parse_args()

    package_dir = args.package_path
    if not os.path.exists(package_dir):
        os.makedirs(package_dir)

    package_name = os.path.basename(os.path.normpath(package_dir))
    if not package_name:
        package_name = "mcp"

    file_path = os.path.join(package_dir, "server.go")
    if os.path.exists(file_path):
        print(f"Error: File {file_path} already exists.")
        sys.exit(1)

    with open(file_path, "w") as f:
        f.write(PACKAGE_TEMPLATE.replace("{{NAME}}", args.name).replace("{{PACKAGE_NAME}}", package_name))

    print("Adding Go SDK dependency (running go get)...")
    try:
        subprocess.run(["go", "get", "github.com/modelcontextprotocol/go-sdk/mcp"], cwd=package_dir, check=True)
        subprocess.run(["go", "mod", "tidy"], cwd=package_dir, check=True)
    except subprocess.CalledProcessError as e:
        print(f"Warning: Could not run go commands. Ensure you are inside a Go module. ({e})")

    print(f"\n[OK] MCP Server package '{args.name}' scaffolded at {package_dir}")
    print("To add tools to this package later, run:")
    print(f"  python scripts/add_tool.py <tool_name> <desc> --file {file_path}")

if __name__ == "__main__":
    main()