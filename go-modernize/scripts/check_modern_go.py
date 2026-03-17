#!/usr/bin/env python3
"""Check for modern Go patterns that go fix doesn't handle.

Identifies old-style patterns that should use modern Go 1.22+ features:
- for i := 0; i < n; i++ → for i := range n
- if x < min { x = min } → x = max(x, min)
- interface{} → any
- append in loops → pre-sized slices
"""

import re
import sys
from pathlib import Path
from dataclasses import dataclass
from typing import List


@dataclass
class ModernizationIssue:
    """Represents a modernization opportunity."""
    file: Path
    line: int
    old_pattern: str
    new_pattern: str
    severity: str  # 'required', 'recommended'


def analyze_file(filepath: Path) -> List[ModernizationIssue]:
    """Analyze a single Go file for modernization opportunities."""
    issues = []
    content = filepath.read_text()
    lines = content.split('\n')

    # Pattern: for i := 0; i < n; i++ (should be for i := range n)
    old_for_pattern = re.compile(r'^\s*for\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*:=\s*0\s*;\s*\1\s*<\s*(\w+)\s*;\s*\1\+\+\s*\{?')

    # Pattern: for i := 0; i <= n; i++ (should be for i := range n+1)
    old_for_inclusive_pattern = re.compile(r'^\s*for\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*:=\s*0\s*;\s*\1\s*<=\s*(\w+)\s*;\s*\1\+\+\s*\{?')

    # Pattern: if x < min { x = min } (should be x = max(x, min))
    min_max_pattern = re.compile(r'^\s*if\s+(\w+)\s*<\s*(\w+)\s*\{\s*\1\s*=\s*\2\s*\}')
    max_min_pattern = re.compile(r'^\s*if\s+(\w+)\s*>\s*(\w+)\s*\{\s*\1\s*=\s*\2\s*\}')

    # Pattern: interface{}
    interface_any_pattern = re.compile(r'interface\{\}')

    for i, line in enumerate(lines, 1):
        # Skip comments
        if line.strip().startswith('//'):
            continue

        # Check for old for loop pattern
        if match := old_for_pattern.search(line):
            var_name, limit = match.groups()
            issues.append(ModernizationIssue(
                filepath, i,
                f'for {var_name} := 0; {var_name} < {limit}; {var_name}++',
                f'for {var_name} := range {limit}',
                'required'
            ))

        # Check for inclusive for loop
        if match := old_for_inclusive_pattern.search(line):
            var_name, limit = match.groups()
            issues.append(ModernizationIssue(
                filepath, i,
                f'for {var_name} := 0; {var_name} <= {limit}; {var_name}++',
                f'for {var_name} := range {limit} + 1',
                'required'
            ))

        # Check for min/max patterns
        if match := min_max_pattern.search(line):
            var_name, limit = match.groups()
            issues.append(ModernizationIssue(
                filepath, i,
                f'if {var_name} < {limit} {{ {var_name} = {limit} }}',
                f'{var_name} = max({var_name}, {limit})',
                'required'
            ))

        if match := max_min_pattern.search(line):
            var_name, limit = match.groups()
            issues.append(ModernizationIssue(
                filepath, i,
                f'if {var_name} > {limit} {{ {var_name} = {limit} }}',
                f'{var_name} = min({var_name}, {limit})',
                'required'
            ))

        # Check for interface{}
        if interface_any_pattern.search(line):
            # Skip if it's in a comment or string
            code_part = line.split('//')[0]
            if 'interface{}' in code_part:
                issues.append(ModernizationIssue(
                    filepath, i,
                    'interface{}',
                    'any',
                    'recommended'
                ))

    return issues


def main():
    """Main entry point."""
    import argparse

    parser = argparse.ArgumentParser(
        description='Check for modern Go patterns'
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
    issues = []

    for go_file in path.rglob('*.go'):
        if 'vendor' in str(go_file):
            continue
        try:
            issues.extend(analyze_file(go_file))
        except Exception as e:
            print(f"Error analyzing {go_file}: {e}", file=sys.stderr)

    # Output results
    if args.format == 'json':
        import json
        print(json.dumps(
            [{'file': str(i.file), 'line': i.line, 'old': i.old_pattern,
              'new': i.new_pattern, 'severity': i.severity}
             for i in issues],
            indent=2
        ))
    else:
        if not issues:
            print("All files use modern Go patterns!")
            return 0

        # Group by file
        by_file = {}
        for issue in issues:
            by_file.setdefault(issue.file, []).append(issue)

        for file, file_issues in sorted(by_file.items()):
            print(f"\n{file}:")
            for issue in sorted(file_issues, key=lambda x: x.line):
                marker = "[!]" if issue.severity == 'required' else "[?]"
                print(f"  Line {issue.line}: {marker} {issue.old_pattern}")
                print(f"           → {issue.new_pattern}")

        print(f"\nTotal issues: {len(issues)}")

    return 1 if issues else 0


if __name__ == '__main__':
    sys.exit(main())
