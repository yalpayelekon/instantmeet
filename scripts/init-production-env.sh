#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

if [[ -f .env.production ]]; then
  echo ".env.production already exists; leaving it unchanged."
  exit 0
fi

acme_email="${1:-}"
if [[ -z "${acme_email}" ]]; then
  echo "Usage: bash scripts/init-production-env.sh you@example.com"
  exit 1
fi

umask 077
cat >.env.production <<EOF
APP_DOMAIN=toplanti.online
LIVEKIT_DOMAIN=livekit.toplanti.online
ACME_EMAIL=${acme_email}
POSTGRES_PASSWORD=$(openssl rand -hex 24)
JWT_SECRET=$(openssl rand -hex 32)
LIVEKIT_API_KEY=$(openssl rand -hex 12)
LIVEKIT_API_SECRET=$(openssl rand -hex 32)
GOOGLE_CLIENT_ID=
GOOGLE_CLIENT_SECRET=
EOF

echo "Created .env.production with fresh secrets."
echo "Add Google OAuth credentials before enabling public sign-in."
