#!/usr/bin/env python3
"""
Detect which addons need to be built based on changed files.
"""

import os
import sys
import json
import re
from pathlib import Path


def get_all_addons(repo_root: Path) -> list[str]:
    """Get list of all available addons by listing subdirectories in addons/."""
    addons_dir = repo_root / "addons"
    if not addons_dir.exists():
        return []

    return sorted(
        [
            d.name
            for d in addons_dir.iterdir()
            if d.is_dir() and not d.name.startswith(".")
        ]
    )


def has_go_changes(changed_files: list[str]) -> bool:
    """Check if Go-related files changed (affects all addons)."""
    go_patterns = [
        r"^go/",  # Changes in go/ (but go/tools/<addon>/ is addon-specific)
        r"go\.mod$",  # go.mod changes
        r"go\.sum$",  # go.sum changes
    ]

    for file_path in changed_files:
        # Skip addon-specific go/tools/<addon>/ paths
        if file_path.startswith("go/tools/"):
            continue

        # Check if file matches any Go pattern
        for pattern in go_patterns:
            if re.search(pattern, file_path):
                return True

    return False


def get_changed_addons(changed_files: list[str], repo_root: Path) -> list[str]:
    """
    Determine which addons need to be built based on changed files.

    Args:
        changed_files: List of file paths that changed
        repo_root: Root directory of the repository

    Returns:
        List of addon names that need to be built
    """
    all_addons = get_all_addons(repo_root)

    if not all_addons:
        print("⚠️  No addons found in addons/ directory", file=sys.stderr, flush=True)
        return []

    # Check if Go-related files changed
    if has_go_changes(changed_files):
        print("⚠️  Go files or dependencies changed - will build all addons", flush=True)
        for addon in all_addons:
            print(f"  ✓ {addon} (Go changes affect all)", flush=True)
        return all_addons

    # Track which addons need to be built
    changed_addons = set()

    # Check each addon for changes
    for addon in all_addons:
        reasons = []

        # Check addon-specific paths
        for file_path in changed_files:
            # Check addons/<addon>/
            if file_path.startswith(f"addons/{addon}/"):
                reasons.append("addons/")
                break
            # Check go/tools/<addon>/
            elif file_path.startswith(f"go/tools/{addon}/"):
                reasons.append("go/tools/")
                break
            # Check web/<addon>/
            elif file_path.startswith(f"web/{addon}/"):
                reasons.append("web/")
                break
            # Special case: gateway also depends on web/auth
            elif addon == "gateway" and file_path.startswith("web/auth/"):
                reasons.append("web/auth/")
                break

        if reasons:
            changed_addons.add(addon)
            print(f"  ✓ {addon} has changes in {', '.join(reasons)}", flush=True)
        else:
            print(f"  - {addon} has no changes (skipped)", flush=True)

    # Convert to sorted list
    result = sorted(changed_addons)

    # Fallback: if no changes detected, build all addons for safety
    if not result:
        print("No addons have changes, but will include all for safety", flush=True)
        return all_addons

    return result


def main():
    """Main entry point for CLI usage."""
    if len(sys.argv) < 2:
        print("Usage: changed_addons.py <changed_files>", file=sys.stderr)
        print("  changed_files: space-separated list of file paths", file=sys.stderr)
        sys.exit(1)

    # Get changed files from arguments (newline-separated to handle spaces in filenames)
    changed_files_str = sys.argv[1]
    changed_files = [f.strip() for f in changed_files_str.split("\n") if f.strip()]

    print(f"Changed files: {' '.join(changed_files)}", flush=True)

    repo_root = Path(__file__).resolve().parent.parent.parent

    # Get changed addons
    changed_addons = get_changed_addons(changed_files, repo_root)

    # Output as JSON array
    json_output = json.dumps(changed_addons)
    print(f"Changed addons to build: {json_output}", flush=True)

    # Write to GitHub Actions output if running in CI
    github_output = os.getenv("GITHUB_OUTPUT")
    if github_output:
        with open(github_output, "a") as f:
            f.write(f"addons={json_output}\n")
    else:
        # For local testing, output to stdout
        print(f"addons={json_output}", flush=True)


if __name__ == "__main__":
    main()
