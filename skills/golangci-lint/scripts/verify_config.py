#!/usr/bin/env python3
"""
Verify a golangci-lint v2 configuration file using `golangci-lint config verify`.
"""

import sys
import subprocess
import argparse

def main():
    parser = argparse.ArgumentParser(description="Verify golangci-lint v2 config file.")
    parser.add_argument("config_path", help="Path to .golangci.yml / .golangci.yaml file")
    args = parser.parse_args()

    cmd = ["golangci-lint", "config", "verify", "-c", args.config_path]
    try:
        res = subprocess.run(cmd, capture_output=True, text=True)
        if res.returncode == 0:
            print(f"[OK] Configuration '{args.config_path}' is valid v2 syntax.")
            sys.exit(0)
        else:
            print(f"[FAIL] Configuration '{args.config_path}' failed verification:")
            if res.stdout:
                print(res.stdout)
            if res.stderr:
                print(res.stderr)
            sys.exit(res.returncode)
    except FileNotFoundError:
        print("[ERROR] golangci-lint binary not found in PATH.")
        sys.exit(1)

if __name__ == "__main__":
    main()
