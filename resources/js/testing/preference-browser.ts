// A minimal browser for unit tests of the preference stores: a cookie jar
// that honours Max-Age=0, a localStorage map, and an <html> element dataset.
// Tests may extend the installed window and document with what they need.

export type PreferenceBrowser = {
	jar: Map<string, string>;
	storage: Map<string, string>;
	html: { dataset: Record<string, string> };
	grantConsent: () => void;
	withdrawConsent: () => void;
};

export const consentCookieValue = (categories: string[]): string =>
	encodeURIComponent(JSON.stringify({ categories, revision: 0 }));

const globals = globalThis as Record<string, unknown>;
let saved: { window: unknown; document: unknown } | null = null;

// restorePreferenceBrowser puts back the globals the first install replaced,
// so later test files see the environment they expect.
export const restorePreferenceBrowser = (): void => {
	if (!saved) {
		return;
	}
	for (const name of ["window", "document"] as const) {
		if (saved[name] === undefined) {
			delete globals[name];
		} else {
			globals[name] = saved[name];
		}
	}
	saved = null;
};

export const installPreferenceBrowser = (
	extra: { window?: object; document?: object } = {},
): PreferenceBrowser => {
	saved ??= { window: globals.window, document: globals.document };
	const jar = new Map<string, string>();
	const storage = new Map<string, string>();
	const html = { dataset: {} as Record<string, string> };
	Object.assign(globalThis, {
		window: {
			location: { protocol: "http:", href: "http://wga.test/" },
			localStorage: {
				getItem: (key: string) => storage.get(key) ?? null,
				setItem: (key: string, value: string) => storage.set(key, value),
				removeItem: (key: string) => storage.delete(key),
			},
			...extra.window,
		},
		document: {
			get cookie() {
				return [...jar].map(([key, value]) => `${key}=${value}`).join("; ");
			},
			set cookie(value: string) {
				const [pair, ...attributes] = value.split("; ");
				const [key, ...rest] = pair.split("=");
				if (attributes.includes("Max-Age=0")) {
					jar.delete(key);
				} else {
					jar.set(key, rest.join("="));
				}
			},
			documentElement: html,
			querySelector: () => null,
			querySelectorAll: () => [],
			addEventListener: () => {},
			...extra.document,
		},
	});
	return {
		jar,
		storage,
		html,
		grantConsent: () =>
			jar.set("cc_cookie", consentCookieValue(["necessary", "preferences"])),
		withdrawConsent: () =>
			jar.set("cc_cookie", consentCookieValue(["necessary"])),
	};
};
