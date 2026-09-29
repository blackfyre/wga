// Completes the in-progress-input contract for enhanced public text-search
// forms ([data-text-search] fields) whose own response replaces the form.
// The server keeps the typed field in place with hx-preserve, but two things
// cannot survive the swap without help:
//
// - The pending debounce timer belongs to the replaced form, and HTMX drops
//   its request once that form has left the document. Text typed while a
//   request was in flight is kept but never searched, so after the form's own
//   response settles, a field whose value differs from the one that request
//   carried is re-announced with an input event, which re-arms the new form's
//   debounce.
// - HTMX snapshots history as innerHTML, which records value attributes. The
//   preserved input keeps its first-render attribute, so when the form's own
//   response settles the attribute is set to the value that request carried.
//   Each later snapshot then records the value of the state it is saved under,
//   not text typed since.
//
// Listeners are document-level and bound once, so they survive every swap.

const FIELD_SELECTOR = "input[data-text-search]";

type RequestedField = { field: HTMLInputElement; value: string };

const requestedFields = new WeakMap<Element, RequestedField[]>();

let initialised = false;

type HtmxDetail = { elt?: unknown; requestConfig?: { elt?: unknown } };

const detailOf = (event: Event): HtmxDetail =>
	(event as CustomEvent<HtmxDetail>).detail ?? {};

export const initTextSearch = (): void => {
	if (initialised) {
		return;
	}
	initialised = true;

	document.addEventListener("htmx:beforeRequest", (event) => {
		const form = detailOf(event).elt;
		if (!(form instanceof HTMLFormElement)) {
			return;
		}
		const fields = Array.from(
			form.querySelectorAll<HTMLInputElement>(FIELD_SELECTOR),
		).map((field) => ({ field, value: field.value }));
		if (fields.length > 0) {
			requestedFields.set(form, fields);
		}
	});

	document.addEventListener("htmx:afterSettle", (event) => {
		const form = detailOf(event).requestConfig?.elt;
		if (!(form instanceof Element)) {
			return;
		}
		const fields = requestedFields.get(form);
		if (!fields) {
			return;
		}
		requestedFields.delete(form);
		for (const { field, value } of fields) {
			if (!field.isConnected) {
				continue;
			}
			field.setAttribute("value", value);
			if (field.value !== value) {
				field.dispatchEvent(new Event("input", { bubbles: true }));
			}
		}
	});
};
