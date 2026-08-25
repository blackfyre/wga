const STORAGE_KEY = "wga-bionic";
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

function stored(): boolean {
	try {
		return window.localStorage.getItem(STORAGE_KEY) === "on";
	} catch {
		return false;
	}
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
		toggle.classList.toggle("bg-primary", on);
		toggle.classList.toggle("text-primary-content", on);
		toggle.classList.toggle("border-primary", on);
		toggle.classList.toggle("bg-base-100", !on);
		toggle.classList.toggle("text-base-content/75", !on);
		toggle.classList.toggle("border-base-content/20", !on);
	}
}

function set(on: boolean, persist: boolean): void {
	enabled = on;
	if (persist) {
		try {
			window.localStorage.setItem(STORAGE_KEY, on ? "on" : "off");
		} catch {
			// Storage can be unavailable in private browsing modes.
		}
		document.cookie = `wga_bionic=${on ? "on" : "off"}; path=/; max-age=31536000; samesite=lax`;
	}

	document.documentElement.dataset.bionicReading = String(on);
	if (on) {
		apply(document.body);
	} else {
		clear();
	}
	updateControls(on);
}

export function initBionicReading(): void {
	if (initialised) {
		return;
	}
	initialised = true;
	set(stored(), false);

	document.addEventListener("click", (event) => {
		if (!(event.target instanceof Element)) {
			return;
		}

		if (!event.target.closest("[data-wga-bionic-toggle]")) {
			return;
		}

		set(!enabled, true);
	});

	document.addEventListener("htmx:afterSwap", (event) => {
		updateControls(enabled);
		if (!enabled) {
			return;
		}

		const target = (event as HtmxAfterSwapEvent).detail.target;
		if (target) {
			apply(target);
		}
	});
}
