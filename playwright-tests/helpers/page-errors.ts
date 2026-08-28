import { type Page, expect } from "@playwright/test";

// Strict console-error and uncaught-pageerror capture shared by the public-shell
// acceptance suites. Errors accumulate per worker (each Playwright worker runs
// one test at a time in its own process), are reset before each test, and must
// be empty when the test finishes. Guard every page — the `page` fixture and any
// manually created context/page — so a failed asset load, uncaught exception, or
// console.error can never pass unnoticed.
const consoleErrors: string[] = [];
const pageErrors: string[] = [];

export function guardPageErrors(page: Page): void {
	page.on("console", (message) => {
		if (message.type() === "error") {
			consoleErrors.push(message.text());
		}
	});
	page.on("pageerror", (error) => {
		pageErrors.push(error.message);
	});
}

export function resetErrorCapture(): void {
	consoleErrors.length = 0;
	pageErrors.length = 0;
}

export function expectNoPageErrors(): void {
	expect(pageErrors).toEqual([]);
	expect(consoleErrors).toEqual([]);
}
