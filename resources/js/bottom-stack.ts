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

const measureBottomStack = () => {
	let offset = 0;
	const items = prepareItems();
	resizeObserver?.disconnect();
	for (const item of items) {
		resizeObserver?.observe(item);
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
	observer.observe(document.body, { childList: true, subtree: true });
	window.addEventListener("resize", refreshBottomStack);
	refreshBottomStack();
};
