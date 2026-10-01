#!/bin/sh
# Create a startable development .env from .env.example in the current
# directory. An existing .env is never modified. The new file receives a
# freshly generated postcard token key, because configuration validation
# requires one even in development.
set -eu

# Succeeds when .env assigns the named setting a non-empty value, ignoring
# surrounding whitespace, quotes, and an optional export prefix.
env_sets() {
	grep -Eq "^[[:space:]]*(export[[:space:]]+)?$1[[:space:]]*=[[:space:]]*[\"']?[^\"'[:space:]#]" .env
}

existing_env() {
	echo ".env already exists; leaving it unchanged."
	# An older app:init-env copied .env.example verbatim, and Claude Code
	# copies that file into new worktrees, so flag a keyring the server rejects.
	if ! env_sets WGA_POSTCARD_TOKEN_KEYS || ! env_sets WGA_POSTCARD_TOKEN_ACTIVE_KEY_ID; then
		echo "warning: .env does not set WGA_POSTCARD_TOKEN_KEYS and WGA_POSTCARD_TOKEN_ACTIVE_KEY_ID;" >&2
		echo "warning: the server will not start unless the environment provides them." >&2
		echo "warning: delete .env and rerun mise run app:init-env, or set both as described in .env.example." >&2
	fi
	exit 0
}

if [ -e .env ]; then
	existing_env
fi

if [ ! -f .env.example ]; then
	echo ".env.example not found in $(pwd)." >&2
	exit 1
fi

# 32 random bytes as canonical, unpadded Base64URL.
key=$(head -c 32 /dev/urandom | base64 | tr -d '\n=' | tr '+/' '-_')
if [ "${#key}" -ne 43 ]; then
	echo "Failed to generate a postcard token key." >&2
	exit 1
fi
key_id=dev

tmp=$(mktemp .env.XXXXXX)
trap 'rm -f "$tmp"' EXIT INT TERM

sed \
	-e "s|^WGA_POSTCARD_TOKEN_KEYS=\$|WGA_POSTCARD_TOKEN_KEYS='{\"${key_id}\":\"${key}\"}'|" \
	-e "s|^WGA_POSTCARD_TOKEN_ACTIVE_KEY_ID=\$|WGA_POSTCARD_TOKEN_ACTIVE_KEY_ID=${key_id}|" \
	.env.example >"$tmp"

# Refuse to replace a .env created concurrently after the check above.
if ! ln "$tmp" .env 2>/dev/null; then
	existing_env
fi
chmod 600 .env

echo "Created .env with a generated development postcard token key."
