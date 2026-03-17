#!/usr/bin/env python3
import os
import sys
import argparse

ADD_RESOURCE_TEMPLATE = """	server.AddResource(mcp.Resource{
		URI:         "{{RESOURCE_URI}}",
		Name:        "{{RESOURCE_NAME}}",
		Description: "{{RESOURCE_DESCRIPTION}}",
		MimeType:    "text/plain",
	}, func(ctx context.Context, request mcp.ReadResourceRequest) ([]interface{}, error) {
		return []interface{}{
			mcp.TextResourceContents{
				ResourceContents: mcp.ResourceContents{
					URI:      request.URI,
					MimeType: "text/plain",
				},
				Text: fmt.Sprintf("Resource content for %s goes here", request.URI),
			},
		}, nil
	})
"""

def main():
    parser = argparse.ArgumentParser(description="Add a new resource to an existing Go MCP server")
    parser.add_argument("name", help="Name of the resource")
    parser.add_argument("uri", help="URI of the resource (e.g., custom://resource)")
    parser.add_argument("description", help="Description of the resource")
    parser.add_argument("--file", default="main.go", help="Path to main.go file (default: main.go)")

    args = parser.parse_args()
    add_resource(args.name, args.uri, args.description, args.file)

def add_resource(name, uri, description, file_path):
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

    resource_code = (ADD_RESOURCE_TEMPLATE
                     .replace("{{RESOURCE_NAME}}", name)
                     .replace("{{RESOURCE_URI}}", uri)
                     .replace("{{RESOURCE_DESCRIPTION}}", description))

    # Inject code right before the marker
    new_content = content.replace(insertion_marker, resource_code + "\n\t" + insertion_marker)

    with open(file_path, "w") as f:
        f.write(new_content)

    print(f"\n[OK] Added resource '{name}' ({uri}) to {file_path}")
    print("You may want to run 'go fmt ./...' to format the file.")

if __name__ == "__main__":
    main()