#!/usr/bin/env python3
"""Analyze Go code for documentation completeness.

Extracts public APIs using `go doc` and identifies:
- Exported functions/types without doc comments
- Incomplete doc comments (just the name)
- Missing package documentation
"""

import subprocess
import re
import sys
from pathlib import Path
from dataclasses import dataclass
from typing import List


@dataclass
class DocIssue:
    """Represents a documentation issue."""
    package: str
    name: str
    issue_type: str  # 'missing', 'incomplete', 'package_missing'
    line: int = 0


def run_go_doc(package: str) -> str:
    """Run go doc on a package and return output."""
    try:
        result = subprocess.run(
            ['go', 'doc', '-all', package],
            capture_output=True,
            text=True,
            cwd=Path.cwd()
        )
        return result.stdout
    except subprocess.CalledProcessError as e:
        print(f"Error running go doc on {package}: {e}", file=sys.stderr)
        return ""


def get_packages() -> List[str]:
    """Get all Go packages in the project."""
    try:
        result = subprocess.run(
            ['go', 'list', './...'],
            capture_output=True,
            text=True
        )
        return [p.strip() for p in result.stdout.split('\n') if p.strip()]
    except subprocess.CalledProcessError:
        return []


def parse_doc_output(output: str, package: str) -> List[DocIssue]:
    """Parse go doc output to find documentation issues."""
    issues = []
    lines = output.split('\n')

    # Check for package documentation
    has_package_doc = False
    for i, line in enumerate(lines):
        if line.startswith('package '):
            # Check if there's a comment before package declaration
            if i > 0 and lines[i-1].startswith('//'):
                has_package_doc = True
            break

    if not has_package_doc and output:
        issues.append(DocIssue(package, package, 'package_missing'))

    # Pattern to match exported identifiers
    # Look for "type Name" or "func Name" without preceding comment
    exported_pattern = re.compile(r'^(type|func)\s+([A-Z][A-Za-z0-9_]*)')

    i = 0
    while i < len(lines):
        line = lines[i]

        # Check if this is an exported declaration
        match = exported_pattern.match(line)
        if match:
            name = match.group(1)

            # Check if previous non-empty line is a comment
            has_doc = False
            j = i - 1
            while j >= 0:
                prev_line = lines[j].strip()
                if not prev_line:
                    j -= 1
                    continue
                if prev_line.startswith('//'):
                    has_doc = True
                    # Check if doc is just the name (incomplete)
                    doc_content = prev_line[2:].strip()
                    if doc_content == name or doc_content.startswith(f"{name} "):
                        # It's just repeating the name - might be incomplete
                        pass
                break

            if not has_doc:
                issues.append(DocIssue(package, name, 'missing', i + 1))

        i += 1

    return issues


def analyze_file(filepath: Path) -> List[DocIssue]:
    """Analyze a single Go file for documentation issues."""
    issues = []
    content = filepath.read_text()
    lines = content.split('\n')

    package_name = None
    exported_type_pattern = re.compile(r'^type\s+([A-Z][A-Za-z0-9_]*)')
    exported_func_pattern = re.compile(r'^func\s+([A-Z][A-Za-z0-9_]*)')
    exported_method_pattern = re.compile(r'^func\s+\([^)]+\)\s+([A-Z][A-Za-z0-9_]*)')

    # Find package name
    for line in lines:
        if line.startswith('package '):
            package_name = line.split()[1]
            break

    if not package_name:
        return issues

    # Check for package comment
    has_package_comment = False
    for i, line in enumerate(lines):
        if line.startswith('package '):
            # Look backwards for package comment
            for j in range(i - 1, -1, -1):
                if lines[j].startswith('//'):
                    has_package_comment = True
                    break
                elif lines[j].strip() and not lines[j].startswith('//'):
                    break
            break

    if not has_package_comment:
        issues.append(DocIssue(
            package_name,
            filepath.stem,
            'package_missing',
            1
        ))

    # Check exported types and functions
    for i, line in enumerate(lines):
        # Skip comments and empty lines
        stripped = line.strip()
        if not stripped or stripped.startswith('//'):
            continue

        match = (exported_type_pattern.match(stripped) or
                 exported_func_pattern.match(stripped) or
                 exported_method_pattern.match(stripped))

        if match:
            name = match.group(1)

            # Look backwards for doc comment
            has_doc = False
            is_incomplete = False
            for j in range(i - 1, -1, -1):
                prev = lines[j].strip()
                if not prev:
                    continue
                if prev.startswith('//'):
                    has_doc = True
                    doc_text = prev[2:].strip()
                    # Check if doc is just the name
                    if doc_text == name or doc_text.startswith(f"{name} "):
                        is_incomplete = True
                    break
                else:
                    break

            if not has_doc:
                issues.append(DocIssue(
                    package_name,
                    name,
                    'missing',
                    i + 1
                ))
            elif is_incomplete:
                issues.append(DocIssue(
                    package_name,
                    name,
                    'incomplete',
                    i + 1
                ))

    return issues


def main():
    """Main entry point."""
    import argparse

    parser = argparse.ArgumentParser(
        description='Analyze Go code for documentation completeness'
    )
    parser.add_argument(
        'path',
        nargs='?',
        default='.',
        help='Path to analyze (default: current directory)'
    )
    parser.add_argument(
        '--format',
        choices=['text', 'json'],
        default='text',
        help='Output format'
    )
    parser.add_argument(
        '--package',
        help='Analyze specific package only'
    )

    args = parser.parse_args()

    issues = []

    if args.package:
        output = run_go_doc(args.package)
        issues.extend(parse_doc_output(output, args.package))
    else:
        # Find all Go files
        path = Path(args.path)
        for go_file in path.rglob('*.go'):
            # Skip vendor and test files if needed
            if 'vendor' in str(go_file):
                continue
            issues.extend(analyze_file(go_file))

    # Output results
    if args.format == 'json':
        import json
        print(json.dumps(
            [{'package': i.package, 'name': i.name, 'type': i.issue_type, 'line': i.line}
             for i in issues],
            indent=2
        ))
    else:
        if not issues:
            print("All public APIs are documented!")
            return 0

        # Group by package
        by_package = {}
        for issue in issues:
            by_package.setdefault(issue.package, []).append(issue)

        for package, pkg_issues in sorted(by_package.items()):
            print(f"\n{package}:")
            for issue in sorted(pkg_issues, key=lambda x: x.line):
                if issue.issue_type == 'missing':
                    print(f"  Line {issue.line}: {issue.name} - missing documentation")
                elif issue.issue_type == 'incomplete':
                    print(f"  Line {issue.line}: {issue.name} - incomplete documentation")
                elif issue.issue_type == 'package_missing':
                    print(f"  Missing package documentation")

        print(f"\nTotal issues: {len(issues)}")

    return 1 if issues else 0


if __name__ == '__main__':
    sys.exit(main())
