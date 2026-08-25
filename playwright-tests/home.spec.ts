import { type Locator, type Page, expect, test } from "@playwright/test";

const homeWidths = [390, 834, 1440] as const;
const artworkHrefPattern = /\/artists\/[^/]+\/[^/]+$/;

async function assertNoHorizontalOverflow(page: Page) {
	const dimensions = await page.evaluate(() => ({
		clientWidth: document.documentElement.clientWidth,
		scrollWidth: document.documentElement.scrollWidth,
	}));
	expect(dimensions.scrollWidth).toBeLessThanOrEqual(dimensions.clientWidth);
}

async function assertReducedMotion(locator: Locator) {
	const duration = await locator.evaluate((element) =>
		Number.parseFloat(getComputedStyle(element).transitionDuration),
	);
	expect(duration).toBeLessThanOrEqual(0.001);
}

async function tabTo(page: Page, target: Locator) {
	for (let index = 0; index < 80; index += 1) {
		if (
			await target.evaluate((element) => element === document.activeElement)
		) {
			return;
		}
		await page.keyboard.press("Tab");
	}
	throw new Error("target was not reachable with Tab");
}

function featuredLink(page: Page) {
	return page.locator("aside[aria-labelledby='work-of-the-day-title'] > a");
}

test.describe("home collection entry", () => {
	test("composes the collection argument at every responsive tier without overflow", async ({
		page,
	}) => {
		for (const width of homeWidths) {
			await page.setViewportSize({ width, height: 900 });
			await page.goto("/");
			await expect(
				page.getByRole("heading", { name: /Explore artists/ }),
			).toBeVisible();
			await expect(
				page.getByText(
					"Paintings, sculpture and architecture from the 3rd century to the early 20th",
				),
			).toBeVisible();
			await expect(
				page.locator("aside[aria-labelledby='work-of-the-day-title']"),
			).toBeVisible();
			await assertNoHorizontalOverflow(page);
		}
	});

	test("exposes complete counts, featured metadata, and recent additions", async ({
		page,
	}) => {
		await page.goto("/");

		for (const label of ["ARTWORKS", "ARTISTS", "SCHOOLS", "PERIOD"]) {
			const count = page
				.locator("section[aria-label='Collection counts'] dt")
				.filter({ hasText: label })
				.locator("..")
				.locator("dd");
			await expect(count).toHaveText(/\d[\d,]*|3RD–19TH/);
		}

		await expect(page.locator("#work-of-the-day-title")).toBeVisible();
		const featured = featuredLink(page);
		await expect(featured).toHaveAttribute("href", artworkHrefPattern);
		await expect(featured.locator("span").first()).toHaveText(/\S+/);
		await expect(featured.locator("span").nth(1)).toHaveText(/\S+ · \S+/);

		const recent = page.locator("#recent-additions > li");
		const recentCount = await recent.count();
		expect(recentCount).toBeGreaterThan(0);
		expect(recentCount).toBeLessThanOrEqual(4);
		for (let index = 0; index < recentCount; index += 1) {
			const link = recent.nth(index).getByRole("link");
			await expect(link).toHaveAttribute("href", artworkHrefPattern);
			await expect(link.locator("span").first()).toHaveText(/\S+/);
			await expect(link.locator("span").nth(1)).toHaveText(/\S+ · \S+/);
			await expect(link.locator("img")).toHaveAttribute("loading", "lazy");
		}
	});

	test("uses exact discovery destinations and makes them keyboard reachable", async ({
		page,
	}) => {
		await page.goto("/");
		const discovery = page.locator("nav[aria-label='Discover the collection']");
		const ctas = [
			{ name: "BROWSE ARTISTS →", href: "/artists" },
			{ name: "BROWSE ARTWORKS →", href: "/artworks" },
			{ name: "FIND INSPIRATION →", href: "/inspire" },
		];
		for (const cta of ctas) {
			const link = discovery.locator("a").filter({ hasText: cta.name });
			await expect(link).toHaveAttribute("href", cta.href);
			await tabTo(page, link);
			await expect(link).toBeFocused();
			await expect(link).toHaveCSS("outline-style", /solid|dotted|dashed/);
		}
	});

	test("keeps the same-day featured destination stable across reload", async ({
		page,
	}) => {
		await page.goto("/");
		const href = await featuredLink(page).getAttribute("href");
		expect(href).toMatch(artworkHrefPattern);
		await page.reload();
		await expect(featuredLink(page)).toHaveAttribute("href", href as string);
	});

	test("supports dark theme and reduced motion", async ({ page }) => {
		await page.emulateMedia({ colorScheme: "dark", reducedMotion: "reduce" });
		await page.goto("/");
		await expect(page.locator("html")).toHaveAttribute(
			"data-theme",
			"wga-rams-dark",
		);
		await assertReducedMotion(featuredLink(page));
	});

	test("reflows at 200 percent text size without horizontal overflow", async ({
		page,
	}) => {
		await page.setViewportSize({ width: 390, height: 844 });
		await page.goto("/");
		await page.evaluate(() => {
			document.documentElement.style.fontSize = "2em";
		});
		await assertNoHorizontalOverflow(page);
		await expect(page.getByRole("main")).toBeVisible();
		await expect(
			page.getByRole("heading", { name: /Explore artists/ }),
		).toBeVisible();
	});

	test.describe("without JavaScript", () => {
		test.use({ javaScriptEnabled: false });

		test("keeps complete content and ordinary artwork links", async ({
			page,
		}) => {
			await page.goto("/");
			await expect(
				page.getByRole("heading", { name: /Explore artists/ }),
			).toBeVisible();
			await expect(page.locator("#work-of-the-day-title")).toBeVisible();
			await expect(page.locator("#recent-additions > li")).toHaveCount(4);
			const discovery = page.locator(
				"nav[aria-label='Discover the collection']",
			);
			for (const href of ["/artists", "/artworks", "/inspire"]) {
				const link = discovery.locator(`a[href='${href}']`);
				await expect(link).toHaveCount(1);
				await expect(link).toHaveAttribute("href", href);
			}
			await expect(featuredLink(page)).toHaveAttribute(
				"href",
				artworkHrefPattern,
			);
			await expect(page.locator("#recent-additions a").first()).toHaveAttribute(
				"href",
				artworkHrefPattern,
			);
			await assertNoHorizontalOverflow(page);
		});
	});
});
