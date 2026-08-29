import { expect, test } from "@playwright/test";

const toursURL = process.env.WGA_TASK86_TOURS_URL;
const task86Tours = toursURL ? test.describe : test.describe.skip;

const rebuilt = "synthetic-rebuilt-tour-task86";
const legacy = "synthetic-legacy-tour-task86";
const widths = [390, 834, 1440];
test.use({ reducedMotion: "reduce" });

async function expectNoOverflow(page) {
	const dimensions = await page.evaluate(() => ({
		clientWidth: document.documentElement.clientWidth,
		scrollWidth: document.documentElement.scrollWidth,
	}));
	expect(dimensions.scrollWidth).toBeLessThanOrEqual(dimensions.clientWidth);
}

async function expectLandmarkSnapshot(
	page,
	selector: string,
	required: string[],
) {
	const snapshot = await page.locator(selector).ariaSnapshot();
	for (const text of required) {
		expect(snapshot).toContain(text);
	}
}

task86Tours("Task 8.6 Guided Tours", () => {
	test("index filters, grouping, four facts, and synthetic disclosure", async ({
		page,
	}) => {
		await page.goto(`${toursURL}/tours`);
		await expect(
			page.getByRole("heading", { name: "Guided Tours" }),
		).toBeVisible();
		await expect(
			page.getByRole("navigation", { name: "Tour filters" }).getByRole("link"),
		).toHaveCount(5);
		await expect(page.locator("#tours dl dt")).toHaveText([
			"WRITTEN BY",
			"SHAPE",
			"SOURCES",
			"UPKEEP",
		]);
		for (const label of [
			"REBUILT PAGE BY PAGE",
			"THE REST OF THE SERIES",
			"5 PAGES",
		]) {
			await expect(page.locator("#tours")).toContainText(label);
		}
		await expect(page.locator("#tours")).toContainText(
			"Synthetic Rebuilt Tour (Task 8.6 Fixture)",
		);
		await expect(page.locator("#tours")).toContainText(
			"Synthetic Legacy Tour (Task 8.6 Fixture)",
		);
		await expect(page.locator("#tours")).toContainText("Not editorial content");
		await expect(page.locator("#tours")).not.toContainText(
			/real source|historical fact|authentic/i,
		);
		await page
			.getByRole("navigation", { name: "Tour filters" })
			.getByRole("link", { name: "Survey", exact: true })
			.click();
		await expect(page).toHaveURL(/\/tours\?kind=survey$/);
		await expect(page.locator("#tours")).toContainText(
			"Synthetic Rebuilt Tour (Task 8.6 Fixture)",
		);
		await expect(page.locator("#tours")).not.toContainText(
			"Synthetic Legacy Tour (Task 8.6 Fixture)",
		);
	});

	for (const [address, marker] of [
		[1, "Synthetic Rebuilt Tour (Task 8.6 Fixture)"],
		[2, "Synthetic Text Page"],
		[3, "Synthetic Picture Page"],
		[4, "Synthetic Index Page"],
		[5, "Sources"],
	] as const) {
		test(`rebuilt route ${address} has page context and controls`, async ({
			page,
		}) => {
			const route =
				address === 1
					? `${toursURL}/tours/${rebuilt}`
					: `${toursURL}/tours/${rebuilt}/${address}`;
			await page.goto(route);
			const expectedPath =
				address === 1 ? `/tours/${rebuilt}` : `/tours/${rebuilt}/${address}`;
			await expect(page).toHaveURL(new RegExp(`${expectedPath}$`));
			await expect(page.locator("#tour")).toContainText(marker);
			await expect(page.locator('[role="progressbar"]')).toHaveAttribute(
				"aria-valuenow",
				String(address),
			);
			await expect(page.locator('[role="progressbar"]')).toHaveAttribute(
				"aria-valuemax",
				"5",
			);
			await expect(
				page.getByRole("navigation", { name: "Tour pages" }),
			).toBeVisible();
			const contents = page.locator("#tour details ol a");
			await expect(contents).toHaveCount(5);
			for (const [index, href] of [
				[1, `/tours/${rebuilt}`],
				[2, `/tours/${rebuilt}/2`],
				[3, `/tours/${rebuilt}/3`],
				[4, `/tours/${rebuilt}/4`],
				[5, `/tours/${rebuilt}/5`],
			] as const) {
				await expect(contents.nth(index - 1)).toHaveAttribute("href", href);
			}
			await expect(
				page.locator('#tour details ol a[aria-current="page"]'),
			).toHaveCount(1);
			await expect(contents.nth(address - 1)).toHaveAttribute(
				"aria-current",
				"page",
			);
			await expect(page.locator("#tour")).toContainText("Synthetic");
			await expect(page.locator("#tour")).not.toContainText(
				/historical fact|authentic/i,
			);
		});
	}

	test("picture route exposes fixed 1400/2000 profiles and plate geometry", async ({
		page,
	}) => {
		await page.goto(`${toursURL}/tours/${rebuilt}/3`);
		const plate = page.locator("#tour figure > div");
		await expect(plate).toHaveClass(/h-\[300px\]/);
		await expect(plate.locator("img")).toHaveAttribute("src", /thumb=1400x0/);
		await expect(plate.locator("img")).toHaveAttribute(
			"data-zoom-url",
			/thumb=2000x0/,
		);
		await expect(plate).toHaveCSS("height", /.+/);
		await expect(
			page.getByRole("link", { name: "OPEN THE FULL RECORD →" }),
		).toHaveAttribute("href", "/artworks/synthetic-work");
	});

	test("legacy route is explicitly original-layout and preserves safe destination", async ({
		page,
	}) => {
		await page.goto(`${toursURL}/tours/${legacy}`);
		await expect(page.locator("#tour")).toContainText(
			"STILL IN ITS ORIGINAL LAYOUT",
		);
		await expect(page.locator("#tour")).toContainText(
			"not yet been rebuilt page by page",
		);
		await expect(
			page.getByRole("link", { name: "Open the original tour" }),
		).toHaveAttribute("href", "https://example.org/synthetic-legacy-original");
		await expect(page.locator('[role="progressbar"]')).toHaveAttribute(
			"aria-valuemax",
			"1",
		);
		await expect(page.locator("#tour")).not.toContainText("PAGE 01 OF 05");
	});

	test("tour keyboard, landmark snapshot, reduced motion, and HTMX fragment", async ({
		page,
	}) => {
		await page.goto(`${toursURL}/tours/${rebuilt}/2`);
		await page.emulateMedia({ reducedMotion: "reduce" });
		await page.locator('#tour a[href*="/3"]').first().focus();
		await expect(page.locator('#tour a[href*="/3"]').first()).toBeFocused();
		await expectLandmarkSnapshot(page, "#tour", [
			"article",
			"GUIDED TOURS",
			"Tour pages",
		]);
		expect(
			await page.evaluate(
				() => matchMedia("(prefers-reduced-motion: reduce)").matches,
			),
		).toBe(true);
		const response = await page.request.get(`${toursURL}/tours/${rebuilt}/3`, {
			headers: { "HX-Request": "true", "HX-Target": "tour" },
		});
		const body = await response.text();
		expect(body).toContain('id="tour"');
		expect(body).not.toContain("<html");
	});

	test("ArrowRight and ArrowLeft turn addressed pages in the browser", async ({
		page,
	}) => {
		await page.goto(`${toursURL}/tours/${rebuilt}/2`);
		await page.locator('#tour a[data-tour-nav="next"]').focus();
		await page.keyboard.press("ArrowRight");
		await expect(page).toHaveURL(new RegExp(`/tours/${rebuilt}/3$`));
		await expect(page.locator("#tour")).toContainText("Synthetic Picture Page");
		await page.locator('#tour a[data-tour-nav="prev"]').focus();
		await page.keyboard.press("ArrowLeft");
		await expect(page).toHaveURL(new RegExp(`/tours/${rebuilt}/2$`));
		await expect(page.locator("#tour")).toContainText("Synthetic Text Page");
	});

	test.describe("ordinary no-JavaScript tour navigation", () => {
		test.use({ javaScriptEnabled: false });
		test("follows page turns with stable addresses", async ({ page }) => {
			await page.goto(`${toursURL}/tours/${rebuilt}/1`);
			await page
				.getByRole("link", { name: /START THE TOUR/ })
				.click({ force: true });
			await expect(page).toHaveURL(new RegExp(`/tours/${rebuilt}/2$`));
			await expect(page.locator("#tour")).toContainText("Synthetic Text Page");
		});
		test("filters the index and follows stable ordinary links", async ({
			page,
		}) => {
			await page.goto(`${toursURL}/tours`);
			await page
				.getByRole("navigation", { name: "Tour filters" })
				.getByRole("link", { name: "Survey", exact: true })
				.click({ force: true });
			await expect(page).toHaveURL(/\/tours\?kind=survey$/);
			await page
				.getByRole("link", {
					name: "Synthetic Rebuilt Tour (Task 8.6 Fixture)",
				})
				.click({ force: true });
			await expect(page).toHaveURL(new RegExp(`/tours/${rebuilt}$`));
		});
	});

	for (const width of widths) {
		test(`tour reflows without overflow at ${width}px and 200% text`, async ({
			page,
		}) => {
			await page.setViewportSize({ width, height: 900 });
			await page.goto(`${toursURL}/tours/${rebuilt}/3`);
			await expectNoOverflow(page);
			await page.evaluate(() => {
				document.documentElement.style.fontSize = "2em";
			});
			await expect(
				page.getByRole("heading", { name: "Synthetic Picture Page" }),
			).toBeVisible();
			await expectNoOverflow(page);
		});
	}
});

test.describe("Task 8.6 Timeline", () => {
	test("six lanes, positive source-backed artists/works/movements, and honest unavailable lanes", async ({
		page,
	}) => {
		await page.goto("/timeline?from=100&to=1994");
		await expect(
			page.locator('#timeline section[aria-label="Timeline lanes"] li'),
		).toHaveCount(6);
		for (const label of [
			"ARTISTS",
			"WORKS",
			"MOVEMENTS",
			"BUILDINGS",
			"EVENTS",
			"MUSIC",
		]) {
			await expect(
				page.locator('#timeline section[aria-label="Timeline lanes"]'),
			).toContainText(label);
		}
		await expect(
			page
				.getByRole("region", { name: "Artists in this window" })
				.locator('a[href^="/artists/"]'),
		).not.toHaveCount(0);
		await expect(
			page
				.getByRole("region", { name: "Works in this window" })
				.locator('a[href^="/artists/"]'),
		).not.toHaveCount(0);
		await expect(
			page.locator('#timeline section[aria-label="Timeline lanes"]'),
		).toContainText("SHOWN");
		for (const index of [0, 1, 2]) {
			await expect(
				page
					.locator('#timeline section[aria-label="Timeline lanes"] li')
					.nth(index)
					.locator("span")
					.first(),
			).toHaveText(/[1-9]/);
		}
		for (const lane of [
			"BUILDINGS IN THIS WINDOW",
			"EVENTS IN THIS WINDOW",
			"MUSIC IN THIS WINDOW",
		]) {
			await expect(page.getByRole("region", { name: lane })).toContainText(
				/No approved|no approved/,
			);
		}
		await expect(
			page.getByRole("region", { name: "Events in this window" }),
		).not.toContainText(/war broke out|revolution|battle of/i);
	});

	test("URL window, density table, periods, bounded marks, disclosure, and canonical pagination", async ({
		page,
	}) => {
		await page.goto("/timeline?from=100&to=1994");
		await expect(page).toHaveURL(/from=100&to=1994$/);
		await expect(page.locator("#timeline")).toContainText("100–1994");
		await expect(page.locator("#timeline-window")).toHaveAttribute(
			"method",
			"GET",
		);
		await expect(page.locator("#timeline table caption")).toContainText(
			"100 to 1994",
		);
		await expect(
			page.getByRole("region", { name: "Art periods in this window" }),
		).toBeVisible();
		await expect(
			page.locator('ul[aria-label^="Published works dated"] li'),
		).toHaveCount(48);
		await expect(page.locator("#timeline")).toContainText(
			"BARS SHOW CATALOGUE DENSITY PER DECADE",
		);
		await expect(
			page.getByRole("navigation", { name: "Works pagination" }),
		).toBeVisible();
		await expect(
			page.getByRole("navigation", { name: "Works pagination" }),
		).toContainText("PAGE 1 OF");
		await expect(
			page
				.locator(
					'section[aria-label="Works in this window"] a[href^="/artists/"]',
				)
				.first(),
		).toHaveAttribute(
			"href",
			/^\/artists\/[^/]+-[A-Za-z0-9]{15}\/[^/]+-[A-Za-z0-9]{15}$/,
		);
	});

	test("timeline keyboard, landmarks, ARIA snapshot, reduced motion and HTMX full/fragment parity", async ({
		page,
	}) => {
		await page.goto("/timeline?from=100&to=1994");
		await page.emulateMedia({ reducedMotion: "reduce" });
		await page.locator('input[name="from"]').focus();
		await expect(page.locator('input[name="from"]')).toBeFocused();
		await page.keyboard.press("Tab");
		await expect(page.locator('input[name="to"]')).toBeFocused();
		await expectLandmarkSnapshot(page, "#timeline", [
			"Timeline",
			"Timeline lanes",
		]);
		expect(
			await page.evaluate(
				() => matchMedia("(prefers-reduced-motion: reduce)").matches,
			),
		).toBe(true);
		const full = await page.request.get("/timeline?from=100&to=1994");
		const fragment = await page.request.get("/timeline?from=100&to=1994", {
			headers: { "HX-Request": "true", "HX-Target": "timeline" },
		});
		expect(await full.text()).toContain("<html");
		expect(await fragment.text()).toContain('id="timeline"');
		expect(await fragment.text()).not.toContain("<html");
		await page.reload();
		await expect(page.locator("#timeline")).toContainText("100–1994");
	});

	test.describe("ordinary no-JavaScript timeline parity", () => {
		test.use({ javaScriptEnabled: false });
		test("submits a shareable window and retains positive entries", async ({
			page,
		}) => {
			await page.goto("/timeline?from=100&to=1994");
			await page.locator('input[name="from"]').fill("1600");
			await page.locator('input[name="to"]').fill("1700");
			await page.getByRole("button", { name: "APPLY WINDOW" }).click();
			await expect(page).toHaveURL(/from=1600&to=1700$/);
			await expect(page.locator("#timeline")).toContainText("1600–1700");
			await expect(
				page.locator(
					'section[aria-label="Works in this window"] a[href^="/artists/"]',
				),
			).not.toHaveCount(0);
		});
	});

	for (const width of widths) {
		test(`timeline reflows without overflow at ${width}px and 200% text`, async ({
			page,
		}) => {
			await page.setViewportSize({ width, height: 900 });
			await page.goto("/timeline?from=100&to=1994");
			await expectNoOverflow(page);
			await page.evaluate(() => {
				document.documentElement.style.fontSize = "2em";
			});
			await expect(
				page.getByRole("heading", { name: "Timeline" }),
			).toBeVisible();
			await expectNoOverflow(page);
		});
	}
});
