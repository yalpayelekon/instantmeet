#!/usr/bin/env bash
set -euo pipefail

base_url="${1:-https://toplanti.online}"
base_url="${base_url%/}"
fail=0

echo "== InstantMeet production smoke against ${base_url} =="

check_json_ok() {
  local path="$1"
  local body
  body="$(curl --fail --silent --show-error --max-time 15 "${base_url}${path}")"
  if [[ "${body}" != *'"status":"ok"'* && "${body}" != *'"status": "ok"'* ]]; then
    echo "FAIL ${path}: expected {\"status\":\"ok\"}, got: ${body}"
    fail=1
    return
  fi
  echo "OK   ${path}: ${body}"
}

check_google_login() {
  local code headers
  headers="$(mktemp)"
  code="$(curl --silent --show-error --max-time 20 \
    --output /dev/null --write-out '%{http_code}' \
    --dump-header "${headers}" \
    "${base_url}/api/login/google" || true)"

  if grep -qi '^Location:.*accounts\.google\.com' "${headers}"; then
    echo "OK   /api/login/google → Google OAuth redirect (${code})"
    rm -f "${headers}"
    return
  fi

  body="$(curl --silent --show-error --max-time 15 "${base_url}/api/login/google" || true)"
  if [[ "${body}" == *'Google OAuth is not configured'* ]]; then
    echo "FAIL /api/login/google: Google OAuth is not configured on the host."
    echo "     Set GOOGLE_CLIENT_ID / GOOGLE_CLIENT_SECRET in .env.production and redeploy."
    fail=1
  else
    echo "FAIL /api/login/google: expected Google redirect, got HTTP ${code}"
    echo "     body: ${body}"
    head -n 20 "${headers}" || true
    fail=1
  fi
  rm -f "${headers}"
}

check_json_ok /healthz || true
check_google_login || true

echo
if [[ "${fail}" -ne 0 ]]; then
  echo "Smoke failed. Complete the manual two-user checklist in docs/deploy.md after fixing the issues above."
  exit 1
fi

echo "API smoke passed."
echo "Still run the manual two-user checklist in docs/deploy.md (create, admit, media, chat, end)."
