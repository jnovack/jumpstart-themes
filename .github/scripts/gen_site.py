#!/usr/bin/env python3
"""
gen_site.py — Generate index.html for the jumpstart-themes GitHub Pages site.

Usage:
    python3 gen_site.py --site-dir <dir> [--version <tag>] [--repo <owner/repo>]

The script scans <dir> for *-letter.pdf files (one per set) and assembles a
Bootstrap 5 / Font Awesome dark-mode page with download buttons and inline
PDF viewers.

To register a friendly display name for a new set, add an entry to SET_NAMES.
"""

import argparse
import glob
import os
import re
import sys
import textwrap


# ── Set metadata ──────────────────────────────────────────────────────────────
# Map lowercase set code → (display name, short description)
# Add a new entry here whenever a new assets/<code>.txt file is added.
SET_META: dict[str, tuple[str, str]] = {
    "jtla": (
        "Avatar: The Last Airbender",
        "The Last Airbender Jumpstart expansion — air, water, earth, and fire themes.",
    ),
    "jmsh": (
        "Marvel Super Heroes",
        "Bringing some of the biggest names in Marvel to Magic.",
    ),
}

# Fallback for sets not listed above
DEFAULT_DESC = "MTG Jumpstart theme decks."


# ── Helpers ───────────────────────────────────────────────────────────────────

def set_display(code: str) -> tuple[str, str]:
    """Return (name, description) for a lowercase set code."""
    meta = SET_META.get(code)
    if meta:
        return meta
    return code.upper(), DEFAULT_DESC


def read_file(path: str, default: str = "?") -> str:
    try:
        return open(path).read().strip()
    except OSError:
        return default


# ── HTML fragments ────────────────────────────────────────────────────────────

def set_card(s: dict) -> str:
    code      = s["code"]        # "JTLA"
    lower     = s["lower"]       # "jtla"
    name      = s["name"]
    desc      = s["desc"]
    count     = s["theme_count"]
    pdf       = s["pdf"]
    zip_file  = s["zip"]
    has_zip   = s["has_zip"]
    has_prev  = s["has_preview"]

    preview_html = ""
    if has_prev:
        preview_html = (
            f'\n        <img src="{lower}-preview.png"'
            f' class="card-img-top" alt="{code} first page preview"'
            f' loading="lazy"'
            f' style="border-bottom:1px solid var(--bs-border-color)">'
        )

    zip_btn = ""
    if has_zip:
        zip_btn = (
            f'<a href="{zip_file}" class="btn btn-primary">'
            f'<i class="fa-solid fa-file-zipper me-2"></i>Download Labels</a>'
        )

    return textwrap.dedent(f"""\
        <div class="col-md-6 col-xl-4">
          <div class="card h-100 shadow-sm">
            <div class="card-header d-flex align-items-start gap-2 py-3">
              <span class="badge bg-secondary font-monospace fs-6 mt-1">{code}</span>
              <div>
                <div class="fw-semibold lh-sm">{name}</div>
                <div class="text-secondary small">{desc}</div>
              </div>
            </div>{preview_html}
            <div class="card-body">
              <p class="text-secondary small mb-3">
                <i class="fa-solid fa-layer-group me-1"></i>{count}&nbsp;themes
              </p>
              <div class="d-flex flex-wrap gap-2">
                {zip_btn}
                <a href="{pdf}" target="_blank" class="btn btn-outline-primary">
                  <i class="fa-solid fa-print me-2"></i>Print Letter Pages
                </a>
              </div>
            </div>
            <div class="card-footer p-0 border-top">
              <iframe src="{pdf}#toolbar=0&navpanes=0&scrollbar=0&view=FitH"
                      class="w-100 rounded-bottom"
                      style="height:420px;border:none;display:block"
                      title="{code} letter pages"
                      loading="lazy"></iframe>
            </div>
          </div>
        </div>""")


_CSS = """\
    body { background: #0d1117; }
    .hero {
      background: linear-gradient(135deg, #161b22 0%, #1f2937 100%);
      border-bottom: 1px solid #30363d;
    }
    .card { background: #161b22; border: 1px solid #30363d; }
    .card-header { background: #1f2937; border-bottom: 1px solid #30363d; }
    .card-footer { background: #161b22; border-color: #30363d !important; }
    footer { border-color: #30363d !important; }
    .badge { letter-spacing: .05em; }
"""


def generate_html(sets: list[dict], version: str, repo: str) -> str:
    if sets:
        cards = "\n".join(set_card(s) for s in sets)
        grid  = f"    <div class=\"row g-4\">\n{cards}\n    </div>"
    else:
        grid = textwrap.dedent("""\
            <div class="col-12 text-center py-5 text-secondary">
              <i class="fa-solid fa-box-open fa-3x mb-3 d-block"></i>
              <p class="fs-5">No sets found — trigger a release to populate this page.</p>
            </div>""")

    version_badge = ""
    if version and version != "refs/heads/main":
        version_badge = (
            f'<span class="badge bg-outline-secondary text-secondary border border-secondary ms-2">'
            f'{version}</span>'
        )

    repo_url = f"https://github.com/{repo}" if repo else "#"

    return f"""\
<!DOCTYPE html>
<html lang="en" data-bs-theme="dark">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Jumpstart Decklist Labels</title>
  <meta name="description"
        content="Thermal-printer-ready labels for MTG Jumpstart theme decks.">
  <link rel="stylesheet"
        href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/css/bootstrap.min.css"
        integrity="sha384-QWTKZyjpPEjISv5WaRU9OFeRpok6YctnYmDr5pNlyT2bRjXh0JMhjY6hW+ALEwIH"
        crossorigin="anonymous">
  <link rel="stylesheet"
        href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.5.1/css/all.min.css"
        crossorigin="anonymous">
  <style>
{_CSS}
  </style>
</head>
<body>

<header class="hero py-5 mb-5">
  <div class="container text-center">
    <h1 class="display-5 fw-bold mb-2">
      <i class="fa-solid fa-tags me-3 text-primary"></i>Jumpstart Decklist Labels
    </h1>
    <p class="lead text-secondary mb-4">
      Thermal-printer-ready labels for Magic: The Gathering Jumpstart theme decks.<br>
      <small>Print individually to a PEDOOLO 482BT via Bluetooth or any CUPS thermal printer.</small>
    </p>
    <div class="d-flex justify-content-center gap-3 flex-wrap">
      <a href="{repo_url}" class="btn btn-outline-secondary">
        <i class="fa-brands fa-github me-2"></i>View Source
      </a>
      <a href="{repo_url}/releases" class="btn btn-outline-secondary">
        <i class="fa-solid fa-download me-2"></i>All Releases
      </a>
    </div>
  </div>
</header>

<main class="container pb-5">
{grid}
</main>

<footer class="text-center text-secondary py-4 border-top">
  <small>
    Built with
    <a href="{repo_url}" class="text-secondary text-decoration-none">decklist-print</a>
    &mdash; labels generated from embedded set files at release time.
  </small>
</footer>

<script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/js/bootstrap.bundle.min.js"
        integrity="sha384-YvpcrYf0tY3lHB60NNkmXc4s9bIOgUxi8T/jzmB7dTUJPr2LVnBIZcbr7wVkCub"
        crossorigin="anonymous"></script>
</body>
</html>
"""


# ── Main ──────────────────────────────────────────────────────────────────────

def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--site-dir",  default=".",  help="directory containing built assets")
    ap.add_argument("--version",   default="",   help="release tag / version string")
    ap.add_argument("--repo",      default="",   help="GitHub repo in owner/name format")
    args = ap.parse_args()

    site_dir = args.site_dir

    # Discover sets from the letter PDFs produced by the build step.
    pdfs = sorted(glob.glob(os.path.join(site_dir, "*-letter.pdf")))

    sets = []
    for pdf_path in pdfs:
        pdf_name = os.path.basename(pdf_path)
        lower    = re.sub(r"-letter\.pdf$", "", pdf_name)
        code     = lower.upper()
        name, desc = set_display(lower)

        count = read_file(os.path.join(site_dir, f"{lower}-theme-count.txt"))
        zip_name = f"{code}.zip"

        sets.append({
            "code":        code,
            "lower":       lower,
            "name":        name,
            "desc":        desc,
            "theme_count": count,
            "pdf":         pdf_name,
            "zip":         zip_name,
            "has_zip":     os.path.exists(os.path.join(site_dir, zip_name)),
            "has_preview": os.path.exists(os.path.join(site_dir, f"{lower}-preview.png")),
        })

    html = generate_html(sets, args.version, args.repo)

    out_path = os.path.join(site_dir, "index.html")
    with open(out_path, "w") as fh:
        fh.write(html)

    print(f"Generated {out_path} — {len(sets)} set(s)", file=sys.stderr)


if __name__ == "__main__":
    main()
