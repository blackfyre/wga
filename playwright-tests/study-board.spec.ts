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

	page.once("dialog", (dialog) => dialog.accept());
	await page.getByRole("link", { name: "CLEAR", exact: true }).click();
	await expect(page).toHaveURL(
		(url) => url.pathname === "/study-board" && url.search === "",
	);
	await expect(page.locator("[data-study-board-empty]")).toBeVisible();
	expect(
		await page.evaluate((key) => window.localStorage.getItem(key), storageKey),
	).toBeNull();
});

test("adds without record navigation, shows the fixed shelf, and reorders both views", async ({
	page,
}) => {
	test.setTimeout(60000);
	await page.goto("/artworks");
	const actions = page.locator("[data-study-board-add]");
	await expect(actions.first()).toBeVisible({ timeout: 30000 });
	const first = await actions.nth(0).getAttribute("data-study-board-add");
	const second = await actions.nth(1).getAttribute("data-study-board-add");
	const secondTitle = await actions
		.nth(1)
		.getAttribute("data-study-board-title");
	expect(first).toBeTruthy();
	expect(second).toBeTruthy();
	expect(secondTitle).toBeTruthy();

	await actions.nth(0).click();
	await expect(page).toHaveURL((url) => url.pathname === "/artworks");
	await expect(page.getByRole("region", { name: "Study Board" })).toContainText(
		"1 OF 12",
	);
	await expect
		.poll(() =>
			page.evaluate(() =>
				Number.parseFloat(
					document.body.style.getPropertyValue("--wga-bottom-stack-height"),
				),
			),
		)
		.toBeGreaterThan(0);
	await expect(actions.nth(0)).toHaveText("ON STUDY BOARD ✓");
	await expect(actions.nth(0)).not.toHaveClass(/hover:border-wga-accent/);
	await expect(actions.nth(0)).not.toHaveClass(/hover:bg-wga-accent-tint/);

	await page.keyboard.press("Control+k");
	await page
		.getByRole("searchbox", { name: "Search sections, artists and works" })
		.fill(secondTitle ?? "");
	const paletteAction = page.locator(
		`#kbd-palette-records [data-study-board-add='${second}']`,
	);
	await expect(paletteAction).toHaveText("ADD TO STUDY BOARD +");
	await page.keyboard.press("Tab");
	await expect(paletteAction).toBeFocused();
	await page.keyboard.press("Enter");
	await expect(paletteAction).toHaveText("ON STUDY BOARD ✓");
	await expect(page.getByRole("region", { name: "Study Board" })).toContainText(
		"2 OF 12",
	);
	await page.keyboard.press("Escape");
	await page.getByRole("link", { name: "OPEN BOARD →" }).click();
	await expect(page).toHaveURL(
		(url) => url.searchParams.get("board") === `${first},${second}`,
	);

	await expect(page.getByRole("button", { name: "MATRIX" })).toHaveAttribute(
		"aria-pressed",
		"true",
	);
	await page.getByRole("button", { name: "BOARD" }).click();
	const board = page.locator("[data-study-board-panel='board']");
	await expect(board).toBeVisible();
	await board
		.locator(`[data-study-board-work='${second}']`)
		.getByRole("button", { name: "Move earlier" })
		.click();
	await expect(page).toHaveURL(
		(url) => url.searchParams.get("board") === `${second},${first}`,
	);

	await page.getByRole("button", { name: "BOARD" }).click();
	await page
		.locator("[data-study-board-panel='board']")
		.locator(`[data-study-board-work='${second}']`)
		.getByRole("button", { name: "REMOVE" })
		.click();
	await expect(page).toHaveURL(
		(url) => url.searchParams.get("board") === first,
	);
});

test("reports capacity without mutating or navigating", async ({ page }) => {
	test.setTimeout(60000);
	await page.goto("/artworks");
	const actions = page.locator("[data-study-board-add]");
	await expect(actions.nth(12)).toBeVisible({ timeout: 30000 });
	const ids = await actions.evaluateAll((buttons) =>
		buttons.map(
			(button) => (button as HTMLElement).dataset.studyBoardAdd ?? "",
		),
	);
	expect(ids.length).toBeGreaterThan(12);
	await page.evaluate(
		([key, values]) => window.localStorage.setItem(key, values.join(",")),
		[storageKey, ids.slice(0, 12)] as [string, string[]],
	);
	await page.reload();
	const outside = page.locator(`[data-study-board-add='${ids[12]}']`);
	await expect(outside).toHaveText("STUDY BOARD FULL");
	await expect(outside).toBeDisabled();
	const beforeURL = page.url();
	const beforeState = await page.evaluate(
		(key) => localStorage.getItem(key),
		storageKey,
	);
	await outside.click({ force: true });
	expect(page.url()).toBe(beforeURL);
	expect(
		await page.evaluate((key) => localStorage.getItem(key), storageKey),
	).toBe(beforeState);
});
