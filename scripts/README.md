# scripts/

## gen_jumpstart.py

Generates a jumpstart card-list file (e.g. `assets/jmsc.txt`) from a directory of theme deck lists.

Each theme file in the input directory is read, card names are resolved against an Elasticsearch `cards` index to find the set code and collector number, and the results are written in the format used by `assets/jtla.txt`.

### Prerequisites

- Python 3.10+
- `curl` available on `PATH`
- Elasticsearch reachable at the configured URL with a `cards` index

### Usage

```text
python3 scripts/gen_jumpstart.py <themes_dir> [--sets SET,...] [--es-url URL] [--output FILE]
```

| Argument | Default | Description |
| --- | --- | --- |
| `themes_dir` | _(required)_ | Directory of theme `.txt` files (e.g. `internal/themes/msc`) |
| `--sets` | themes dir name | Comma-separated ES set codes to search (e.g. `msc,msh`) |
| `--es-url` | `http://elasticsearch:9200` | Elasticsearch base URL |
| `--output` | `assets/j<dirname>.txt` | Output file path |

### Examples

Generate `assets/jmsc.txt` from the MSC Commander themes, searching both the
MSC and MSH sets:

```bash
python3 scripts/gen_jumpstart.py internal/themes/msc --sets msc,msh
```

Generate `assets/jtla.txt` from the TLA themes, searching TLA and TLE:

```bash
python3 scripts/gen_jumpstart.py internal/themes/tla --sets tla,tle
```

Write to a custom path:

```bash
python3 scripts/gen_jumpstart.py internal/themes/msc --sets msc,msh --output /tmp/preview.txt
```

### Output format

Each non-basic card gets one line with all themes it appears in listed as hashtags:

```text
1 Helicarrier Strike (MSH) 15 #Agents of Shield #Battalion #Marvelous
```

Basic lands get one line per theme (quantities vary per deck):

```text
7 Plains (MSC) 866 #Agents of Shield
7 Plains (MSC) 866 #Battalion
```

Non-basic cards are sorted alphabetically. Basic lands follow, sorted by
quantity then land name then theme name.

### Theme file detection

A `.txt` file is treated as a theme deck list only if its first non-blank line
is `Deck`. Any other file (shell scripts, notes, placeholder lists) is skipped
automatically.

### Collector number selection

When a card appears in multiple printings across the searched sets, the printing
with the **lowest numeric collector number** is used. This selects the base
printing and avoids extended-art or alternate-frame variants, which carry higher
numbers.
