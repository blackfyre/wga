import { type Page, expect, test } from "@playwright/test";

const referencePages = [
	{ path: "/pages/about", title: "About" },
	{ path: "/pages/privacy-policy", title: "Privacy policy" },
	{ path: "/contributors", title: "Contributors" },
] as const;

async function expectNoHorizontalOverflow(page: Page) {
	const dimensions = await page.evaluate(() => ({
		clientWidth: document.documentElement.clientWidth,
		scrollWidth: document.documentElement.scrollWidth,
	}));
	expect(dimensions.scrollWidth).toBeLessThanOrEqual(dimensions.clientWidth);
}

test.describe("reference destinations", () => {
	for (const reference of referencePages) {
		test(`${reference.path} provides a canonical, titled public document`, async ({
			page,
		}) => {
			const response = await page.goto(reference.path);

			expect(response?.status()).toBe(200);
			await expect(page).toHaveTitle(new RegExp(reference.title, "i"));
			await expect(page.locator("link[rel='canonical']")).toHaveAttribute(
				"href",
				new RegExp(`${reference.path}$`),
			);
			await expect(
				page.getByRole("heading", {
					level: 1,
					name: new RegExp(`^${reference.title}$`, "i"),
				}),
			).toBeVisible();
			await expect(page.getByRole("main")).toBeVisible();
			await expect(page.locator("main p").first()).toBeVisible();
			await expect(page.locator("header a[href='/']:visible")).toBeVisible();
		});
	}

	test("static documents expose complete content and usable table-of-contents links", async ({
		page,
	}) => {
		for (const path of ["/pages/about", "/pages/privacy-policy"]) {
			await page.goto(path);
			const article = page.locator("main article");
			const contents = page.getByRole("navigation", { name: "Contents" });

			await expect(article).toBeVisible();
			expect(await article.locator("h2, h3, p").count()).toBeGreaterThan(1);
			await expect(contents).toBeVisible();
			const links = contents.getByRole("link");
			expect(await links.count()).toBeGreaterThan(0);
			for (let index = 0; index < (await links.count()); index += 1) {
				const link = links.nth(index);
				const href = await link.getAttribute("href");
				expect(href).toMatch(/^#[\w-]+$/);
				await expect(page.locator(href as string)).toHaveCount(1);
			}
		}
	});

	test("missing public destinations use the shared recovery page while API misses stay technical", async ({
		page,
	}) => {
		for (const path of [
			"/pages/no-such-reference-page",
			"/no-such-public-page",
		]) {
			const response = await page.goto(path);
			expect(response?.status()).toBe(404);
			await expect(
				page.getByRole("heading", {
					level: 1,
					name: "This record is not in the collection.",
				}),
			).toBeVisible();
			const recovery = page.getByRole("link", {
				name: /return to the gallery/i,
			});
			await expect(recovery).toHaveAttribute("href", "/");
			await recovery.focus();
			await expect(recovery).toBeFocused();
			await page.keyboard.press("Enter");
			await expect(page).toHaveURL(/\/$/);
		}

		const response = await page.goto("/api/no-such-public-endpoint");
		expect(response?.status()).toBe(404);
		expect(response?.headers()["content-type"]).not.toMatch(/text\/html/i);
		await expect(
			page.getByRole("heading", {
				name: "This record is not in the collection.",
			}),
		).toHaveCount(0);
	});

	test("reference pages remain usable across Rams viewports and preferences", async ({
		page,
	}) => {
		await page.setViewportSize({ width: 390, height: 900 });
		await page.goto(referencePages[0].path);
		await page.getByRole("button", { name: "DARK", exact: true }).click();
		await expect(page.locator("html")).toHaveAttribute(
			"data-theme",
			"wga-rams-dark",
		);

		for (const width of [390, 834, 1440]) {
			for (const reference of referencePages) {
				await page.setViewportSize({ width, height: 900 });
				await page.goto(reference.path);
				await expect(page.locator("html")).toHaveAttribute(
					"data-theme",
					"wga-rams-dark",
				);
				await expect(
					page.getByRole("heading", {
						level: 1,
						name: new RegExp(`^${reference.title}$`, "i"),
					}),
				).toBeVisible();
				await expectNoHorizontalOverflow(page);
			}
		}

		await page.setViewportSize({ width: 390, height: 844 });
		await page.goto("/pages/about");
		await page.locator("html").evaluate((element) => {
			element.style.fontSize = "200%";
		});
		await expectNoHorizontalOverflow(page);

		await page.emulateMedia({ reducedMotion: "reduce" });
		await page.goto("/pages/privacy-policy");
		expect(
			await page
				.locator("html")
				.evaluate((element) => getComputedStyle(element).scrollBehavior),
		).toBe("auto");
		await expectNoHorizontalOverflow(page);
	});
});

test.describe("reference destinations without JavaScript", () => {
	test.use({ javaScriptEnabled: false });

	test("static documents and contributors retain their complete server-rendered content", async ({
		page,
	}) => {
		for (const reference of referencePages) {
			const response = await page.goto(reference.path);
			expect(response?.status()).toBe(200);
			await expect(
				page.getByRole("heading", {
					level: 1,
					name: new RegExp(`^${reference.title}$`, "i"),
				}),
			).toBeVisible();
			await expect(page.locator("main p").first()).toBeVisible();
		}

		await page.goto("/pages/about");
		await expect(
			page.getByRole("navigation", { name: "Contents" }),
		).toBeVisible();
	});
});
