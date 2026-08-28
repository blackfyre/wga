import { type Page, expect, test } from "@playwright/test";

const producerRuntime = Boolean(process.env.WGA_REAL_PRODUCER);
const catalogueLimit = 24;

type DiscoveredRecords = {
	holding: string;
	wide: string;
	narrow: string;
};

test.describe("tasks 6.6/7.7 real producer evidence", () => {
	test.skip(
		!producerRuntime,
		"Set WGA_REAL_PRODUCER=1 when the approved producer database is running",
	);
	test.setTimeout(120000);
	let discovered: DiscoveredRecords;

	async function catalogueRecords(page: Page) {
		await page.goto("/artworks");
		const links = await page
			.locator("#artwork-search-results [data-kbd-href]")
			.evaluateAll((elements) =>
				elements
					.map((link) => link.getAttribute("href"))
					.filter(
						(href): href is string =>
							href?.startsWith("/artists/") && href.split("/").length >= 4,
					),
			);
		return [...new Set(links)].slice(0, catalogueLimit);
	}

	async function findRecord(
		page: Page,
		predicate: (display: string, zoom: string) => boolean,
	) {
		for (const path of await catalogueRecords(page)) {
			const response = await page.goto(path);
			if (response?.status() !== 200) continue;
			const image = page.locator("figure img").first();
			if ((await image.count()) !== 1) continue;
			const display = (await image.getAttribute("src")) ?? "";
			const zoom = (await image.getAttribute("data-zoom-url")) ?? "";
			if (predicate(display, zoom)) return path;
		}
		throw new Error(
			"No eligible image record found in the bounded catalogue page",
		);
	}

	async function recordFromExactTitle(page: Page, title: string, id: string) {
		await page.goto(`/artworks?q=${encodeURIComponent(title)}`);
		const matches = page
			.locator("#artwork-search-results [data-kbd-href]")
			.filter({ hasText: title });
		await expect(matches).toHaveCount(1);
		const href = await matches.getAttribute("href");
		expect(href).toBeTruthy();
		expect(href).toContain(id);
		return href as string;
	}

	async function discoverRecords(page: Page): Promise<DiscoveredRecords> {
		const wide = await recordFromExactTitle(
			page,
			"Kolowrat Wedding",
			"rea135c19d9c553",
		);
		const narrow = await recordFromExactTitle(
			page,
			"Ludovico Maria Sforza",
			"r80a20580970725",
		);
		const candidates = await catalogueRecords(page);
		let holding = "";
		for (const path of candidates) {
			const response = await page.goto(path);
			if (response?.status() !== 200) continue;
			if (
				!holding &&
				(await page
					.getByRole("link", { name: /FIND MORE \d+ IN THE ARTWORK SEARCH/ })
					.count()) === 1
			) {
				holding = path;
			}
			if (holding) return { holding, wide, narrow };
		}
		throw new Error(
			"Bounded producer discovery did not find holding, wide, and original-only records",
		);
	}

	test.beforeAll(async ({ browser }) => {
		const page = await browser.newPage();
		discovered = await discoverRecords(page);
		await page.close();
	});

	test("search sort and counted relationship holding use producer records", async ({
		page,
	}) => {
		await page.goto("/artworks");
		const sort = page.getByRole("link", { name: "TITLE" });
		await sort.click();
		await expect(page).toHaveURL(/sort=title/);
		await expect(
			page.getByRole("link", { name: "Reverse sort direction" }),
		).toContainText("A–Z");

		const record = discovered.holding;
		await page.goto(record);
		const holding = page.getByRole("link", {
			name: /FIND MORE \d+ IN THE ARTWORK SEARCH/,
		});
		await expect(holding).toHaveCount(1);
		await expect(holding).toHaveAttribute(
			"href",
			/\/artworks\?(artist|venue|period)=/,
		);
	});

	test("wide producer records use exact display and zoom rendition profiles", async ({
		page,
	}) => {
		const record = discovered.wide;
		await page.goto(record);
		const image = page.locator("figure img").first();
		await expect(image).toHaveAttribute("src", /thumb=1400x0/);
		await expect(image).toHaveAttribute("data-zoom-url", /thumb=2000x0/);
	});

	test("Dual preserves independent panes and image-size hand-offs", async ({
		page,
	}) => {
		const record = discovered.wide;
		await page.goto(
			`/dual-mode?wide=1&left=${encodeURIComponent(record)}&right=${encodeURIComponent(record)}`,
		);
		await expect(page.locator("#dual-left figure img")).toHaveAttribute(
			"src",
			/thumb=1100x0/,
		);
		await expect(page.locator("#dual-right figure img")).toHaveAttribute(
			"src",
			/thumb=1100x0/,
		);
		for (const [label, profile] of [
			["700", "700x0"],
			["1100", "1100x0"],
			["1600", "1600x0"],
		] as const) {
			const size = page.locator("#dual-right a").filter({
				hasText: new RegExp(label),
			});
			await expect(size).toHaveCount(1);
			await size.click();
			await expect(page.locator("#dual-right figure img")).toHaveAttribute(
				"src",
				new RegExp(`thumb=${profile}`),
			);
		}
		await page.reload();
		await expect(page.locator("#dual-left")).toBeVisible();
		await expect(page.locator("#dual-right")).toBeVisible();
	});

	test("original-only producer records keep original image URLs", async ({
		page,
	}) => {
		await page.goto(discovered.narrow);
		const image = page.locator("figure img").first();
		await expect(image).not.toHaveAttribute("src", /thumb=/);
		await expect(image).not.toHaveAttribute("data-zoom-url", /thumb=/);
	});

	test("ordinary zoom and holding links navigate without JavaScript", async ({
		browser,
	}) => {
		const context = await browser.newContext({ javaScriptEnabled: false });
		const page = await context.newPage();
		await page.goto(discovered.wide);
		const image = page.locator("[data-viewer] img");
		const zoomURL = await image.getAttribute("data-zoom-url");
		if (!zoomURL) throw new Error("Discovered wide record has no zoom URL");
		await expect(page.locator("[data-viewer]")).toHaveAttribute(
			"href",
			zoomURL,
		);
		await page.goto(discovered.holding);
		const holding = page.getByRole("link", {
			name: /FIND MORE \d+ IN THE ARTWORK SEARCH/,
		});
		await holding.click();
		await expect(page).toHaveURL(/\/artworks\?(artist|venue|period)=/);
		await context.close();
	});
});
