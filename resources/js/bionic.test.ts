import { afterAll, beforeEach, expect, test } from "bun:test";
import {
	BIONIC_STORAGE_KEY,
	forgetBionicReading,
	rememberBionicReading,
} from "./bionic";
import { resetPreferenceConsentForTests } from "./preference-consent";
import {
	installPreferenceBrowser,
	type PreferenceBrowser,
	restorePreferenceBrowser,
} from "./testing/preference-browser";

afterAll(restorePreferenceBrowser);

let browser: PreferenceBrowser;
beforeEach(() => {
	browser = installPreferenceBrowser();
	resetPreferenceConsentForTests();
});

test("remembers the shown bionic state only with consent", () => {
	browser.html.dataset.bionicReading = "true";
	rememberBionicReading();
	expect(browser.storage.has(BIONIC_STORAGE_KEY)).toBe(false);

	browser.grantConsent();
	rememberBionicReading();
	expect(browser.storage.get(BIONIC_STORAGE_KEY)).toBe("on");

	browser.html.dataset.bionicReading = "false";
	rememberBionicReading();
	expect(browser.storage.get(BIONIC_STORAGE_KEY)).toBe("off");
});

test("forgetting deletes the stored choice and leaves the page alone", () => {
	browser.grantConsent();
	browser.storage.set(BIONIC_STORAGE_KEY, "on");
	browser.html.dataset.bionicReading = "true";
	forgetBionicReading();
	expect(browser.storage.has(BIONIC_STORAGE_KEY)).toBe(false);
	expect(browser.html.dataset.bionicReading).toBe("true");
});
