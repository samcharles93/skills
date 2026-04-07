#!/usr/bin/env python3
"""Generate a typed MCP tool stub for an existing Go package."""

from __future__ import annotations

import argparse
import keyword
import re
import sys
from pathlib import Path


TOOL_TEMPLATE = """package __PACKAGE__

import (
\t"context"

\t"github.com/modelcontextprotocol/go-sdk/mcp"
)

type __TYPE__Input struct {
\tQuery string `json:"query" jsonschema:"input provided by the caller"`
}

type __TYPE__Output struct {
\tSummary string `json:"summary" jsonschema:"result returned to the caller"`
}

func __TYPE__(
\tctx context.Context,
\treq *mcp.CallToolRequest,
\tinput __TYPE__Input,
) (*mcp.CallToolResult, __TYPE__Output, error) {
\t_ = ctx
\t_ = req

\t// TODO: replace this stub with real business logic.
\ttext := "stub result for: " + input.Query
\treturn &mcp.CallToolResult{
\t\tContent: []mcp.Content{&mcp.TextContent{Text: text}},
\t}, __TYPE__Output{Summary: text}, nil
}
"""


def pascal_case(value: str) -> str:
    parts = re.split(r"[^a-zA-Z0-9]+", value)
    cleaned = [part for part in parts if part]
    if not cleaned:
        raise ValueError("tool name must contain letters or digits")
    result = "".join(part[:1].upper() + part[1:] for part in cleaned)
    if result[:1].isdigit() or keyword.iskeyword(result.lower()):
        result = "Tool" + result
    return result


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Create a typed tool handler stub for the official Go MCP SDK."
    )
    parser.add_argument("tool_name", help="MCP tool name, for example summarize_text")
    parser.add_argument(
        "--package",
        default="main",
        help="Go package name for the generated file (default: main)",
    )
    parser.add_argument(
        "--output",
        required=True,
        help="Path to the Go file that should be written",
    )
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    type_name = pascal_case(args.tool_name)
    output_path = Path(args.output).expanduser().resolve()
    if output_path.exists():
        print(f"[ERROR] Output already exists: {output_path}", file=sys.stderr)
        return 1

    output_path.parent.mkdir(parents=True, exist_ok=True)
    content = (
        TOOL_TEMPLATE.replace("__PACKAGE__", args.package)
        .replace("__TYPE__", type_name)
    )
    output_path.write_text(content)

    print(f"[OK] Wrote {output_path}")
    print("")
    print("Add this registration near server setup:")
    print(
        f'mcp.AddTool(server, &mcp.Tool{{Name: "{args.tool_name}", Description: "TODO: describe the tool"}}, {type_name})'
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
