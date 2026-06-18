#!/usr/bin/env python3
"""
Generate a jumpstart card list from a themes directory.

Usage:
    python3 scripts/gen_jumpstart.py internal/themes/jmsh --sets msc,msh

Output is written to assets/{dirname}.txt relative to the repo root, or to
the path given by --output.
"""

import argparse
import json
import os
import subprocess
import sys
from pathlib import Path

SMALL_WORDS = {"a", "an", "the", "and", "but", "or", "for", "nor",
               "on", "at", "to", "by", "in", "of", "up", "as", "with"}

BASIC_LANDS = {"Island", "Mountain", "Plains", "Swamp", "Forest"}


def title_case(name: str) -> str:
    """Convert kebab-case to title case, keeping small words lowercase."""
    words = name.replace("-", " ").split()
    result = []
    for i, word in enumerate(words):
        if i == 0 or word.lower() not in SMALL_WORDS:
            result.append(word.capitalize())
        else:
            result.append(word.lower())
    return " ".join(result)


def normalize_name(name: str) -> str:
    """Normalize curly apostrophes/quotes to ASCII equivalents."""
    return name.replace("‘", "'").replace("’", "'")


def is_theme_file(path: Path) -> bool:
    """Return True if the file is a valid jumpstart theme file (first non-blank line is 'Deck')."""
    try:
        with open(path) as f:
            for line in f:
                line = line.strip()
                if line:
                    return line == "Deck"
    except (OSError, UnicodeDecodeError):
        pass
    return False


def read_theme_file(path: Path, theme_name: str) -> list:
    """Return list of (qty, card_name, theme_name) from a theme file."""
    entries = []
    with open(path) as f:
        for line in f:
            line = line.strip()
            if not line or line == "Deck":
                continue
            parts = line.split(" ", 1)
            if len(parts) != 2:
                continue
            try:
                qty = int(parts[0])
            except ValueError:
                continue
            entries.append((qty, normalize_name(parts[1].strip()), theme_name))
    return entries


def es_search(es_url: str, query: dict) -> dict:
    """Run an elasticsearch search via curl (handles redirects and self-signed TLS)."""
    proc = subprocess.run(
        ["curl", "-skL", f"{es_url}/cards/_search",
         "-H", "Content-Type: application/json",
         "-d", json.dumps(query)],
        capture_output=True, text=True, check=True,
    )
    return json.loads(proc.stdout)


def collector_number_key(num: str) -> tuple:
    """Numeric sort key for collector numbers (handles suffixes like '415a')."""
    digits = "".join(c for c in num if c.isdigit())
    return (int(digits) if digits else float("inf"), num)


def fetch_cards(es_url: str, sets: list) -> dict:
    """Return {card_name: (SET, collector_number)} for all cards in the given sets.

    When a card appears in multiple printings across the provided sets, the
    printing with the lowest numeric collector number is chosen — that is
    always the base (non-extended-art, non-alternate) printing.
    """
    query = {
        "size": 10000,
        "_source": ["name", "set", "collector_number"],
        "query": {"terms": {"set": [s.lower() for s in sets]}},
    }
    result = es_search(es_url, query)

    # Collect every printing per card name, then pick the lowest collector number.
    candidates: dict[str, list] = {}
    for hit in result["hits"]["hits"]:
        src = hit["_source"]
        candidates.setdefault(src["name"], []).append(
            (src["set"], src["collector_number"])
        )

    card_map: dict = {}
    for name, printings in candidates.items():
        best_set, best_num = min(printings, key=lambda p: collector_number_key(p[1]))
        card_map[name] = (best_set.upper(), best_num)

    total = result["hits"]["total"]["value"]
    set_labels = ", ".join(s.upper() for s in sets)
    print(f"Fetched {len(card_map)} unique names ({total} hits) from {set_labels}", file=sys.stderr)
    return card_map


def find_repo_root(start: Path) -> Path:
    """Walk up from start until we find go.mod, .git, or an assets/ directory."""
    current = start.resolve()
    while current != current.parent:
        if (current / "go.mod").exists() or (current / ".git").exists() or (current / "assets").is_dir():
            return current
        current = current.parent
    return start.resolve()


def build_output(themes_dir: Path, card_map: dict) -> list:
    """Read all theme files and return formatted output lines."""
    non_basic: dict[str, set] = {}
    basic_entries: list = []

    for path in sorted(themes_dir.iterdir()):
        if path.suffix != ".txt" or not is_theme_file(path):
            continue
        theme_name = title_case(path.stem)
        for qty, card_name, tname in read_theme_file(path, theme_name):
            if card_name in BASIC_LANDS:
                basic_entries.append((qty, card_name, tname))
            else:
                non_basic.setdefault(card_name, set()).add(tname)

    all_names = set(non_basic) | {name for _, name, _ in basic_entries}
    for name in sorted(all_names - set(card_map)):
        print(f"WARNING: not found in ES: {name!r}", file=sys.stderr)

    lines = []

    for card_name in sorted(non_basic):
        set_code, num = card_map.get(card_name, ("???", "?"))
        tags = " ".join(f"#{t}" for t in sorted(non_basic[card_name]))
        lines.append(f"1 {card_name} ({set_code}) {num} {tags}")

    basic_entries.sort(key=lambda x: (x[0], x[1], x[2]))
    for qty, card_name, theme in basic_entries:
        set_code, num = card_map.get(card_name, ("???", "?"))
        lines.append(f"{qty} {card_name} ({set_code}) {num} #{theme}")

    return lines


def main():
    parser = argparse.ArgumentParser(
        description="Generate a jumpstart card list from a themes directory."
    )
    parser.add_argument(
        "themes_dir",
        type=Path,
        help="Directory of theme .txt files (e.g. internal/themes/jmsh)",
    )
    parser.add_argument(
        "--sets",
        help=(
            "Comma-separated ES set codes to search "
            "(default: the themes directory name, e.g. msc). "
            "Add companion sets as needed: --sets msc,msh"
        ),
    )
    parser.add_argument(
        "--es-url",
        default="http://elasticsearch:9200",
        help="Elasticsearch base URL (default: http://elasticsearch:9200)",
    )
    parser.add_argument(
        "--output",
        type=Path,
        help="Output file path (default: assets/{dirname}.txt relative to repo root)",
    )
    args = parser.parse_args()

    themes_dir: Path = args.themes_dir
    if not themes_dir.is_dir():
        print(f"error: {themes_dir} is not a directory", file=sys.stderr)
        sys.exit(1)

    sets = [s.strip() for s in args.sets.split(",")] if args.sets else [themes_dir.name]

    output_path: Path
    if args.output:
        output_path = args.output
    else:
        repo_root = find_repo_root(themes_dir)
        output_path = repo_root / "assets" / f"j{themes_dir.name}.txt"

    card_map = fetch_cards(args.es_url, sets)
    lines = build_output(themes_dir, card_map)

    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text("\n".join(lines) + "\n")
    print(f"Wrote {len(lines)} lines to {output_path}", file=sys.stderr)


if __name__ == "__main__":
    main()
