// Every remembered preference (search toolbar, colour scheme, palette,
// bionic reading, Study Board) is optional storage, gated on the
// cookie-consent "preferences" category. This module is the single gate:
//
// - It reads the CookieConsent record live on every use, because the stores
//   start before the consent library has loaded and because another tab may
//   grant or withdraw consent at any time.
// - Its storage helpers refuse to read or write without consent; removing is
//   always allowed.
// - Each store registers how to remember the state its page shows and how to
//   forget every stored copy. Registering without consent forgets at once, so
//   a stale copy never survives. The consent dialog then reports decisions
//   through applyPreferenceConsent.

export const CONSENT_COOKIE_NAME = "cc_cookie";
export const PREFERENCES_CATEGORY = "preferences";

export type PreferenceStore = {
	// remember stores the state the page currently shows.
	remember: () => void;
	// forget deletes every stored copy; the page keeps its state.
	forget: () => void;
};

const isRecord = (value: unknown): value is Record<string, unknown> =>
	typeof value === "object" && value !== null && !Array.isArray(value);

export const readCookie = (cookies: string, name: string): string | null => {
	for (const part of cookies.split(";")) {
		const [key, ...rest] = part.trim().split("=");
		if (key === name) {
			try {
				return decodeURIComponent(rest.join("="));
			} catch {
				return null;
			}
		}
	}
	return null;
};

// consentAllowsPreferences parses the CookieConsent record. An absent or
// unreadable record means no consent.
export const consentAllowsPreferences = (cookies: string): boolean => {
	const raw = readCookie(cookies, CONSENT_COOKIE_NAME);
	if (!raw) {
		return false;
	}
	try {
		const record: unknown = JSON.parse(raw);
		return (
			isRecord(record) &&
			Array.isArray(record.categories) &&
			record.categories.includes(PREFERENCES_CATEGORY)
		);
	} catch {
		return false;
	}
};

export const preferenceStorageAllowed = (): boolean => {
	try {
		return consentAllowsPreferences(document.cookie);
	} catch {
		return false;
	}
};

export const readPreference = (key: string): string | null => {
	if (!preferenceStorageAllowed()) {
		return null;
	}
	try {
		return window.localStorage.getItem(key);
	} catch {
		return null;
	}
};

export const writePreference = (key: string, value: string): void => {
	if (!preferenceStorageAllowed()) {
		return;
	}
	try {
		window.localStorage.setItem(key, value);
	} catch {
		// Storage can be unavailable; the page keeps its in-memory state.
	}
};

export const removePreference = (key: string): void => {
	try {
		window.localStorage.removeItem(key);
	} catch {
		// Nothing to remove when storage is unavailable.
	}
};

export const cookieAttributes = (maxAge: number): string => {
	const secure = window.location.protocol === "https:" ? "; Secure" : "";
	return `Path=/; Max-Age=${maxAge}; SameSite=Lax${secure}`;
};

export const expireCookie = (name: string): void => {
	// biome-ignore lint/suspicious/noDocumentCookie: the Cookie Store API is not available in every supported browser.
	document.cookie = `${name}=; ${cookieAttributes(0)}`;
};

// syncPreferenceStorageState mirrors the consent state on <html> so the
// Preferences panel's storage note can say whether its choices are kept.
export const syncPreferenceStorageState = (
	allowed: boolean = preferenceStorageAllowed(),
): void => {
	document.documentElement.dataset.wgaPreferenceStorage = allowed
		? "on"
		: "off";
};

const stores: PreferenceStore[] = [];
// remembering records whether this page last knew consent to be granted, so a
// first grant can be told apart from the library confirming existing consent
// on page load. It is taken when the first store registers, before the
// consent library runs.
let remembering: boolean | null = null;

export const registerPreferenceStore = (store: PreferenceStore): void => {
	const allowed = preferenceStorageAllowed();
	if (remembering === null) {
		remembering = allowed;
		syncPreferenceStorageState(allowed);
	}
	stores.push(store);
	if (!allowed) {
		store.forget();
	}
};

// applyPreferenceConsent applies a decision from the cookie-consent dialog.
// A first grant stores what each page shows, which an explicit URL may have
// set; confirming consent that already existed keeps the remembered state.
// Withdrawing deletes every stored copy while the page keeps its state.
export const applyPreferenceConsent = (allowed: boolean): void => {
	syncPreferenceStorageState(allowed);
	if (allowed) {
		const firstGrant = remembering !== true;
		remembering = true;
		if (firstGrant) {
			for (const store of stores) {
				store.remember();
			}
		}
		return;
	}
	remembering = false;
	for (const store of stores) {
		store.forget();
	}
};

// resetPreferenceConsentForTests clears the module state between unit tests.
export const resetPreferenceConsentForTests = (): void => {
	stores.length = 0;
	remembering = null;
};
