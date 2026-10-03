import { expect, type Locator, type Page, test } from "@playwright/test";

const homeWidths = [390, 834, 1440] as const;
const artworkHrefPattern = /\/artists\/[^/]+\/[^/]+$/;

async function assertNoHorizontalOverflow(page: Page) {
	const dimensions = await page.evaluate(() => ({
		clientWidth: document.documentElement.clientWidth,
		scrollWidth: document.documentElement.scrollWidth,
	}));
	expect(dimensions.scrollWidth).toBeLessThanOrEqual(dimensions.clientWidth);
}

async function tokenPx(page: Page, token: string) {
	return page.evaluate((name) => {
		const probe = document.createElement("span");
		probe.style.fontSize = `var(${name})`;
		document.body.append(probe);
		const size = getComputedStyle(probe).fontSize;
		probe.remove();
		return size;
	}, token);
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
				page.getByText("00 — PROVIDING EXPERIENCE SINCE 1996", {
					exact: true,
				}),
			).toBeVisible();
			await expect(
				page.getByRole("link", { name: "For agents · llms.txt" }),
			).toHaveAttribute("href", "/llms.txt");
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

		for (const label of ["REPRODUCTIONS", "ARTISTS", "SCHOOLS", "CENTURY"]) {
			const count = page
				.locator("section[aria-label='Collection counts'] dt")
				.filter({ hasText: label })
				.locator("..")
				.locator("dd");
			await expect(count).toHaveText(/^(\d[\d,]*|3rd–early 20th)$/);
		}

		await expect(page.locator("#work-of-the-day-title")).toBeVisible();
		const featured = featuredLink(page);
		await expect(featured).toHaveAttribute("href", artworkHrefPattern);
		await expect(featured.locator("span").first()).toHaveText(/\S+/);
		const caption = featured.locator("div > span").nth(1);
		await expect(caption).toHaveText(/\S+ · \S+ →$/);
		await expect(caption.locator("span.uppercase")).toHaveCSS(
			"text-transform",
			"uppercase",
		);
		await expect(caption.locator("span[aria-hidden='true']")).toHaveText("→");

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
		await expect(discovery.locator("a")).toHaveCount(2);
		const ctas = [
			{ name: "BROWSE ARTISTS →", href: "/artists" },
			{ name: "COMPARE TWO WORKS →", href: "/dual-mode" },
		];
		for (const cta of ctas) {
			const link = discovery.locator("a").filter({ hasText: cta.name });
			await expect(link).toHaveAttribute("href", cta.href);
			await expect(link).toHaveCSS("height", "48px");
			await tabTo(page, link);
			await expect(link).toBeFocused();
			await expect(link).toHaveCSS("outline-style", /solid|dotted|dashed/);
		}
	});

	test("offers the design's feature cards with real destinations at every responsive tier", async ({
		page,
	}) => {
		const cards = [
			{
				label: "TWO WINDOWS",
				title: "Run two copies of the collection",
				links: [{ name: "OPEN DUAL MODE →", href: "/dual-mode" }],
			},
			{
				label: "TIMELINE",
				title: "See what overlapped in a given fifty years",
				links: [{ name: "OPEN THE TIMELINE →", href: "/timeline" }],
			},
			{
				label: "POSTCARD SERVICE",
				title: "Send any work as a postcard",
				links: [
					{ name: "SEND A POSTCARD →", href: "/postcard" },
					{ name: "MEET THE CONTRIBUTORS →", href: "/contributors" },
				],
			},
		];
		for (const width of homeWidths) {
			await page.setViewportSize({ width, height: 900 });
			await page.goto("/");
			const region = page.locator("section[aria-label='More ways to explore']");
			const titleSize = await tokenPx(page, "--t-22");
			const linkSize = await tokenPx(page, "--t-11");
			const sections = region.locator(":scope > section");
			await expect(sections).toHaveCount(cards.length);
			for (const [index, card] of cards.entries()) {
				const section = sections.nth(index);
				await expect(section.locator("p").first()).toHaveText(card.label);
				await expect(section.getByRole("heading", { level: 2 })).toHaveText(
					card.title,
				);
				await expect(section.getByRole("heading", { level: 2 })).toHaveCSS(
					"font-size",
					titleSize,
				);
				await expect(section.locator("a")).toHaveCount(card.links.length);
				for (const [linkIndex, expected] of card.links.entries()) {
					const link = section.locator("a").nth(linkIndex);
					await expect(link).toHaveText(expected.name);
					await expect(link).toHaveAttribute("href", expected.href);
					await expect(link).toHaveCSS("text-align", "right");
					await expect(link).toHaveCSS("font-size", linkSize);
					await expect(link).toHaveCSS(
						"margin-top",
						linkIndex === 0 ? "0px" : "10px",
					);
					await expect(link).toBeVisible();
				}
			}
			await expect(region).not.toContainText("Period music optional");
			await expect(page.getByText("Help sustain the archive")).toHaveCount(0);
			await expect(
				page.getByRole("link", { name: "ALL WORKS →" }),
			).toHaveAttribute("href", "/artworks");
			await assertNoHorizontalOverflow(page);
		}
	});

	test("feature card and hero destinations resolve to real pages", async ({
		page,
	}) => {
		for (const name of [
			"COMPARE TWO WORKS →",
			"OPEN DUAL MODE →",
			"OPEN THE TIMELINE →",
			"SEND A POSTCARD →",
			"MEET THE CONTRIBUTORS →",
			"ALL WORKS →",
		]) {
			await page.goto("/");
			const href = await page
				.getByRole("link", { name, exact: true })
				.getAttribute("href");
			const response = await page.request.get(href as string);
			expect(response.status(), `${name} → ${href}`).toBe(200);
		}
		await page.goto("/");
		for (const href of ["/inspire", "/artworks"]) {
			await expect(
				page.locator(`nav[aria-label='Primary navigation'] a[href='${href}']`),
			).not.toHaveCount(0);
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
		await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
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
			for (const href of ["/artists", "/dual-mode"]) {
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
