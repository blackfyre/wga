import type { Page } from "@playwright/test";
import { grantPreferenceConsent } from "./preference-consent";

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
