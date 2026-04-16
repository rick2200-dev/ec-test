#!/bin/bash
# Phase 1.4: fail CI when a service's source references another service's
# Postgres schema. Enforces the boundary Phase 1.1 established via GRANTs
# (one role per service) and Phase 1.2 completed for the order service.
#
# Allowlist entries document transitional violations that a later phase
# will remove — drop the entry as soon as that phase lands.
set -euo pipefail

# Schemas each service owns. Own-schema references are always fine.
declare -A OWNED=(
  [auth]='auth_svc'
  [catalog]='catalog_svc'
  [inventory]='inventory_svc'
  [order]='order_svc'
  [subscription]='subscription_svc'
  [inquiry]='inquiry_svc'
  [review]='review_svc'
  [shipping]='shipping_svc'
  [notification]='notification_svc'
)

# Transitional allowlist. Value is a regex of allowed cross-schema matches.
# Remove each entry when its phase eliminates the violation.
declare -A ALLOW=(
  # Phase 2.1 (catalog outbox): seller_plan_boost matview reads auth +
  # subscription. Swap for local projection when outbox lands.
  [catalog]='(auth_svc\.sellers|auth_svc\.seller_subscriptions|subscription_svc\.subscription_plans)'
  # Phase 2.2 (search/recommend local read model): both services project
  # catalog data and search joins auth sellers for ranking. Replace with a
  # dedicated indexing pipeline.
  [search]='(catalog_svc\.(products|skus|categories|seller_plan_boost)|auth_svc\.sellers)'
  [recommend]='catalog_svc\.(products|skus|popular_products|product_categories)'
  # Phase 2.1: subscription refreshes the catalog-owned matview when plans
  # change. Goes away when catalog owns its own projection.
  [subscription]='catalog_svc\.refresh_seller_plan_boost'
)

# Any reference to a *_svc schema.
SCHEMAS_RE='(auth|catalog|inventory|order|subscription|inquiry|review|shipping|notification)_svc\.'

exit_code=0
for svc_dir in backend/services/*/; do
  svc=$(basename "$svc_dir")
  owned="${OWNED[$svc]:-}"
  allow="${ALLOW[$svc]:-}"

  raw=$(grep -rEn "$SCHEMAS_RE" "$svc_dir" --include='*.go' --include='*.sql' 2>/dev/null || true)
  # Skip generated proto code under api/gen/; it never references DB schemas
  # but some services embed schema names in comments that could match.
  raw=$(printf '%s\n' "$raw" | grep -v '/api/gen/' || true)
  # Drop pure-comment lines. Format is file:lineno:content — strip the prefix
  # then check if the content is a Go (//) or SQL (--) comment.
  raw=$(printf '%s\n' "$raw" | awk -F: '
    NF < 3 { print; next }
    {
      content = $3
      for (i = 4; i <= NF; i++) content = content ":" $i
      t = content
      sub(/^[ \t]+/, "", t)
      if (t ~ /^\/\// || t ~ /^--/) next
      print
    }' || true)

  filtered="$raw"
  if [ -n "$owned" ]; then
    filtered=$(printf '%s\n' "$filtered" | grep -Ev "${owned}\\." || true)
  fi
  if [ -n "$allow" ]; then
    filtered=$(printf '%s\n' "$filtered" | grep -Ev "$allow" || true)
  fi
  # Drop empty lines left by the filters above.
  filtered=$(printf '%s\n' "$filtered" | sed '/^$/d')

  if [ -n "$filtered" ]; then
    echo "=== ${svc}: schema boundary violations ==="
    echo "$filtered"
    exit_code=1
  fi
done

if [ $exit_code -eq 0 ]; then
  echo "OK: no cross-schema references outside the documented allowlist."
fi
exit $exit_code
