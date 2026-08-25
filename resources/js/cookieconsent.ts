import * as CookieConsent from "vanilla-cookieconsent";

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
		},
		language: {
			default: "en",
			translations: {
				en: {
					consentModal: {
						title: "COOKIES",
						description:
							'Essential cookies keep this site working. Analytics cookies are not in use. Read our <a href="/pages/privacy-policy">privacy policy</a>.',
						acceptNecessaryBtn: "ACCEPT ESSENTIAL COOKIES",
						showPreferencesBtn: "COOKIE PREFERENCES",
					},
					preferencesModal: {
						title: "COOKIE PREFERENCES",
						acceptNecessaryBtn: "ACCEPT ESSENTIAL COOKIES",
						savePreferencesBtn: "SAVE PREFERENCES",
						closeIconLabel: "Close cookie preferences",
						sections: [
							{
								title: "Cookie use",
								description:
									"We only use essential cookies needed for this site to work.",
							},
							{
								title: "Strictly necessary cookies",
								description:
									"These cookies are required for the site to function and cannot be disabled.",
								linkedCategory: "necessary",
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
