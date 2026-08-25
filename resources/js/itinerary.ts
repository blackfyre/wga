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

/** Updates a "N characters left" output next to a textarea.
 *
 * The remaining count is measured in Unicode code points, matching the Go
 * `[]rune` length the server enforces. `[...value]` iterates code points rather
 * than UTF-16 code units, so an astral character (for example an emoji) counts
 * once and the visible countdown stays consistent with what is persisted.
 */
export function countdown(field: HTMLTextAreaElement, outputId: string): void {
	const output = document.getElementById(outputId);
	if (!output) {
		return;
	}
	const max = Number(field.getAttribute("maxlength") ?? 600);
	output.textContent = String(Math.max(0, max - [...field.value].length));
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

	const viewers = document.querySelectorAll<HTMLElement>(
		"[data-itinerary-viewer]:not([data-itinerary-bound])",
	);

	for (const viewer of Array.from(viewers)) {
		viewer.dataset.itineraryBound = "true";
		prefetchNeighbours(viewer);
	}

	registerItineraryKeyboard();
}
