import { expect, type Page, test } from "@playwright/test";

// An enhanced request cannot solve an edge (Cloudflare) managed challenge, so
// a challenged navigation must fall back to a full document load.

const challengeBody =
	"<!DOCTYPE html><html><head><title>Just a moment...</title></head><body>challenge</body></html>";

async function answerEnhancedArtworks(
	page: Page,
	headers: Record<string, string>,
) {
	let answered = 0;
	await page.route(/\/artworks(\?|$)/, async (route) => {
		const request = route.request();
		if (request.headers()["hx-request"] === "true" && answered === 0) {
			answered++;
			await route.fulfill({
				status: 403,
				contentType: "text/html; charset=UTF-8",
				headers,
				body: challengeBody,
			});
			return;
		}
		await route.continue();
	});
}

test("a challenged navigation link falls back to a full page load", async ({
	page,
}) => {
	await page.goto("/artists");
	await answerEnhancedArtworks(page, { "cf-mitigated": "challenge" });
	const documentLoad = page.waitForRequest(
		(request) =>
			request.resourceType() === "document" &&
			new URL(request.url()).pathname === "/artworks",
	);

	await page.locator("a[href='/artworks']:visible").first().click();

	const request = await documentLoad;
	expect(request.headers()["hx-request"]).toBeUndefined();
	await expect(page).toHaveURL(/\/artworks$/);
	await expect(page.locator("h1").first()).toHaveText("Artworks");
});

test("a plain 403 on an enhanced navigation does not navigate", async ({
	page,
}) => {
	await page.goto("/artists");
	await answerEnhancedArtworks(page, {});
	const forbidden = page.waitForResponse(
		(response) =>
			response.status() === 403 &&
			new URL(response.url()).pathname === "/artworks",
	);

	await page.locator("a[href='/artworks']:visible").first().click();
	await forbidden;
	await page.waitForTimeout(500);

	await expect(page).toHaveURL(/\/artists$/);
	await expect(page.locator("h1").first()).toHaveText("Artists");
});
