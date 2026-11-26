#!/bin/sh
# Creates local NPM proxy hosts for the VPN panel stack (like VPS 1, but on localhost).
# Domains: vpn.local -> user-ui:3000, admin.vpn.local -> admin-ui:3000, api.vpn.local -> panel-api:8080
#
# Usage:  sh deploy/npm/setup-local.sh
# Env:    NPM_URL (default http://127.0.0.1:81), NPM_EMAIL (default admin@example.com),
#         NPM_PASSWORD (default changeme)
set -e

NPM_URL="${NPM_URL:-http://127.0.0.1:81}"
NPM_EMAIL="${NPM_EMAIL:-admin@example.com}"
NPM_PASSWORD="${NPM_PASSWORD:-changeme}"

if ! curl -sf "$NPM_URL/api/" >/dev/null 2>&1; then
  echo "NPM is not reachable at $NPM_URL. Start the control plane first:"
  echo "  docker compose -f deploy/docker-compose.control.yml up -d"
  exit 1
fi

TOKEN=$(curl -sf -X POST "$NPM_URL/api/tokens" -H "Content-Type: application/json" \
  -d "{\"identity\":\"$NPM_EMAIL\",\"secret\":\"$NPM_PASSWORD\"}" | sed -E 's/.*"token":"([^"]+)".*/\1/') || {
  echo "NPM login failed for $NPM_EMAIL. Set NPM_EMAIL/NPM_PASSWORD if you changed them."; exit 1; }

add_host() {
  DOMAIN="$1"; HOST="$2"; PORT="$3"
  if curl -sf -H "Authorization: Bearer $TOKEN" "$NPM_URL/api/nginx/proxy-hosts" | grep -q "\"$DOMAIN\""; then
    echo "exists: $DOMAIN"
  else
    curl -sf -X POST "$NPM_URL/api/nginx/proxy-hosts" \
      -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
      -d "{\"domain_names\":[\"$DOMAIN\"],\"forward_scheme\":\"http\",\"forward_host\":\"$HOST\",\"forward_port\":$PORT,\"block_exploits\":true,\"allow_websocket_upgrade\":true,\"access_list_id\":1,\"certificate_id\":0,\"ssl_forced\":false,\"http2_support\":false,\"hsts_enabled\":false,\"hsts_subdomains\":false}" \
      >/dev/null && echo "created: $DOMAIN -> $HOST:$PORT"
  fi
}

add_host vpn.local user-ui 3000
add_host admin.vpn.local admin-ui 3000
add_host api.vpn.local panel-api 8080

echo ""
echo "Add to /etc/hosts:"
echo "127.0.0.1 vpn.local admin.vpn.local api.vpn.local"
