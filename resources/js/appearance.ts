export type Scheme = "light" | "dark";

export const PALETTE_NAMES = [
	"bone",
	"classic",
	"verdigris",
	"gothic",
	"renaissance",
	"baroque",
	"rococo",
	"classical",
	"impressionist",
	"catppuccin",
	"tokyo",
] as const;

export type Palette = (typeof PALETTE_NAMES)[number];

export const SCHEME_STORAGE_KEY = "wga-theme";
export const PALETTE_STORAGE_KEY = "wga-palette";
export const SCHEME_COOKIE_NAME = "wga_theme";
export const PALETTE_COOKIE_NAME = "wga_palette";

const DEFAULT_PALETTE: Palette = "bone";
const DARK_ONLY_PALETTES: ReadonlySet<Palette> = new Set(["baroque", "tokyo"]);

const THEME_NAMES: Record<Palette, Record<Scheme, string>> = {
	bone: { light: "wga-rams", dark: "wga-rams-dark" },
	classic: { light: "wga-classic", dark: "wga-classic-dark" },
	verdigris: { light: "wga-verdigris", dark: "wga-verdigris-dark" },
	gothic: { light: "wga-gothic", dark: "wga-gothic-dark" },
	renaissance: {
		light: "wga-renaissance",
		dark: "wga-renaissance-dark",
	},
	baroque: { light: "wga-baroque", dark: "wga-baroque" },
	rococo: { light: "wga-rococo", dark: "wga-rococo-dark" },
	classical: { light: "wga-classical", dark: "wga-classical-dark" },
	impressionist: {
		light: "wga-impressionist",
		dark: "wga-impressionist-dark",
	},
	catppuccin: { light: "wga-catppuccin", dark: "wga-catppuccin-dark" },
	tokyo: { light: "wga-tokyo", dark: "wga-tokyo" },
};

let initialised = false;
let sessionScheme: Scheme | null = null;
let sessionPalette: Palette | null = null;
let preferencesInvoker: HTMLElement | null = null;
const boundPanels = new WeakSet<HTMLDialogElement>();

export function parseScheme(value: string | null): Scheme | null {
	if (value === "light" || value === "wga_light") {
		return "light";
	}
	if (value === "dark" || value === "wga_dark") {
		return "dark";
	}
	return null;
}

export function parsePalette(value: string | null): Palette | null {
	if (!value) {
		return null;
	}
	if ((PALETTE_NAMES as readonly string[]).includes(value)) {
		return value as Palette;
	}
	return null;
}

export function isDarkOnlyPalette(palette: Palette): boolean {
	return DARK_ONLY_PALETTES.has(palette);
}

export function effectiveScheme(scheme: Scheme, palette: Palette): Scheme {
	if (isDarkOnlyPalette(palette)) {
		return "dark";
	}
	return scheme;
}

export function resolveThemeName(scheme: Scheme, palette: Palette): string {
	return THEME_NAMES[palette][scheme];
}

function cookieValue(name: string): string | null {
	const prefix = `${name}=`;
	for (const item of document.cookie.split(";")) {
		const cookie = item.trim();
		if (cookie.startsWith(prefix)) {
			return decodeURIComponent(cookie.slice(prefix.length));
		}
	}
	return null;
}

function localStorageValue(key: string): string | null {
	try {
		return window.localStorage.getItem(key);
	} catch {
		return null;
	}
}

function writeLocalStorage(key: string, value: string): void {
	try {
		window.localStorage.setItem(key, value);
	} catch {
		// The in-memory choice still applies when storage is unavailable.
	}
}

function removeLocalStorage(key: string): void {
	try {
		window.localStorage.removeItem(key);
	} catch {
		// There is nothing else to clear when storage is unavailable.
	}
}

function writeCookie(name: string, value: string): void {
	document.cookie = `${name}=${encodeURIComponent(value)}; path=/; max-age=31536000; samesite=lax`;
}

function clearCookie(name: string): void {
	document.cookie = `${name}=; path=/; max-age=0; samesite=lax`;
}

function storedScheme(): Scheme | null {
	if (sessionScheme) {
		return sessionScheme;
	}
	const localValue = parseScheme(localStorageValue(SCHEME_STORAGE_KEY));
	if (localValue) {
		return localValue;
	}
	return parseScheme(cookieValue(SCHEME_COOKIE_NAME));
}

function storedPalette(): Palette | null {
	if (sessionPalette) {
		return sessionPalette;
	}
	const localValue = parsePalette(localStorageValue(PALETTE_STORAGE_KEY));
	if (localValue) {
		return localValue;
	}
	return parsePalette(cookieValue(PALETTE_COOKIE_NAME));
}

export function currentScheme(): Scheme {
	const explicit = storedScheme();
	if (explicit) {
		return explicit;
	}
	if (window.matchMedia("(prefers-color-scheme: dark)").matches) {
		return "dark";
	}
	return "light";
}

export function currentPalette(): Palette {
	return storedPalette() ?? DEFAULT_PALETTE;
}

function paletteLabel(palette: Palette): string {
	const control = document.querySelector<HTMLElement>(
		`[data-wga-palette="${palette}"]`,
	);
	return control?.dataset.wgaPaletteLabel?.trim() || palette.toUpperCase();
}

function markSchemeControls(scheme: Scheme, palette: Palette): void {
	const appliedScheme = effectiveScheme(scheme, palette);
	for (const control of document.querySelectorAll<HTMLButtonElement>(
		"[data-wga-scheme], [data-wga-theme]",
	)) {
		const value = control.dataset.wgaScheme ?? control.dataset.wgaTheme;
		const active = value === appliedScheme;
		const disabled = value === "light" && isDarkOnlyPalette(palette);
		control.setAttribute("aria-pressed", String(active));
		control.disabled = disabled;
		control.classList.toggle("bg-primary", active);
		control.classList.toggle("text-primary-content", active);
		control.classList.toggle("bg-base-100", !active);
		control.classList.toggle("text-base-content/75", !active);
		if (disabled) {
			control.title = `${paletteLabel(palette)} is a dark-only palette`;
		} else {
			control.removeAttribute("title");
		}
	}
}

function markPaletteControls(palette: Palette): void {
	for (const control of document.querySelectorAll<HTMLElement>(
		"[data-wga-palette]",
	)) {
		const active = control.dataset.wgaPalette === palette;
		control.setAttribute("aria-checked", String(active));
		control.classList.toggle("bg-primary/10", active);
		control.classList.toggle(
			"shadow-[inset_3px_0_0_var(--color-primary)]",
			active,
		);
		control.classList.toggle("bg-base-100", !active);
		control.classList.toggle("hover:bg-base-200", !active);

		const label = control.querySelector<HTMLElement>("[data-wga-palette-name]");
		label?.classList.toggle("text-primary", active);
		label?.classList.toggle("text-base-content", !active);

		let marker = control.querySelector<HTMLElement>(
			"[data-wga-palette-in-use]",
		);
		if (active) {
			if (!marker) {
				marker = document.createElement("span");
				marker.dataset.wgaPaletteInUse = "true";
				marker.className =
					"ml-auto shrink-0 font-mono text-[9px] tracking-[1.5px] text-primary";
				marker.textContent = "IN USE";
				control.append(marker);
			}
		} else {
			marker?.remove();
		}
	}
}

function markSchemeExplanation(palette: Palette): void {
	for (const explanation of document.querySelectorAll<HTMLElement>(
		"[data-wga-scheme-explanation]",
	)) {
		if (isDarkOnlyPalette(palette)) {
			explanation.textContent = `${paletteLabel(palette)} has no light build, so light is unavailable while it is chosen.`;
		} else {
			explanation.textContent =
				"Follows your system setting until you choose here.";
		}
	}
}

function markPreferencesSummary(scheme: Scheme, palette: Palette): void {
	const parts = [
		paletteLabel(palette),
		effectiveScheme(scheme, palette).toUpperCase(),
	];
	if (document.documentElement.dataset.bionicReading === "true") {
		parts.push("BIONIC");
	}
	for (const summary of document.querySelectorAll<HTMLElement>(
		"[data-wga-preferences-summary]",
	)) {
		summary.textContent = parts.join(" · ");
	}

	const paletteControl = document.querySelector<HTMLElement>(
		`[data-wga-palette="${palette}"]`,
	);
	const paletteSwatch = paletteControl?.querySelector<HTMLElement>(
		"[data-wga-palette-swatch], span[style]",
	);
	if (!paletteSwatch?.style.background) {
		return;
	}
	for (const swatch of document.querySelectorAll<HTMLElement>(
		"[data-wga-preferences-swatch]",
	)) {
		swatch.style.background = paletteSwatch.style.background;
	}
}

function revealControls(): void {
	for (const control of document.querySelectorAll<HTMLElement>(
		"[data-wga-preferences-control], [data-wga-theme-toggle]",
	)) {
		control.classList.remove("hidden");
		control.classList.add("flex");
		control.removeAttribute("aria-hidden");
	}
}

function bindPreferencesPanel(): void {
	for (const panel of document.querySelectorAll<HTMLDialogElement>(
		"[data-wga-preferences-panel], #wga-preferences",
	)) {
		if (boundPanels.has(panel)) {
			continue;
		}
		boundPanels.add(panel);
		panel.addEventListener("close", () => {
			if (preferencesInvoker?.isConnected) {
				preferencesInvoker.focus();
			}
			preferencesInvoker = null;
		});
	}
}

export function reconcileAppearancePreferences(): void {
	const scheme = currentScheme();
	const palette = currentPalette();
	document.documentElement.dataset.theme = resolveThemeName(scheme, palette);
	markSchemeControls(scheme, palette);
	markPaletteControls(palette);
	markSchemeExplanation(palette);
	markPreferencesSummary(scheme, palette);
	revealControls();
	bindPreferencesPanel();
}

export function setScheme(scheme: Scheme): void {
	sessionScheme = scheme;
	writeLocalStorage(SCHEME_STORAGE_KEY, scheme);
	writeCookie(SCHEME_COOKIE_NAME, scheme);
	reconcileAppearancePreferences();
}

export function clearScheme(): void {
	sessionScheme = null;
	removeLocalStorage(SCHEME_STORAGE_KEY);
	clearCookie(SCHEME_COOKIE_NAME);
	reconcileAppearancePreferences();
}

export function setPalette(palette: Palette): void {
	sessionPalette = palette;
	writeLocalStorage(PALETTE_STORAGE_KEY, palette);
	writeCookie(PALETTE_COOKIE_NAME, palette);
	reconcileAppearancePreferences();
}

export function clearPalette(): void {
	sessionPalette = null;
	removeLocalStorage(PALETTE_STORAGE_KEY);
	clearCookie(PALETTE_COOKIE_NAME);
	reconcileAppearancePreferences();
}

function preferencesPanel(): HTMLDialogElement | null {
	return document.querySelector<HTMLDialogElement>(
		"[data-wga-preferences-panel], #wga-preferences",
	);
}

export function openPreferences(invoker?: HTMLElement): void {
	const panel = preferencesPanel();
	if (!panel || panel.open) {
		return;
	}
	preferencesInvoker = invoker ?? null;
	panel.showModal();
	window.requestAnimationFrame(() => {
		if (!panel.open) {
			return;
		}
		const initialFocus = panel.querySelector<HTMLElement>(
			"[data-wga-preferences-initial-focus], [data-wga-preferences-close]",
		);
		initialFocus?.focus();
	});
}

export function closePreferences(): void {
	const panel = preferencesPanel();
	if (panel?.open) {
		panel.close();
	}
}

function migrateLegacyScheme(): void {
	const storedValue = localStorageValue(SCHEME_STORAGE_KEY);
	const scheme = parseScheme(storedValue);
	if (!scheme || storedValue === scheme) {
		return;
	}
	writeLocalStorage(SCHEME_STORAGE_KEY, scheme);
	writeCookie(SCHEME_COOKIE_NAME, scheme);
}

function handleClick(event: MouseEvent): void {
	if (!(event.target instanceof Element)) {
		return;
	}

	const open = event.target.closest<HTMLElement>("[data-wga-preferences-open]");
	if (open) {
		openPreferences(open);
		return;
	}
	if (event.target.closest("[data-wga-preferences-close]")) {
		closePreferences();
		return;
	}

	const schemeControl = event.target.closest<HTMLElement>(
		"[data-wga-scheme], [data-wga-theme]",
	);
	if (schemeControl) {
		const scheme = parseScheme(
			schemeControl.dataset.wgaScheme ?? schemeControl.dataset.wgaTheme ?? null,
		);
		if (
			scheme &&
			!(scheme === "light" && isDarkOnlyPalette(currentPalette()))
		) {
			setScheme(scheme);
		}
		return;
	}

	const paletteControl =
		event.target.closest<HTMLElement>("[data-wga-palette]");
	const palette = parsePalette(paletteControl?.dataset.wgaPalette ?? null);
	if (palette) {
		setPalette(palette);
	}
}

export function initialiseAppearancePreferences(): void {
	if (initialised) {
		reconcileAppearancePreferences();
		return;
	}
	initialised = true;
	migrateLegacyScheme();
	reconcileAppearancePreferences();
	document.addEventListener("click", handleClick);
	document.addEventListener("wga:preferences-changed", () => {
		reconcileAppearancePreferences();
	});
	document.addEventListener("htmx:afterSwap", () => {
		reconcileAppearancePreferences();
	});
	window
		.matchMedia("(prefers-color-scheme: dark)")
		.addEventListener("change", () => {
			if (!storedScheme()) {
				reconcileAppearancePreferences();
			}
		});
}
