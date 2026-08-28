import { expect, test } from "@playwright/test";

const viewports = [390, 834, 1440];

for (const width of viewports) {
	test.describe(`artwork search task 7.1 at ${width}px`, () => {
		test.use({ viewport: { width, height: 900 } });
		test.setTimeout(60000);

		async function openSearch(page) {
			await page.goto("/artworks");
			await expect(page.locator("#artwork-search")).toBeVisible();
		}

		function collectionOptions(page) {
			return page.locator(
				"#artwork-filters input[name='venue']:not([value=''])",
			);
		}

		async function chooseCollection(page) {
			const details = page
				.locator("#artwork-filters > details")
				.filter({ has: page.locator("summary", { hasText: "COLLECTION" }) });
			if (!(await details.getAttribute("open"))) {
				await details.locator("summary").click();
			}
			await expect(details).toHaveAttribute("open", "");
			const option = collectionOptions(page).first();
			await expect(option).toHaveCount(1);
			const value = await option.getAttribute("value");
			expect(value).toBeTruthy();
			const label = option.locator("..");
			await expect(label).toBeVisible();
			const name = (await label.innerText()).replace(/\d+\s*$/, "").trim();
			return { label, name, option, value: value as string };
		}

		test("renders eight semantic disclosures with canonical initial state", async ({
			page,
		}) => {
			await openSearch(page);
			const disclosures = page.locator("#artwork-filters > details");
			await expect(disclosures).toHaveCount(8);
			await expect(disclosures.locator("summary").nth(0)).toContainText(
				"TITLE OR ARTIST",
			);
			await expect(disclosures.locator("summary").nth(1)).toContainText(
				"TECHNIQUE",
			);
			await expect(disclosures.locator("summary").nth(2)).toContainText(
				"SCHOOL",
			);
			await expect(disclosures.locator("summary").nth(3)).toContainText("FORM");
			await expect(disclosures.locator("summary").nth(4)).toContainText("TYPE");
			await expect(disclosures.locator("summary").nth(5)).toContainText(
				"PERIOD",
			);
			await expect(disclosures.locator("summary").nth(6)).toContainText(
				"COLLECTION",
			);
			await expect(disclosures.locator("summary").nth(7)).toContainText(
				"YEAR RANGE",
			);
			await expect(disclosures.nth(0)).toHaveAttribute("open", "");
			await expect(disclosures.nth(1)).not.toHaveAttribute("open", "");
			await expect(disclosures.nth(2)).toHaveAttribute("open", "");
			await expect(disclosures.nth(3)).not.toHaveAttribute("open", "");
			await expect(disclosures.nth(4)).not.toHaveAttribute("open", "");
			await expect(disclosures.nth(5)).not.toHaveAttribute("open", "");
			await expect(disclosures.nth(6)).not.toHaveAttribute("open", "");
			await expect(disclosures.nth(7)).not.toHaveAttribute("open", "");
			await expect(
				page.locator("input[name='tone'], input[name='location']"),
			).toHaveCount(0);
			await expect(page.getByText("TONE", { exact: true })).toHaveCount(0);
			await expect(page.getByText("LOCATION", { exact: true })).toHaveCount(0);
		});

		test("selecting a collection swaps the full block and retains state", async ({
			page,
		}) => {
			await openSearch(page);
			const collection = await chooseCollection(page);
			const oldBlock = await page.locator("#artwork-search").elementHandle();
			const response = page.waitForResponse(
				(item) =>
					item.url().includes("/artworks") &&
					new URL(item.url()).searchParams.get("venue") === collection.value,
			);
			await collection.label.click();
			await response;
			await expect(page).toHaveURL(
				(url) => url.searchParams.get("venue") === collection.value,
			);
			await expect
				.poll(
					async () =>
						oldBlock &&
						(await oldBlock.evaluate((element) => element.isConnected)),
				)
				.toBe(false);
			await expect(page.locator("#artwork-search")).toContainText("1 ACTIVE");
			await expect(
				page.locator("#artwork-filters > details").nth(6),
			).toHaveAttribute("open", "");
			await expect(
				page.locator(`input[name='venue'][value='${collection.value}']`),
			).toBeChecked();
			await expect(
				page.locator("#artwork-search-results p[aria-live]"),
			).toContainText(/WORK/);
		});

		test("collection name search only narrows the rail", async ({ page }) => {
			await openSearch(page);
			const collection = await chooseCollection(page);
			await collection.label.click();
			await expect(page).toHaveURL((url) => url.searchParams.has("venue"));
			const beforeCount = await page
				.locator("#artwork-search-results p[aria-live]")
				.innerText();
			const beforeActive = await page.locator("#artwork-filters").innerText();
			const query = page.locator("input[name='venue_q']");
			await query.fill(collection.name.split(/\s+/)[0]);
			await expect(page).toHaveURL((url) => url.searchParams.has("venue_q"));
			await expect(
				page.locator("#artwork-search-results p[aria-live]"),
			).toHaveText(beforeCount);
			await expect(page.locator("#artwork-filters")).toContainText("1 ACTIVE");
			await expect(page.locator("#artwork-filters")).not.toContainText(
				/2 ACTIVE/,
			);
			await expect(page.locator("input[name='venue_q']")).toHaveValue(
				collection.name.split(/\s+/)[0],
			);
			await expect(page.locator("#artwork-filters")).toContainText(
				beforeActive.match(/\d+ ACTIVE/)?.[0] ?? "1 ACTIVE",
			);
		});

		test("year, sort, and view state round-trip through a filter change", async ({
			page,
		}) => {
			await openSearch(page);
			await page.goto("/artworks?year_from=1600&year_to=1700");
			await expect(page.locator("input[name='year_from']")).toHaveValue("1600");
			await expect(page.locator("input[name='year_to']")).toHaveValue("1700");
			await page.getByRole("link", { name: "LIST" }).click();
			await expect(page).toHaveURL(
				(url) => url.searchParams.get("view") === "list",
			);
			await page.getByRole("link", { name: "TITLE" }).click();
			await expect(page).toHaveURL(
				(url) => url.searchParams.get("sort") === "title",
			);
			const collection = await chooseCollection(page);
			await collection.label.click();
			await expect(page).toHaveURL(
				(url) =>
					url.searchParams.get("venue") === collection.value &&
					url.searchParams.get("view") === "list" &&
					url.searchParams.get("sort") === "title" &&
					url.searchParams.get("year_from") === "1600" &&
					url.searchParams.get("year_to") === "1700",
			);
			await expect(page.locator("[data-view='list']")).toBeVisible();
			await expect(page.locator("input[name='year_from']")).toHaveValue("1600");
			await expect(page.locator("input[name='year_to']")).toHaveValue("1700");
		});

		test("sort criterion exposes an explicit direction toggle and persists it", async ({
			page,
		}) => {
			await openSearch(page);
			const direction = page.getByRole("link", {
				name: "Reverse sort direction",
			});
			await expect(direction).toHaveText("↑ ARCHIVE ORDER");
			await expect(direction).toHaveAttribute("href", /dir=desc/);

			await page.getByRole("link", { name: "TITLE" }).click();
			await expect(page).toHaveURL(/sort=title/);
			await expect(
				page.getByRole("link", { name: "Reverse sort direction" }),
			).toHaveText("↑ A–Z");
			await page.getByRole("link", { name: "Reverse sort direction" }).click();
			await expect(page).toHaveURL(/sort=title.*dir=desc|dir=desc.*sort=title/);
			await expect(
				page.getByRole("link", { name: "Reverse sort direction" }),
			).toHaveText("↓ Z–A");
		});

		test("unknown direct collection is an honest active empty result", async ({
			page,
		}) => {
			await page.goto("/artworks?venue=task71-browser-unknown-collection");
			await expect(page).toHaveURL(/venue=task71-browser-unknown-collection/);
			await expect(page.locator("#artwork-search-results")).toContainText(
				"NO MATCHING WORKS",
			);
			await expect(
				page.locator("#artwork-search-results p[aria-live]"),
			).toContainText("0 WORKS MATCH");
			await expect(page.locator("#artwork-filters")).toContainText("1 ACTIVE");
			const unknown = page.locator(
				"input[name='venue'][value='task71-browser-unknown-collection']",
			);
			await expect(unknown).toBeChecked();
			await expect(unknown.locator("..")).toContainText("Unknown collection");
		});

		test("has no horizontal overflow", async ({ page }) => {
			await openSearch(page);
			const overflow = await page.evaluate(
				() => document.documentElement.scrollWidth > window.innerWidth,
			);
			expect(overflow).toBe(false);
		});
	});
}

for (const width of viewports) {
	test(`ordinary GET submits to /artworks without JavaScript at ${width}px`, async ({
		browser,
	}) => {
		const context = await browser.newContext({
			javaScriptEnabled: false,
			viewport: { width, height: 900 },
		});
		const page = await context.newPage();
		await page.goto("/artworks");
		const details = page
			.locator("#artwork-filters > details")
			.filter({ has: page.locator("summary", { hasText: "COLLECTION" }) });
		const summary = details.locator("summary");
		await summary.focus();
		await summary.press("Enter");
		await expect(details).toHaveAttribute("open", "");
		const collection = page
			.locator("#artwork-filters input[name='venue']:not([value=''])")
			.first();
		await expect(collection).toHaveCount(1);
		const value = await collection.getAttribute("value");
		await expect(collection.locator("..")).toBeVisible();
		await collection.focus();
		await collection.press("Space");
		await expect(collection).toBeChecked();
		const query = page.locator("#artwork-filters input[name='q']");
		await query.fill("Synthetic");
		await expect(query).toHaveValue("Synthetic");
		const apply = page.getByRole("button", { name: "APPLY FILTERS" });
		await apply.focus();
		await apply.press("Enter");
		await expect(page).toHaveURL(
			(url) =>
				url.pathname === "/artworks" &&
				url.searchParams.get("q") === "Synthetic" &&
				url.searchParams.get("venue") === value,
		);
		await expect(page.locator("#artwork-search")).toBeVisible();
		await context.close();
	});
}
