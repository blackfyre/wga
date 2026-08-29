import { expect, test } from "@playwright/test";

const artworkURL =
	"/artists/aachen-hans-von-r49032850f1b20c/boy-with-grapes-r4e965fe756506e";
const dualURL = `/dual-mode?wide=1&left=${encodeURIComponent(artworkURL)}&right=${encodeURIComponent(artworkURL)}`;

test.use({ viewport: { width: 390, height: 844 } });

test("discloses a contained artwork palette through hover, focus, and tap", async ({
	page,
}) => {
	await page.goto(artworkURL);
	const bar = page.locator("[data-wga-palette-bar]");
	const swatches = bar.locator("[data-wga-palette-swatch]");
	const first = swatches.first();
	const firstTooltip = first.locator("[data-wga-palette-tooltip]");

	await expect(swatches).toHaveCount(6);
	await expect(first).toHaveAttribute(
		"aria-label",
		/#0C0909, 63% of the surface/,
	);

	await first.hover();
	await expect(firstTooltip).toBeVisible();
	await first.focus();
	await expect(firstTooltip).toBeVisible();

	await first.click();
	await expect(first).toHaveAttribute("aria-expanded", "true");
	await expect(firstTooltip).toBeVisible();

	const last = swatches.last();
	const lastTooltip = last.locator("[data-wga-palette-tooltip]");
	await lastTooltip
		.locator("span")
		.first()
		.evaluate((label) => {
			label.textContent = "A deliberately long source-supplied colour name";
		});
	await last.click();
	await expect(first).toHaveAttribute("aria-expanded", "false");
	await expect(firstTooltip).not.toBeVisible();
	await expect(last).toHaveAttribute("aria-expanded", "true");
	await expect(lastTooltip).toBeVisible();
	const lastSwatchBounds = await last.boundingBox();
	const bounds = await lastTooltip.boundingBox();
	expect(bounds).not.toBeNull();
	if (bounds && lastSwatchBounds) {
		expect(bounds.x).toBeGreaterThanOrEqual(0);
		expect(bounds.x + bounds.width).toBeLessThanOrEqual(390);
		expect(bounds.width).toBe(220);
		expect(bounds.width).toBeGreaterThan(lastSwatchBounds.width);
	}

	await expect(bar.locator("xpath=following-sibling::p")).toHaveText(
		/HOVER OR TAP A SWATCH FOR NAME, SHARE AND HEX/,
	);
});

test("uses equal contained palette bands in each narrow Dual pane", async ({
	page,
}) => {
	await page.goto(dualURL);

	const bars = page.locator("[data-wga-palette-bar]");
	await expect(bars).toHaveCount(2);
	for (const bar of await bars.all()) {
		const swatches = bar.locator("[data-wga-palette-swatch]");
		await expect(swatches).toHaveCount(6);
		for (const swatch of await swatches.all()) {
			await expect(swatch).toHaveAttribute("style", /flex:1/);
			await expect(swatch).not.toHaveAttribute("style", /flex-grow/);
		}
	}

	const last = bars.first().locator("[data-wga-palette-swatch]").last();
	const tooltip = last.locator("[data-wga-palette-tooltip]");
	await last.focus();
	await expect(tooltip).toBeVisible();
	await last.click();
	await expect(last).toHaveAttribute("aria-expanded", "true");
	const bounds = await tooltip.boundingBox();
	expect(bounds).not.toBeNull();
	if (bounds) {
		expect(bounds.x).toBeGreaterThanOrEqual(0);
		expect(bounds.x + bounds.width).toBeLessThanOrEqual(390);
	}
});
