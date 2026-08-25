// Tour reading enhancement.
//
// A tour page is server-rendered one addressed page at a time with ordinary
// rel="prev"/rel="next" links, so it works without JavaScript. This helper only
// adds ArrowLeft/ArrowRight page turning on top; its absence cannot block
// reading or navigation. Escape dismissal of the deliberate artwork viewer is
// owned by the shared viewer focus trap, not by this helper.
import logger from "./logger";

export function registerTourHelpers(): void {
	const readings = document.querySelectorAll<HTMLElement>(
		"[data-tour-reading]:not([data-tour-bound])",
	);

	for (const reading of Array.from(readings)) {
		reading.dataset.tourBound = "true";

		const handleKeydown = (event: KeyboardEvent) => {
			if (!reading.isConnected) {
				document.removeEventListener("keydown", handleKeydown);
				return;
			}

			const target = event.target as HTMLElement | null;
			// Leave editable controls alone, and never turn pages while the
			// deliberate artwork viewer surface owns the keyboard.
			if (
				target?.closest(
					"input, textarea, select, [contenteditable], .wga-viewer-surface",
				)
			) {
				return;
			}

			if (event.key !== "ArrowLeft" && event.key !== "ArrowRight") {
				return;
			}

			const nav = reading.querySelector<HTMLAnchorElement>(
				event.key === "ArrowLeft"
					? '[data-tour-nav="prev"][href]'
					: '[data-tour-nav="next"][href]',
			);
			const href = nav?.getAttribute("href");
			if (href) {
				event.preventDefault();
				window.location.href = href;
			}
		};

		document.addEventListener("keydown", handleKeydown);

		logger.debug("Tour reading helpers registered");
	}
}
