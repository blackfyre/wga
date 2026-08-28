#!/bin/sh
set -eu

if [ -n "${SENTRY_AUTH_TOKEN:-}" ] && [ -n "${WGA_SENTRY_BROWSER_DSN:-}" ]; then
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
