import {
	expireCookie,
	readPreference,
	registerPreferenceStore,
	removePreference,
	writePreference,
} from "./preference-consent";

export const BIONIC_STORAGE_KEY = "wga-bionic";
// The cookie releases before #208 set. Nothing reads it any more.
const LEGACY_COOKIE = "wga_bionic";
const PROSE_SELECTOR = "p, [data-bionic]";
const SKIP_SELECTOR =
	"[data-bionic-mark], b, strong, em, i, mark, [data-bionic='off'], nav, footer, figure, [class~='font-mono'], code, pre, form, button, input, select, textarea";
const WORD = /([A-Za-z\u00C0-\u024F\u2019']+)/;
const LETTER = /[A-Za-z\u00C0-\u024F]/;

type HtmxAfterSwapEvent = CustomEvent<{ target?: Element }>;

let initialised = false;
let enabled = false;

function wordHeadLength(word: string): number {
	if (word.length <= 3) {
		return 1;
	}

	return Math.min(word.length - 1, Math.round(word.length * 0.4));
}

function fillMark(mark: HTMLElement, text: string): void {
	text.split(WORD).forEach((part, index) => {
		if (!part) {
			return;
		}

		if (index % 2 === 0) {
			mark.append(document.createTextNode(part));
			return;
		}

		const headLength = wordHeadLength(part);
		const head = document.createElement("b");
		head.textContent = part.slice(0, headLength);
		mark.append(head, document.createTextNode(part.slice(headLength)));
	});
}

function apply(root: ParentNode): void {
	const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT, {
		acceptNode(node) {
			if (!node.nodeValue || !LETTER.test(node.nodeValue)) {
				return NodeFilter.FILTER_REJECT;
			}

			const parent = node.parentElement;
			if (!parent || parent.closest(SKIP_SELECTOR)) {
				return NodeFilter.FILTER_REJECT;
			}

			if (!parent.closest(PROSE_SELECTOR)) {
				return NodeFilter.FILTER_REJECT;
			}

			return NodeFilter.FILTER_ACCEPT;
		},
	});
	const nodes: Text[] = [];

	while (walker.nextNode()) {
		nodes.push(walker.currentNode as Text);
	}

	for (const node of nodes) {
		const text = node.nodeValue || "";
		const mark = document.createElement("span");
		mark.dataset.bionicMark = "true";
		mark.dataset.bionicSource = text;
		fillMark(mark, text);
		node.replaceWith(mark);
	}
}

function clear(): void {
	for (const mark of document.querySelectorAll<HTMLElement>(
		"[data-bionic-mark]",
	)) {
		mark.replaceWith(document.createTextNode(mark.dataset.bionicSource || ""));
	}
	document.body.normalize();
}

// Remembering is optional storage: readPreference and writePreference refuse
// without cookie-consent "preferences", so the choice then lasts for the
// current page only.
function storedBionicReading(): boolean {
	return readPreference(BIONIC_STORAGE_KEY) === "on";
}

// rememberBionicReading stores the state the page shows.
export function rememberBionicReading(): void {
	writePreference(
		BIONIC_STORAGE_KEY,
		document.documentElement.dataset.bionicReading === "true" ? "on" : "off",
	);
}

export function forgetBionicReading(): void {
	removePreference(BIONIC_STORAGE_KEY);
}

function updateControls(on: boolean): void {
	for (const control of document.querySelectorAll<HTMLElement>(
		"[data-wga-bionic-control]",
	)) {
		control.classList.remove("hidden");
		control.classList.add("flex");
	}

	for (const toggle of document.querySelectorAll<HTMLElement>(
		"[data-wga-bionic-toggle]",
	)) {
		toggle.setAttribute("aria-checked", String(on));
		toggle.classList.toggle("bg-wga-accent-bg", on);
		toggle.classList.toggle("text-wga-inv-fg", on);
		toggle.classList.toggle("border-wga-accent", on);
		toggle.classList.toggle("bg-wga-bg", !on);
		toggle.classList.toggle("text-wga-ink/75", !on);
		toggle.classList.toggle("border-wga-ink/20", !on);
	}
}

export function currentBionicReading(): boolean {
	return enabled;
}

export function setBionicReading(on: boolean, persist = true): void {
	enabled = on;
	if (persist) {
		writePreference(BIONIC_STORAGE_KEY, on ? "on" : "off");
	}

	document.documentElement.dataset.bionicReading = String(on);
	if (on) {
		apply(document.body);
	} else {
		clear();
	}
	updateControls(on);
	document.dispatchEvent(new CustomEvent("wga:preferences-changed"));
}

export function toggleBionicReading(): void {
	setBionicReading(!enabled);
}

export function initBionicReading(): void {
	if (initialised) {
		return;
	}
	initialised = true;
	expireCookie(LEGACY_COOKIE);
	registerPreferenceStore({
		remember: rememberBionicReading,
		forget: forgetBionicReading,
	});
	setBionicReading(storedBionicReading(), false);

	document.addEventListener("click", (event) => {
		if (!(event.target instanceof Element)) {
			return;
		}

		if (!event.target.closest("[data-wga-bionic-toggle]")) {
			return;
		}

		toggleBionicReading();
	});

	document.addEventListener("htmx:afterSwap", (event) => {
		updateControls(enabled);
		if (!enabled) {
			return;
		}

		const target = (event as HtmxAfterSwapEvent).detail?.target;
		if (target) {
			apply(target);
		}
	});
}
