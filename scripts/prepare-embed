#!/usr/bin/env python3

import os
import sys
import argparse

sys.path.insert(
    0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "scripts/lib"))
)

from ha_addons.build import get_components, build_web


def main():
    parser = argparse.ArgumentParser(
        description="Prepare web embed assets for all components."
    )
    parser.add_argument(
        "--fast", action="store_true", help="Skip building if assets already exist."
    )
    parser.add_argument(
        "component",
        nargs="?",
        default=None,
        help="Component name to install (optional)",
    )
    args = parser.parse_args()

    component_names = get_components(args.component)

    for component_name in component_names:
        build_web(component_name, fast=args.fast)


if __name__ == "__main__":
    main()
