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
	await form.getByRole("searchbox").fill("Synthetic Artwork 01-01");
	await form.getByRole("button", { name: "APPLY FILTERS" }).click();

	await expect(page).toHaveURL(/\/artworks\?.*q=Synthetic\+Artwork\+01-01/);
	await expectArtworkResults(page);
	await context.close();
});
