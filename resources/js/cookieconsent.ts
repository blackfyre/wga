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

const PREFERENCES_MODAL = "preferencesModal";
const PREFERENCES_CLOSE_SELECTOR = "#cc-main .pm__close-btn";

// The library's close control is an icon; the panel's CLOSE reads like the
// site's own Preferences panel. The aria-label keeps the full name.
const labelPreferencesClose = (modal: HTMLElement) => {
	const close = modal.querySelector<HTMLElement>(".pm__close-btn");
	close?.replaceChildren("CLOSE");
};

// The library moves focus only on a transitionend, which the site's 0ms
// modal transition never fires, and reveals each modal a few frames after its
// show event. Focus is retried each frame until the target can take it, and
// held there briefly while the library settles.
// Only the latest request runs, so closing the panel straight after opening
// it cannot let the pending move to CLOSE take focus back from the invoker.
let focusRequest = 0;

// Focus counts as lost when it sits on the page body, on an element the
// library has just hidden, or on the non-interactive sentinel the library
// focuses 100 ms after showing either dialog. A visible control the visitor
// moved to is left alone.
const LIBRARY_CONTROLS = "button, a[href], input, select, textarea";

const focusLost = (active: Element | null) =>
	active === null ||
	active === document.body ||
	!active.isConnected ||
	(active.closest("#cc-main") !== null && !active.matches(LIBRARY_CONTROLS)) ||
	(active instanceof HTMLElement &&
		typeof active.checkVisibility === "function" &&
		!active.checkVisibility({ visibilityProperty: true }));

const focusWhenVisible = (target: () => HTMLElement | null) => {
	focusRequest += 1;
	const request = focusRequest;
	const started = performance.now();
	let settled: number | null = null;
	const attempt = () => {
		if (request !== focusRequest) {
			return;
		}
		const element = target();
		const active = document.activeElement;
		if (
			element &&
			active !== element &&
			(settled === null || focusLost(active))
		) {
			element.focus({ preventScroll: true });
		}
		const now = performance.now();
		if (document.activeElement === element && settled === null) {
			settled = now;
		}
		const settling = settled !== null && now - settled < 500;
		const searching = settled === null && now - started < 2000;
		if (settling || searching) {
			window.requestAnimationFrame(attempt);
		}
	};
	attempt();
};

// The control that opened the panel: the notice's PREFERENCES or the footer's
// Cookie settings. Focus returns there when the panel closes.
let preferencesInvoker: HTMLElement | null = null;

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
				layout: "bar",
				position: "right",
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
		onModalReady: ({ modalName, modal }) => {
			if (modalName === PREFERENCES_MODAL) {
				labelPreferencesClose(modal);
			}
		},
		// The notice steps aside while the panel is open, so nothing outside
		// the panel stays operable, and returns if the panel closes before a
		// choice is recorded.
		onModalShow: ({ modalName }) => {
			if (modalName !== PREFERENCES_MODAL) {
				return;
			}
			preferencesInvoker =
				document.activeElement instanceof HTMLElement &&
				document.activeElement !== document.body
					? document.activeElement
					: null;
			CookieConsent.hide();
			focusWhenVisible(() =>
				document.querySelector<HTMLElement>(PREFERENCES_CLOSE_SELECTOR),
			);
		},
		onModalHide: ({ modalName }) => {
			if (modalName !== PREFERENCES_MODAL) {
				return;
			}
			const invoker = preferencesInvoker;
			preferencesInvoker = null;
			const noticeReturns = !CookieConsent.validConsent();
			if (noticeReturns) {
				CookieConsent.show();
			}
			// A choice made after opening the panel from the notice closes the
			// notice for good, so its PREFERENCES button cannot take focus back;
			// focus moves to Cookie settings, where the choice can be revisited.
			const invokerLeftWithNotice =
				!noticeReturns && invoker?.closest("#cc-main .cm") !== null;
			const target =
				invoker?.isConnected && !invokerLeftWithNotice
					? invoker
					: document.querySelector<HTMLElement>("[data-wga-cookie-settings]");
			if (target) {
				focusWhenVisible(() => target);
			}
		},
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
