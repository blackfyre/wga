#!/bin/sh
set -eu

if [ -n "${WGA_SENTRY_BROWSER_DSN:-}" ]; then
	: "${SENTRY_AUTH_TOKEN:?SENTRY_AUTH_TOKEN must be set when browser Sentry monitoring is enabled}"
	release=$(cat /usr/local/share/wga/release)
	sentry-cli sourcemaps upload \
		--org web-gallery-of-art-modernisation \
		--project wga-browser \
		--release "$release" \
		--url-prefix "~/assets/js" \
		--validate \
		--wait \
		/usr/local/share/wga/sourcemaps
fi

exec "$@"
