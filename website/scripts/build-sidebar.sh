#!/bin/sh
# SPDX-FileCopyrightText: 2016-2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>
# SPDX-License-Identifier: EUPL-1.2
#
# Generate the website sidebar partial by walking the content tree.
#
# Caddy 2.6 templates have no string-trimming functions, so we can't strip
# ".md" from filenames at request time. Instead we emit the sidebar once
# at container start (and at image build) as a Caddy template that still
# carries the per-link `active` checks evaluated per request.
#
# Inputs (env vars, with defaults):
#   CONTENT  — directory to walk (default: /srv/content/patterns)
#   OUT      — output file        (default: /srv/generated/sidebar.html)
#   URL_BASE — URL prefix used in links (default: /patterns)

set -eu

CONTENT="${CONTENT:-/srv/content/patterns}"
OUT="${OUT:-/srv/generated/sidebar.html}"
URL_BASE="${URL_BASE:-/patterns}"

mkdir -p "$(dirname "$OUT")"

# Read `title:` from a YAML front-matter block (between the first pair of
# `---` lines). Falls back to empty if absent.
extract_title() {
    awk '
        BEGIN { fm = 0 }
        /^---[[:space:]]*$/ {
            fm++
            if (fm == 2) exit
            next
        }
        fm == 1 && /^title:[[:space:]]/ {
            sub(/^title:[[:space:]]+/, "")
            sub(/[[:space:]]+$/, "")
            # strip surrounding quotes if present
            if (match($0, /^".*"$/) || match($0, /^'\''.*'\''$/)) {
                $0 = substr($0, 2, length($0) - 2)
            }
            print
            exit
        }
    ' "$1"
}

# "development/frontend" -> "Development / Frontend"
prettify() {
    echo "$1" | awk -F/ '{
        for (i = 1; i <= NF; i++) {
            $i = toupper(substr($i, 1, 1)) substr($i, 2)
        }
        OFS = " / "
        $1 = $1
        print
    }'
}

# HTML-escape a string for safe insertion into element text or an
# attribute value.
html_escape() {
    printf '%s' "$1" | sed -e 's/&/\&amp;/g' -e 's/</\&lt;/g' -e 's/>/\&gt;/g' -e 's/"/\&quot;/g'
}

# Top-of-file header + always-present links.
{
    cat <<'EOF'
{{- /*
SPDX-FileCopyrightText: 2016-2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>
SPDX-License-Identifier: EUPL-1.2

Auto-generated at container start by website/scripts/build-sidebar.sh.
Do not edit by hand — changes here are overwritten on the next start.
*/ -}}
{{- $current := .OriginalReq.URL.Path -}}

<ul class="sidebar-top">
  <li><a href="/about"{{ if eq $current "/about" }} class="active" aria-current="page"{{ end }}>About</a></li>
  <li><a href="/patterns"{{ if eq $current "/patterns" }} class="active" aria-current="page"{{ end }}>Patterns</a></li>
</ul>

EOF

    # All directories under $CONTENT that hold at least one .md file
    # (excluding the content root, whose index.md backs /patterns itself).
    cd "$CONTENT"
    find . -mindepth 2 -type f -name '*.md' \
        | sed 's|^\./||; s|/[^/]*$||' \
        | sort -u \
        | while IFS= read -r dir; do
            entries=""
            # POSIX glob; `-e` test handles the "no match" case.
            for f in "$dir"/*.md; do
                [ -e "$f" ] || continue
                base=$(basename "$f" .md)
                [ "$base" = "index" ] && continue
                title=$(extract_title "$f")
                [ -z "$title" ] && title="$base"
                title_html=$(html_escape "$title")
                href="$URL_BASE/$dir/$base"
                entries="${entries}$(printf '  <li><a href="%s"{{ if eq $current "%s" }} class="active" aria-current="page"{{ end }}>%s</a></li>\n' \
                    "$href" "$href" "$title_html")
"
            done
            if [ -n "$entries" ]; then
                label=$(html_escape "$(prettify "$dir")")
                printf '<h3 class="sidebar-section">%s</h3>\n' "$label"
                printf '<ul class="sidebar-list">\n'
                printf '%s' "$entries"
                printf '</ul>\n\n'
            fi
        done
} > "$OUT"

echo "build-sidebar: wrote $OUT (from $CONTENT)" >&2
