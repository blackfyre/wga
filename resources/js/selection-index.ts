// Progressive enhancement for a large-holding artist record's ON THIS PAGE
// rail (nav[data-selection-index]).
//
// The server renders every form row as a fragment link and every foldable
// form's selection entries open, so the index works without JavaScript. This
// module adds a +/− control with aria-expanded to each foldable form, keeps
// only the form being read open, and highlights that form's row. The current
// form is tracked with an IntersectionObserver on the form blocks
// ([data-selection-index-block]) only, never on the tiles inside them.
//
// The rail is replaced whenever #mc-area is swapped. Each initialisation binds
// the rail currently in the document, rebuilding any controls restored from
// the HTMX history cache, and disconnects the observer of a rail that has left
// the document.

const RAIL_SELECTOR = "nav[data-selection-index]";
const FORM_SELECTOR = "[data-selection-index-form]";
const ROW_SELECTOR = "[data-selection-index-row]";
const ENTRIES_SELECTOR = "[data-selection-index-entries]";
const BLOCK_SELECTOR = "[data-selection-index-block]";
const TOGGLE_ATTRIBUTE = "data-selection-index-toggle";
// readingBandMargin is the observer's root margin for a viewport height. The
// reading band sits in the upper part of the viewport: a block counts as the
// one being read while it crosses the band between 10% and 40% from the top.
// The margins are pixels derived from the height, not percentages.
export const readingBandMargin = (viewportHeight: number): string => {
	const height = Math.max(0, Math.round(viewportHeight));
	const top = Math.round(height * 0.1);
	const bottom = Math.round(height * 0.6);
	return `-${top}px 0px -${bottom}px 0px`;
};

// currentFormId picks the form being read: the first form, in document order,
// whose block crosses the reading band. When no block crosses it (between
// blocks, or above the first one), the previous choice stands.
export const currentFormId = (
	order: readonly string[],
	visible: ReadonlySet<string>,
	previous: string | null,
): string | null => {
	for (const id of order) {
		if (visible.has(id)) {
			return id;
		}
	}
	return previous;
};

// foldPresentation is the control's text and aria-expanded value for a state.
export const foldPresentation = (
	open: boolean,
): { label: string; expanded: "true" | "false" } =>
	open ? { label: "−", expanded: "true" } : { label: "+", expanded: "false" };

type FoldableForm = {
	id: string;
	row: HTMLAnchorElement | null;
	toggle: HTMLButtonElement | null;
	entries: HTMLElement | null;
};

type ActiveRail = {
	rail: Element;
	disconnect: () => void;
};

let active: ActiveRail | null = null;

const setOpen = (form: FoldableForm, open: boolean) => {
	if (!form.toggle || !form.entries) {
		return;
	}
	const presentation = foldPresentation(open);
	form.toggle.textContent = presentation.label;
	form.toggle.setAttribute("aria-expanded", presentation.expanded);
	form.entries.hidden = !open;
};

const bindRail = (rail: Element, doc: Document): ActiveRail => {
	const forms: FoldableForm[] = [];
	for (const item of rail.querySelectorAll<HTMLElement>(FORM_SELECTOR)) {
		const id = item.dataset.selectionIndexForm ?? "";
		const rowContainer = item.querySelector<HTMLElement>(ROW_SELECTOR);
		const row = rowContainer?.querySelector<HTMLAnchorElement>("a") ?? null;
		const entries = item.querySelector<HTMLElement>(ENTRIES_SELECTOR);
		// Controls restored from the history cache are rebuilt, not reused,
		// because their listeners did not survive the snapshot.
		for (const stale of item.querySelectorAll(`[${TOGGLE_ATTRIBUTE}]`)) {
			stale.remove();
		}
		let toggle: HTMLButtonElement | null = null;
		if (entries && rowContainer) {
			toggle = doc.createElement("button");
			toggle.type = "button";
			toggle.setAttribute(TOGGLE_ATTRIBUTE, "");
			toggle.setAttribute("aria-controls", entries.id);
			const name = (row?.textContent ?? "").split(" · ")[0]?.trim() ?? "";
			toggle.setAttribute("aria-label", `Selections in ${name}`);
			toggle.className =
				"shrink-0 w-5 cursor-pointer font-mono text-(length:--t-13) leading-none text-wga-ink/70 hover:text-wga-accent focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-wga-accent";
			rowContainer.append(toggle);
		}
		const form: FoldableForm = { id, row, toggle, entries };
		toggle?.addEventListener("click", () => {
			setOpen(form, form.entries?.hidden ?? false);
		});
		forms.push(form);
	}

	let current: string | null = null;
	const highlight = (next: string | null) => {
		for (const form of forms) {
			const isCurrent = form.id === next;
			if (isCurrent) {
				form.row?.setAttribute("aria-current", "location");
			} else {
				form.row?.removeAttribute("aria-current");
			}
			setOpen(form, isCurrent);
		}
	};
	// Only the form being read is open; before any block is reached, none is.
	highlight(current);

	const order = forms.map((form) => form.id);
	const blocks = order
		.map((id) => doc.getElementById(id))
		.filter(
			(block): block is HTMLElement => block?.matches(BLOCK_SELECTOR) === true,
		);
	if (blocks.length === 0 || typeof IntersectionObserver === "undefined") {
		return { rail, disconnect: () => {} };
	}

	const view = doc.defaultView;
	const visible = new Set<string>();
	let observer: IntersectionObserver | null = null;
	let observedHeight = -1;
	const disconnect = () => {
		observer?.disconnect();
		observer = null;
		view?.removeEventListener("resize", observe);
	};
	// The band is built from the viewport height in pixels, so it never
	// depends on how an engine resolves percentage margins. A height change
	// rebuilds the observer, which reports every block's state afresh.
	function observe() {
		const height = view?.innerHeight ?? 0;
		if (height === observedHeight) {
			return;
		}
		observedHeight = height;
		observer?.disconnect();
		visible.clear();
		const next = new IntersectionObserver(
			(records) => {
				if (!rail.isConnected) {
					disconnect();
					return;
				}
				for (const record of records) {
					if (record.isIntersecting) {
						visible.add(record.target.id);
					} else {
						visible.delete(record.target.id);
					}
				}
				const form = currentFormId(order, visible, current);
				if (form !== current) {
					current = form;
					highlight(current);
				}
			},
			{ rootMargin: readingBandMargin(height) },
		);
		for (const block of blocks) {
			next.observe(block);
		}
		observer = next;
	}
	observe();
	view?.addEventListener("resize", observe);
	return { rail, disconnect };
};

// initSelectionIndex binds the large-holding rail in the document, if any. It
// is idempotent: calling it again for the same, still-connected rail does
// nothing.
export const initSelectionIndex = (doc: Document = document) => {
	const rail = doc.querySelector(RAIL_SELECTOR);
	if (active && (active.rail === rail || !rail) && active.rail.isConnected) {
		return;
	}
	active?.disconnect();
	active = rail ? bindRail(rail, doc) : null;
};

let registered = false;

// registerSelectionIndex binds the current rail and rebinds after every HTMX
// swap or history restore. It registers its listeners once.
export const registerSelectionIndex = () => {
	if (registered) {
		return;
	}
	registered = true;
	const init = () => initSelectionIndex(document);
	document.addEventListener("htmx:load", init);
	document.addEventListener("htmx:afterSwap", init);
	document.addEventListener("htmx:historyRestore", init);
	if (document.readyState === "loading") {
		document.addEventListener("DOMContentLoaded", init, { once: true });
	} else {
		init();
	}
};
