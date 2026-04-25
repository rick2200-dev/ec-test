#!/usr/bin/env bash
# call_copilot.sh — wrapper around the GitHub Copilot CLI for skill-eval.
#
# IMPORTANT: This wrapper is the ONE place to adjust the actual `copilot` CLI
# invocation. The exact flag names depend on the version of the CLI installed
# in the runner. Run `copilot --help` and update the `copilot ...` line below
# if your version differs.
#
# Inputs (named flags):
#   --system <file>       path to a file holding the system prompt
#   --user <file>         path to a file holding the user prompt
#   --model <id>          model id (e.g. claude-sonnet-4.6)
#   --max-tokens <n>      max output tokens (best-effort; pass-through)
#   --out <file>          path to write the assistant response (verbatim stdout)
#   --workdir <dir>       working directory the CLI is invoked from (isolation
#                         knob — typically a clean tmp dir per call so the
#                         agentic CLI doesn't auto-discover the repo's files)
#
# Exit codes:
#   0  success — stdout written to --out
#   1  invocation failure or empty output
#
# Isolation contract:
#   The caller is expected to pass --workdir pointing to an empty directory
#   that has NO repo content (i.e. NOT the runner's checkout path). The
#   wrapper cd's into that directory before invoking the CLI so any tool the
#   CLI uses (file read, glob, search) sees only the empty workspace.

set -euo pipefail

SYSTEM_FILE=""
USER_FILE=""
MODEL=""
MAX_TOKENS="2048"
OUT=""
WORKDIR=""

while [ $# -gt 0 ]; do
  case "$1" in
    --system)     SYSTEM_FILE="$2"; shift 2 ;;
    --user)       USER_FILE="$2";   shift 2 ;;
    --model)      MODEL="$2";       shift 2 ;;
    --max-tokens) MAX_TOKENS="$2";  shift 2 ;;
    --out)        OUT="$2";         shift 2 ;;
    --workdir)    WORKDIR="$2";     shift 2 ;;
    *)            echo "unknown arg: $1" >&2; exit 2 ;;
  esac
done

[ -s "$SYSTEM_FILE" ] || { echo "system file empty or missing: $SYSTEM_FILE" >&2; exit 1; }
[ -s "$USER_FILE"   ] || { echo "user file empty or missing: $USER_FILE"   >&2; exit 1; }
[ -n "$MODEL"       ] || { echo "--model is required" >&2; exit 1; }
[ -n "$OUT"         ] || { echo "--out is required"   >&2; exit 1; }
[ -n "$WORKDIR"     ] || { echo "--workdir is required (isolation)" >&2; exit 1; }

mkdir -p "$WORKDIR" "$(dirname "$OUT")"

# Resolve to absolute paths BEFORE we cd into the isolation workdir
SYSTEM_ABS=$(realpath "$SYSTEM_FILE")
USER_ABS=$(realpath "$USER_FILE")
OUT_ABS=$(realpath -m "$OUT")

cd "$WORKDIR"

# ============================================================================
# CLI INVOCATION — uses the official GitHub Copilot CLI (npm: @github/copilot)
# ============================================================================
# Reference: https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-command-reference
#            https://docs.github.com/en/copilot/how-tos/copilot-cli/allowing-tools
#
# Flags used:
#   -p / --prompt=PROMPT       Execute a prompt programmatically (exits after completion)
#   --model=MODEL              Model id (e.g. claude-sonnet-4.6 or "auto")
#   --no-ask-user              Disable ask_user tool (autonomous run)
#   -s                         Silent (suppress metadata; output = response only)
#   --add-dir=PATH             Restrict file-system tool access to listed dirs
#   --allow-tool='read'        ALLOW read only (constrained to --add-dir paths)
#   --deny-tool='shell, write, edit, web_fetch, web_search'
#                              EXPLICITLY DENY shell + write + URL fetch + web search.
#                              This is critical: --allow-all-tools would let a
#                              prompt-injected SKILL.md run shell commands and
#                              cat the checked-out repo (.claude/skills/<other>/...)
#                              or exfiltrate the COPILOT_GITHUB_TOKEN via web_fetch.
#                              We allow ONLY `read`, scoped to the empty isolation
#                              workdir via --add-dir.
#
# IMPORTANT spec gaps:
#   - There is NO --system-prompt flag. We embed the system context inside the
#     -p prompt with explicit "## Operating Context" / "## Task" headers.
#   - There is NO --max-tokens flag. The MAX_TOKENS arg is recorded into the
#     tokens.json sidecar (for accounting only); the model uses its own ceiling.
#
# Authentication:
#   The CLI reads the token from env vars in this order of precedence:
#     1. COPILOT_GITHUB_TOKEN
#     2. GH_TOKEN
#     3. GITHUB_TOKEN
#   The token MUST belong to a user with an active Copilot subscription. The
#   default Actions-injected GITHUB_TOKEN (github-actions[bot]) does NOT have
#   Copilot, so the workflow passes a user PAT via COPILOT_GITHUB_TOKEN.

SYSTEM_PROMPT=$(cat "$SYSTEM_ABS")
USER_PROMPT=$(cat "$USER_ABS")

if ! command -v copilot >/dev/null 2>&1; then
  echo "copilot CLI not found in PATH. Add a 'Setup Copilot CLI' step that installs @github/copilot." >&2
  exit 1
fi

# Combine system + user into a single prompt (no --system-prompt flag exists)
COMBINED_PROMPT=$(printf '## Operating Context\n%s\n\n## Task\n%s\n' "$SYSTEM_PROMPT" "$USER_PROMPT")

# Single hand-off point. The order of flags matters: -p must come last and
# its value is the entire combined prompt.
copilot \
  --model "$MODEL" \
  --allow-tool 'read' \
  --deny-tool 'shell, write, edit, web_fetch, web_search' \
  --no-ask-user \
  --add-dir "$WORKDIR" \
  -s \
  -p "$COMBINED_PROMPT" \
  > "$OUT_ABS" \
  2> "${OUT_ABS}.stderr"

# ============================================================================

if [ ! -s "$OUT_ABS" ]; then
  echo "copilot CLI returned empty output to $OUT_ABS" >&2
  exit 1
fi

# ============================================================================
# Token usage sidecar — written next to --out as <out>.tokens.json
# ----------------------------------------------------------------------------
# We always record character counts (cheap, deterministic) and a rough token
# estimate (4 chars/token, English-skewed; treat as ballpark only).
#
# If your `copilot` CLI version emits actual token counts (e.g. on stderr in
# verbose mode, or via a `--report-usage` flag), capture them here and add
# `actual_input_tokens` / `actual_output_tokens` to the JSON. score.py prefers
# actual_* over estimated_* when present.
# ============================================================================
SYSTEM_CHARS=$(wc -c < "$SYSTEM_ABS" | tr -d ' ')
USER_CHARS=$(wc -c < "$USER_ABS" | tr -d ' ')
OUTPUT_CHARS=$(wc -c < "$OUT_ABS" | tr -d ' ')
INPUT_CHARS=$((SYSTEM_CHARS + USER_CHARS))

cat > "${OUT_ABS}.tokens.json" <<EOF
{
  "model": "$MODEL",
  "system_chars": $SYSTEM_CHARS,
  "user_chars": $USER_CHARS,
  "input_chars": $INPUT_CHARS,
  "output_chars": $OUTPUT_CHARS,
  "estimated_input_tokens": $((INPUT_CHARS / 4)),
  "estimated_output_tokens": $((OUTPUT_CHARS / 4)),
  "max_tokens_setting": $MAX_TOKENS
}
EOF
