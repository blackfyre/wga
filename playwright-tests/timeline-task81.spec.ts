import { expect, test } from "@playwright/test";

const viewports = [390, 834, 1440];
const unavailableLanes = [
	{
		label: "BUILDINGS",
		region: "BUILDINGS IN THIS WINDOW",
		note: "No approved source-backed building records have been supplied",
	},
	{
		label: "EVENTS",
		region: "EVENTS IN THIS WINDOW",
		note: "No approved source-backed historical-event records have been supplied",
	},
	{
		label: "MUSIC",
		region: "MUSIC IN THIS WINDOW",
		note: "no approved art-period mapping",
	},
] as const;

async function expectNoHorizontalOverflow(page) {
	const dimensions = await page.evaluate(() => ({
		clientWidth: document.documentElement.clientWidth,
		scrollWidth: document.documentElement.scrollWidth,
	}));
	expect(dimensions.scrollWidth).toBeLessThanOrEqual(dimensions.clientWidth);
}

test("timeline exposes six labelled lanes and honest unavailable states", async ({
	page,
}) => {
	await page.goto("/timeline");

	const lanes = page.locator(
		'#timeline section[aria-label="Timeline lanes"] li',
	);
	await expect(lanes).toHaveCount(6);
	await expect(lanes).toContainText([
		"ARTISTS",
		"WORKS",
		"MOVEMENTS",
		"BUILDINGS",
		"EVENTS",
		"MUSIC",
	]);

	for (const lane of unavailableLanes) {
		const region = page.getByRole("region", { name: lane.region });
		await expect(region).toBeVisible();
		await expect(region).toContainText(lane.note);
		await expect(region).not.toContainText(
			/war broke out|revolution|battle of/i,
		);
	}
});

test("selected window has canonical URL and preserves server-rendered chronology", async ({
	page,
}) => {
	await page.goto("/timeline?from=1600&to=1700");
	await expect(page).toHaveURL(/\/timeline\?from=1600&to=1700$/);
	await expect(page.locator("#timeline")).toContainText("1600–1700");
	await expect(page.locator("#timeline-window")).toHaveAttribute(
		"action",
		"/timeline",
	);
	await expect(page.locator("#timeline-window")).toHaveAttribute(
		"method",
		"GET",
	);
	await expect(page.locator('input[name="from"]')).toHaveValue("1600");
	await expect(page.locator('input[name="to"]')).toHaveValue("1700");
});

test("seeded artist and artwork entries retain canonical public links", async ({
	page,
}) => {
	await page.goto("/timeline");

	const artistLinks = page.locator(
		'#timeline section[aria-label="Artists in this window"] a[href^="/artists/"]',
	);
	if ((await artistLinks.count()) > 0) {
		await expect(artistLinks.first()).toHaveAttribute(
			"href",
			/^\/artists\/[^/]+-[a-zA-Z0-9]{15}$/,
		);
	}

	const artworkLinks = page.locator(
		'#timeline section[aria-label="Works in this window"] a[href^="/artists/"]',
	);
	if ((await artworkLinks.count()) > 0) {
		await expect(artworkLinks.first()).toHaveAttribute(
			"href",
			/^\/artists\/[^/]+-[a-zA-Z0-9]{15}\/[^/]+-[a-zA-Z0-9]{15}$/,
		);
	}
});

test.describe("ordinary no-JavaScript timeline window submission", () => {
	test.use({ javaScriptEnabled: false });

	test("submits the selected window as a normal GET", async ({ page }) => {
		await page.goto("/timeline");
		await page.locator('input[name="from"]').fill("1600");
		await page.locator('input[name="to"]').fill("1700");
		await page.getByRole("button", { name: "APPLY WINDOW" }).click();
		await expect(page).toHaveURL(/\/timeline\?from=1600&to=1700$/);
		await expect(page.locator("#timeline")).toContainText("1600–1700");
	});
});

test("timeline range controls expose keyboard-visible focus", async ({
	page,
}) => {
	await page.goto("/timeline");

	const from = page.locator('input[name="from"]');
	const to = page.locator('input[name="to"]');
	await from.focus();
	await expect(from).toBeFocused();
	await page.keyboard.press("Tab");
	await expect(to).toBeFocused();
});

for (const width of viewports) {
	test(`timeline has no document overflow at ${width}px`, async ({ page }) => {
		await page.setViewportSize({ width, height: 900 });
		await page.goto("/timeline");
		await expect(page.getByRole("heading", { name: "Timeline" })).toBeVisible();
		await expectNoHorizontalOverflow(page);
	});
}

test("timeline reflows at 200% text at 390px", async ({ page }) => {
	await page.setViewportSize({ width: 390, height: 900 });
	await page.goto("/timeline");
	await page.evaluate(() => {
		document.documentElement.style.fontSize = "2em";
	});
	await expect(page.getByRole("heading", { name: "Timeline" })).toBeVisible();
	await expectNoHorizontalOverflow(page);
	await expect(
		page.getByRole("region", { name: "MUSIC IN THIS WINDOW" }),
	).toBeVisible();
});
