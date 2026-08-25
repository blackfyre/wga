import { afterEach, expect, test } from "bun:test";
import { initDualHorizontalScroll } from "./dual";

type KeydownEvent = {
	key: string;
	target: unknown;
	metaKey?: boolean;
	ctrlKey?: boolean;
	altKey?: boolean;
	preventDefault(): void;
	stopPropagation(): void;
};

type KeydownListener = (event: KeydownEvent) => void;

class FakeHTMLElement {
	isContentEditable = false;
	scrollWidth = 640;
	matches(_selector: string): boolean {
		return false;
	}
	closest(_selector: string): FakeHTMLElement | null {
		return null;
	}
	scrollBy(_options: ScrollToOptions): void {}
	scrollTo(_options: ScrollToOptions): void {}
}

// Restore the globals this test installs so they cannot leak into other test
// files sharing the same process.
const savedDocument = globalThis.document;
const savedWindow = globalThis.window;
const savedHTMLElement = globalThis.HTMLElement;

afterEach(() => {
	globalThis.document = savedDocument;
	globalThis.window = savedWindow;
	globalThis.HTMLElement = savedHTMLElement;
});

test("dual horizontal scroll handles movement, limits, exclusion, isolation and idempotence", () => {
	const listeners: Array<{
		name: string;
		fn: KeydownListener;
		capture?: boolean;
	}> = [];
	const scrollCalls: ScrollToOptions[] = [];

	const scroller = new FakeHTMLElement();
	scroller.scrollWidth = 640;
	scroller.closest = (_selector: string) => scroller;
	scroller.scrollBy = (options: ScrollToOptions) => {
		scrollCalls.push(options);
	};
	scroller.scrollTo = (options: ScrollToOptions) => {
		scrollCalls.push(options);
	};

	const document = {
		activeElement: scroller as unknown as FakeHTMLElement,
		addEventListener: (
			name: string,
			fn: KeydownListener,
			capture?: boolean,
		) => {
			listeners.push({ name, fn, capture });
		},
	};

	globalThis.HTMLElement = FakeHTMLElement as unknown as typeof HTMLElement;
	globalThis.document = document as unknown as Document;
	let reduced = false;
	globalThis.window = {
		matchMedia: (_query: string) => ({ matches: reduced }),
	} as unknown as Window & typeof globalThis;

	// Idempotence: the second call must not register a second listener.
	initDualHorizontalScroll();
	initDualHorizontalScroll();

	const keydown = listeners.filter((listener) => listener.name === "keydown");
	expect(keydown).toHaveLength(1);
	expect(keydown[0].capture).toBe(true);

	const dispatch = (key: string, target: unknown, mod = false) => {
		let prevented = false;
		let stopped = false;
		const event: KeydownEvent = {
			key,
			target,
			metaKey: mod,
			ctrlKey: false,
			altKey: false,
			preventDefault: () => {
				prevented = true;
			},
			stopPropagation: () => {
				stopped = true;
			},
		};
		keydown[0].fn(event);
		return { prevented, stopped };
	};

	// Movement: ArrowRight scrolls right and isolates the event.
	scrollCalls.length = 0;
	let result = dispatch("ArrowRight", scroller);
	expect(scrollCalls).toHaveLength(1);
	expect(scrollCalls[0].left).toBe(120);
	expect(scrollCalls[0].behavior).toBe("smooth");
	expect(result.prevented).toBe(true);
	expect(result.stopped).toBe(true);

	// Movement: ArrowLeft scrolls left.
	scrollCalls.length = 0;
	result = dispatch("ArrowLeft", scroller);
	expect(scrollCalls).toHaveLength(1);
	expect(scrollCalls[0].left).toBe(-120);
	expect(result.prevented).toBe(true);
	expect(result.stopped).toBe(true);

	// Limits: Home and End scroll to the explicit bounds.
	scrollCalls.length = 0;
	dispatch("Home", scroller);
	expect(scrollCalls[0]).toEqual({ left: 0, behavior: "smooth" });

	scrollCalls.length = 0;
	dispatch("End", scroller);
	expect(scrollCalls[0]).toEqual({ left: 640, behavior: "smooth" });

	// Reduced motion: scroll requests use the instant "auto" behaviour.
	reduced = true;
	scrollCalls.length = 0;
	result = dispatch("ArrowRight", scroller);
	expect(scrollCalls).toHaveLength(1);
	expect(scrollCalls[0]).toEqual({ left: 120, behavior: "auto" });
	expect(result.prevented).toBe(true);
	expect(result.stopped).toBe(true);
	reduced = false;

	// Editable exclusion: an editable descendant must not scroll or consume keys.
	const input = new FakeHTMLElement();
	input.isContentEditable = true;
	input.closest = () => scroller;
	scrollCalls.length = 0;
	result = dispatch("ArrowRight", input);
	expect(scrollCalls).toHaveLength(0);
	expect(result.prevented).toBe(false);
	expect(result.stopped).toBe(false);

	// Modified keys pass through untouched.
	scrollCalls.length = 0;
	result = dispatch("ArrowRight", scroller, true);
	expect(scrollCalls).toHaveLength(0);
	expect(result.prevented).toBe(false);

	// Non-scroller focus: no action and the key is left for the global handler.
	const elsewhere = new FakeHTMLElement();
	elsewhere.closest = () => null;
	document.activeElement = elsewhere as unknown as FakeHTMLElement;
	scrollCalls.length = 0;
	result = dispatch("ArrowRight", elsewhere);
	expect(scrollCalls).toHaveLength(0);
	expect(result.prevented).toBe(false);
	expect(result.stopped).toBe(false);
});
