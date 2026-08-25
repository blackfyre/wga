// Horizontal keyboard scrolling for the Dual Mode record tables. Each Dual
// index table is wrapped in a focusable, labelled region
// ([data-dual-horizontal-scroll]) that scrolls horizontally inside a narrow
// pane. The shared keyboard layer treats ArrowLeft/ArrowRight as list movement
// across the whole document, so this handler intercepts those keys while the
// region itself is focused and scrolls the region instead, stopping propagation
// so the global pane/list navigation never consumes them.

const SCROLL_STEP = 120;

const editableTarget = (target: EventTarget | null): boolean => {
	if (!(target instanceof HTMLElement)) {
		return false;
	}
	return (
		target.isContentEditable ||
		target.matches("input, textarea, select, [contenteditable='true']")
	);
};

const reducedMotion = (): boolean =>
	typeof window !== "undefined" &&
	window.matchMedia("(prefers-reduced-motion: reduce)").matches;

let initialised = false;

export const initDualHorizontalScroll = (): void => {
	if (initialised) {
		return;
	}
	initialised = true;

	document.addEventListener(
		"keydown",
		(event) => {
			if (!(event.target instanceof HTMLElement)) {
				return;
			}
			const scroller = event.target.closest<HTMLElement>(
				"[data-dual-horizontal-scroll]",
			);
			if (!scroller) {
				return;
			}
			// Act only while the labelled region itself holds focus, and never
			// inside an editable descendant.
			if (document.activeElement !== scroller || editableTarget(event.target)) {
				return;
			}
			if (event.metaKey || event.ctrlKey || event.altKey) {
				return;
			}

			const behavior: ScrollBehavior = reducedMotion() ? "auto" : "smooth";

			switch (event.key) {
				case "ArrowRight":
					scroller.scrollBy({ left: SCROLL_STEP, behavior });
					event.preventDefault();
					event.stopPropagation();
					break;
				case "ArrowLeft":
					scroller.scrollBy({ left: -SCROLL_STEP, behavior });
					event.preventDefault();
					event.stopPropagation();
					break;
				case "Home":
					scroller.scrollTo({ left: 0, behavior });
					event.preventDefault();
					event.stopPropagation();
					break;
				case "End":
					scroller.scrollTo({ left: scroller.scrollWidth, behavior });
					event.preventDefault();
					event.stopPropagation();
					break;
			}
		},
		true,
	);
};
