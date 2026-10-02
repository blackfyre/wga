import type { BrowserContext, Page } from "@playwright/test";

// Every remembered preference (search toolbar, colour scheme, palette, bionic
// reading, Study Board) needs the optional "preferences" cookie-consent
// category. These helpers record or change that consent the way
// CookieConsent stores it, before or between page scripts.

const consentCookie = (categories: string[]) => {
	const now = new Date().toISOString();
	return {
		name: "cc_cookie",
		value: encodeURIComponent(
			JSON.stringify({
				categories,
				revision: 0,
				data: null,
				consentTimestamp: now,
				consentId: "00000000-0000-4000-8000-000000000000",
				services: Object.fromEntries(categories.map((name) => [name, []])),
				languageCode: "en",
				lastConsentTimestamp: now,
				expirationTime: Date.now() + 182 * 24 * 60 * 60 * 1000,
			}),
		),
		url: `${process.env.WGA_PROTOCOL}://${process.env.WGA_HOSTNAME}`,
	};
};

const contextOf = (target: Page | BrowserContext): BrowserContext =>
	"context" in target ? target.context() : target;

// grantPreferenceConsent records consent to every category, as ACCEPT ALL
// does, without the consent library's callbacks running.
export async function grantPreferenceConsent(
	target: Page | BrowserContext,
): Promise<void> {
	await contextOf(target).addCookies([
		consentCookie(["necessary", "preferences"]),
	]);
}

// denyPreferenceConsent records consent to the necessary category only, as
// DENY does, so the notice stays closed while preferences are not stored.
export async function denyPreferenceConsent(
	target: Page | BrowserContext,
): Promise<void> {
	await contextOf(target).addCookies([consentCookie(["necessary"])]);
}

// The localStorage entries the "preferences" category governs.
export const PREFERENCE_STORAGE_KEYS = [
	"wga-aw-prefs",
	"wga-theme",
	"wga-palette",
	"wga-bionic",
	"wga-study-board",
] as const;

export async function storedPreferences(
	page: Page,
): Promise<Record<string, string | null>> {
	return page.evaluate(
		(keys) =>
			Object.fromEntries(
				keys.map((key) => [key, window.localStorage.getItem(key)]),
			),
		[...PREFERENCE_STORAGE_KEYS],
	);
}
