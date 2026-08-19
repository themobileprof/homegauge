#!/usr/bin/env bash
# Enable /mortgage on themobileprof.com (run once on the server, requires sudo).
# Usage: bash deploy/enable-nginx-mortgage.sh [path/to/nginx-mortgage.conf]
#
# If the server needs a sudo password and you can't get a real TTY (e.g. piping
# over ssh), set SUDO_PASS in your environment before running:
#   SUDO_PASS=mypassword bash deploy/enable-nginx-mortgage.sh
set -euo pipefail

SNIPPET_SRC="${1:-$(dirname "$0")/nginx-mortgage.conf}"
SNIPPET_DST=/etc/nginx/snippets/mortgage.conf
SITE=/etc/nginx/sites-available/themobileprof.com.conf

# Wrap sudo so it works both interactively and non-interactively (piped ssh).
_sudo() {
  if [[ -n "${SUDO_PASS:-}" ]]; then
    echo "$SUDO_PASS" | sudo -S "$@" 2>/dev/null
  else
    sudo "$@"
  fi
}

if [[ ! -f "$SITE" ]]; then
  echo "ERROR: $SITE not found — adjust the SITE variable for your server." >&2
  exit 1
fi

_sudo cp "$SNIPPET_SRC" "$SNIPPET_DST"
echo "Copied snippet → $SNIPPET_DST"

if ! grep -q 'snippets/mortgage.conf' "$SITE"; then
  _sudo python3 - <<'PY'
from pathlib import Path
path = Path("/etc/nginx/sites-available/themobileprof.com.conf")
text = path.read_text()
include_line = "    include snippets/mortgage.conf;"
if "snippets/mortgage.conf" in text:
    print("Include already present — nothing to change.")
else:
    # Prefer inserting after finco include if present
    finco = "    include snippets/finco.conf;"
    ssl_listen = "    listen [::]:443 ssl; # managed by Certbot"
    ssl_listen2 = "    listen 443 ssl; # managed by Certbot"
    if finco in text:
        text = text.replace(finco, finco + "\n" + include_line, 1)
        print("Inserted include after snippets/finco.conf")
    elif ssl_listen in text:
        text = text.replace(ssl_listen, include_line + "\n\n    " + ssl_listen.strip(), 1)
        print("Inserted include before SSL listen line")
    elif ssl_listen2 in text:
        text = text.replace(ssl_listen2, include_line + "\n\n    " + ssl_listen2.strip(), 1)
        print("Inserted include before SSL listen line")
    else:
        raise SystemExit("Could not find insertion point — add manually:\n  " + include_line)
    path.write_text(text)
PY
else
  echo "Include already present in $SITE — nothing to change."
fi

_sudo nginx -t
_sudo systemctl reload nginx
echo ""
echo "OK — https://themobileprof.com/mortgage"
