import * as CookieConsent from "vanilla-cookieconsent";
import {
	applyPreferenceConsent,
	PREFERENCES_CATEGORY,
} from "./preference-consent";

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

const applyConsentDecision = () =>
	applyPreferenceConsent(CookieConsent.acceptedCategory(PREFERENCES_CATEGORY));

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
		onConsent: applyConsentDecision,
		onChange: applyConsentDecision,
		language: {
			default: "en",
			translations: {
				en: {
					consentModal: {
						title: "COOKIES",
						description:
							'Essential cookies keep this site working. Analytics cookies are not in use. With your permission, optional preference storage remembers your artwork search toolbar, colour scheme, palette, reading aid, and Study Board on this device. DENY, or turning it off under PREFERENCES, deletes them. Read our <a href="/pages/privacy-policy">privacy policy</a>.',
						acceptAllBtn: "ACCEPT ALL",
						acceptNecessaryBtn: "DENY",
						showPreferencesBtn: "PREFERENCES",
					},
					preferencesModal: {
						title: "COOKIE PREFERENCES",
						acceptAllBtn: "ACCEPT ALL",
						acceptNecessaryBtn: "DENY",
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
									"Remembers, on this device, your artwork search toolbar (sort, view, and whether result actions are shown), your colour scheme and palette, the bionic reading aid, and your Study Board. They are kept in this browser's local storage and, for the search toolbar, the wga_aw_prefs cookie. Turning it off, or DENY, deletes them all; the settings then apply to the current visit only.",
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
