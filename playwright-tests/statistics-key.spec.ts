import { expect, type Page, test } from "@playwright/test";

// Each Statistics key row must draw exactly one swatch, and that swatch must
// show the colour the chart draws the same series with (design-audit ST-1).
// The canvases record the colours they were drawn with in
// data-series-colours; a hatch is recorded as "hatch:<colour>".

type KeyReport = {
	failures: string[];
	rows: number;
};

async function waitForCharts(page: Page): Promise<void> {
	for (const id of [
		"art-form-chart",
		"artworks-by-period-chart",
		"artists-by-period-chart",
	]) {
		await expect(page.locator(`#${id}`)).toHaveAttribute(
			"data-series-colours",
			/.+/,
		);
	}
}

async function compareKeysWithCharts(page: Page): Promise<KeyReport> {
	return page.evaluate(() => {
		const failures: string[] = [];
		let rows = 0;

		// Normalise any CSS colour through the canvas colour parser, which
		// serialises hex and rgb() inputs to the same form.
		const parser = document.createElement("canvas").getContext("2d");
		const toRgb = (colour: string): string => {
			if (!parser) return colour;
			parser.fillStyle = "#000000";
			parser.fillStyle = colour;
			return String(parser.fillStyle);
		};

		const chartColours = (id: string): string[] =>
			JSON.parse(
				document.getElementById(id)?.getAttribute("data-series-colours") ??
					"[]",
			);

		const swatchColour = (swatch: Element): string => {
			const style = getComputedStyle(swatch);
			if (swatch.getAttribute("data-series-fill") === "hatch") {
				const match = style.backgroundImage.match(/rgba?\([^)]*\)/);
				return `hatch:${toRgb(match ? match[0] : style.backgroundImage)}`;
			}
			return toRgb(style.backgroundColor);
		};

		const chartColour = (value: string): string =>
			value.startsWith("hatch:")
				? `hatch:${toRgb(value.slice("hatch:".length))}`
				: toRgb(value);

		const countSwatches = (row: Element, label: string): Element | null => {
			rows++;
			const swatches = row.querySelectorAll("[data-series-token]");
			const header = row.querySelector("th") ?? row;
			const before = getComputedStyle(header, "::before").content;
			if (swatches.length !== 1) {
				failures.push(`${label}: ${swatches.length} swatches`);
			}
			if (before !== "none" && before !== "normal") {
				failures.push(`${label}: extra ::before swatch (${before})`);
			}
			return swatches[0] ?? null;
		};

		// Art-form key rows against the donut segments, in row order.
		const donut = chartColours("art-form-chart");
		const formRows = Array.from(
			document.querySelectorAll("#art-form-summary tbody tr"),
		);
		if (formRows.length === 0) failures.push("art-form key is empty");
		if (donut.length !== formRows.length) {
			failures.push(
				`donut has ${donut.length} colours for ${formRows.length} key rows`,
			);
		}
		formRows.forEach((row, index) => {
			const swatch = countSwatches(row, `art form row ${index}`);
			if (!swatch || donut[index] === undefined) return;
			const key = swatchColour(swatch);
			const chart = chartColour(donut[index]);
			if (key !== chart) {
				failures.push(
					`art form row ${index}: key ${key} != chart ${chart} (raw ${donut[index]})`,
				);
			}
		});

		// School keys against the stacked datasets, by school name.
		for (const id of ["artworks-by-period-chart", "artists-by-period-chart"]) {
			const canvas = document.getElementById(id);
			const section = canvas?.closest("section");
			const key = section?.querySelector("[data-school-key]");
			const dataId =
				id === "artworks-by-period-chart"
					? "artworks-period-data"
					: "artists-period-data";
			const data = JSON.parse(
				document.getElementById(dataId)?.getAttribute("data-json") ?? "[]",
			) as { school: string }[];
			const present = new Set(data.map((row) => row.school));
			const items = Array.from(key?.querySelectorAll("[data-school]") ?? []);
			if (items.length === 0) failures.push(`${id}: school key is empty`);
			const drawn = items
				.map((item) => item.getAttribute("data-school") ?? "")
				.filter((school) => present.has(school));
			const colours = chartColours(id);
			if (colours.length !== drawn.length) {
				failures.push(
					`${id}: ${colours.length} datasets for ${drawn.length} schools`,
				);
			}
			for (const item of items) {
				const school = item.getAttribute("data-school") ?? "";
				const swatch = countSwatches(item, `${id} ${school}`);
				const index = drawn.indexOf(school);
				if (!swatch || index === -1 || colours[index] === undefined) continue;
				const keyColour = swatchColour(swatch);
				const chart = chartColour(colours[index]);
				if (keyColour !== chart) {
					failures.push(`${id} ${school}: key ${keyColour} != chart ${chart}`);
				}
			}
		}

		return { failures, rows };
	});
}

for (const scheme of ["light", "dark"] as const) {
	test(`statistics keys draw one swatch matching the chart in ${scheme}`, async ({
		page,
	}) => {
		await page.emulateMedia({ colorScheme: scheme, reducedMotion: "reduce" });
		await page.goto("/statistics");
		await expect(page.locator("html")).toHaveAttribute("data-theme", scheme);
		await waitForCharts(page);

		const report = await compareKeysWithCharts(page);
		expect(report.rows).toBeGreaterThan(0);
		expect(report.failures).toEqual([]);
	});
}

test("statistics charts follow the key after a palette change", async ({
	page,
}) => {
	await page.emulateMedia({ colorScheme: "light", reducedMotion: "reduce" });
	await page.goto("/statistics");
	await waitForCharts(page);
	const before = await page
		.locator("#art-form-chart")
		.getAttribute("data-series-colours");

	await page.locator("[data-wga-preferences-open]").click();
	await page.locator('[data-wga-palette="verdigris"]').click();
	await expect(page.locator("html")).toHaveAttribute(
		"data-palette",
		"verdigris",
	);
	await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
	await expect(page.locator("#art-form-chart")).not.toHaveAttribute(
		"data-series-colours",
		before ?? "",
	);

	const report = await compareKeysWithCharts(page);
	expect(report.failures).toEqual([]);
});

test("the school charts draw Other as a hatch, as the key does", async ({
	page,
}) => {
	await page.goto("/statistics");
	await waitForCharts(page);

	await expect(page.locator("[data-school-key]")).toHaveCount(2);
	await expect(
		page.locator('[data-school="Other"] [data-series-fill="hatch"]'),
	).toHaveCount(2);
	for (const id of ["artworks-by-period-chart", "artists-by-period-chart"]) {
		await expect(page.locator(`#${id}`)).toHaveAttribute(
			"data-series-colours",
			/"hatch:/,
		);
	}
});
