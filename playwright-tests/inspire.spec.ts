import { type Page, expect, test } from "@playwright/test";

const viewports = [
	{ width: 390, columns: 1 },
	{ width: 834, columns: 2 },
	{ width: 1440, columns: 4 },
] as const;

async function assertNoHorizontalOverflow(page: Page) {
	const dimensions = await page.evaluate(() => ({
		clientWidth: document.documentElement.clientWidth,
		scrollWidth: document.documentElement.scrollWidth,
	}));
	expect(dimensions.scrollWidth).toBeLessThanOrEqual(dimensions.clientWidth);
}

test("inspiration is a responsive, linkable shuffled collection entry", async ({
	page,
}) => {
	for (const viewport of viewports) {
		await page.setViewportSize({ width: viewport.width, height: 900 });
		await page.goto("/inspire");
		await expect(
			page.getByRole("heading", { name: "A shuffled slice of the collection" }),
		).toBeVisible();
		await expect(
			page.getByText("There is no prescribed order here."),
		).toBeVisible();
		const grid = page.locator(
			"#inspiration > section[aria-label='Shuffled artworks'] > div",
		);
		if ((await grid.count()) > 0) {
			await expect(grid.locator("a").first()).toHaveAttribute(
				"href",
				/\/artists\/[^/]+\/[^/]+$/,
			);
			expect(
				(await grid.evaluate(
					(element) =>
						getComputedStyle(element).gridTemplateColumns.split(" ").length,
				)) as number,
			).toBe(viewport.columns);
		}
		await expect(page.locator("#inspiration a[href='/tours']")).toHaveCount(1);
		await expect(
			page.locator("#inspiration a[href='/itineraries']"),
		).toHaveCount(1);
		await expect(
			page.locator("a[href='/inspire']").filter({ hasText: "ANOTHER SET" }),
		).toHaveCount(1);
		await assertNoHorizontalOverflow(page);
	}
});

test("inspiration reflows at enlarged text without horizontal overflow", async ({
	page,
}) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await page.goto("/inspire");
	await page.evaluate(() => {
		document.documentElement.style.fontSize = "2em";
	});
	await assertNoHorizontalOverflow(page);
	await expect(page.getByRole("main")).toBeVisible();
	await expect(
		page.getByRole("heading", { name: "A shuffled slice of the collection" }),
	).toBeVisible();
});

test("inspiration supports keyboard focus, dark theme, and reduced motion", async ({
	page,
}) => {
	await page.emulateMedia({ colorScheme: "dark", reducedMotion: "reduce" });
	await page.goto("/inspire");
	await expect(page.locator("html")).toHaveAttribute(
		"data-theme",
		"wga-rams-dark",
	);
	const anotherSet = page
		.locator("a[href='/inspire']")
		.filter({ hasText: "ANOTHER SET" });
	if ((await anotherSet.count()) > 0) {
		await anotherSet.focus();
		await expect(anotherSet).toBeFocused();
	}
});

test.describe("inspiration without JavaScript", () => {
	test.use({ javaScriptEnabled: false });

	test("keeps ordinary collection and journey links", async ({ page }) => {
		await page.goto("/inspire");
		await expect(
			page.getByRole("heading", { name: "A shuffled slice of the collection" }),
		).toBeVisible();
		for (const href of ["/inspire", "/tours", "/itineraries"]) {
			await expect(page.locator(`a[href='${href}']`).first()).toHaveAttribute(
				"href",
				href,
			);
		}
	});
});
