#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

if [[ ! -f .env.production ]]; then
  echo "Missing .env.production."
  echo "Run: bash scripts/init-production-env.sh you@example.com"
  echo "Then set GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET."
  exit 1
fi

# shellcheck disable=SC1091
set -a
source .env.production
set +a

require_nonempty() {
  local name="$1"
  local value="${!name-}"
  if [[ -z "${value}" ]]; then
    echo "Missing required ${name} in .env.production."
    exit 1
  fi
}

require_nonempty APP_DOMAIN
require_nonempty LIVEKIT_DOMAIN
require_nonempty ACME_EMAIL
require_nonempty POSTGRES_PASSWORD
require_nonempty JWT_SECRET
require_nonempty LIVEKIT_API_KEY
require_nonempty LIVEKIT_API_SECRET
require_nonempty GRAFANA_ADMIN_PASSWORD
require_nonempty GOOGLE_CLIENT_ID
require_nonempty GOOGLE_CLIENT_SECRET

if [[ "${#JWT_SECRET}" -lt 32 ]]; then
  echo "JWT_SECRET must be at least 32 characters."
  exit 1
fi

if [[ "${#GRAFANA_ADMIN_PASSWORD}" -lt 12 ]]; then
  echo "GRAFANA_ADMIN_PASSWORD must be at least 12 characters."
  exit 1
fi
echo "Validated .env.production (including Google OAuth)."
docker compose --env-file .env.production -f docker-compose.prod.yml config --quiet

build_succeeded=false
for attempt in 1 2 3; do
  echo "Building production images (attempt ${attempt}/3)..."
  if docker compose --env-file .env.production -f docker-compose.prod.yml build; then
    build_succeeded=true
    break
  fi
  if [[ "${attempt}" -lt 3 ]]; then
    echo "Build failed; retrying in 5 seconds. Existing containers remain untouched."
    sleep 5
  fi
done

if [[ "${build_succeeded}" != "true" ]]; then
  echo "Production build failed after 3 attempts. Existing containers were not replaced."
  exit 1
fi

docker compose --env-file .env.production -f docker-compose.prod.yml up -d --no-build --remove-orphans
docker compose --env-file .env.production -f docker-compose.prod.yml ps
