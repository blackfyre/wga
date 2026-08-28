import { type Locator, type Page, expect, test } from "@playwright/test";

import {
	expectNoPageErrors,
	guardPageErrors,
	resetErrorCapture,
} from "./helpers/page-errors";

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

async function assertNoDuplicateIds(page: Page) {
	const duplicates = await page.evaluate(() => {
		const seen = new Set<string>();
		const repeated = new Set<string>();
		for (const element of document.querySelectorAll<HTMLElement>("[id]")) {
			if (seen.has(element.id)) {
				repeated.add(element.id);
			}
			seen.add(element.id);
		}
		return [...repeated];
	});
	expect(duplicates).toEqual([]);
}

test.describe("shared public shell", () => {
	test.beforeEach(async ({ page }) => {
		resetErrorCapture();
		guardPageErrors(page);
	});

	test.afterEach(() => {
		expectNoPageErrors();
	});

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
		const preferencesOpen = page.locator("[data-wga-preferences-open]");
		await tabTo(page, preferencesOpen);
		await page.keyboard.press("Enter");
		await expect(page.locator("#wga-preferences")).toHaveJSProperty(
			"open",
			true,
		);
		const dark = page.locator('[data-wga-scheme="dark"]');
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
		// Focus stays trapped inside the preferences dialog on Shift+Tab.
		await page.keyboard.press("Shift+Tab");
		const focusInsideDialog = await page.evaluate(
			() => document.activeElement?.closest("#wga-preferences") !== null,
		);
		expect(focusInsideDialog).toBe(true);
		// Escape closes the dialog and restores focus to the footer trigger.
		await page.keyboard.press("Escape");
		await expect(page.locator("#wga-preferences")).toHaveJSProperty(
			"open",
			false,
		);
		await expect(preferencesOpen).toBeFocused();
	});

	test("keeps Rams dark roles readable at every shell tier", async ({
		page,
	}) => {
		for (const width of shellWidths) {
			await page.setViewportSize({ width, height: 900 });
			await page.goto("/");
			await page.locator("[data-wga-preferences-open]").click();
			await page.locator('[data-wga-scheme="dark"]').click();
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

	test("keeps landmarks and controls reachable at 200 percent text", async ({
		page,
	}) => {
		const relevant = page.locator(
			"header, main, footer, header a[href], footer a[href], header button, footer button, header summary, footer summary",
		);

		await page.setViewportSize({ width: 390, height: 844 });
		await page.goto("/");
		await page.evaluate(() => {
			document.documentElement.style.fontSize = "2em";
		});
		expect(
			await page.evaluate(
				() => getComputedStyle(document.documentElement).fontSize,
			),
		).toBe("32px");
		await assertNoHorizontalOverflow(page);
		await assertInBounds(relevant, 390);
		await expect(page.locator("header")).toBeVisible();
		await expect(page.getByRole("main")).toBeVisible();
		await expect(page.locator("footer")).toBeVisible();
		await expect(
			page.getByRole("link", { name: "Artists" }).last(),
		).toBeVisible();
	});

	test("enlarges the computed root text at a true 400 percent", async ({
		page,
	}) => {
		const relevant = page.locator(
			"header, main, footer, header a[href], footer a[href], header button, footer button, header summary, footer summary",
		);

		await page.setViewportSize({ width: 1280, height: 900 });
		await page.goto("/");
		await page.evaluate(() => {
			document.documentElement.style.fontSize = "4em";
		});
		// The enlargement must be real: the computed root font-size is four
		// times the browser default, not merely a set style that no-ops.
		expect(
			await page.evaluate(
				() => getComputedStyle(document.documentElement).fontSize,
			),
		).toBe("64px");
		await assertNoHorizontalOverflow(page);
		await assertInBounds(relevant, 1280);
		await expect(page.locator("header")).toBeVisible();
		await expect(page.getByRole("main")).toBeVisible();
		await expect(page.locator("footer")).toBeVisible();
		await expect(
			page.getByRole("link", { name: "Artists" }).last(),
		).toBeVisible();
	});

	test("reflows without horizontal overflow at the effective 400 percent narrow viewport", async ({
		page,
	}) => {
		// 400% text at a 1280px reference viewport yields an effective 320px
		// content width (WCAG 1.4.10). Exercise that width directly so narrow
		// reflow is proven independently of root text scaling.
		const relevant = page.locator(
			"header, main, footer, header a[href], footer a[href], header button, footer button, header summary, footer summary",
		);

		await page.setViewportSize({ width: 320, height: 900 });
		await page.goto("/");
		await page.evaluate(() => {
			document.documentElement.style.fontSize = "4em";
		});
		await assertNoHorizontalOverflow(page);
		await assertInBounds(relevant, 320);
		await expect(page.locator("header")).toBeVisible();
		await expect(page.getByRole("main")).toBeVisible();
		await expect(page.locator("footer")).toBeVisible();
	});

	test("retains native disclosure and honest unavailable controls without JavaScript", async ({
		browser,
	}) => {
		const context = await browser.newContext({
			javaScriptEnabled: false,
			viewport: { width: 390, height: 844 },
		});
		const page = await context.newPage();
		guardPageErrors(page);
		await page.goto("/");
		const menu = page.locator("header details[data-kbd-mobile-navigation]");
		await expect(menu).not.toHaveAttribute("open", "");
		await menu.locator("summary[aria-label='Open primary navigation']").click();
		await expect(menu).toHaveAttribute("open", "");
		await expect(
			menu.getByRole("link", { name: "ARTISTS", exact: true }),
		).toBeVisible();
		await expect(page.locator("[data-wga-preferences-control]")).toBeHidden();
		await expect(page.locator("[data-wga-preferences-open]")).toBeHidden();
		await expect(page.locator("[data-wga-cookie-settings]")).toBeHidden();
		await expect(page.locator("[data-wga-cookie-settings]")).toHaveAttribute(
			"tabindex",
			"-1",
		);
		await context.close();
	});

	test("mounts one main area, unique ids, and an honest tray/toast stack", async ({
		page,
	}) => {
		await page.setViewportSize({ width: 390, height: 844 });
		await page.goto("/");

		await expect(page.locator("#mc-area")).toHaveCount(1);
		await assertNoDuplicateIds(page);

		await expect(page.locator("#itinerary-tray")).toHaveCount(1);
		await expect(page.locator("#toast-container")).toHaveCount(1);

		// With an empty draft the tray bar is absent and the fixed toast
		// container keeps its base class without a tray-clearance offset.
		await expect(page.locator("#itinerary-tray [role='region']")).toHaveCount(
			0,
		);
		await expect(page.locator("#toast-container")).toHaveClass(/toast/);
		await expect(page.locator("#toast-container")).not.toHaveClass(/bottom-28/);
		await expect(page.locator("#mc-area")).not.toHaveClass(/pb-28/);

		// Public toast notifications stack in the shared container in order.
		await page.evaluate(() => {
			document.body.dispatchEvent(
				new CustomEvent("notification:toast", {
					detail: { closeDialog: false, message: "First", type: "info" },
				}),
			);
			document.body.dispatchEvent(
				new CustomEvent("notification:toast", {
					detail: { closeDialog: false, message: "Second", type: "success" },
				}),
			);
		});
		const toasts = page.locator("#toast-container [role='alert']");
		await expect(toasts).toHaveCount(2);
		await expect(toasts.nth(0)).toContainText("First");
		await expect(toasts.nth(1)).toContainText("Second");
	});

	test("reserves layout and stacks toasts above the active itinerary tray", async ({
		page,
	}) => {
		await page.setViewportSize({ width: 390, height: 844 });
		await page.goto("/inspire");

		// A deterministic synthetic-work add drives the real HTMX add workflow,
		// producing both an active tray and a real toast from the response.
		const add = page
			.locator("section[aria-label='Shuffled artworks'] article")
			.first()
			.getByRole("button", { name: "ADD TO AN ITINERARY +" });
		await add.scrollIntoViewIfNeeded();
		await add.click();

		const tray = page.locator("#itinerary-tray");
		await expect(tray.locator("[role='region']")).toHaveCount(1);
		await expect(tray).toContainText("ITINERARY DRAFT · 1 OF 15");

		// The non-empty tray reserves bottom clearance on both the main area
		// and the fixed toast container, mirroring the server-side offsets.
		await expect(page.locator("#mc-area")).toHaveClass(/pb-28/);
		await expect(page.locator("#toast-container")).toHaveClass(/bottom-28/);

		const toasts = page.locator("#toast-container [role='alert']");
		await expect(toasts).toHaveCount(1);
		await expect(toasts).toContainText("Added to your itinerary.");

		// A second toast stacks in order above the first, inside the tray-lifted
		// container.
		await page.evaluate(() => {
			document.body.dispatchEvent(
				new CustomEvent("notification:toast", {
					detail: { closeDialog: false, message: "Second", type: "success" },
				}),
			);
		});
		await expect(toasts).toHaveCount(2);
		await expect(toasts.nth(0)).toContainText("Added to your itinerary.");
		await expect(toasts.nth(1)).toContainText("Second");

		// Geometry: every stacked toast sits fully above the fixed tray bar,
		// never rendering behind it.
		const trayBox = await tray.locator("[role='region']").boundingBox();
		expect(trayBox).not.toBeNull();
		if (trayBox) {
			for (const toast of await toasts.all()) {
				const box = await toast.boundingBox();
				expect(box).not.toBeNull();
				if (box) {
					expect(box.y + box.height).toBeLessThanOrEqual(trayBox.y + 1);
				}
			}
		}
	});
});
