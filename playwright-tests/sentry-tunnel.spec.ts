import { expect, test } from "@playwright/test";

// Browser Sentry delivery goes through the first-party relay so content
// blockers that reject sentry.io do not drop events. The check needs a server
// started with WGA_SENTRY_BROWSER_DSN and is skipped otherwise, so default and
// CI runs do not cover it; delivery to Sentry itself is never exercised here.
test("browser errors are sent to the first-party relay, not sentry.io", async ({
	page,
}) => {
	const direct: string[] = [];
	await page.route(/sentry\.io/, (route) => {
		direct.push(route.request().url());
		return route.abort("blockedbyclient");
	});

	await page.goto("/artworks");
	const dsn = await page
		.locator('meta[name="sentry-dsn"]')
		.getAttribute("content");
	test.skip(!dsn, "browser Sentry DSN is not configured for this server");

	const relayed = page.waitForRequest(
		(request) =>
			request.method() === "POST" &&
			new URL(request.url()).pathname === "/diagnostics/browser" &&
			(request.postData() ?? "").includes("sentry tunnel acceptance check"),
	);
	await page.evaluate(() => {
		setTimeout(() => {
			throw new Error("sentry tunnel acceptance check");
		});
	});
	const request = await relayed;

	expect(new URL(request.url()).origin).toBe(new URL(page.url()).origin);
	const header = JSON.parse((request.postData() ?? "").split("\n")[0]);
	expect(header.dsn).toBe(dsn);
	expect(direct).toEqual([]);

	// The relay must accept the envelope. Without a reachable Sentry host in
	// test runs, acceptance shows up as a forwarded result or a gateway error.
	const response = await request.response();
	expect(response).not.toBeNull();
	expect([400, 403, 413, 415, 429]).not.toContain(response?.status());
});
