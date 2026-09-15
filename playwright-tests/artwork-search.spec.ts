import { expect, test } from "@playwright/test";

test.describe.configure({ mode: "serial" });
test.setTimeout(60000);

async function expectArtworkResults(page) {
	await expect(
		page.locator("#artwork-search-results [data-view='grid']"),
	).toBeVisible({
		timeout: 30000,
	});
}

const artworkSearchForm = (page) => page.locator("#artwork-filters");

async function chooseFilter(page, name, value) {
	let input = page.locator(`[name='${name}'][value='${value}']`);
	if (
		(await input.count()) === 0 &&
		["art_school", "art_form"].includes(name)
	) {
		const label = name === "art_school" ? "SCHOOL" : "FORM";
		const disclosure = page
			.locator("#artwork-filters > details")
			.filter({ has: page.locator("summary", { hasText: label }) });
		if (
			!(await disclosure.evaluate(
				(details: HTMLDetailsElement) => details.open,
			))
		) {
			await disclosure.locator("summary").click();
		}
		await disclosure.getByRole("link", { name: /^SHOW ALL/ }).click();
		input = page.locator(`[name='${name}'][value='${value}']`);
		await expect(input).toHaveCount(1);
	}
	const facet = input.locator("xpath=ancestor::details[1]");
	if (!(await facet.evaluate((details: HTMLDetailsElement) => details.open))) {
		await facet.locator("summary").click();
	}

	const response = page.waitForResponse(
		(response) =>
			new URL(response.url()).pathname === "/artworks" &&
			new URL(response.url()).searchParams.getAll(name).includes(value),
	);
	await input.locator("..").click();
	await response;
	await expect(page).toHaveURL((url) =>
		url.searchParams.getAll(name).includes(value),
	);
	await expectArtworkResults(page);
}

test("school facet accepts repeated selections", async ({ page }) => {
	await page.goto("/artworks");
	await chooseFilter(page, "art_school", "bohemian");
	const second = page
		.locator("input[name='art_school']:not(:checked):not(:disabled)")
		.first();
	const secondValue = await second.getAttribute("value");
	expect(secondValue).toBeTruthy();
	await chooseFilter(page, "art_school", secondValue as string);

	await expect(page).toHaveURL(
		(url) => url.searchParams.getAll("art_school").length === 2,
	);
	await expect(page.locator("input[name='art_school']:checked")).toHaveCount(2);
	await expect(
		page
			.getByText("SCHOOL", { exact: true })
			.locator("xpath=ancestor::details[1]")
			.locator("summary"),
	).toContainText("2 SELECTED");
});

test("sort and view controls use the compact reference rank", async ({
	page,
}) => {
	await page.goto("/artworks");
	const titleAscending = page.getByRole("link", { name: "TITLE A–Z" });
	await expect(titleAscending).toHaveCSS("height", "32px");
	await expect(page.getByRole("link", { name: "GRID", exact: true })).toHaveCSS(
		"height",
		"32px",
	);
});

test("active sort control toggles its named direction", async ({ page }) => {
	await page.goto("/artworks");
	const titleAscending = page.getByRole("link", { name: "TITLE A–Z" });
	await titleAscending.click();
	await expect(page).toHaveURL((url) => url.searchParams.get("dir") === "desc");
	await page.getByRole("link", { name: "TITLE Z–A" }).click();
	await expect(page).toHaveURL(
		(url) => !url.searchParams.has("sort") && !url.searchParams.has("dir"),
	);
	await expect(
		page.getByRole("link", { name: "Reverse sort direction" }),
	).toHaveCount(0);
});

test("artwork search", async ({ page }) => {
	await page.goto("/artworks");
	await expect(page.locator("h1")).toHaveText("Artworks");
	await artworkSearchForm(page)
		.getByRole("searchbox")
		.fill("Synthetic Artwork 01-01");
	await expectArtworkResults(page);
	await expect(page.locator("#search-result-container")).toContainText(
		"1 WORK MATCHES",
	);
});

test("search cards request portrait thumbnails", async ({ page }) => {
	await page.goto("/artworks");
	await expectArtworkResults(page);

	await expect(
		page.locator("#artwork-search-results [data-view='grid'] img").first(),
	).toHaveAttribute("src", /thumb=500x0/);
});

test("grid results expose reference metadata and independent workspace actions", async ({
	page,
}) => {
	await page.goto("/artworks?art_school=bohemian");
	await expectArtworkResults(page);
	const card = page
		.locator("#artwork-search-results [data-view='grid'] article")
		.first();
	await expect(
		card.locator('[data-artwork-result-meta="identity"]'),
	).not.toHaveText("NOT RECORDED");
	await expect(
		card.locator('[data-artwork-result-meta="classification"]'),
	).not.toHaveText("NOT RECORDED");
	const actions = card.locator("[data-artwork-workspace-actions]");
	await expect(actions.locator("button")).toHaveCount(2);
	for (const action of await actions.locator("button").all()) {
		await expect(action).toHaveCSS("height", "33px");
	}

	const itinerary = actions.locator("[data-itinerary-search-add]");
	await expect(itinerary).toHaveText("ADD TO ITINERARY +");
	await itinerary.hover();
	await expect(itinerary).toHaveClass(/hover:bg-wga-accent-tint/);
	await itinerary.focus();
	const added = page.waitForResponse(
		(response) => new URL(response.url()).pathname === "/itineraries/draft/add",
	);
	await itinerary.press("Enter");
	expect((await added).ok()).toBe(true);
	await expect(itinerary).toBeDisabled();
	await expect(itinerary).toHaveText("IN ITINERARY ✓");
	expect(new URL(page.url()).pathname).toBe("/artworks");
	await expect(page.locator("#itinerary-tray")).toContainText("1 OF 15");
	await itinerary.evaluate((button: HTMLButtonElement) => button.click());
	await expect(page.locator("#itinerary-tray")).toContainText("1 OF 15");

	const studyBoard = actions.locator("[data-study-board-add]");
	await expect(studyBoard).toHaveText("ADD TO STUDY BOARD +");
	await studyBoard.click();
	await expect(studyBoard).toBeDisabled();
	await expect(studyBoard).toHaveText("ON STUDY BOARD ✓");
	expect(new URL(page.url()).pathname).toBe("/artworks");
	await studyBoard.evaluate((button: HTMLButtonElement) => button.click());
	expect(
		await page.evaluate(
			() => window.localStorage.getItem("wga-study-board")?.split(",").length,
		),
	).toBe(1);
});

test("dense results expose desktop columns and retain actions responsively", async ({
	page,
}) => {
	await page.setViewportSize({ width: 1440, height: 900 });
	await page.goto("/artworks?art_school=bohemian&view=list");
	const row = page
		.locator("#artwork-search-results [data-view='list'] li")
		.first();
	for (const label of ["SCHOOL", "FORM", "TYPE"]) {
		await expect(row.getByText(label, { exact: true })).toBeVisible();
	}
	await expect(
		row.locator("[data-artwork-workspace-actions] button"),
	).toHaveCount(2);

	await page.setViewportSize({ width: 800, height: 900 });
	for (const label of ["SCHOOL", "FORM", "TYPE"]) {
		await expect(row.getByText(label, { exact: true })).toBeHidden();
	}
	await expect(row.locator("[data-artwork-workspace-actions]")).toBeVisible();
});

test("artwork and artist searches show nine inert placeholders across repeated swaps", async ({
	page,
}) => {
	await page.goto("/artworks?art_school=bohemian");
	await page.route(/\/artworks\?.*/, async (route) => {
		await new Promise((resolve) => setTimeout(resolve, 700));
		await route.continue();
	});
	const artworkSkeleton = page.locator("#artwork-search-skeleton");
	const artworkCount = page.locator(
		"#artwork-search-results p[aria-live='polite']",
	);
	const initialArtworkCount = await artworkCount.textContent();
	await artworkSearchForm(page).getByRole("searchbox").fill("zzzzzz");
	await expect(artworkSkeleton).toBeVisible();
	await expect(artworkSkeleton).toHaveAttribute("aria-hidden", "true");
	await expect(artworkSkeleton.locator(":scope > div > div")).toHaveCount(9);
	await expect(
		page.locator("#search-result-container .wga-search-current"),
	).toBeHidden();
	await expect(artworkSkeleton).toBeHidden({ timeout: 30000 });
	await expect
		.poll(() => artworkCount.textContent())
		.not.toBe(initialArtworkCount);
	const firstArtworkCount = await artworkCount.textContent();
	await artworkSearchForm(page).getByRole("searchbox").fill("a");
	await expect(artworkSkeleton).toBeVisible();
	await expect(artworkSkeleton).toBeHidden({ timeout: 30000 });
	await expect
		.poll(() => artworkCount.textContent())
		.not.toBe(firstArtworkCount);

	await page.goto("/artists");
	await page.route(/\/artists\?.*/, async (route) => {
		await new Promise((resolve) => setTimeout(resolve, 700));
		await route.continue();
	});
	const artistSkeleton = page.locator("#artist-search-skeleton");
	const artistCount = page.locator("#artists p[aria-live='polite']");
	const initialArtistCount = await artistCount.textContent();
	await page.locator("#artist-filters").getByRole("searchbox").fill("zzzzzz");
	await expect(artistSkeleton).toBeVisible();
	await expect(artistSkeleton).toHaveAttribute("aria-hidden", "true");
	await expect(artistSkeleton.locator(":scope > div > div")).toHaveCount(9);
	await expect(page.locator("#artists .wga-search-current")).toBeHidden();
	await expect(artistSkeleton).toBeHidden({ timeout: 30000 });
	await expect
		.poll(() => artistCount.textContent())
		.not.toBe(initialArtistCount);
});

test("artform search", async ({ page }) => {
	await page.goto("/artworks");
	await chooseFilter(page, "art_form", "architecture");
});

test("art type search", async ({ page }) => {
	await page.goto("/artworks");
	await chooseFilter(page, "art_type", "synthetic-test-type");
});

test("art school search", async ({ page }) => {
	await page.goto("/artworks");
	await chooseFilter(page, "art_school", "bohemian");
});

test("art type and school combined search", async ({ page }) => {
	await page.goto("/artworks");
	await chooseFilter(page, "art_type", "synthetic-test-type");
	await chooseFilter(page, "art_school", "bohemian");
});

test("artist name search", async ({ page }) => {
	await page.goto("/artworks");
	await artworkSearchForm(page)
		.getByRole("searchbox")
		.fill("Synthetic Artist 01");
	await expectArtworkResults(page);
});

test("artwork date range search", async ({ page }) => {
	await page.goto("/artworks");
	const yearFrom = page.locator("[name='year_from']");
	await yearFrom.evaluate((input: HTMLInputElement) => {
		input.value = "1800";
		input.dispatchEvent(new Event("input", { bubbles: true }));
	});
	await expect(page.locator("output[for='year_from year_to']")).toHaveText(
		"1800–1900",
	);
});

test("reset clears the artwork search form", async ({ page }) => {
	await page.goto("/artworks");
	const form = artworkSearchForm(page);
	await form.getByRole("searchbox").fill("Synthetic Artwork 01-01");
	await expectArtworkResults(page);
	await page.getByRole("link", { name: "RESET" }).click();

	await expect(page).toHaveURL(/\/artworks$/);
	await expect(form.getByRole("searchbox")).toHaveValue("");
	await expect(page.locator("input[name='art_school']:checked")).toHaveCount(0);
	await expect(page.locator("[name='year_from']")).toHaveValue("200");
	await expect(page.locator("#search-result-container")).toContainText(
		/works match/i,
	);
});

test("artwork search form works without JavaScript", async ({ browser }) => {
	const context = await browser.newContext({ javaScriptEnabled: false });
	const page = await context.newPage();

	await page.goto("/artworks");
	const form = artworkSearchForm(page);
	await form.getByRole("searchbox").fill("a");
	await form.getByRole("button", { name: "APPLY FILTERS" }).click();

	await expect(page).toHaveURL(/\/artworks\?.*q=a/);
	await expectArtworkResults(page);

	await page.goto("/artists");
	const artistForm = page.locator("#artist-filters");
	await artistForm.getByRole("searchbox").fill("a");
	await artistForm.getByRole("button", { name: "APPLY FILTERS" }).click();
	await expect(page).toHaveURL(/\/artists\?.*q=a/);
	await expect(page.locator("#artists .wga-search-current")).toContainText(
		"ARTISTS",
	);
	await context.close();
});
