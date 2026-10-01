import type { Page } from "@playwright/test";

// Remembering artwork search choices needs the optional "preferences"
// cookie-consent category. This records that consent the way CookieConsent
// stores it, before any page script runs.
export async function grantPreferenceConsent(page: Page): Promise<void> {
	const now = new Date().toISOString();
	await page.context().addCookies([
		{
			name: "cc_cookie",
			value: encodeURIComponent(
				JSON.stringify({
					categories: ["necessary", "preferences"],
					revision: 0,
					data: null,
					consentTimestamp: now,
					consentId: "00000000-0000-4000-8000-000000000000",
					services: { necessary: [], preferences: [] },
					languageCode: "en",
					lastConsentTimestamp: now,
					expirationTime: Date.now() + 182 * 24 * 60 * 60 * 1000,
				}),
			),
			url: `${process.env.WGA_PROTOCOL}://${process.env.WGA_HOSTNAME}`,
		},
	]);
}

// Artwork search result actions are opt-in. Specifications that exercise the
// per-result itinerary or Study Board controls opt in the way a visitor's
// remembered choice does, before any page script runs. An existing remembered
// choice is left alone, so later navigations keep what the page itself wrote.
export async function showArtworkResultActions(page: Page): Promise<void> {
	await grantPreferenceConsent(page);
	await page.addInitScript(() => {
		const key = "wga-aw-prefs";
		if (window.localStorage.getItem(key) === null) {
			window.localStorage.setItem(
				key,
				JSON.stringify({
					sort: "title",
					dir: "asc",
					view: "grid",
					actions: true,
				}),
			);
		}
	});
}
