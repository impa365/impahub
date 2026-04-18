#!/bin/sh
# Generate runtime config from environment variables
cat <<EOF > /usr/share/nginx/html/config.js
window.__ENV__ = {
  VITE_API_URL: "${VITE_API_URL:-}",
};
EOF

# Start nginx
exec nginx -g 'daemon off;'
