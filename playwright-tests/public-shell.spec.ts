import { type Locator, type Page, expect, test } from "@playwright/test";

const shellWidths = [390, 834, 1440] as const;
const deferredHrefs = [
	"/timeline",
	"/tours",
	"/itineraries",
	"/statistics",
	"/glossary",
	"/guestbook",
	"/postcard",
	"/pages/about",
];

async function assertNoHorizontalOverflow(page: Page) {
	const dimensions = await page.evaluate(() => ({
		clientWidth: document.documentElement.clientWidth,
		scrollWidth: document.documentElement.scrollWidth,
	}));
	expect(dimensions.scrollWidth).toBeLessThanOrEqual(dimensions.clientWidth);
}

async function assertInBounds(locator: Locator, width: number) {
	const count = await locator.count();
	for (let index = 0; index < count; index += 1) {
		const item = locator.nth(index);
		if (!(await item.isVisible())) {
			continue;
		}
		const box = await item.boundingBox();
		expect(box).not.toBeNull();
		if (box === null) {
			throw new Error("visible shell element has no bounding box");
		}
		expect(box.x).toBeGreaterThanOrEqual(0);
		expect(box.x + box.width).toBeLessThanOrEqual(width);
	}
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

test.describe("shared public shell", () => {
	test("composes identity, navigation, and footer by viewport tier", async ({
		page,
	}) => {
		await page.setViewportSize({ width: 390, height: 900 });
		await page.goto("/");
		await expect(page.locator("header a[href='/']").first()).toBeVisible();
		await expect(
			page.locator("header > nav[aria-label='Primary navigation']"),
		).toBeHidden();
		const mobileDetails = page.locator(
			"header details[data-kbd-mobile-navigation]",
		);
		await expect(mobileDetails).not.toHaveAttribute("open", "");
		await mobileDetails
			.locator("summary[aria-label='Open primary navigation']")
			.click();
		await expect(mobileDetails).toHaveAttribute("open", "");
		const mobileNavigation = mobileDetails.getByRole("navigation", {
			name: "Primary navigation",
		});
		await expect(mobileNavigation).toBeVisible();
		const mobileBox = await mobileDetails.boundingBox();
		expect(mobileBox).not.toBeNull();
		if (mobileBox === null) {
			throw new Error("mobile navigation has no bounding box");
		}
		expect(mobileBox.x + mobileBox.width).toBeLessThanOrEqual(390);
		const navigationBox = await mobileNavigation.boundingBox();
		expect(navigationBox).not.toBeNull();
		if (navigationBox === null) {
			throw new Error("mobile navigation has no bounding box");
		}
		expect(navigationBox.x).toBeGreaterThanOrEqual(0);
		expect(navigationBox.x + navigationBox.width).toBeLessThanOrEqual(390);
		await expect(page.locator("header")).toBeVisible();
		await expect(page.getByRole("main")).toBeVisible();
		await expect(page.locator("footer")).toBeVisible();

		await page.setViewportSize({ width: 834, height: 900 });
		await page.goto("/");
		await expect(
			page.locator("header > nav[aria-label='Primary navigation']"),
		).toBeVisible();
		await expect(page.locator("footer > div").first()).toHaveCSS(
			"grid-template-columns",
			/\S+\s+\S+$/,
		);
		const mediumColumns = await page
			.locator("footer > div")
			.first()
			.evaluate(
				(element) =>
					getComputedStyle(element).gridTemplateColumns.split(" ").length,
			);
		expect(mediumColumns).toBe(2);

		await page.setViewportSize({ width: 1440, height: 900 });
		await page.goto("/");
		const desktopNavigation = page.locator(
			"header > nav[aria-label='Primary navigation']",
		);
		await expect(desktopNavigation).toBeVisible();
		await expect(
			desktopNavigation.getByText("MORE", { exact: true }),
		).toBeVisible();
		const largeColumns = await page
			.locator("footer > div")
			.first()
			.evaluate(
				(element) =>
					getComputedStyle(element).gridTemplateColumns.split(" ").length,
			);
		expect(largeColumns).toBe(4);
	});

	test("exposes ordinary navigation and deferred destinations", async ({
		page,
	}) => {
		await page.setViewportSize({ width: 1440, height: 900 });
		await page.goto("/");
		const navigation = page.getByRole("navigation", {
			name: "Primary navigation",
		});
		await expect(
			navigation.getByRole("link", { name: "ARTWORKS", exact: true }),
		).toHaveAttribute("href", "/artworks");
		await expect(
			navigation.getByRole("link", { name: "ARTISTS", exact: true }),
		).toHaveAttribute("href", "/artists");
		for (const href of deferredHrefs) {
			await expect(page.locator(`footer a[href='${href}']`)).toHaveCount(1);
		}
	});

	test("supports keyboard-only disclosure, links, and footer preferences", async ({
		page,
	}) => {
		await page.setViewportSize({ width: 390, height: 844 });
		await page.goto("/");

		const summary = page.locator(
			"summary[aria-label='Open primary navigation']",
		);
		await tabTo(page, summary);
		await expect(summary).toBeFocused();
		await expect(summary).toHaveCSS("outline-style", /solid|dotted|dashed/);
		await page.keyboard.press("Enter");
		await expect(
			page.locator("header details[data-kbd-mobile-navigation]"),
		).toHaveAttribute("open", "");
		await page.keyboard.press("Shift+Tab");
		await expect(page.locator("header a[href='/']").first()).toBeFocused();
		await page.keyboard.press("Tab");
		await expect(summary).toBeFocused();

		const primaryArtists = page
			.getByRole("navigation", { name: "Primary navigation" })
			.getByRole("link", {
				name: "ARTISTS",
				exact: true,
			});
		await tabTo(page, primaryArtists);
		await page.keyboard.press("Enter");
		await expect(page).toHaveURL(/\/artists$/);

		await page.goto("/");
		const secondaryArtists = page.locator("footer a[href='/artists']");
		await tabTo(page, secondaryArtists);
		await page.keyboard.press("Enter");
		await expect(page).toHaveURL(/\/artists$/);

		await page.goto("/");
		const dark = page.getByRole("button", { name: "DARK", exact: true });
		await tabTo(page, dark);
		await expect(dark).toBeFocused();
		await expect(dark).toHaveCSS("outline-style", /solid|dotted|dashed/);
		await page.keyboard.press("Enter");
		await expect(page.locator("html")).toHaveAttribute(
			"data-theme",
			"wga-rams-dark",
		);
		const bionic = page.getByRole("switch", { name: "Bionic reading" });
		await tabTo(page, bionic);
		await page.keyboard.press("Space");
		await expect(bionic).toHaveAttribute("aria-checked", "true");
		await page.keyboard.press("Shift+Tab");
		await expect(dark).toBeFocused();
	});

	test("keeps Rams dark roles readable at every shell tier", async ({
		page,
	}) => {
		for (const width of shellWidths) {
			await page.setViewportSize({ width, height: 900 });
			await page.goto("/");
			await page.getByRole("button", { name: "DARK", exact: true }).click();
			await expect(page.locator("html")).toHaveAttribute(
				"data-theme",
				"wga-rams-dark",
			);
			const roles = await page.evaluate(() => {
				const root = getComputedStyle(document.documentElement);
				const body = getComputedStyle(document.body);
				const main = getComputedStyle(
					document.querySelector("main") as Element,
				);
				const surfaceFor = (element: Element) => {
					const background = getComputedStyle(element).backgroundColor;
					if (background === "rgba(0, 0, 0, 0)") {
						return body.backgroundColor;
					}
					return background;
				};
				return {
					colorScheme: root.colorScheme,
					base: root.getPropertyValue("--color-base-100").trim(),
					content: root.getPropertyValue("--color-base-content").trim(),
					primary: root.getPropertyValue("--color-primary").trim(),
					headerColor: getComputedStyle(
						document.querySelector("header") as Element,
					).color,
					headerSurface: surfaceFor(
						document.querySelector("header") as Element,
					),
					mainColor: main.color,
					mainSurface: surfaceFor(document.querySelector("main") as Element),
					footerColor: getComputedStyle(
						document.querySelector("footer") as Element,
					).color,
					footerSurface: surfaceFor(
						document.querySelector("footer") as Element,
					),
				};
			});
			expect(roles.colorScheme).toBe("dark");
			expect(roles.base).toBe("#1a1814");
			expect(roles.content).toBe("#edeae1");
			expect(roles.primary).not.toBe("");
			expect(roles.headerColor).toBe("rgb(237, 234, 225)");
			expect(roles.headerSurface).toBe("rgb(26, 24, 20)");
			expect(roles.mainColor).toBe("rgb(237, 234, 225)");
			expect(roles.mainSurface).toBe("rgb(26, 24, 20)");
			expect(roles.footerColor).toBe("rgb(237, 234, 225)");
			expect(roles.footerSurface).toBe("rgb(26, 24, 20)");
			await assertNoHorizontalOverflow(page);
		}
	});

	test("keeps landmarks and controls reachable at 200 percent and 400 percent reflow", async ({
		page,
	}) => {
		await page.setViewportSize({ width: 390, height: 844 });
		await page.goto("/");
		const cookieAccept = page.getByRole("button", { name: /accept/i }).first();
		if ((await cookieAccept.count()) > 0) {
			await cookieAccept.click();
		}
		await page.evaluate(() => {
			document.documentElement.style.fontSize = "2em";
		});
		await assertNoHorizontalOverflow(page);
		let relevant = page.locator(
			"header, main, footer, header a[href], footer a[href], header button, footer button, header summary, footer summary",
		);
		await assertInBounds(relevant, 390);
		await expect(page.locator("header")).toBeVisible();
		await expect(page.getByRole("main")).toBeVisible();
		await expect(page.locator("footer")).toBeVisible();
		await expect(
			page.getByRole("link", { name: "Artists" }).last(),
		).toBeVisible();

		await page.setViewportSize({ width: 320, height: 844 });
		await page.evaluate(() => {
			document.documentElement.style.fontSize = "1em";
		});
		await assertNoHorizontalOverflow(page);
		relevant = page.locator(
			"header, main, footer, header a[href], footer a[href], header button, footer button, header summary, footer summary",
		);
		await assertInBounds(relevant, 320);
		await expect(page.locator("header")).toBeVisible();
		await expect(page.getByRole("main")).toBeVisible();
		await expect(page.locator("footer")).toBeVisible();
		await expect(
			page.getByRole("link", { name: "Artists" }).last(),
		).toBeVisible();
	});

	test("retains native disclosure and honest unavailable controls without JavaScript", async ({
		browser,
	}) => {
		const context = await browser.newContext({
			javaScriptEnabled: false,
			viewport: { width: 390, height: 844 },
		});
		const page = await context.newPage();
		await page.goto("/");
		const menu = page.locator("header details[data-kbd-mobile-navigation]");
		await expect(menu).not.toHaveAttribute("open", "");
		await menu.locator("summary[aria-label='Open primary navigation']").click();
		await expect(menu).toHaveAttribute("open", "");
		await expect(
			menu.getByRole("link", { name: "ARTISTS", exact: true }),
		).toBeVisible();
		await expect(page.locator("[data-wga-theme-toggle]")).toBeHidden();
		await expect(page.locator("[data-wga-cookie-settings]")).toBeHidden();
		await expect(page.locator("[data-wga-cookie-settings]")).toHaveAttribute(
			"tabindex",
			"-1",
		);
		await context.close();
	});
});
