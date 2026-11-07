import { expect, type Page, test } from "@playwright/test";

// A browser holding an entry module from an earlier release can request a
// dynamic chunk that the current release no longer serves. The entry URL is
// versioned by content and revalidates, hashed chunks stay immutable, and a
// failed dynamic import reloads once before offering a manual retry.

const bootstrapChunk = /\/assets\/js\/bootstrap-[^/]+\.js(\?|$)/;
const recoveryKey = "wga.dynamic-module-recovery";

function countDocumentLoads(page: Page, pathname: string): () => number {
	let loads = 0;
	page.on("request", (request) => {
		if (
			request.resourceType() === "document" &&
			new URL(request.url()).pathname === pathname
		) {
			loads++;
		}
	});
	return () => loads;
}

test("the entry module revalidates and hashed chunks are immutable", async ({
	page,
}) => {
	const entry = page.waitForResponse(
		(response) => new URL(response.url()).pathname === "/assets/js/app.js",
	);
	const chunk = page.waitForResponse((response) =>
		bootstrapChunk.test(response.url()),
	);

	await page.goto("/artists");

	const entrySource = await page
		.locator('script[type="module"][src^="/assets/js/app.js"]')
		.getAttribute("src");
	expect(entrySource).toMatch(/^\/assets\/js\/app\.js\?v=[0-9a-f]{12}$/);
	expect(new URL((await entry).url()).search).toBe(
		new URL(entrySource ?? "", page.url()).search,
	);
	expect((await entry).headers()["cache-control"]).toBe("no-cache");
	expect((await chunk).headers()["cache-control"]).toBe(
		"public, max-age=31536000, immutable",
	);
});

test("a transient dynamic-module failure reloads once and keeps the URL", async ({
	page,
}) => {
	let failures = 0;
	await page.route(bootstrapChunk, async (route) => {
		if (failures === 0) {
			failures++;
			await route.abort();
			return;
		}
		await route.continue();
	});
	const documentLoads = countDocumentLoads(page, "/artists");
	const recovered = page.waitForResponse(
		(response) => bootstrapChunk.test(response.url()) && response.ok(),
	);

	await page.goto("/artists?letter=b");
	await recovered;

	await expect(page).toHaveURL(/\/artists\?letter=b$/);
	expect(documentLoads()).toBe(2);
	await expect
		.poll(() =>
			page.evaluate((key) => window.sessionStorage.getItem(key), recoveryKey),
		)
		.toBeNull();
	await expect(page.getByRole("button", { name: "Reload page" })).toHaveCount(
		0,
	);
});

test("a persistent dynamic-module failure offers a reload action instead of looping", async ({
	page,
}) => {
	await page.route(bootstrapChunk, (route) => route.abort());
	const documentLoads = countDocumentLoads(page, "/artists");

	await page.goto("/artists?letter=b");

	const alert = page.getByRole("alert").filter({ hasText: "assets changed" });
	await expect(alert).toBeVisible();
	await expect(page).toHaveURL(/\/artists\?letter=b$/);
	await page.waitForTimeout(500);
	expect(documentLoads()).toBe(2);

	const reload = page.waitForRequest(
		(request) =>
			request.resourceType() === "document" &&
			new URL(request.url()).pathname === "/artists",
	);
	await alert.getByRole("button", { name: "Reload page" }).click();
	await reload;
	expect(documentLoads()).toBe(3);
});
