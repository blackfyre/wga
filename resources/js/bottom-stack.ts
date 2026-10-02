const ITEM_SELECTOR = "[data-wga-bottom-stack-item]";
const COOKIE_NOTICE_SELECTOR = "#cc-main .cm";

let frame = 0;
let observer: MutationObserver | null = null;
let resizeObserver: ResizeObserver | null = null;

const numericData = (element: HTMLElement, key: string) => {
	const value = Number(element.dataset[key] ?? "0");
	return Number.isFinite(value) ? value : 0;
};

const visible = (element: HTMLElement) =>
	element.getClientRects().length > 0 &&
	getComputedStyle(element).visibility !== "hidden";

const prepareItems = () => {
	for (const notice of document.querySelectorAll<HTMLElement>(
		COOKIE_NOTICE_SELECTOR,
	)) {
		notice.dataset.wgaBottomStackItem = "cookie";
		notice.dataset.wgaBottomStackOrder = "100";
		notice.dataset.wgaBottomStackGap = "16";
	}

	return [...document.querySelectorAll<HTMLElement>(ITEM_SELECTOR)].sort(
		(first, second) =>
			numericData(first, "wgaBottomStackOrder") -
			numericData(second, "wgaBottomStackOrder"),
	);
};

// Observing a sized element queues an initial notification, so re-observing
// every item on each measurement would loop forever. Only new items are
// observed and removed ones released.
const observed = new Set<HTMLElement>();

const reconcileObserved = (items: HTMLElement[]) => {
	if (!resizeObserver) {
		return;
	}
	const current = new Set(items);
	for (const item of observed) {
		if (!current.has(item)) {
			resizeObserver.unobserve(item);
			observed.delete(item);
		}
	}
	for (const item of items) {
		if (!observed.has(item)) {
			resizeObserver.observe(item);
			observed.add(item);
		}
	}
};

const measureBottomStack = () => {
	let offset = 0;
	const items = prepareItems();
	reconcileObserved(items);
	for (const item of items) {
		if (!visible(item)) {
			item.style.removeProperty("--wga-bottom-stack-offset");
			continue;
		}
		offset += numericData(item, "wgaBottomStackGap");
		item.style.setProperty("--wga-bottom-stack-offset", `${offset}px`);
		offset += item.getBoundingClientRect().height;
	}
	document.body.style.setProperty(
		"--wga-bottom-stack-height",
		`${Math.ceil(offset)}px`,
	);
};

const refreshBottomStack = () => {
	window.cancelAnimationFrame(frame);
	frame = window.requestAnimationFrame(measureBottomStack);
};

export const initBottomStack = () => {
	if (observer) {
		refreshBottomStack();
		return;
	}
	resizeObserver = new ResizeObserver(refreshBottomStack);
	observer = new MutationObserver(refreshBottomStack);
	// CookieConsent inserts its notice hidden and reveals it later by toggling
	// classes on <html> and its wrapper, which changes neither the DOM tree nor
	// the notice's size. Measuring writes only style properties, so watching
	// class and hidden changes cannot feed back into itself.
	observer.observe(document.documentElement, {
		childList: true,
		subtree: true,
		attributes: true,
		attributeFilter: ["class", "hidden"],
	});
	window.addEventListener("resize", refreshBottomStack);
	refreshBottomStack();
};
