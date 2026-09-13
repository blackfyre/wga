import { expect, test } from "@playwright/test";

const storageKey = "wga-study-board";

async function publishedArtworkIDs(page): Promise<string[]> {
	await page.goto("/artworks");
	const links = page.locator(
		"#artwork-search-results [data-view='grid'] a.wga-record-card",
	);
	await expect(links.first()).toBeVisible({ timeout: 30000 });
	const hrefs = await links.evaluateAll((elements) =>
		elements.map((element) => (element as HTMLAnchorElement).pathname),
	);
	const ids = hrefs
		.map((href) => href.match(/([A-Za-z0-9_-]{15})$/)?.[1] ?? "")
		.filter(Boolean);
	return [...new Set(ids)].slice(0, 2);
}

test("restores, shares, canonicalises, and clears transient board state", async ({
	context,
	page,
}) => {
	test.setTimeout(60000);
	await context.grantPermissions(["clipboard-read", "clipboard-write"]);
	const ids = await publishedArtworkIDs(page);
	expect(ids).toHaveLength(2);

	await page.evaluate(
		([key, id]) => window.localStorage.setItem(key, id),
		[storageKey, ids[0]],
	);
	await page.goto("/study-board");
	await expect(page).toHaveURL(
		(url) => url.searchParams.get("board") === ids[0],
	);
	await expect(page.locator(`[data-study-board-work='${ids[0]}']`)).toHaveCount(
		1,
	);

	await page.evaluate(
		([key, id]) => window.localStorage.setItem(key, id),
		[storageKey, ids[0]],
	);
	await page.goto(`/study-board?board=${ids[1]}`);
	await expect(page.locator(`[data-study-board-work='${ids[1]}']`)).toHaveCount(
		1,
	);
	await expect(page.locator(`[data-study-board-work='${ids[0]}']`)).toHaveCount(
		0,
	);
	expect(
		await page.evaluate((key) => window.localStorage.getItem(key), storageKey),
	).toBe(ids[1]);

	await page.goto(`/study-board?board=missing00000001,${ids[1]},${ids[1]}`);
	await expect(page).toHaveURL(
		(url) => url.searchParams.get("board") === ids[1],
	);
	await expect(page.locator(`[data-study-board-work='${ids[1]}']`)).toHaveCount(
		1,
	);

	const canonicalURL = page.url();
	await page.getByRole("button", { name: "COPY LINK" }).click();
	await expect(page.getByRole("button", { name: "COPIED" })).toBeVisible();
	expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(
		canonicalURL,
	);

	await page.getByRole("link", { name: "CLEAR", exact: true }).click();
	await expect(page).toHaveURL(
		(url) => url.pathname === "/study-board" && url.search === "",
	);
	await expect(page.locator("[data-study-board-empty]")).toBeVisible();
	expect(
		await page.evaluate((key) => window.localStorage.getItem(key), storageKey),
	).toBeNull();
});
