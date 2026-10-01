import * as CookieConsent from "vanilla-cookieconsent";
import { PREFERENCES_CATEGORY, setSearchPrefsConsent } from "./search-prefs";

const revealSettingsControl = () => {
	for (const control of document.querySelectorAll<HTMLElement>(
		"[data-wga-cookie-settings]",
	)) {
		control.classList.remove("hidden");
		control.removeAttribute("aria-hidden");
		control.removeAttribute("tabindex");
	}
};

const hasConsentUI = () => document.querySelector("#cc-main .cm") !== null;

const applyPreferenceConsent = () =>
	setSearchPrefsConsent(CookieConsent.acceptedCategory(PREFERENCES_CATEGORY));

export const initCookieConsent = async () => {
	const result = (await CookieConsent.run({
		guiOptions: {
			consentModal: {
				layout: "box inline",
				position: "bottom left",
				equalWeightButtons: false,
				flipButtons: false,
			},
			preferencesModal: {
				layout: "box",
				equalWeightButtons: false,
				flipButtons: false,
			},
		},
		categories: {
			necessary: {
				enabled: true,
				readOnly: true,
			},
			[PREFERENCES_CATEGORY]: {
				enabled: false,
				readOnly: false,
			},
		},
		onConsent: applyPreferenceConsent,
		onChange: applyPreferenceConsent,
		language: {
			default: "en",
			translations: {
				en: {
					consentModal: {
						title: "COOKIES",
						description:
							'Essential cookies keep this site working. Analytics cookies are not in use. With your permission, optional preference storage remembers your artwork search sort, view, and whether result actions are shown, on this device; turning it off in cookie preferences deletes it. Read our <a href="/pages/privacy-policy">privacy policy</a>.',
						acceptAllBtn: "ACCEPT ALL",
						acceptNecessaryBtn: "ACCEPT ESSENTIAL COOKIES",
						showPreferencesBtn: "COOKIE PREFERENCES",
					},
					preferencesModal: {
						title: "COOKIE PREFERENCES",
						acceptAllBtn: "ACCEPT ALL",
						acceptNecessaryBtn: "ACCEPT ESSENTIAL COOKIES",
						savePreferencesBtn: "SAVE PREFERENCES",
						closeIconLabel: "Close cookie preferences",
						sections: [
							{
								title: "Cookie use",
								description:
									"Essential cookies keep this site working. Optional preference storage stays off unless you turn it on.",
							},
							{
								title: "Strictly necessary cookies",
								description:
									"These cookies are required for the site to function and cannot be disabled.",
								linkedCategory: "necessary",
							},
							{
								title: "Preference storage",
								description:
									"Remembers your artwork search sort, view, and whether result actions are shown, in the wga_aw_prefs cookie and this browser's local storage. Turning it off deletes both; the settings then apply to the current page only.",
								linkedCategory: PREFERENCES_CATEGORY,
							},
							{
								title: "More information",
								description:
									'Read our <a href="/pages/privacy-policy">privacy policy</a>.',
							},
						],
					},
				},
			},
		},
	})) as unknown;

	if (result === false || (!hasConsentUI() && !CookieConsent.validConsent())) {
		return false;
	}

	revealSettingsControl();
	return true;
};
