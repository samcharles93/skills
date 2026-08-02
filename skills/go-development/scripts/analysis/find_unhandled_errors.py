#!/usr/bin/env python3
"""Find unhandled errors in Go code.

Identifies:
- Errors assigned to _ (discarded)
- Function calls that return errors but aren't checked
- Missing error checks after function calls
"""

import re
import sys
from pathlib import Path
from dataclasses import dataclass
from typing import List, Set


@dataclass
class ErrorIssue:
    """Represents an unhandled error."""
    file: Path
    line: int
    func_call: str
    issue_type: str  # 'discarded', 'unchecked', 'assigned_not_checked'


def find_error_returns_in_file(filepath: Path) -> Set[str]:
    """Find all functions in a file that return errors."""
    error_funcs = set()
    content = filepath.read_text()
    lines = content.split('\n')

    # Pattern for function declarations returning error
    func_pattern = re.compile(
        r'^func\s+(?:\([^)]+\)\s+)?([A-Z][A-Za-z0-9_]*)\s*\([^)]*\)\s*(?:\([^)]*\)|[^\{]*)error'
    )

    for line in lines:
        match = func_pattern.search(line)
        if match:
            error_funcs.add(match.group(1))

    return error_funcs


def analyze_file(filepath: Path, project_error_funcs: Set[str]) -> List[ErrorIssue]:
    """Analyze a single Go file for unhandled errors."""
    issues = []
    content = filepath.read_text()
    lines = content.split('\n')

    # Patterns to match
    discarded_pattern = re.compile(r'^\s*_,?\s*:\s*=\s*([A-Z][A-Za-z0-9_]*|[a-z][A-Za-z0-9_]*)\s*\(')
    unchecked_pattern = re.compile(r'^\s*([A-Z][A-Za-z0-9_]*|[a-z][A-Za-z0-9_]*)\s*\([^)]*\)\s*$')

    # Common functions that return errors
    common_error_funcs = {
        'Close', 'Write', 'Read', 'WriteString', 'WriteByte', 'WriteRune',
        'Flush', 'Sync', 'Seek', 'Truncate', 'Chmod', 'Chown',
        'Marshal', 'Unmarshal', 'Encode', 'Decode', 'Parse', 'Format',
        'Scan', 'Scanln', 'Fscan', 'Fscanln', 'Sscan', 'Sscanln',
        'Lock', 'Unlock', 'RLock', 'RUnlock',
        'AddComponent', 'RemoveComponent', 'GetComponent',
    } | project_error_funcs

    for i, line in enumerate(lines, 1):
        stripped = line.strip()

        # Skip comments and empty lines
        if not stripped or stripped.startswith('//'):
            continue

        # Check for discarded errors: _, _ = func() or _ = func()
        if discarded_match := discarded_pattern.match(stripped):
            func_name = discarded_match.group(1)
            if func_name in common_error_funcs or func_name.startswith('err'):
                issues.append(ErrorIssue(
                    filepath, i, func_name, 'discarded'
                ))

        # Check for unchecked function calls that likely return errors
        # Look for patterns like "func()" on their own line
        elif unchecked_match := unchecked_pattern.match(stripped):
            func_name = unchecked_match.group(1)
            if func_name in common_error_funcs:
                # Check if next line doesn't handle the error
                if i < len(lines):
                    next_line = lines[i].strip() if i < len(lines) else ""
                    if not (next_line.startswith('if err') or
                            next_line.startswith('err =') or
                            next_line.startswith('err :=')):
                        issues.append(ErrorIssue(
                            filepath, i, func_name, 'unchecked'
                        ))

    return issues


def find_project_error_funcs(path: Path) -> Set[str]:
    """Scan project for all functions that return errors."""
    error_funcs = set()

    for go_file in path.rglob('*.go'):
        if 'vendor' in str(go_file):
            continue
        try:
            funcs = find_error_returns_in_file(go_file)
            error_funcs.update(funcs)
        except Exception:
            pass

    return error_funcs


def main():
    """Main entry point."""
    import argparse

    parser = argparse.ArgumentParser(
        description='Find unhandled errors in Go code'
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

    args = parser.parse_args()

    path = Path(args.path)

    # First pass: find all functions that return errors
    print("Scanning for error-returning functions...", file=sys.stderr)
    project_error_funcs = find_project_error_funcs(path)
    print(f"Found {len(project_error_funcs)} functions returning errors", file=sys.stderr)

    # Second pass: find unhandled errors
    print("Analyzing for unhandled errors...", file=sys.stderr)
    issues = []
    for go_file in path.rglob('*.go'):
        if 'vendor' in str(go_file):
            continue
        try:
            issues.extend(analyze_file(go_file, project_error_funcs))
        except Exception as e:
            print(f"Error analyzing {go_file}: {e}", file=sys.stderr)

    # Output results
    if args.format == 'json':
        import json
        print(json.dumps(
            [{'file': str(i.file), 'line': i.line, 'call': i.func_call, 'type': i.issue_type}
             for i in issues],
            indent=2
        ))
    else:
        if not issues:
            print("No unhandled errors found!")
            return 0

        # Group by file
        by_file = {}
        for issue in issues:
            by_file.setdefault(issue.file, []).append(issue)

        for file, file_issues in sorted(by_file.items()):
            print(f"\n{file}:")
            for issue in sorted(file_issues, key=lambda x: x.line):
                if issue.issue_type == 'discarded':
                    print(f"  Line {issue.line}: {issue.func_call}() - error discarded")
                elif issue.issue_type == 'unchecked':
                    print(f"  Line {issue.line}: {issue.func_call}() - error not checked")

        print(f"\nTotal issues: {len(issues)}")

    return 1 if issues else 0


if __name__ == '__main__':
    sys.exit(main())
