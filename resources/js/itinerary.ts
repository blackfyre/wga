// Itinerary slideshow enhancement.
//
// The public itinerary is server-rendered one stop at a time with ordinary
// rel="prev"/rel="next" links, so it works without JavaScript. This helper only
// adds keyboard navigation (Arrow keys and Escape) and prefetches the two
// neighbouring stops on top. Its absence cannot block reading or navigation.
//
// Keyboard navigation listens at document scope so the Arrow keys and Escape
// still work when focus sits on the page body rather than inside the viewer.
//
// The keyboard binder is idempotent: it is called synchronously from the
// application entry so Arrow keys work the instant a directly-loaded viewer is
// parsed (rather than only after the async bootstrap chunk settles), and it is
// called again by `registerItineraryHelpers` after an HTMX swap. The
// `data-itinerary-keyboard` guard keeps those overlapping calls to a single
// listener per viewer.
import logger from "./logger";

function closestHref(target: EventTarget | null): string {
	const element = target as Element | null;
	const link = element?.closest?.("a[href]") as HTMLAnchorElement | null;
	return link?.getAttribute("href") ?? "";
}

function prefetchNeighbours(root: Element): void {
	const neighbours = root.querySelectorAll<HTMLAnchorElement>(
		"[data-itinerary-prefetch][href]",
	);

	// The viewer exposes at most two neighbours (previous and next).
	for (const link of Array.from(neighbours).slice(0, 2)) {
		const href = link.getAttribute("href");
		if (!href) {
			continue;
		}
		const prefetch = document.createElement("link");
		prefetch.rel = "prefetch";
		prefetch.href = href;
		document.head.appendChild(prefetch);
	}
}

function bindCopyLinks(): void {
	const buttons = document.querySelectorAll<HTMLButtonElement>(
		"[data-copy-itinerary]:not([data-copy-bound])",
	);

	for (const button of Array.from(buttons)) {
		button.dataset.copyBound = "true";
		button.addEventListener("click", async () => {
			const target = document.querySelector<HTMLElement>(
				button.dataset.copyTarget || "",
			);
			if (!target) {
				return;
			}

			let copied = false;
			try {
				await navigator.clipboard.writeText(target.innerText);
				copied = true;
			} catch {
				const field = document.createElement("textarea");
				field.value = target.innerText;
				document.body.appendChild(field);
				field.select();
				copied = document.execCommand("copy");
				field.remove();
			}

			if (!copied) {
				logger.warn("Unable to copy itinerary link");
				return;
			}

			button.textContent = "COPIED";
			setTimeout(() => {
				button.textContent = "COPY LINK";
			}, 2000);
		});
	}
}

function bindBuilderTabs(): void {
	const editors = Array.from(
		document.querySelectorAll<HTMLElement>("[data-itinerary-editor]"),
	);
	if (editors.length === 0) {
		return;
	}

	const tabs = Array.from(
		document.querySelectorAll<HTMLAnchorElement>(
			"[data-itinerary-tab]:not([data-itinerary-tab-bound])",
		),
	);
	for (const tab of tabs) {
		tab.dataset.itineraryTabBound = "true";
		tab.addEventListener("click", (event) => {
			event.preventDefault();
			const selected = tab.dataset.itineraryTab;
			if (selected === undefined) {
				return;
			}

			for (const editor of editors) {
				editor.classList.toggle(
					"hidden",
					editor.dataset.itineraryEditor !== selected,
				);
			}
			for (const candidate of document.querySelectorAll<HTMLAnchorElement>(
				"[data-itinerary-tab]",
			)) {
				if (candidate.dataset.itineraryTab === selected) {
					candidate.setAttribute("aria-current", "page");
				} else {
					candidate.removeAttribute("aria-current");
				}
			}
			for (const item of document.querySelectorAll<HTMLElement>(
				"[data-itinerary-tab-item]",
			)) {
				const active = item.dataset.itineraryTabItem === selected;
				item.classList.toggle("bg-primary/6", active);
				item.classList.toggle("border-base-content", active);
				item.classList.toggle("border-base-content/15", !active);
			}
			for (const label of document.querySelectorAll<HTMLElement>(
				"[data-itinerary-tab-label]",
			)) {
				const active = label.dataset.itineraryTabLabel === selected;
				label.classList.toggle("text-primary", active);
				label.classList.toggle("text-base-content/40", !active);
			}
		});
	}
}

function bindBuilderNarrationStatus(): void {
	const narrations = document.querySelectorAll<HTMLTextAreaElement>(
		"[data-itinerary-narration]:not([data-itinerary-narration-bound])",
	);
	for (const narration of narrations) {
		narration.dataset.itineraryNarrationBound = "true";
		narration.addEventListener("input", () => {
			const stopID = narration.dataset.itineraryNarration;
			if (stopID === undefined) {
				return;
			}
			const count = Array.from(narration.value).length;
			for (const status of document.querySelectorAll<HTMLElement>(
				"[data-itinerary-narration-status]",
			)) {
				if (status.dataset.itineraryNarrationStatus !== stopID) {
					continue;
				}
				status.textContent =
					count === 0 ? "NO NARRATION YET" : `${count} CHARS`;
				status.classList.toggle("text-warning", count === 0);
				status.classList.toggle("text-base-content/60", count > 0);
			}
		});
	}
}

function bindBuilderVisibility(): void {
	const radios = document.querySelectorAll<HTMLInputElement>(
		"[data-itinerary-visibility] input[name='listed']:not([data-itinerary-visibility-bound])",
	);
	for (const radio of radios) {
		radio.dataset.itineraryVisibilityBound = "true";
		radio.addEventListener("change", () => {
			for (const candidate of document.querySelectorAll<HTMLInputElement>(
				"[data-itinerary-visibility] input[name='listed']",
			)) {
				const label = candidate.closest<HTMLElement>(
					"[data-itinerary-visibility]",
				);
				if (label === null) {
					continue;
				}
				label.classList.toggle("border-primary", candidate.checked);
				label.classList.toggle("bg-primary", candidate.checked);
				label.classList.toggle("text-primary-content", candidate.checked);
				label.classList.toggle("border-control", !candidate.checked);
				label.classList.toggle("text-base-content/70", !candidate.checked);
			}
		});
	}
}

function bindViewerKeyboard(viewer: HTMLElement): void {
	const handleKeydown = (event: KeyboardEvent) => {
		if (!viewer.isConnected) {
			document.removeEventListener("keydown", handleKeydown);
			return;
		}

		const target = event.target as HTMLElement | null;
		if (target?.closest("input, textarea, select, [contenteditable]")) {
			return;
		}

		if (event.key === "ArrowLeft" || event.key === "ArrowRight") {
			const nav = viewer.querySelector<HTMLAnchorElement>(
				event.key === "ArrowLeft"
					? '[data-itinerary-nav="prev"][href]'
					: '[data-itinerary-nav="next"][href]',
			);
			if (nav) {
				event.preventDefault();
				// Dispatch a real click rather than assigning window.location:
				// the link carries hx-get, so htmx's own delegated click
				// handler swaps just #mc-area in place. A location assignment
				// forces a full document reload, which is the flash this
				// avoids.
				nav.click();
			}
			return;
		}

		if (event.key === "Escape") {
			const exit = closestHref(event.target);
			const exitHref =
				viewer
					.querySelector<HTMLAnchorElement>("[data-itinerary-exit][href]")
					?.getAttribute("href") ?? exit;
			if (exitHref) {
				event.preventDefault();
				window.location.href = exitHref;
			}
		}
	};

	document.addEventListener("keydown", handleKeydown);
}

/** Binds Arrow/Escape keyboard navigation for every unbound viewer.
 *
 * Idempotent: each bound viewer gains `data-itinerary-keyboard` and is then
 * excluded by the selector, so repeated calls — the synchronous entry followed
 * by the async bootstrap, or successive HTMX swaps — never attach a second
 * listener to the same viewer.
 */
export function registerItineraryKeyboard(): void {
	const viewers = document.querySelectorAll<HTMLElement>(
		"[data-itinerary-viewer]:not([data-itinerary-keyboard])",
	);

	for (const viewer of Array.from(viewers)) {
		viewer.dataset.itineraryKeyboard = "bound";
		bindViewerKeyboard(viewer);
		logger.debug("Itinerary viewer keyboard registered");
	}
}

export function registerItineraryHelpers(): void {
	bindCopyLinks();
	bindBuilderTabs();
	bindBuilderNarrationStatus();
	bindBuilderVisibility();

	const viewers = document.querySelectorAll<HTMLElement>(
		"[data-itinerary-viewer]:not([data-itinerary-bound])",
	);

	for (const viewer of Array.from(viewers)) {
		viewer.dataset.itineraryBound = "true";
		prefetchNeighbours(viewer);
	}

	registerItineraryKeyboard();
}
