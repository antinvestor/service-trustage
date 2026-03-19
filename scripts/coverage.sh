#!/usr/bin/env bash

set -euo pipefail

mode="${1:-app}"
coverage_min="${COVERAGE_MIN:-85}"

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

raw_profile="coverage.out"
app_profile="coverage_handwritten.out"

test_targets=(
  ./apps/default/tests
  ./apps/formstore/tests
  ./apps/queue/tests
  ./apps/default/service/authz
  ./apps/formstore/service/authz
  ./apps/queue/service/authz
  ./apps/default/service/handlers
  ./apps/default/service/queues
  ./apps/formstore/service/business
  ./apps/default/service/business
  ./connector/adapters
  ./pkg/cacheutil
  ./pkg/cryptoutil
  ./pkg/telemetry
  ./dsl
)

build_coverpkg() {
  go list ./... \
    | grep -v '/gen/' \
    | grep -v '/proto/' \
    | grep -v '/baml_client' \
    | grep -v '/cmd$' \
    | grep -v '/config$' \
    | grep -v '/tests/testketo$' \
    | grep -v '/service/cache$' \
    | grep -v '/apps/formstore/service/handlers$' \
    | grep -v '/apps/queue/service/handlers$' \
    | paste -sd, -
}

extract_total() {
  local profile="$1"
  go tool cover -func="$profile" | awk '/^total:/ {print $3}'
}

run_raw() {
  go test ./... -coverprofile="$raw_profile"
  local total
  total="$(extract_total "$raw_profile")"
  printf 'Raw whole-repo coverage: %s\n' "$total"
}

run_app() {
  local coverpkg
  coverpkg="$(build_coverpkg)"

  go test "${test_targets[@]}" -coverpkg="$coverpkg" -coverprofile="$app_profile"

  local total
  total="$(extract_total "$app_profile")"

  printf 'Handwritten app coverage: %s\n' "$total"
  printf 'Profile: %s\n' "$app_profile"
}

run_check() {
  run_app

  local total
  total="$(extract_total "$app_profile")"
  local numeric_total
  numeric_total="${total%%%}"

  if awk -v actual="$numeric_total" -v minimum="$coverage_min" 'BEGIN { exit !(actual + 0 >= minimum + 0) }'; then
    printf 'Coverage gate passed: %.1f%% >= %s%%\n' "$numeric_total" "$coverage_min"
    return 0
  fi

  printf 'Coverage gate failed: %.1f%% < %s%%\n' "$numeric_total" "$coverage_min" >&2
  exit 1
}

case "$mode" in
  raw)
    run_raw
    ;;
  app)
    run_app
    ;;
  check)
    run_check
    ;;
  *)
    printf 'usage: %s [raw|app|check]\n' "${BASH_SOURCE[0]}" >&2
    exit 2
    ;;
esac
