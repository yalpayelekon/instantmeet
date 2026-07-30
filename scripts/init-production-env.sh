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

# Generate each secret independently; do not reuse values.
POSTGRES_PASSWORD=$(openssl rand -hex 24)
JWT_SECRET=$(openssl rand -hex 32)
LIVEKIT_API_KEY=$(openssl rand -hex 12)
LIVEKIT_API_SECRET=$(openssl rand -hex 32)
GRAFANA_ADMIN_PASSWORD=$(openssl rand -hex 24)

# Required before deploy.sh will succeed.
# Google Cloud Console → APIs & Services → Credentials → OAuth 2.0 Client ID (Web).
# Authorized redirect URI:
#   https://toplanti.online/api/auth/google/callback
GOOGLE_CLIENT_ID=
GOOGLE_CLIENT_SECRET=
EOF

echo "Created .env.production with fresh secrets."
echo
echo "Next steps:"
echo "  1. Create a Google OAuth Web client."
echo "  2. Add redirect URI https://toplanti.online/api/auth/google/callback"
echo "  3. Set GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET in .env.production"
echo "  4. Run: bash scripts/deploy.sh"
echo "  5. Run: bash scripts/smoke-production.sh"
