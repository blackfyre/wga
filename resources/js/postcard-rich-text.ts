import DOMPurify from "dompurify";
import Squire from "squire-rte";
import logger from "./logger";

const ALLOWED_TAGS = ["p", "b", "i", "ul", "ol", "li"];
const mounted = new Map<HTMLElement, () => void>();
let initialised = false;

type Command = "bold" | "italic" | "ul" | "ol";

const formatTag: Record<Command, string> = {
	bold: "B",
	italic: "I",
	ul: "UL",
	ol: "OL",
};

const toggleCommand: Record<
	Command,
	(editor: Squire, active: boolean) => void
> = {
	bold: (editor, active) => (active ? editor.removeBold() : editor.bold()),
	italic: (editor, active) =>
		active ? editor.removeItalic() : editor.italic(),
	ul: (editor, active) =>
		active ? editor.removeList() : editor.makeUnorderedList(),
	ol: (editor, active) =>
		active ? editor.removeList() : editor.makeOrderedList(),
};

function sanitiseToFragment(html: string): DocumentFragment {
	return DOMPurify.sanitize(html, {
		ALLOWED_TAGS,
		ALLOWED_ATTR: [],
		ALLOW_ARIA_ATTR: false,
		ALLOW_DATA_ATTR: false,
		RETURN_DOM_FRAGMENT: true,
	}) as DocumentFragment;
}

function plainTextParagraph(text: string): string {
	const paragraph = document.createElement("p");
	paragraph.textContent = text;
	return paragraph.outerHTML;
}

function textOffset(
	block: HTMLElement,
	container: Node,
	offset: number,
): number {
	const range = document.createRange();
	range.selectNodeContents(block);
	range.setEnd(container, offset);
	return range.toString().length;
}

function textBoundary(
	block: HTMLElement,
	target: number,
): { node: Node; offset: number } | null {
	const walker = document.createTreeWalker(block, NodeFilter.SHOW_TEXT);
	let traversed = 0;
	for (let node = walker.nextNode(); node; node = walker.nextNode()) {
		const length = (node.nodeValue ?? "").length;
		if (traversed + length >= target) {
			return { node, offset: Math.min(length, target - traversed) };
		}
		traversed += length;
	}
	return null;
}

type SelectionAnchor = {
	startBlockIndex: number;
	startBlockText: string;
	endBlockIndex: number;
	endBlockText: string;
	start: number;
	end: number;
};

function edgeTextNode(node: Node, atEnd: boolean): Text | null {
	let edge = node;
	while (edge.hasChildNodes()) {
		edge = atEnd ? (edge.lastChild as Node) : (edge.firstChild as Node);
	}
	return edge instanceof Text ? edge : null;
}

function selectionBoundary(
	container: Node,
	offset: number,
	atEnd: boolean,
): { node: Node; offset: number } | null {
	if (container.nodeType === Node.TEXT_NODE) return { node: container, offset };
	const before = offset > 0 ? container.childNodes[offset - 1] : null;
	const after =
		offset < container.childNodes.length ? container.childNodes[offset] : null;
	const child = atEnd ? (before ?? after) : (after ?? before);
	const useEnd = child === before;
	const node = child ? edgeTextNode(child, useEnd) : null;
	if (!node) return null;
	return { node, offset: useEnd ? (node.nodeValue?.length ?? 0) : 0 };
}

function normalise(surface: HTMLElement): void {
	for (const list of surface.querySelectorAll("ul, ol")) {
		if (list.querySelector("li")) continue;
		const paragraph = document.createElement("p");
		while (list.firstChild) paragraph.appendChild(list.firstChild);
		list.parentNode?.replaceChild(paragraph, list);
	}
}

function mount(root: HTMLElement): void {
	if (mounted.has(root)) return;
	const surface = root.querySelector<HTMLElement>("[data-rte-surface]");
	const source = root.querySelector<HTMLTextAreaElement>("[data-rte-source]");
	const toolbar = root.querySelector<HTMLElement>("[data-rte-toolbar]");
	const counter = root.querySelector<HTMLElement>("[data-rte-count]");
	const form = root.closest("form");
	if (!surface || !source || !toolbar || !counter || !form) {
		logger.error("Postcard rich-text field is incomplete");
		return;
	}

	const limit = Number.parseInt(root.dataset.rteLimit ?? "300", 10);
	const editor = new Squire(surface, {
		blockTag: "P",
		sanitizeToDOMFragment: (html) => sanitiseToFragment(html),
	});
	const initial = source.value.trim().startsWith("<")
		? source.value
		: plainTextParagraph(source.value);
	editor.setHTML(initial);
	surface.hidden = false;
	toolbar.hidden = false;
	source.hidden = true;
	source.removeAttribute("required");

	let anchor: SelectionAnchor | null = null;
	const rememberSelection = () => {
		const selection = window.getSelection();
		if (!selection?.rangeCount) return;
		const range = selection.getRangeAt(0);
		anchor = null;
		const start = selectionBoundary(
			range.startContainer,
			range.startOffset,
			false,
		);
		const end = selectionBoundary(range.endContainer, range.endOffset, true);
		const startBlock = start?.node.parentElement?.closest<HTMLElement>("p, li");
		const endBlock = end?.node.parentElement?.closest<HTMLElement>("p, li");
		if (
			!start ||
			!end ||
			!startBlock ||
			!endBlock ||
			!surface.contains(startBlock) ||
			!surface.contains(endBlock)
		)
			return;
		const blocks = Array.from(surface.querySelectorAll<HTMLElement>("p, li"));
		anchor = {
			startBlockIndex: blocks.indexOf(startBlock),
			startBlockText: startBlock.textContent ?? "",
			endBlockIndex: blocks.indexOf(endBlock),
			endBlockText: endBlock.textContent ?? "",
			start: textOffset(startBlock, start.node, start.offset),
			end: textOffset(endBlock, end.node, end.offset),
		};
	};
	const restoreSelection = (): Range | null => {
		if (!anchor) return null;
		const blocks = Array.from(surface.querySelectorAll<HTMLElement>("p, li"));
		const startBlock =
			blocks.at(anchor.startBlockIndex) ??
			blocks.find(
				(candidate) => candidate.textContent === anchor?.startBlockText,
			) ??
			blocks.at(-1);
		const endBlock =
			blocks.at(anchor.endBlockIndex) ??
			blocks.find(
				(candidate) => candidate.textContent === anchor?.endBlockText,
			) ??
			blocks.at(-1);
		if (!startBlock || !endBlock) return null;
		const start = textBoundary(startBlock, anchor.start);
		if (!start) return null;
		const end = textBoundary(endBlock, anchor.end);
		const range = document.createRange();
		range.setStart(start.node, start.offset);
		if (end) range.setEnd(end.node, end.offset);
		else range.collapse(true);
		return range;
	};
	const reflectState = () => {
		for (const button of toolbar.querySelectorAll<HTMLElement>(
			"[data-rte-cmd]",
		)) {
			const command = button.dataset.rteCmd as Command;
			button.setAttribute(
				"aria-pressed",
				String(editor.hasFormat(formatTag[command])),
			);
		}
	};
	const sync = () => {
		normalise(surface);
		source.value = editor.getHTML();
		const count = Array.from(surface.textContent ?? "").length;
		const remaining = limit - count;
		counter.textContent = `${remaining} ${remaining === 1 ? "character" : "characters"} remaining`;
		counter.classList.toggle("text-wga-error", remaining < 0);
		surface.toggleAttribute("data-rte-empty", count === 0);
		reflectState();
	};
	const onToolbarMouseDown = (event: MouseEvent) => event.preventDefault();
	const onToolbarClick = (event: MouseEvent) => {
		const button = (event.target as HTMLElement | null)?.closest<HTMLElement>(
			"[data-rte-cmd]",
		);
		const command = button?.dataset.rteCmd as Command | undefined;
		if (!command) return;
		const range = restoreSelection();
		if (!range) return;
		const active = editor.hasFormat(formatTag[command]);
		if (
			(command === "bold" || command === "italic") &&
			!active &&
			range.collapsed
		)
			return;
		try {
			editor.setSelection(range);
			toggleCommand[command](editor, active);
		} catch (error) {
			logger.warn("Unable to apply postcard formatting", error);
			return;
		}
		rememberSelection();
		sync();
	};
	const onSubmit = (event: SubmitEvent) => {
		sync();
		if (
			event.submitter instanceof HTMLButtonElement &&
			event.submitter.formNoValidate
		)
			return;
		const count = Array.from(surface.textContent ?? "").length;
		if (count > 0 && count <= limit) return;
		event.preventDefault();
		surface.focus();
	};

	editor.addEventListener("input", sync);
	editor.addEventListener("pathChange", reflectState);
	editor.addEventListener("select", reflectState);
	document.addEventListener("selectionchange", rememberSelection);
	surface.addEventListener("keyup", rememberSelection);
	surface.addEventListener("mouseup", rememberSelection);
	toolbar.addEventListener("mousedown", onToolbarMouseDown);
	toolbar.addEventListener("click", onToolbarClick);
	form.addEventListener("submit", onSubmit);
	sync();

	mounted.set(root, () => {
		editor.removeEventListener("input", sync);
		editor.removeEventListener("pathChange", reflectState);
		editor.removeEventListener("select", reflectState);
		document.removeEventListener("selectionchange", rememberSelection);
		surface.removeEventListener("keyup", rememberSelection);
		surface.removeEventListener("mouseup", rememberSelection);
		toolbar.removeEventListener("mousedown", onToolbarMouseDown);
		toolbar.removeEventListener("click", onToolbarClick);
		form.removeEventListener("submit", onSubmit);
		editor.destroy();
		mounted.delete(root);
	});
}

function mountAll(root: ParentNode = document): void {
	if (root instanceof HTMLElement && root.matches("[data-wga-rte]"))
		mount(root);
	for (const field of root.querySelectorAll<HTMLElement>("[data-wga-rte]"))
		mount(field);
}

function cleanupWithin(element: Element): void {
	for (const [root, cleanup] of mounted) {
		if (root === element || element.contains(root)) cleanup();
	}
}

export function initPostcardRichText(): void {
	if (initialised) return;
	initialised = true;
	mountAll();
	document.addEventListener("htmx:load", (event) =>
		mountAll(event.target as ParentNode),
	);
	document.addEventListener("htmx:beforeCleanupElement", (event) =>
		cleanupWithin(event.target as Element),
	);
}
