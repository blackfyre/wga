import { expect, test } from "@playwright/test";

import {
	expectNoPageErrors,
	guardPageErrors,
	resetErrorCapture,
} from "./helpers/page-errors";

test.beforeEach(async ({ page }) => {
	resetErrorCapture();
	guardPageErrors(page);
});

test.afterEach(() => {
	expectNoPageErrors();
});

test("desktop navigation opens the catalogue", async ({ page }) => {
	await page.goto("/");
	await page
		.getByRole("navigation", { name: "Primary navigation" })
		.getByRole("link", { name: "ARTWORKS", exact: true })
		.click();

	await expect(page).toHaveURL(/\/artworks$/);
	await expect(page.getByRole("heading", { name: "Artworks" })).toBeVisible();
});

test("footer links navigate through real HTMX requests", async ({ page }) => {
	await page.goto("/");
	const htmxRequests: string[] = [];
	page.on("request", (request) => {
		if (request.headers()["hx-request"] === "true") {
			htmxRequests.push(request.url());
		}
	});

	// A footer link carries an ordinary href plus hx-get; clicking it must be
	// a genuine HTMX swap of #mc-area, not a synthetic lifecycle event or a
	// full document reload.
	await page.locator("footer a[href='/artists']").click();

	await expect(page).toHaveURL(/\/artists$/);
	await expect(page.getByRole("heading", { name: "Artists" })).toBeVisible();
	expect(htmxRequests.some((url) => new URL(url).pathname === "/artists")).toBe(
		true,
	);
});

test("desktop navigation underline sits on the navigation edge", async ({
	page,
}) => {
	await page.goto("/");
	const navigation = page.locator(
		"header > nav[aria-label='Primary navigation']",
	);
	const artworkLink = navigation.getByRole("link", {
		name: "ARTWORKS",
		exact: true,
	});

	await artworkLink.hover();
	await expect(artworkLink).toHaveCSS("border-bottom-color", "rgb(0, 51, 102)");

	const [navigationBottom, linkBottom] = await Promise.all([
		navigation.evaluate((element) => element.getBoundingClientRect().bottom),
		artworkLink.evaluate((element) => element.getBoundingClientRect().bottom),
	]);
	expect(Math.abs(navigationBottom - linkBottom)).toBeLessThanOrEqual(1);
});

test("navigation matches the prototype type scale", async ({ page }) => {
	await page.goto("/");
	const artworkLink = page
		.locator("header > nav[aria-label='Primary navigation']")
		.getByRole("link", { name: "ARTWORKS", exact: true });

	await expect(artworkLink).toHaveCSS("font-size", "12px");
	await expect(artworkLink).toHaveCSS("letter-spacing", "1.5px");
	await expect(page.locator("header a[href='/'] span span").last()).toHaveCSS(
		"font-size",
		"10px",
	);
});

test("desktop header matches the prototype baseline", async ({ page }) => {
	await page.setViewportSize({ width: 1440, height: 900 });
	await page.goto("/");

	const bottom = await page
		.locator("header")
		.evaluate((element) => element.getBoundingClientRect().bottom);
	expect(Math.abs(bottom - 148)).toBeLessThanOrEqual(1);
});

test("desktop header controls share the search baseline", async ({ page }) => {
	await page.setViewportSize({ width: 1440, height: 900 });
	await page.goto("/");

	const desktopControls = page.locator("header > div > div.hidden");
	const controls = [
		desktopControls.locator("form button[type='submit']"),
		desktopControls.locator("[data-keyboard-open]"),
		desktopControls.locator("[data-keyboard-help]"),
	];
	const bottoms = await Promise.all(
		controls.map((control) =>
			control.evaluate((element) => element.getBoundingClientRect().bottom),
		),
	);
	expect(Math.max(...bottoms) - Math.min(...bottoms)).toBeLessThanOrEqual(1);
});

test("header search submits a global query and reports an honest empty result", async ({
	page,
}) => {
	await page.goto("/");
	const query = "no-such-record-zzz-0000";
	await page.getByRole("searchbox", { name: "Search collection" }).fill(query);
	await page.getByRole("button", { name: "SEARCH", exact: true }).click();

	await expect(page).toHaveURL(/\/search\?q=no-such-record-zzz-0000/);
	await expect(page.locator("#global-search-results")).toBeVisible();
	await expect(page.locator("#global-search-results")).toContainText("ARTISTS");
	await expect(page.locator("#global-search-results")).toContainText("WORKS");
	await expect(page.locator("#global-search-results")).toContainText(
		"No artist matches that.",
	);
	await expect(page.locator("#global-search-results")).toContainText(
		"No work matches that.",
	);
});

test("navigation follows the available reference destinations", async ({
	page,
}) => {
	await page.goto("/");
	const navigation = page.locator(
		"header > nav[aria-label='Primary navigation']",
	);
	const more = navigation.locator("details[aria-label='More destinations']");
	await more.locator("summary").click();

	await expect(more.getByRole("link", { name: "POSTCARDS" })).toHaveAttribute(
		"href",
		"/postcard",
	);
	await expect(more.getByRole("link", { name: "ABOUT" })).toHaveAttribute(
		"href",
		"/pages/about",
	);
});

test("mobile navigation opens by keyboard", async ({ page }) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await page.goto("/");

	const menu = page.locator("header details[data-kbd-mobile-navigation]");
	const summary = menu.locator("summary[aria-label='Open primary navigation']");
	await summary.focus();
	await expect(summary).toBeFocused();
	await summary.press("Enter");
	await expect(
		page.getByRole("navigation", { name: "Primary navigation" }),
	).toBeVisible();
});

test("mobile toggle exposes a 44px touch target", async ({ page }) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await page.goto("/");

	const summary = page.locator(
		"header details[data-kbd-mobile-navigation] summary[aria-label='Open primary navigation']",
	);
	const box = await summary.boundingBox();
	expect(box).not.toBeNull();
	if (box) {
		expect(box.width).toBeGreaterThanOrEqual(44);
		expect(box.height).toBeGreaterThanOrEqual(44);
	}
});

for (const viewport of [
	{ width: 375, height: 812 },
	{ width: 390, height: 844 },
]) {
	test(`mobile disclosure keeps the logo fixed and paints a full-width opaque panel at ${viewport.width}px`, async ({
		page,
	}) => {
		await page.setViewportSize(viewport);
		await page.goto("/");

		const menu = page.locator("header details[data-kbd-mobile-navigation]");
		const summary = menu.locator(
			"summary[aria-label='Open primary navigation']",
		);
		const logo = page.locator("header a[href='/'][class*='col-start-1']");
		const nav = menu.locator("nav[data-mobile-navigation]");

		const closedLogoTop = await logo.evaluate(
			(element) => element.getBoundingClientRect().top,
		);

		await summary.click();
		await expect(menu).toHaveAttribute("open", "");

		const openLogoTop = await logo.evaluate(
			(element) => element.getBoundingClientRect().top,
		);
		expect(Math.abs(openLogoTop - closedLogoTop)).toBeLessThanOrEqual(1);

		const logoBox = await logo.boundingBox();
		const summaryBox = await summary.boundingBox();
		const navBox = await nav.boundingBox();
		expect(logoBox).not.toBeNull();
		expect(summaryBox).not.toBeNull();
		expect(navBox).not.toBeNull();
		if (!logoBox || !summaryBox || !navBox) {
			return;
		}

		// panel begins below the logo and toggle row
		expect(navBox.y).toBeGreaterThanOrEqual(logoBox.y + logoBox.height);
		expect(navBox.y).toBeGreaterThanOrEqual(summaryBox.y + summaryBox.height);

		// panel spans the full header content width (logo left edge to toggle right edge)
		expect(Math.abs(navBox.x - logoBox.x)).toBeLessThanOrEqual(1);
		const navRight = navBox.x + navBox.width;
		const gridRight = summaryBox.x + summaryBox.width;
		expect(Math.abs(navRight - gridRight)).toBeLessThanOrEqual(1);

		// panel background is opaque
		const backgroundColor = await nav.evaluate(
			(element) => getComputedStyle(element).backgroundColor,
		);
		expect(backgroundColorAlpha(backgroundColor)).toBe(1);

		// logo and toggle do not intersect (the verifier failing case)
		expect(rectsIntersect(logoBox, summaryBox)).toBe(false);

		// logo and panel do not intersect
		expect(rectsIntersect(logoBox, navBox)).toBe(false);

		// a representative point in the panel resolves to panel content, not the logo
		const probe = await page.evaluate(
			({ x, y }) => {
				const element = document.elementFromPoint(x, y);
				return {
					inPanel: !!element?.closest("[data-mobile-navigation]"),
					inLogo: !!element?.closest("header a[href='/']"),
				};
			},
			{ x: navBox.x + 24, y: navBox.y + 24 },
		);
		expect(probe.inPanel).toBe(true);
		expect(probe.inLogo).toBe(false);

		// complete mobile brand is present, visible, and not clipped/ellipsised
		const wgaMark = logo.locator("span").first();
		const title = logo.locator("span").nth(1).locator("span").nth(0);
		const tagline = logo.locator("span").nth(1).locator("span").nth(1);
		await expect(wgaMark).toBeVisible();
		await expect(wgaMark).toHaveText("WGA");
		await expect(title).toBeVisible();
		await expect(title).toHaveText("WEB GALLERY OF ART");
		await expect(tagline).toBeVisible();
		await expect(tagline).toHaveText("EUROPEAN ART, 3RD CENTURY – EARLY 20TH");
		for (const brand of [title, tagline]) {
			const metrics = await brand.evaluate((el) => ({
				scrollWidth: el.scrollWidth,
				clientWidth: el.clientWidth,
				scrollHeight: el.scrollHeight,
				clientHeight: el.clientHeight,
				textOverflow: getComputedStyle(el).textOverflow,
			}));
			expect(metrics.scrollWidth).toBeLessThanOrEqual(metrics.clientWidth);
			expect(metrics.scrollHeight).toBeLessThanOrEqual(metrics.clientHeight);
			expect(metrics.textOverflow).not.toBe("ellipsis");
		}
	});
}

for (const viewport of [
	{ width: 375, height: 812 },
	{ width: 390, height: 844 },
]) {
	test(`mobile toggle hit-tests across its full surface and toggles at ${viewport.width}px`, async ({
		page,
	}) => {
		await page.setViewportSize(viewport);
		await page.goto("/");

		const menu = page.locator("header details[data-kbd-mobile-navigation]");
		const summary = menu.locator(
			"summary[aria-label='Open primary navigation']",
		);
		const box = await summary.boundingBox();
		expect(box).not.toBeNull();
		if (!box) {
			return;
		}

		// a grid of probes across edges, corners and centre resolves to the summary
		const probes = [
			{ dx: 2, dy: 2 },
			{ dx: box.width - 2, dy: 2 },
			{ dx: 2, dy: box.height - 2 },
			{ dx: box.width - 2, dy: box.height - 2 },
			{ dx: box.width / 2, dy: 2 },
			{ dx: box.width / 2, dy: box.height - 2 },
			{ dx: 2, dy: box.height / 2 },
			{ dx: box.width - 2, dy: box.height / 2 },
			{ dx: box.width / 2, dy: box.height / 2 },
		];
		for (const probe of probes) {
			const hitsSummary = await page.evaluate(
				({ x, y }) => {
					const element = document.elementFromPoint(x, y);
					return !!element?.closest(
						"summary[aria-label='Open primary navigation']",
					);
				},
				{ x: box.x + probe.dx, y: box.y + probe.dy },
			);
			expect(hitsSummary).toBe(true);
		}

		// tapping toggles open then closed at left edge, centre, and right edge
		const togglePoints = [
			{ dx: 2, dy: box.height / 2 },
			{ dx: box.width / 2, dy: box.height / 2 },
			{ dx: box.width - 2, dy: box.height / 2 },
		];
		for (const point of togglePoints) {
			await page.mouse.click(box.x + point.dx, box.y + point.dy);
			await expect(menu).toHaveAttribute("open", "");
			await page.mouse.click(box.x + point.dx, box.y + point.dy);
			await expect(menu).not.toHaveAttribute("open", "");
		}
	});
}

test("mobile navigation closes after following a link", async ({ page }) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await page.goto("/");
	const menu = page.locator("header details[data-kbd-mobile-navigation]");

	await menu.locator("summary[aria-label='Open primary navigation']").click();
	await expect(menu).toHaveAttribute("open", "");
	await page
		.getByRole("navigation", { name: "Primary navigation" })
		.getByRole("link", { name: "ARTISTS", exact: true })
		.click();

	await expect(page).toHaveURL(/\/artists$/);
	await expect(menu).not.toHaveAttribute("open", "");
	const activeLink = page.locator(
		"[data-mobile-navigation] a[href='/artists']",
	);
	await expect(activeLink).toHaveAttribute("aria-current", "page");
	await expect(activeLink).toHaveCSS("background-color", "rgb(0, 51, 102)");
	await expect(activeLink).toHaveCSS("padding-left", "12px");
});

test("mobile navigation stays open while using global search", async ({
	page,
}) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await page.goto("/");
	const menu = page.locator("header details[data-kbd-mobile-navigation]");

	await menu.locator("summary[aria-label='Open primary navigation']").click();
	await menu.getByRole("searchbox", { name: "Search collection" }).click();

	await expect(menu).toHaveAttribute("open", "");
});

test("mobile navigation highlights the current page when opened", async ({
	page,
}) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await page.goto("/artists");
	await page
		.locator("header details[data-kbd-mobile-navigation]")
		.locator("summary[aria-label='Open primary navigation']")
		.click();

	const activeLink = page.locator(
		"[data-mobile-navigation] a[href='/artists']",
	);
	await expect(activeLink).toHaveAttribute("aria-current", "page");
	await expect(activeLink).toHaveCSS("background-color", "rgb(0, 51, 102)");
});

test("reduced motion removes page-enter animation delay", async ({ page }) => {
	await page.emulateMedia({ reducedMotion: "reduce" });
	await page.goto("/");

	await expect(page.locator("#mc-area")).toHaveCSS(
		"animation-duration",
		"1e-05s",
	);
});

test("navigation works without JavaScript", async ({ browser }) => {
	const context = await browser.newContext({ javaScriptEnabled: false });
	const page = await context.newPage();
	guardPageErrors(page);
	await page.goto("/");
	await page
		.getByRole("navigation", { name: "Primary navigation" })
		.getByRole("link", { name: "ARTISTS", exact: true })
		.click();

	await expect(page).toHaveURL(/\/artists$/);
	await expect(page.getByRole("heading", { name: "Artists" })).toBeVisible();
	await context.close();
});

test("footer follows the public site map", async ({ page }) => {
	await page.goto("/");
	const footer = page.locator("footer");

	await expect(footer.getByText("BROWSE", { exact: true })).toBeVisible();
	await expect(footer.getByText("SERVICES", { exact: true })).toBeVisible();
	await expect(footer.getByText("ABOUT", { exact: true })).toBeVisible();
	await expect(footer.getByRole("link", { name: "Artists" })).toHaveAttribute(
		"href",
		"/artists",
	);
	await expect(footer.getByRole("link", { name: "Postcards" })).toHaveAttribute(
		"href",
		"/postcard",
	);
	await expect(footer.getByText("Period music", { exact: true })).toBeVisible();
	await expect(footer.getByRole("link", { name: "Period music" })).toHaveCount(
		0,
	);
	await expect(page.getByRole("checkbox", { name: "Dark mode" })).toHaveCount(
		0,
	);
});

const mobileInventory = [
	["ARTISTS", "/artists"],
	["ARTWORKS", "/artworks"],
	["DUAL MODE", "/dual-mode"],
	["TIMELINE", "/timeline"],
	["INSPIRATION", "/inspire"],
	["GUIDED TOURS", "/tours"],
	["ITINERARIES", "/itineraries"],
	["BUILD AN ITINERARY", "/itineraries/new"],
	["STATISTICS", "/statistics"],
	["GLOSSARY", "/glossary"],
	["GUESTBOOK", "/guestbook"],
	["POSTCARDS", "/postcard"],
	["ABOUT", "/pages/about"],
	["CONTRIBUTORS", "/contributors"],
	["PRIVACY POLICY", "/pages/privacy-policy"],
] as const;

test("mobile navigation renders the flat prototype inventory in order", async ({
	page,
}) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await page.goto("/");
	const menu = page.locator("header details[data-kbd-mobile-navigation]");
	await menu.locator("summary[aria-label='Open primary navigation']").click();

	const links = menu.locator("[data-mobile-navigation] a");
	await expect(links).toHaveCount(mobileInventory.length);
	for (let index = 0; index < mobileInventory.length; index += 1) {
		const link = links.nth(index);
		await expect(link).toHaveAttribute("href", mobileInventory[index][1]);
		await expect(link.locator("span").first()).toHaveText(
			mobileInventory[index][0],
		);
		await expect(link.locator("span[aria-hidden='true']")).toHaveText("→");
	}
	await expect(menu.locator("details")).toHaveCount(0);
});

test("mobile navigation marks the most specific current route", async ({
	page,
}) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await page.goto("/itineraries/new");
	const menu = page.locator("header details[data-kbd-mobile-navigation]");
	await menu.locator("summary[aria-label='Open primary navigation']").click();

	const build = menu.locator("a[href='/itineraries/new']");
	const itineraries = menu.locator("a[href='/itineraries']");
	await expect(build).toHaveAttribute("aria-current", "page");
	await expect(itineraries).not.toHaveAttribute("aria-current", "page");

	await page.goto("/itineraries");
	await menu.locator("summary[aria-label='Open primary navigation']").click();
	await expect(menu.locator("a[href='/itineraries']")).toHaveAttribute(
		"aria-current",
		"page",
	);
	await expect(menu.locator("a[href='/itineraries/new']")).not.toHaveAttribute(
		"aria-current",
		"page",
	);
});

test("Escape closes the mobile disclosure and restores focus to its summary", async ({
	page,
}) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await page.goto("/");
	await page.waitForFunction(
		() => document.documentElement.dataset.keyboardNavigationReady === "true",
	);
	const menu = page.locator("header details[data-kbd-mobile-navigation]");
	const summary = menu.locator("summary[aria-label='Open primary navigation']");

	await page.keyboard.press("/");
	await expect(menu).toHaveAttribute("open", "");
	await expect(menu.locator("[data-kbd-search]")).toBeFocused();

	await page.keyboard.press("Escape");
	await expect(menu).not.toHaveAttribute("open", "");
	await expect(summary).toBeFocused();
});

test("open mobile disclosure closes at the 45rem boundary and moves focus to desktop search", async ({
	page,
}) => {
	await page.setViewportSize({ width: 719, height: 844 });
	await page.goto("/");
	await page.waitForFunction(
		() => document.documentElement.dataset.keyboardNavigationReady === "true",
	);
	const menu = page.locator("header details[data-kbd-mobile-navigation]");
	await menu.locator("summary[aria-label='Open primary navigation']").click();
	await expect(menu).toHaveAttribute("open", "");
	await menu.locator("[data-kbd-search]").focus();

	await page.setViewportSize({ width: 720, height: 900 });
	await expect(menu).not.toHaveAttribute("open", "");
	await expect(
		page.locator("header [data-kbd-search]").filter({ visible: true }),
	).toBeFocused();
	await expect(
		page.locator("header > nav[aria-label='Primary navigation']"),
	).toBeVisible();
	await expect(
		page
			.locator("header > nav[aria-label='Primary navigation']")
			.getByText("MORE", { exact: true }),
	).toBeVisible();
});

test("mobile disclosure toggles by touch and closes on link tap", async ({
	browser,
}) => {
	const context = await browser.newContext({
		viewport: { width: 390, height: 844 },
		hasTouch: true,
	});
	const page = await context.newPage();
	guardPageErrors(page);
	await page.goto("/");
	const menu = page.locator("header details[data-kbd-mobile-navigation]");
	await menu.locator("summary[aria-label='Open primary navigation']").tap();
	await expect(menu).toHaveAttribute("open", "");

	await menu.getByRole("link", { name: "ARTISTS", exact: true }).tap();
	await expect(page).toHaveURL(/\/artists$/);
	await expect(menu).not.toHaveAttribute("open", "");
	await context.close();
});

test("open mobile navigation stays in bounds in dark mode", async ({
	page,
}) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await page.goto("/");
	await page.locator("[data-wga-preferences-open]").click();
	await page.locator('[data-wga-scheme="dark"]').click();
	await expect(page.locator("html")).toHaveAttribute(
		"data-theme",
		"wga-rams-dark",
	);
	await page.keyboard.press("Escape");
	const menu = page.locator("header details[data-kbd-mobile-navigation]");
	await menu.locator("summary[aria-label='Open primary navigation']").click();
	await expect(menu).toHaveAttribute("open", "");

	const dimensions = await page.evaluate(() => ({
		clientWidth: document.documentElement.clientWidth,
		scrollWidth: document.documentElement.scrollWidth,
	}));
	expect(dimensions.scrollWidth).toBeLessThanOrEqual(dimensions.clientWidth);
	for (const link of await menu.locator("[data-mobile-navigation] a").all()) {
		const box = await link.boundingBox();
		expect(box).not.toBeNull();
		if (box) {
			expect(box.x).toBeGreaterThanOrEqual(0);
			expect(box.x + box.width).toBeLessThanOrEqual(390);
		}
	}
});

test("mobile navigation works without JavaScript", async ({ browser }) => {
	const context = await browser.newContext({
		javaScriptEnabled: false,
		viewport: { width: 390, height: 844 },
	});
	const page = await context.newPage();
	guardPageErrors(page);
	await page.goto("/");
	const menu = page.locator("header details[data-kbd-mobile-navigation]");
	await menu.locator("summary[aria-label='Open primary navigation']").click();
	await expect(menu).toHaveAttribute("open", "");
	await expect(
		menu.getByRole("link", { name: "CONTRIBUTORS" }),
	).toHaveAttribute("href", "/contributors");
	await menu.getByRole("link", { name: "ARTISTS", exact: true }).click();
	await expect(page).toHaveURL(/\/artists$/);
	await expect(page.getByRole("heading", { name: "Artists" })).toBeVisible();
	await context.close();
});

function backgroundColorAlpha(color: string): number {
	const match = color.match(/rgba?\(([^)]+)\)/);
	if (!match) {
		return 1;
	}
	const parts = match[1].split(",").map((part) => Number(part.trim()));
	return parts.length >= 4 ? parts[3] : 1;
}

function rectsIntersect(
	a: { x: number; y: number; width: number; height: number },
	b: { x: number; y: number; width: number; height: number },
): boolean {
	return !(
		a.x + a.width <= b.x ||
		b.x + b.width <= a.x ||
		a.y + a.height <= b.y ||
		b.y + b.height <= a.y
	);
}
