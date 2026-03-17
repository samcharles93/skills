#!/usr/bin/env python3
import os
import sys
import argparse

ADD_TOOL_TEMPLATE = """	server.AddTool(mcp.Tool{
		Name:        "{{TOOL_NAME}}",
		Description: "{{TOOL_DESCRIPTION}}",
		InputSchema: mcp.JSonschema{
			Type: "object",
			Properties: map[string]interface{}{
				"message": map[string]interface{}{
					"type":        "string",
					"description": "A sample parameter",
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

		return mcp.NewToolResultText(fmt.Sprintf("Tool {{TOOL_NAME}} executed with message: %s", args.Message)), nil
	})
"""

def main():
    parser = argparse.ArgumentParser(description="Add a new tool to an existing Go MCP server")
    parser.add_argument("name", help="Name of the tool")
    parser.add_argument("description", help="Description of the tool")
    parser.add_argument("--file", default="main.go", help="Path to main.go file (default: main.go)")

    args = parser.parse_args()
    add_tool(args.name, args.description, args.file)

def add_tool(name, description, file_path):
    if not os.path.exists(file_path):
        print(f"Error: File '{file_path}' not found.")
        sys.exit(1)

    with open(file_path, "r") as f:
        content = f.read()

    insertion_marker = "// 3. Start the server using stdio transport"
    if insertion_marker not in content:
        print(f"Error: Could not find insertion marker '{insertion_marker}' in {file_path}.")
        print("Please ensure the file is a scaffolded main.go with the marker comment.")
        sys.exit(1)

    tool_code = ADD_TOOL_TEMPLATE.replace("{{TOOL_NAME}}", name).replace("{{TOOL_DESCRIPTION}}", description)

    new_content = content.replace(insertion_marker, tool_code + "\n\t" + insertion_marker)

    with open(file_path, "w") as f:
        f.write(new_content)

    print(f"\n[OK] Added tool '{name}' to {file_path}")
    print("You may want to run 'go fmt ./...' to format the file.")

if __name__ == "__main__":
    main()