import { afterAll, beforeEach, describe, expect, test } from "bun:test";
import {
	clearScheme,
	effectiveScheme,
	forgetAppearance,
	isDarkOnlyPalette,
	PALETTE_NAMES,
	PALETTE_STORAGE_KEY,
	parsePalette,
	parseScheme,
	rememberAppearance,
	SCHEME_STORAGE_KEY,
	setPalette,
	setScheme,
} from "./appearance";
import { resetPreferenceConsentForTests } from "./preference-consent";
import {
	installPreferenceBrowser,
	type PreferenceBrowser,
	restorePreferenceBrowser,
} from "./testing/preference-browser";

afterAll(restorePreferenceBrowser);

describe("remembered appearance", () => {
	let browser: PreferenceBrowser;
	beforeEach(() => {
		browser = installPreferenceBrowser({
			window: {
				getComputedStyle: () => ({ getPropertyValue: () => "" }),
				matchMedia: () => ({ matches: false, addEventListener: () => {} }),
			},
		});
		resetPreferenceConsentForTests();
		clearScheme();
	});

	test("choices apply to the page but are not stored without consent", () => {
		setScheme("dark");
		setPalette("gothic");
		expect(browser.html.dataset.theme).toBe("dark");
		expect(browser.html.dataset.palette).toBe("gothic");
		expect(browser.storage.size).toBe(0);
	});

	test("a grant stores the palette shown and the chosen scheme", () => {
		setScheme("dark");
		setPalette("verdigris");
		browser.grantConsent();
		rememberAppearance();
		expect(browser.storage.get(SCHEME_STORAGE_KEY)).toBe("dark");
		expect(browser.storage.get(PALETTE_STORAGE_KEY)).toBe("verdigris");
	});

	test("a scheme that follows the system stays unstored", () => {
		// A dark-only palette forces data-theme to dark; that is not a choice.
		setPalette("baroque");
		browser.grantConsent();
		rememberAppearance();
		expect(browser.html.dataset.theme).toBe("dark");
		expect(browser.storage.has(SCHEME_STORAGE_KEY)).toBe(false);
		expect(browser.storage.get(PALETTE_STORAGE_KEY)).toBe("baroque");
	});

	test("forgetting deletes both copies and keeps the page's appearance", () => {
		browser.grantConsent();
		setScheme("dark");
		setPalette("rococo");
		expect(browser.storage.size).toBe(2);
		forgetAppearance();
		expect(browser.storage.size).toBe(0);
		expect(browser.html.dataset.theme).toBe("dark");
		expect(browser.html.dataset.palette).toBe("rococo");
	});
});

describe("appearance preference resolution", () => {
	test("accepts current and legacy scheme values", () => {
		expect(parseScheme("light")).toBe("light");
		expect(parseScheme("dark")).toBe("dark");
		expect(parseScheme("wga_light")).toBe("light");
		expect(parseScheme("wga_dark")).toBe("dark");
		expect(parseScheme("invalid")).toBeNull();
	});

	test("accepts only the eleven stable palette keys", () => {
		expect(PALETTE_NAMES).toHaveLength(11);
		for (const palette of PALETTE_NAMES) {
			expect(parsePalette(palette)).toBe(palette);
		}
		expect(parsePalette("unknown")).toBeNull();
	});

	test("forces only Baroque and Tokyo to a dark effective scheme", () => {
		expect(isDarkOnlyPalette("baroque")).toBeTrue();
		expect(isDarkOnlyPalette("tokyo")).toBeTrue();
		expect(isDarkOnlyPalette("bone")).toBeFalse();
		expect(effectiveScheme("light", "baroque")).toBe("dark");
		expect(effectiveScheme("light", "tokyo")).toBe("dark");
		expect(effectiveScheme("light", "bone")).toBe("light");
	});
});
