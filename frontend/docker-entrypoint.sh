#!/bin/sh
set -eu

# Rewrite config.js from env at container start — never baked into the image build.
API_BASE_URL="${API_BASE_URL:-/api/v1}"
cat > /usr/share/nginx/html/config.js <<EOF
window.__APP_CONFIG__ = {
  apiBaseUrl: "${API_BASE_URL}",
};
EOF

exec nginx -g 'daemon off;'
