import { test, expect } from "@playwright/test";

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
	const input = page.locator(`[name='${name}'][value='${value}']`);
	const facet = input.locator("xpath=ancestor::details[1]");
	if (!(await facet.evaluate((details: HTMLDetailsElement) => details.open))) {
		await facet.locator("summary").click();
	}

	const response = page.waitForResponse(
		(response) =>
			new URL(response.url()).pathname === "/artworks" &&
			new URL(response.url()).searchParams.get(name) === value,
	);
	await input.locator("..").click();
	await response;
	await expect(page).toHaveURL((url) => url.searchParams.get(name) === value);
	await expectArtworkResults(page);
}

test("active filter chip follows the selected radio", async ({ page }) => {
	await page.goto("/artworks");
	const all = page.locator("[name='art_school'][value='']");
	const school = page.locator("[name='art_school'][value='bohemian']");

	await expect(all).toBeChecked();
	await school.check({ force: true });

	await expect(school).toBeChecked();
	await expect(all).not.toBeChecked();
	await expect(school.locator("..")).toHaveCSS(
		"background-color",
		"rgb(0, 51, 102)",
	);
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
	await expect(page.locator("[name='art_school'][value='']")).toBeChecked();
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
