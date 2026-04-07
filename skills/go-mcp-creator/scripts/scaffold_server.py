#!/usr/bin/env python3
"""Scaffold a small, current Go MCP server from bundled templates."""

from __future__ import annotations

import argparse
import os
import shutil
import subprocess
import sys
from pathlib import Path


DEFAULT_SDK_VERSION = "1.4.1"
TEMPLATE_CHOICES = ("stdio-server",)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Scaffold a Go MCP server using bundled verified templates."
    )
    parser.add_argument("name", help="Directory name for the new project")
    parser.add_argument(
        "--path",
        default=".",
        help="Parent directory where the project should be created",
    )
    parser.add_argument(
        "--module",
        help="Go module path to write into go.mod (defaults to the project name)",
    )
    parser.add_argument(
        "--server-name",
        help="MCP implementation name (defaults to the project name with dashes)",
    )
    parser.add_argument(
        "--sdk-version",
        default=DEFAULT_SDK_VERSION,
        help=f"go-sdk version to pin (default: {DEFAULT_SDK_VERSION})",
    )
    parser.add_argument(
        "--template",
        default="stdio-server",
        choices=TEMPLATE_CHOICES,
        help="Template to use",
    )
    parser.add_argument(
        "--skip-tidy",
        action="store_true",
        help="Write files only; do not run `go mod tidy`",
    )
    return parser.parse_args()


def copy_template(template_dir: Path, dest_dir: Path) -> None:
    for source in template_dir.rglob("*"):
        relative = source.relative_to(template_dir)
        target = dest_dir / relative
        if source.is_dir():
            target.mkdir(parents=True, exist_ok=True)
            continue
        shutil.copy2(source, target)


def replace_placeholders(project_dir: Path, replacements: dict[str, str]) -> None:
    for path in project_dir.rglob("*"):
        if path.is_dir():
            continue
        text = path.read_text()
        for key, value in replacements.items():
            text = text.replace(key, value)
        if path.suffix == ".tmpl":
            path = path.with_suffix("")
        path.write_text(text)

    for tmpl in list(project_dir.rglob("*.tmpl")):
        tmpl.unlink()


def run_go_mod_tidy(project_dir: Path) -> None:
    subprocess.run(["go", "mod", "tidy"], cwd=project_dir, check=True)


def main() -> int:
    args = parse_args()

    script_dir = Path(__file__).resolve().parent
    skill_dir = script_dir.parent
    template_dir = skill_dir / "assets" / args.template
    if not template_dir.exists():
        print(f"[ERROR] Missing template directory: {template_dir}", file=sys.stderr)
        return 1

    project_dir = Path(args.path).expanduser().resolve() / args.name
    if project_dir.exists():
        print(f"[ERROR] Destination already exists: {project_dir}", file=sys.stderr)
        return 1

    project_dir.mkdir(parents=True)
    copy_template(template_dir, project_dir)

    module = args.module or args.name
    server_name = args.server_name or args.name.replace("_", "-")
    replacements = {
        "__MODULE__": module,
        "__SERVER_NAME__": server_name,
        "__SDK_VERSION__": args.sdk_version,
    }
    replace_placeholders(project_dir, replacements)

    if not args.skip_tidy:
        run_go_mod_tidy(project_dir)

    print(f"[OK] Project created at {project_dir}")
    print("")
    print("Next steps:")
    print(f"  1. cd {project_dir}")
    print("  2. Edit the TODO block in main.go")
    print("  3. Run: go test ./...")
    print(f"  4. Run: {script_dir / 'validate_server.sh'} {project_dir}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
