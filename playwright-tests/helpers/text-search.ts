import {
	expect,
	type Locator,
	type Page,
	type Request,
} from "@playwright/test";

// Shared acceptance checks for enhanced public text-search forms whose own
// response replaces the form (stabilise-public-text-search). Typing uses a
// per-key delay shorter than the 500 ms debounce so the checks exercise the
// real input/debounce/swap path rather than a single `fill` event.

const keyDelay = 90;
const debounceSettle = 900;

export type TextSearch = {
	page: Page;
	field: Locator;
	param: string;
	path: string;
};

function searchRequests(search: TextSearch): Request[] {
	const requests: Request[] = [];
	search.page.on("request", (request) => {
		const url = new URL(request.url());
		if (
			request.headers()["hx-request"] === "true" &&
			url.pathname === search.path &&
			url.searchParams.has(search.param)
		) {
			requests.push(request);
		}
	});
	return requests;
}

function paramOf(request: Request, param: string): string {
	return new URL(request.url()).searchParams.get(param) ?? "";
}

async function expectFocusedAtEnd(field: Locator, value: string) {
	await expect(field).toHaveValue(value);
	await expect(field).toBeFocused();
	const caret = await field.evaluate(
		(element: HTMLInputElement) => element.selectionStart,
	);
	expect(caret).toBe(value.length);
}

// Types a word at a normal pace and asserts that exactly one search request is
// issued for the complete word and that the swap keeps value, focus, and caret.
export async function expectDebouncedSearch(search: TextSearch, word: string) {
	const requests = searchRequests(search);
	await search.field.click();
	const response = search.page.waitForResponse(
		(candidate) =>
			new URL(candidate.url()).pathname === search.path &&
			paramOf(candidate.request(), search.param) === word,
	);
	await search.field.pressSequentially(word, { delay: keyDelay });
	await response;
	await search.page.waitForTimeout(debounceSettle);

	expect(requests.map((request) => paramOf(request, search.param))).toEqual([
		word,
	]);
	await expectFocusedAtEnd(search.field, word);
}

// Types a prefix, lets its request start, then keeps typing while its delayed
// response is swapped in. No typed character may be lost, and the follow-up
// request must carry the complete value.
export async function expectTypingSurvivesInFlightSwap(
	search: TextSearch,
	prefix: string,
	rest: string,
) {
	let delayed = false;
	const matcher = (url: URL) =>
		url.pathname === search.path && url.searchParams.has(search.param);
	await search.page.route(matcher, async (route) => {
		if (!delayed && paramOf(route.request(), search.param) === prefix) {
			delayed = true;
			await new Promise((resolve) => setTimeout(resolve, 300));
		}
		await route.continue();
	});

	await search.field.click();
	const firstRequest = search.page.waitForRequest(
		(request) =>
			new URL(request.url()).pathname === search.path &&
			paramOf(request, search.param) === prefix,
	);
	const firstResponse = search.page.waitForResponse(
		(response) => paramOf(response.request(), search.param) === prefix,
	);
	await search.field.pressSequentially(prefix, { delay: keyDelay });
	await firstRequest;
	const completeResponse = search.page.waitForResponse(
		(response) => paramOf(response.request(), search.param) === prefix + rest,
	);
	await search.field.pressSequentially(rest, { delay: keyDelay });
	await firstResponse;
	await completeResponse;
	await search.page.unroute(matcher);

	await expectFocusedAtEnd(search.field, prefix + rest);
}

// Activates another control that replaces the same block and asserts that the
// field shows the resulting state's value rather than the typed text; then
// returns through history and asserts the earlier value is restored.
export async function expectNavigationAndHistoryUseStateValues(
	search: TextSearch,
	typed: string,
	navigate: () => Promise<void>,
) {
	await expect(search.page).toHaveURL(
		new RegExp(`[?&]${search.param}=${encodeURIComponent(typed)}`),
	);
	const before = search.page.url();
	await navigate();
	await expect(search.page).not.toHaveURL(before);
	const after = new URL(search.page.url()).searchParams.get(search.param) ?? "";
	await expect(search.field).toHaveValue(after);
	expect(after).not.toBe(typed);

	await search.page.goBack();
	await expect(search.page).toHaveURL(before);
	await expect(search.field).toHaveValue(typed);
}

// Searches twice from an empty field, then walks back through history. Each
// earlier state must show its own query, not text typed after it was left.
export async function expectHistoryMatchesEachSearchState(
	search: TextSearch,
	first: string,
	second: string,
) {
	const initial = search.page.url();
	await expect(search.field).toHaveValue("");
	for (const value of [first, first + second]) {
		const response = search.page.waitForResponse(
			(candidate) =>
				new URL(candidate.url()).pathname === search.path &&
				paramOf(candidate.request(), search.param) === value,
		);
		await search.field.click();
		await search.field.press("End");
		await search.field.pressSequentially(
			value.slice(value === first ? 0 : first.length),
			{
				delay: keyDelay,
			},
		);
		await response;
		await search.page.waitForTimeout(debounceSettle);
	}

	await search.page.goBack();
	await expect(search.page).toHaveURL(
		new RegExp(`[?&]${search.param}=${encodeURIComponent(first)}(&|$)`),
	);
	await expect(search.field).toHaveValue(first);

	await search.page.goBack();
	await expect(search.page).toHaveURL(initial);
	await expect(search.field).toHaveValue("");
}
