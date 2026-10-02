import { expect, type Page, test } from "@playwright/test";

// Desktop header layout contract (openspec change fix-header-search-row):
// search and the Go to keyboard cue sit on the logo row at every desktop
// width, and the navigation is a second row.
const widths = [720, 834, 1079, 1080, 1440];
// At 720px the seven destinations and MORE cannot fit on one line; the row
// wraps without hiding destinations (see the change's design.md).
const singleLineNavFrom = 834;

async function headerGeometry(page: Page) {
	return page.evaluate(() => {
		const row = document.querySelector("header > div > div.hidden");
		if (!row) {
			throw new Error("Expected the desktop logo row");
		}
		const box = (selector: string) => {
			const element = row.querySelector(selector);
			if (!element) {
				throw new Error(`Expected ${selector} in the logo row`);
			}
			return element.getBoundingClientRect();
		};
		const cue = row.querySelector("[data-keyboard-open]") as HTMLElement;
		const cueLabel = cue.querySelector("[data-kbd-modifier]") as HTMLElement;
		const nav = document.querySelector(
			"header > nav[aria-label='Primary navigation']",
		) as HTMLElement;
		const items = [
			...nav.querySelectorAll(
				":scope > ul > li > a, :scope > details > summary",
			),
		];
		const navBottoms = new Set(
			items.map((item) => Math.round(item.getBoundingClientRect().bottom)),
		);
		const logo = box("a[href='/']");
		return {
			cue: cue.getBoundingClientRect(),
			cueLabelHeight: cueLabel.getBoundingClientRect().height,
			cueLineHeight: Number.parseFloat(getComputedStyle(cue).lineHeight),
			submit: box("button[type='submit']"),
			input: box("input[type='search']"),
			logo,
			navItems: items.length,
			navLines: navBottoms.size,
			overflow:
				document.documentElement.scrollWidth -
				document.documentElement.clientWidth,
		};
	});
}

for (const width of widths) {
	test(`desktop header keeps search on the logo row at ${width}px`, async ({
		page,
	}) => {
		await page.setViewportSize({ width, height: 900 });
		await page.goto("/");

		const geometry = await headerGeometry(page);

		// The keyboard cue is one line and bottom-aligned with SEARCH.
		expect(geometry.cue.height).toBeLessThanOrEqual(32);
		expect(geometry.cueLabelHeight).toBeLessThanOrEqual(
			geometry.cueLineHeight + 1,
		);
		expect(
			Math.abs(geometry.cue.bottom - geometry.submit.bottom),
		).toBeLessThanOrEqual(1);
		expect(geometry.submit.height).toBeLessThanOrEqual(34);

		// It shares the logo's row: their vertical extents overlap.
		expect(geometry.cue.top).toBeLessThan(geometry.logo.bottom);
		expect(geometry.cue.bottom).toBeGreaterThan(geometry.logo.top);
		expect(geometry.cue.left).toBeGreaterThan(geometry.logo.right);

		expect(geometry.input.width).toBeGreaterThanOrEqual(140);
		expect(geometry.navItems).toBe(8);
		if (width >= singleLineNavFrom) {
			expect(geometry.navLines).toBe(1);
		}
		expect(geometry.overflow).toBe(0);
	});
}

test("desktop brand and MORE follow the design", async ({ page }) => {
	await page.setViewportSize({ width: 1440, height: 900 });
	await page.goto("/");

	const brand = page.locator("header > div > div.hidden > a[href='/'] > span");
	const wordmark = brand.locator(":scope > span").nth(0);
	const strapline = brand.locator(":scope > span").nth(1);

	await expect(wordmark).toHaveText("WEB GALLERY OF ART");
	await expect(wordmark).toHaveCSS("font-size", "15.5px");
	await expect(wordmark).toHaveCSS("font-weight", "600");
	await expect(wordmark).toHaveCSS("letter-spacing", "3px");
	await expect(wordmark).toHaveCSS("line-height", "17.825px");

	await expect(strapline).toHaveText("EUROPEAN ART, 3rd CENTURY – EARLY 20th");
	await expect(strapline).toHaveCSS("font-size", "12px");
	await expect(strapline).toHaveCSS("letter-spacing", "1px");
	await expect(strapline).toHaveCSS("margin-top", "4px");

	const [wordmarkFont, straplineFont, faint, straplineColor] =
		await strapline.evaluate((element) => {
			const probe = document.createElement("span");
			probe.style.color = "var(--wga-faint)";
			document.body.append(probe);
			const faintColor = getComputedStyle(probe).color;
			probe.remove();
			return [
				getComputedStyle(element.previousElementSibling as Element).fontFamily,
				getComputedStyle(element).fontFamily,
				faintColor,
				getComputedStyle(element).color,
			];
		});
	expect(wordmarkFont).toContain("monospace");
	expect(straplineFont).toContain("monospace");
	expect(straplineColor).toBe(faint);

	const summary = page.locator(
		"header > nav details[aria-label='More destinations'] > summary",
	);
	await expect(summary).toHaveText("MORE ▾");
	await expect(summary).toHaveAccessibleName("MORE");
	// No native disclosure triangle: the summary is not a list item with a
	// marker, so only the design's ▾ is drawn.
	await expect(summary).toHaveCSS("list-style-type", "none");
});
