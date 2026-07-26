import * as CookieConsent from "vanilla-cookieconsent";

export const initCookieConsent = () => {
	CookieConsent.run({
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
						title: "We use cookies",
						description:
							'We use essential cookies to make this site work. Read our <a href="/pages/privacy-policy">privacy policy</a>.',
						acceptNecessaryBtn: "Accept essential cookies",
						showPreferencesBtn: "Cookie preferences",
					},
					preferencesModal: {
						title: "Cookie preferences",
						acceptNecessaryBtn: "Accept essential cookies",
						savePreferencesBtn: "Save preferences",
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
	});
};
