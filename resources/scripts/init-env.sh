#!/bin/sh
# Create a startable development .env from .env.example in the current
# directory. An existing .env is never modified. The new file receives a
# freshly generated postcard token key, because configuration validation
# requires one even in development.
set -eu

if [ -e .env ]; then
	echo ".env already exists; leaving it unchanged."
	exit 0
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
	echo ".env already exists; leaving it unchanged."
	exit 0
fi
chmod 600 .env

echo "Created .env with a generated development postcard token key."
