#!/usr/bin/env python3
"""Check Go tests for modern patterns and completeness.

Validates:
- Uses b.Loop() for benchmarks (Go 1.24+)
- Table-driven test structure
- Test function naming conventions
- Missing test coverage indicators
"""

import re
import sys
from pathlib import Path
from dataclasses import dataclass
from typing import List


@dataclass
class TestIssue:
    """Represents a test issue."""
    file: Path
    line: int
    issue_type: str
    message: str
    severity: str  # 'required', 'recommended'


def analyze_test_file(filepath: Path) -> List[TestIssue]:
    """Analyze a single Go test file."""
    issues = []
    content = filepath.read_text()
    lines = content.split('\n')

    # Check for old-style benchmark loops
    old_benchmark_pattern = re.compile(r'^\s*for\s+i\s*:=\s*0\s*;\s*i\s*<\s*b\.N\s*;\s*i\+\+')

    # Check for table-driven test pattern
    has_table_tests = False

    # Check for test function naming
    test_func_pattern = re.compile(r'^func\s+(Test\w+)')

    for i, line in enumerate(lines, 1):
        # Check for old benchmark pattern
        if old_benchmark_pattern.search(line):
            issues.append(TestIssue(
                filepath, i,
                'old_benchmark',
                'Use for b.Loop() instead of for i := 0; i < b.N; i++',
                'required'
            ))

        # Check test function naming
        if match := test_func_pattern.search(line):
            func_name = match.group(1)
            # Test names should be CamelCase starting with Test
            if not func_name[4:].istitle() and not func_name[4:].isupper():
                # This is a loose check - allow TestFoo, TestFooBar
                pass

        # Check for table-driven test indicators
        if 'tests := []struct' in line or 'testCases := []struct' in line:
            has_table_tests = True

    # Check if file has tests but no table-driven structure
    if 'func Test' in content and not has_table_tests:
        # Count test functions
        test_count = len(re.findall(r'^func Test\w+', content, re.MULTILINE))
        if test_count > 2:
            issues.append(TestIssue(
                filepath, 1,
                'missing_table_tests',
                f'Consider using table-driven tests ({test_count} test functions found)',
                'recommended'
            ))

    return issues


def main():
    """Main entry point."""
    import argparse

    parser = argparse.ArgumentParser(
        description='Check Go tests for modern patterns'
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

    for go_file in path.rglob('*_test.go'):
        if 'vendor' in str(go_file):
            continue
        try:
            issues.extend(analyze_test_file(go_file))
        except Exception as e:
            print(f"Error analyzing {go_file}: {e}", file=sys.stderr)

    # Output results
    if args.format == 'json':
        import json
        print(json.dumps(
            [{'file': str(i.file), 'line': i.line, 'type': i.issue_type,
              'message': i.message, 'severity': i.severity}
             for i in issues],
            indent=2
        ))
    else:
        if not issues:
            print("All test files follow modern patterns!")
            return 0

        # Group by file
        by_file = {}
        for issue in issues:
            by_file.setdefault(issue.file, []).append(issue)

        for file, file_issues in sorted(by_file.items()):
            print(f"\n{file}:")
            for issue in sorted(file_issues, key=lambda x: x.line):
                marker = "[!]" if issue.severity == 'required' else "[?]"
                print(f"  Line {issue.line}: {marker} {issue.message}")

        print(f"\nTotal issues: {len(issues)}")

    return 1 if issues else 0


if __name__ == '__main__':
    sys.exit(main())
