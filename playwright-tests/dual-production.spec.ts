import { expect, test } from "@playwright/test";

const artistPath = "/artists/gozzoli-benozzo-r9fb82d431d2a5c";
const artworkPath =
	"/artists/gozzoli-benozzo-r9fb82d431d2a5c/the-mocking-of-christ-detail-r8c3a31f30aefc8";

test("production comparison restores panes, routing, and rendition state", async ({
	page,
}) => {
	const url = `/dual-mode?wide=1&left=${encodeURIComponent(artistPath)}&right=${encodeURIComponent(artworkPath)}&l_size=small&r_size=large&left_render_to=left`;
	await page.goto(url);

	await expect(page.locator("#dual-left")).toContainText("GOZZOLI, Benozzo");
	await expect(page.locator("#dual-right")).toContainText(
		"The Mocking of Christ (detail)",
	);
	await expect(page.locator("#dual-left a[aria-current]")).toHaveCount(1);
	expect(
		await page.locator("#dual-right a[aria-current]").count(),
	).toBeGreaterThan(0);
	await expect(page.locator("#dual-right")).toContainText(
		"REPRODUCTION AT 1600PX WIDE",
	);
	const plate = page.locator("#dual-right img").first();
	await expect(plate).toHaveAttribute("src", /\/api\/files\/artworks\//);
	await expect(plate).toHaveAttribute(
		"data-zoom-url",
		/\/api\/files\/artworks\//,
	);

	await page.reload();
	await expect(page).toHaveURL(/l_size=small/);
	await expect(page).toHaveURL(/r_size=large/);
	await expect(page.locator("#dual-right")).toContainText(
		"The Mocking of Christ (detail)",
	);
});

test("production comparison keeps pane query and sort state addressable", async ({
	page,
}) => {
	await page.goto(
		"/dual-mode?wide=1&l_q=GOZZOLI&l_sort=za&r_q=GOZZOLI&r_sort=birth",
	);
	await expect(page).toHaveURL(/l_q=GOZZOLI/);
	await expect(page).toHaveURL(/r_sort=birth/);
	await expect(page.locator("#dual-left")).toContainText("SORT: Z–A");
	await expect(page.locator("#dual-right")).toContainText("SORT: BIRTH YEAR");
});

test("production comparison applies a pane sort control", async ({ page }) => {
	await page.goto("/dual-mode?wide=1");
	await page.locator("#dual-left a[href*='l_sort=za']").click();
	await expect(page).toHaveURL(/l_sort=za/);
	await expect(page.locator("#dual-left")).toContainText("SORT: Z–A");
});

test("production comparison applies pane search and record navigation without changing its opposite pane", async ({
	page,
}) => {
	await page.goto(
		`/dual-mode?wide=1&right=${encodeURIComponent(artworkPath)}&left_render_to=left`,
	);
	const form = page.locator("form#dual-filters-left");
	await form.locator("input[name='l_q']").fill("GOZZOLI");
	await form.evaluate((element: HTMLFormElement) => element.requestSubmit());
	await expect(page).toHaveURL(/l_q=GOZZOLI/);
	await expect(page.locator("#dual-right")).toContainText(
		"The Mocking of Christ (detail)",
	);

	const record = page
		.locator("#dual-left a", { hasText: "GOZZOLI, Benozzo" })
		.first();
	await record.click();
	await expect(page).toHaveURL(/left=.*gozzoli-benozzo/);
	await expect(page.locator("#dual-left")).toContainText("GOZZOLI, Benozzo");
	await expect(page.locator("#dual-right")).toContainText(
		"The Mocking of Christ (detail)",
	);
});

for (const width of [390, 834, 1440]) {
	test(`production comparison has no overflow at ${width}px`, async ({
		page,
	}) => {
		await page.setViewportSize({ width, height: 900 });
		await page.goto(
			`/dual-mode?wide=1&left=${encodeURIComponent(artistPath)}&right=${encodeURIComponent(artworkPath)}`,
		);
		expect(
			await page.evaluate(
				() =>
					document.documentElement.scrollWidth <=
					document.documentElement.clientWidth,
			),
		).toBe(true);
	});
}
