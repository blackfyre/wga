import { expect, test } from "@playwright/test";

const chartSummaryPairs = [
	{ chart: "art-form-chart", summary: "art-form-summary" },
	{ chart: "artworks-by-period-chart", summary: "artworks-period-summary" },
	{ chart: "artists-by-period-chart", summary: "artists-period-summary" },
] as const;

const periodTablePairs = [
	{ data: "artworks-period-data", summary: "artworks-period-summary" },
	{ data: "artists-period-data", summary: "artists-period-summary" },
] as const;

const schoolColumns = [
	"Italian",
	"French",
	"Dutch",
	"Flemish",
	"German",
	"English",
	"Spanish",
	"Other",
];

test("statistics charts provide text summaries", async ({ page }) => {
	await page.goto("/statistics");
	await expect(page.getByRole("heading", { name: "Statistics" })).toBeVisible();

	for (const { chart, summary } of chartSummaryPairs) {
		await expect(page.locator(`#${chart}`)).toHaveAttribute(
			"aria-describedby",
			summary,
		);
		await expect(page.locator(`#${summary}`)).toBeVisible();
	}
});

test("every chart has a visible table caption", async ({ page }) => {
	await page.goto("/statistics");

	const captions = {
		"#art-form-summary": "Art form distribution data",
		"#artworks-period-summary": "Artworks by school and birth period data",
		"#artists-period-summary": "Artists by school and birth period data",
	};

	for (const [summaryId, captionText] of Object.entries(captions)) {
		const caption = page.locator(`${summaryId} caption`);
		await expect(caption).toBeVisible();
		await expect(caption).toContainText(captionText);
	}
});

test("school abbreviations expose their full names", async ({ page }) => {
	await page.goto("/statistics");

	const table = page.locator("#artworks-period-summary table thead");
	await expect(table.locator('th[title="Italian"]')).toHaveText("IT");
	await expect(table.locator('th[title="Flemish"]')).toHaveText("FL");
	await expect(table.locator('th[title="Other"]')).toHaveText("OTH");
});

test("art form swatches use the Rams series tokens", async ({ page }) => {
	await page.goto("/statistics");

	const swatch = page
		.locator("#art-form-summary tbody tr")
		.first()
		.locator("th span span")
		.first();
	await expect(swatch).toBeVisible();
	await expect(swatch).toHaveAttribute(
		"style",
		/background:var\(--wga-series-\d\)/,
	);
});

test("chart data and accessible tables present equivalent figures", async ({
	page,
}) => {
	await page.goto("/statistics");

	const failures = await page.evaluate(() => {
		const readJson = (id: string): unknown[] => {
			const raw =
				document.getElementById(id)?.getAttribute("data-json") ?? "[]";
			return JSON.parse(raw);
		};

		const schoolColumns = [
			"Italian",
			"French",
			"Dutch",
			"Flemish",
			"German",
			"English",
			"Spanish",
			"Other",
		];

		const failures: string[] = [];

		const artForms = readJson("art-form-data") as {
			name: string;
			count: number;
		}[];
		const artFormLabels = Array.from(
			document.querySelectorAll("#art-form-summary tbody th.art-form-label"),
		).map((element) => element.textContent?.trim() ?? "");
		const artFormCounts = Array.from(
			document.querySelectorAll("#art-form-summary tbody tr td:nth-child(2)"),
		).map((element) => Number(element.textContent));
		const artFormShares = Array.from(
			document.querySelectorAll("#art-form-summary tbody tr td:nth-child(3)"),
		).map((element) => element.textContent?.trim() ?? "");
		if (artForms.length === 0) {
			failures.push("art-form-data is empty");
		}
		if (artFormLabels.length !== artForms.length) {
			failures.push(
				`art form label count ${artFormLabels.length} != ${artForms.length}`,
			);
		}
		const artFormTotal = artForms.reduce((sum, form) => sum + form.count, 0);
		for (let index = 0; index < artForms.length; index++) {
			if (artFormLabels[index] !== artForms[index].name) {
				failures.push(
					`art form row ${index}: label ${artFormLabels[index]} != ${artForms[index].name}`,
				);
			}
			if (artFormCounts[index] !== artForms[index].count) {
				failures.push(
					`art form row ${index}: count ${artFormCounts[index]} != ${artForms[index].count}`,
				);
			}
			const expectedShare = `${(
				(artForms[index].count / artFormTotal) * 100
			).toFixed(1)}%`;
			if (artFormShares[index] !== expectedShare) {
				failures.push(
					`art form row ${index}: share ${artFormShares[index]} != ${expectedShare}`,
				);
			}
		}

		const comparePeriodTable = (dataId: string, tableId: string): void => {
			const rows = readJson(dataId) as {
				period_start: number;
				school: string;
				count: number;
			}[];
			if (rows.length === 0) {
				failures.push(`${dataId} is empty`);
				return;
			}

			const periods: number[] = [];
			const expected = new Map<number, Map<string, number>>();
			for (const row of rows) {
				if (!expected.has(row.period_start)) {
					expected.set(row.period_start, new Map());
					periods.push(row.period_start);
				}
				expected.get(row.period_start)?.set(row.school, row.count);
			}

			const bodyRows = Array.from(
				document.querySelectorAll(`#${tableId} tbody tr`),
			);
			if (bodyRows.length !== periods.length) {
				failures.push(
					`${tableId} row count ${bodyRows.length} != ${periods.length}`,
				);
				return;
			}

			for (let rowIndex = 0; rowIndex < bodyRows.length; rowIndex++) {
				const cells = Array.from(
					bodyRows[rowIndex].querySelectorAll("th, td"),
				).map((element) => element.textContent?.trim() ?? "");
				const period = periods[rowIndex];
				const expectedLabel = `${period}–${period + 49}`;
				if (cells[0] !== expectedLabel) {
					failures.push(
						`${tableId} row ${rowIndex}: period ${cells[0]} != ${expectedLabel}`,
					);
				}

				const schoolCounts = expected.get(period);
				for (let column = 0; column < schoolColumns.length; column++) {
					const expectedCount = schoolCounts?.get(schoolColumns[column]) ?? 0;
					const actualCount = Number(cells[1 + column]);
					if (actualCount !== expectedCount) {
						failures.push(
							`${tableId} row ${rowIndex} ${schoolColumns[column]}: ${actualCount} != ${expectedCount}`,
						);
					}
				}

				const expectedTotal = Array.from(schoolCounts?.values() ?? []).reduce(
					(sum, count) => sum + count,
					0,
				);
				const actualTotal = Number(cells[1 + schoolColumns.length]);
				if (actualTotal !== expectedTotal) {
					failures.push(
						`${tableId} row ${rowIndex} total: ${actualTotal} != ${expectedTotal}`,
					);
				}
			}
		};

		comparePeriodTable("artworks-period-data", "artworks-period-summary");
		comparePeriodTable("artists-period-data", "artists-period-summary");

		return failures;
	});

	expect(failures).toEqual([]);
});

test("a live theme change redraws chart output with dark tokens", async ({
	page,
}) => {
	await page.emulateMedia({ reducedMotion: "reduce" });
	await page.addInitScript(() => {
		(window as unknown as Record<string, unknown>).__wgaCanvasHasPixels = (
			canvas: HTMLCanvasElement,
		): boolean => {
			const context = canvas.getContext("2d");
			if (!context) return false;
			const pixels = context.getImageData(
				0,
				0,
				canvas.width,
				canvas.height,
			).data;
			for (let index = 3; index < pixels.length; index += 4) {
				if (pixels[index] !== 0) return true;
			}
			return false;
		};
	});
	await page.goto("/statistics");

	const canvas = page.locator("#artworks-by-period-chart");
	await canvas.waitFor({ state: "visible" });
	await page.waitForFunction(() => {
		const element = document.getElementById("artworks-by-period-chart");
		if (!(element instanceof HTMLCanvasElement)) return false;
		if (element.width === 0 || element.height === 0) return false;
		const hasPixels = (
			window as unknown as {
				__wgaCanvasHasPixels: (canvas: HTMLCanvasElement) => boolean;
			}
		).__wgaCanvasHasPixels;
		return hasPixels(element);
	});

	const before = await canvas.evaluate((element) =>
		(element as HTMLCanvasElement).toDataURL(),
	);

	await page.locator("[data-wga-preferences-open]").click();
	await page.getByRole("button", { name: "DARK" }).first().click();
	await expect(page.locator("html")).toHaveAttribute(
		"data-theme",
		"wga-rams-dark",
	);

	await page.waitForFunction(
		(previous) => {
			const element = document.getElementById("artworks-by-period-chart");
			if (!(element instanceof HTMLCanvasElement)) return false;
			if (element.width === 0 || element.height === 0) return false;
			const hasPixels = (
				window as unknown as {
					__wgaCanvasHasPixels: (canvas: HTMLCanvasElement) => boolean;
				}
			).__wgaCanvasHasPixels;
			return hasPixels(element) && element.toDataURL() !== previous;
		},
		before,
		{ timeout: 15000 },
	);

	const after = await canvas.evaluate((element) =>
		(element as HTMLCanvasElement).toDataURL(),
	);
	expect(after).not.toBe(before);

	const series0 = await page.evaluate(() =>
		getComputedStyle(document.documentElement)
			.getPropertyValue("--wga-series-0")
			.trim(),
	);
	expect(series0).toBe("#E4EDF5");
});

test("charts use the non-animated path under reduced motion", async ({
	page,
}) => {
	await page.emulateMedia({ reducedMotion: "reduce" });
	await page.goto("/statistics");

	for (const { chart } of chartSummaryPairs) {
		await expect(page.locator(`#${chart}`)).toHaveAttribute(
			"data-chart-animation",
			"none",
		);
	}
});

test("charts use the default animated path without reduced motion", async ({
	page,
}) => {
	await page.emulateMedia({ reducedMotion: "no-preference" });
	await page.goto("/statistics");

	for (const { chart } of chartSummaryPairs) {
		await expect(page.locator(`#${chart}`)).toHaveAttribute(
			"data-chart-animation",
			"animated",
		);
	}
});

test.describe("without JavaScript", () => {
	test.use({ javaScriptEnabled: false });

	test("renders non-empty equivalent figures without JavaScript", async ({
		page,
	}) => {
		await page.goto("/statistics");

		const artFormRows = page.locator("#art-form-summary tbody tr");
		expect(await artFormRows.count()).toBeGreaterThan(0);
		const artFormCounts = await artFormRows.locator("td").allTextContents();
		expect(artFormCounts.some((text) => Number(text) > 0)).toBe(true);

		for (const { summary } of periodTablePairs) {
			const rows = page.locator(`#${summary} tbody tr`);
			expect(await rows.count()).toBeGreaterThan(0);
			const totals = await rows.locator("td:last-child").allTextContents();
			expect(totals.some((text) => Number(text) > 0)).toBe(true);
		}
	});

	test("renders server-produced chart summaries without JavaScript", async ({
		page,
	}) => {
		await page.goto("/statistics");

		// The art-form donut renders a server-produced horizontal bar summary.
		const artFormBars = page.locator('ul[aria-hidden="true"]');
		expect(await artFormBars.count()).toBe(1);
		await expect(artFormBars).toBeVisible();
		expect(await artFormBars.locator("li").count()).toBeGreaterThan(0);
		expect(await artFormBars.locator("li div span").count()).toBeGreaterThan(0);

		// Each stacked-bar chart renders a server-produced CSS column summary on
		// a single, horizontally scrollable chronological axis.
		const periodBars = page.locator('div.overflow-x-auto[aria-hidden="true"]');
		expect(await periodBars.count()).toBe(2);
		for (let index = 0; index < 2; index++) {
			const bars = periodBars.nth(index);
			await expect(bars).toBeVisible();
			expect(await bars.locator("> div > div").count()).toBeGreaterThan(0);
			expect(await bars.locator("span").count()).toBeGreaterThan(0);
		}

		// Each chart carries a shared legend naming all eight schools.
		const legends = page.locator("#statistics section > ul");
		expect(await legends.count()).toBe(2);
		for (const school of schoolColumns) {
			await expect(
				legends.first().getByText(school, { exact: true }),
			).toBeVisible();
		}

		// The art-form table exposes each form's relative share.
		const shareCells = page.locator(
			"#art-form-summary tbody tr td:nth-child(3)",
		);
		expect(await shareCells.count()).toBeGreaterThan(0);
		await expect(shareCells.first()).toHaveText(/^\d+(\.\d+)?%$/);
	});

	for (const width of [390, 834, 1440]) {
		test(`statistics renders without horizontal overflow at ${width}px`, async ({
			page,
		}) => {
			await page.setViewportSize({ width, height: 900 });
			await page.goto("/statistics");

			const dimensions = await page.evaluate(() => ({
				clientWidth: document.documentElement.clientWidth,
				scrollWidth: document.documentElement.scrollWidth,
			}));
			expect(dimensions.scrollWidth).toBeLessThanOrEqual(
				dimensions.clientWidth,
			);
		});
	}
});

for (const width of [390, 834, 1440]) {
	test(`preserves chart-before-table responsive order at ${width}px`, async ({
		page,
	}) => {
		await page.setViewportSize({ width, height: 900 });
		await page.goto("/statistics");

		for (const { chart, summary } of chartSummaryPairs) {
			const domBefore = await page.evaluate(
				([chartId, summaryId]) => {
					const chartElement = document.getElementById(chartId);
					const summaryElement = document.getElementById(summaryId);
					if (!chartElement || !summaryElement) return false;
					return (
						(chartElement.compareDocumentPosition(summaryElement) &
							Node.DOCUMENT_POSITION_FOLLOWING) !==
						0
					);
				},
				[chart, summary],
			);
			expect(domBefore).toBe(true);

			const chartBox = await page.locator(`#${chart}`).boundingBox();
			const summaryBox = await page.locator(`#${summary}`).boundingBox();
			expect(chartBox).not.toBeNull();
			expect(summaryBox).not.toBeNull();
			if (chartBox && summaryBox) {
				// The chart must not sit below its table; a 1px tolerance absorbs
				// sub-pixel flex/canvas rounding on side-by-side desktop rows.
				expect(chartBox.y).toBeLessThanOrEqual(summaryBox.y + 1);
			}
		}

		const dimensions = await page.evaluate(() => ({
			clientWidth: document.documentElement.clientWidth,
			scrollWidth: document.documentElement.scrollWidth,
		}));
		expect(dimensions.scrollWidth).toBeLessThanOrEqual(dimensions.clientWidth);
	});
}

test("art form chart sits beside its summary on desktop", async ({ page }) => {
	await page.setViewportSize({ width: 1440, height: 900 });
	await page.goto("/statistics");
	await page.waitForFunction(
		() =>
			document.getElementById("art-form-chart")?.getBoundingClientRect()
				.width === 240,
	);

	const chartBox = await page.locator("#art-form-chart").boundingBox();
	const summaryBox = await page.locator("#art-form-summary").boundingBox();
	expect(chartBox).not.toBeNull();
	expect(summaryBox).not.toBeNull();
	if (chartBox && summaryBox) {
		expect(chartBox.x).toBeLessThan(summaryBox.x);
		expect(Math.abs(chartBox.y - summaryBox.y)).toBeLessThanOrEqual(1);
	}
	await expect(page.locator(".art-form-label").first()).toBeVisible();

	const periodChart = page.locator("#artworks-by-period-chart");
	await page.waitForFunction(() => {
		const box = document
			.getElementById("artworks-by-period-chart")
			?.getBoundingClientRect();
		return box !== undefined && box.width > 0 && box.width / box.height === 2;
	});
	const periodBox = await periodChart.boundingBox();
	expect(periodBox?.width || 0).toBeCloseTo((periodBox?.height || 0) * 2, 0);
});

test("school period columns remain readable on mobile", async ({ page }) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await page.goto("/statistics");

	const table = page.locator("#artworks-period-summary table");
	await expect(table).toBeVisible();
	expect(
		await table.evaluate((element) => element.scrollWidth),
	).toBeGreaterThanOrEqual(560);
});

test("art form legend fills the mobile content width", async ({ page }) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await page.goto("/statistics");

	const summary = await page.locator("#art-form-summary").boundingBox();
	const section = await page
		.locator("#statistics > section")
		.first()
		.boundingBox();
	expect(summary?.width).toBeCloseTo(section?.width || 0, 0);
});
