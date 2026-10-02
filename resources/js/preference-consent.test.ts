import { afterAll, beforeEach, describe, expect, test } from "bun:test";
import {
	applyPreferenceConsent,
	consentAllowsPreferences,
	expireCookie,
	readPreference,
	registerPreferenceStore,
	removePreference,
	resetPreferenceConsentForTests,
	writePreference,
} from "./preference-consent";
import {
	consentCookieValue,
	installPreferenceBrowser,
	type PreferenceBrowser,
	restorePreferenceBrowser,
} from "./testing/preference-browser";

afterAll(restorePreferenceBrowser);

test("parses only a consent record that accepts preferences", () => {
	const consent = (categories: string[]) =>
		`cc_cookie=${consentCookieValue(categories)}`;
	expect(consentAllowsPreferences("")).toBe(false);
	expect(consentAllowsPreferences(consent(["necessary"]))).toBe(false);
	expect(
		consentAllowsPreferences(`a=1; ${consent(["necessary", "preferences"])}`),
	).toBe(true);
	expect(consentAllowsPreferences("cc_cookie=%7Bbroken")).toBe(false);
	expect(
		consentAllowsPreferences(
			`cc_cookie=${encodeURIComponent('{"categories":"preferences"}')}`,
		),
	).toBe(false);
});

describe("gated storage", () => {
	let browser: PreferenceBrowser;
	beforeEach(() => {
		browser = installPreferenceBrowser();
		resetPreferenceConsentForTests();
	});

	test("refuses to read or write without consent but always removes", () => {
		browser.storage.set("wga-theme", "dark");
		expect(readPreference("wga-theme")).toBeNull();
		writePreference("wga-palette", "gothic");
		expect(browser.storage.has("wga-palette")).toBe(false);
		removePreference("wga-theme");
		expect(browser.storage.has("wga-theme")).toBe(false);
	});

	test("reads consent live, so another tab's decision applies at once", () => {
		writePreference("wga-theme", "dark");
		expect(browser.storage.has("wga-theme")).toBe(false);

		// Another tab grants consent.
		browser.grantConsent();
		writePreference("wga-theme", "dark");
		expect(readPreference("wga-theme")).toBe("dark");

		// Another tab withdraws it; its forget deleted the shared copy, and
		// this tab's later writes are refused.
		browser.withdrawConsent();
		expect(readPreference("wga-theme")).toBeNull();
		writePreference("wga-theme", "light");
		expect(browser.storage.get("wga-theme")).toBe("dark");
	});

	test("expires a cookie", () => {
		browser.jar.set("wga_theme", "dark");
		expireCookie("wga_theme");
		expect(browser.jar.has("wga_theme")).toBe(false);
	});
});

describe("store transitions", () => {
	let browser: PreferenceBrowser;
	let calls: string[];
	const store = () => ({
		remember: () => calls.push("remember"),
		forget: () => calls.push("forget"),
	});
	beforeEach(() => {
		browser = installPreferenceBrowser();
		resetPreferenceConsentForTests();
		calls = [];
	});

	test("registering without consent forgets stale copies", () => {
		registerPreferenceStore(store());
		expect(calls).toEqual(["forget"]);
	});

	test("a first grant remembers; a confirmation does not", () => {
		registerPreferenceStore(store());
		calls = [];
		browser.grantConsent();
		applyPreferenceConsent(true);
		expect(calls).toEqual(["remember"]);
		applyPreferenceConsent(true);
		expect(calls).toEqual(["remember"]);
	});

	test("consent that existed at load is only confirmed", () => {
		browser.grantConsent();
		registerPreferenceStore(store());
		registerPreferenceStore(store());
		applyPreferenceConsent(true);
		expect(calls).toEqual([]);
	});

	test("withdrawal forgets every store, and a later grant remembers again", () => {
		browser.grantConsent();
		registerPreferenceStore(store());
		registerPreferenceStore(store());
		browser.withdrawConsent();
		applyPreferenceConsent(false);
		expect(calls).toEqual(["forget", "forget"]);
		calls = [];
		browser.grantConsent();
		applyPreferenceConsent(true);
		expect(calls).toEqual(["remember", "remember"]);
	});
});
