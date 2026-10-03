import { expect, type Page, test } from "@playwright/test";

const KBD_BAR = 30;
const ITINERARY = 72;
const BOARD = 48;
const widths = [390, 834, 1440];
const floatingGap = (width: number) => (width >= 720 ? 24 : 16);

const stackHeight = (page: Page) =>
	page.evaluate(() =>
		Number.parseFloat(
			getComputedStyle(document.body).getPropertyValue(
				"--wga-bottom-stack-height",
			),
		),
	);

// Fake trays carry the same measured-stack contract as the real ones, so the
// geometry can be checked for every combination without driving the workflows.
const dockTrays = (page: Page, trays: { itinerary: boolean; board: boolean }) =>
	page.evaluate(
		({ trays, heights }) => {
			document.querySelector("#cc-main")?.remove();
			const add = (name: string, order: string, height: number) => {
				const tray = document.createElement("div");
				tray.className = "wga-bottom-stack-item";
				tray.dataset.wgaBottomStackItem = name;
				tray.dataset.wgaBottomStackOrder = order;
				tray.style.cssText = `position:fixed;inset-inline:0;height:${height}px;background:black`;
				document.body.append(tray);
			};
			if (trays.itinerary) add("itinerary", "10", heights.itinerary);
			if (trays.board) add("study-board", "20", heights.board);
		},
		{ trays, heights: { itinerary: ITINERARY, board: BOARD } },
	);

// Distance from the viewport's bottom edge to the top of the highest docked
// surface, and from that top to the bottom edges of FEEDBACK and the toasts.
const floatingGeometry = (page: Page) =>
	page.evaluate(() => {
		const docked = [
			...document.querySelectorAll<HTMLElement>("[data-wga-bottom-stack-item]"),
		].filter((item) => item.getClientRects().length > 0);
		const dockTop = Math.min(
			innerHeight,
			...docked.map((item) => item.getBoundingClientRect().top),
		);
		const feedback = document.querySelector<HTMLElement>(
			"a.wga-feedback-anchor",
		);
		const toast = document.querySelector<HTMLElement>("#toast-container");
		if (!feedback || !toast) {
			throw new Error("floating controls are missing");
		}
		return {
			dockHeight: innerHeight - dockTop,
			feedbackGap: dockTop - feedback.getBoundingClientRect().bottom,
			toastGap: dockTop - toast.getBoundingClientRect().bottom,
		};
	});

const trayCases = [
	{ name: "no tray", itinerary: false, board: false },
	{ name: "the itinerary tray", itinerary: true, board: false },
	{ name: "the board tray", itinerary: false, board: true },
	{ name: "both trays", itinerary: true, board: true },
];

for (const pointer of ["desktop", "touch"] as const) {
	test.describe(`${pointer} pointer`, () => {
		if (pointer === "touch") {
			test.use({ hasTouch: true, isMobile: true });
		}
		const bar = pointer === "desktop" ? KBD_BAR : 0;

		for (const width of widths) {
			test(`shows the keyboard bar only with a desktop pointer at ${width}px`, async ({
				page,
			}) => {
				await page.setViewportSize({ width, height: 844 });
				await page.goto("/");
				const keyboardBar = page.locator(".wga-kbd-bar");
				if (pointer === "desktop") {
					await expect(keyboardBar).toBeVisible();
					await expect(keyboardBar).toHaveCSS("height", `${KBD_BAR}px`);
					const bottom = await keyboardBar.evaluate(
						(element) => innerHeight - element.getBoundingClientRect().bottom,
					);
					expect(bottom).toBeCloseTo(0, 0);
				} else {
					await expect(keyboardBar).toBeHidden();
				}
				await expect.poll(() => stackHeight(page)).toBeCloseTo(bar, 0);
			});

			for (const trays of trayCases) {
				test(`clears ${trays.name} by the floating gap at ${width}px`, async ({
					page,
				}) => {
					await page.setViewportSize({ width, height: 844 });
					await page.goto("/");
					await dockTrays(page, trays);
					const docked =
						bar + (trays.itinerary ? ITINERARY : 0) + (trays.board ? BOARD : 0);
					await expect.poll(() => stackHeight(page)).toBeCloseTo(docked, 0);

					await page.evaluate(() =>
						document.body.dispatchEvent(
							new CustomEvent("notification:toast", {
								detail: { closeDialog: false, message: "Saved", type: "info" },
							}),
						),
					);
					await expect(
						page.locator("#toast-container [role='alert']"),
					).toHaveCount(1);

					const geometry = await floatingGeometry(page);
					expect(geometry.dockHeight).toBeCloseTo(docked, 0);
					expect(geometry.feedbackGap).toBeCloseTo(floatingGap(width), 0);
					expect(geometry.toastGap).toBeCloseTo(floatingGap(width), 0);

					// The final page content stays reachable above every docked surface.
					await page.evaluate(() =>
						window.scrollTo({
							top: document.body.scrollHeight,
							behavior: "instant",
						}),
					);
					const reach = await page.evaluate(() => {
						const last = document.querySelector<HTMLElement>(
							"[data-wga-back-to-top]",
						);
						if (!last) {
							throw new Error("footer return link is missing");
						}
						return innerHeight - last.getBoundingClientRect().bottom;
					});
					expect(reach).toBeGreaterThanOrEqual(docked);
				});
			}
		}
	});
}

test("stops measuring once the permanent bar and a tray have settled", async ({
	page,
}) => {
	await page.addInitScript(() => {
		const win = window as unknown as { stackWrites: number };
		win.stackWrites = 0;
		const setProperty = CSSStyleDeclaration.prototype.setProperty;
		CSSStyleDeclaration.prototype.setProperty = function (name, ...rest) {
			if (name === "--wga-bottom-stack-height") {
				win.stackWrites += 1;
			}
			return setProperty.call(this, name, ...rest);
		};
	});
	await page.setViewportSize({ width: 1440, height: 844 });
	await page.goto("/");
	await dockTrays(page, { itinerary: true, board: false });
	await expect
		.poll(() => stackHeight(page))
		.toBeCloseTo(KBD_BAR + ITINERARY, 0);
	await page.waitForTimeout(300);

	const writes = () =>
		page.evaluate(
			() => (window as unknown as { stackWrites: number }).stackWrites,
		);
	const settled = await writes();
	await page.waitForTimeout(1000);
	expect(await writes()).toBe(settled);
});

for (const pointer of ["desktop", "touch"] as const) {
	test.describe(`${pointer} pre-measurement tray offset`, () => {
		if (pointer === "touch") {
			test.use({ hasTouch: true, isMobile: true });
		}

		test("places an unmeasured tray above the keyboard bar", async ({
			page,
		}) => {
			await page.setViewportSize({ width: 1440, height: 844 });
			await page.goto("/");
			// Without the measurement attribute the script never sets an offset,
			// which is the tray's state before the first frame or without scripts.
			const bottom = await page.evaluate(() => {
				const tray = document.createElement("div");
				tray.className = "wga-bottom-stack-item";
				tray.style.cssText = "position:fixed;inset-inline:0;height:72px";
				document.body.append(tray);
				return getComputedStyle(tray).bottom;
			});
			expect(bottom).toBe(pointer === "desktop" ? `${KBD_BAR}px` : "0px");
		});
	});
}

for (const width of widths) {
	test(`measures and coordinates fixed bottom surfaces at ${width}px`, async ({
		page,
	}) => {
		await page.setViewportSize({ width, height: 844 });
		await page.goto("/");
		await page.evaluate(() => {
			document.querySelector("#cc-main")?.remove();
			const tray = document.createElement("div");
			tray.className = "wga-bottom-stack-item";
			tray.dataset.wgaBottomStackItem = "itinerary";
			tray.dataset.wgaBottomStackOrder = "10";
			tray.style.cssText =
				"position:fixed;inset-inline:0;height:72px;background:black";
			document.body.append(tray);

			const consent = document.createElement("div");
			consent.id = "cc-main";
			consent.innerHTML =
				'<div class="cm cm--bottom" style="display:block!important;visibility:visible;transform:none!important;opacity:1!important;position:fixed;height:120px;width:280px;background:white"></div>';
			document.body.append(consent);
		});
		const noticeOnlyHeight = KBD_BAR + 136;

		await expect
			.poll(() => stackHeight(page))
			.toBeCloseTo(noticeOnlyHeight + 72, 0);
		const geometry = await page.evaluate(() => {
			const tray = document.querySelector<HTMLElement>(
				'[data-wga-bottom-stack-item="itinerary"]',
			);
			const notice = document.querySelector<HTMLElement>("#cc-main .cm");
			const toast = document.querySelector<HTMLElement>("#toast-container");
			if (!toast) {
				throw new Error("shared bottom-stack mounts are missing");
			}
			return {
				trayBottom: tray
					? innerHeight - tray.getBoundingClientRect().bottom
					: -1,
				noticeBottom: notice
					? innerHeight - notice.getBoundingClientRect().bottom
					: -1,
				reservedPadding: Number.parseFloat(
					getComputedStyle(document.body).paddingBottom,
				),
				toastBottom: Number.parseFloat(getComputedStyle(toast).bottom),
			};
		});
		expect(geometry.trayBottom).toBeCloseTo(KBD_BAR, 0);
		expect(geometry.noticeBottom).toBeCloseTo(KBD_BAR + 88, 0);
		expect(geometry.reservedPadding).toBeCloseTo(noticeOnlyHeight + 72, 0);
		expect(geometry.toastBottom).toBeCloseTo(
			noticeOnlyHeight + 72 + floatingGap(width),
			0,
		);

		await page.evaluate(() =>
			document
				.querySelector('[data-wga-bottom-stack-item="itinerary"]')
				?.remove(),
		);
		await expect.poll(() => stackHeight(page)).toBeCloseTo(noticeOnlyHeight, 0);
	});
}

test("footer exposes labelled ordinary community links without JavaScript", async ({
	browser,
}) => {
	const context = await browser.newContext({ javaScriptEnabled: false });
	const page = await context.newPage();
	await page.goto("/");
	for (const link of [
		{
			name: "WGA on GitHub (opens in a new tab)",
			href: "https://github.com/blackfyre/wga/",
		},
		{
			name: "Web Gallery of Art on Mastodon (opens in a new tab)",
			href: "https://mastodon.social/@webgalleryofart",
		},
	]) {
		const destination = page.getByRole("link", { name: link.name });
		await expect(destination).toHaveAttribute("href", link.href);
		await expect(destination).toHaveAttribute("target", "_blank");
		await expect(destination).toHaveAttribute("rel", "noopener");
	}
	await context.close();
});

// The library inserts the real notice hidden and reveals it a moment later by
// toggling classes, so this checks the measurement that follows the reveal
// rather than a pre-shown stand-in.
test.describe("revealed cookie notice", () => {
	test.beforeEach(async ({ context, page }) => {
		await context.clearCookies();
		await page.addInitScript(() => {
			Object.defineProperty(navigator, "webdriver", { get: () => false });
			const win = window as unknown as { stackWrites: number };
			win.stackWrites = 0;
			const setProperty = CSSStyleDeclaration.prototype.setProperty;
			CSSStyleDeclaration.prototype.setProperty = function (name, ...rest) {
				if (name === "--wga-bottom-stack-height") {
					win.stackWrites += 1;
				}
				return setProperty.call(this, name, ...rest);
			};
		});
	});

	test("clears the keyboard bar once CookieConsent reveals it", async ({
		page,
	}) => {
		await page.setViewportSize({ width: 1440, height: 900 });
		await page.goto("/");
		const notice = page.locator("#cc-main .cm");
		await expect(notice).toBeVisible();
		const noticeBottom = () =>
			notice.evaluate(
				(element) => innerHeight - element.getBoundingClientRect().bottom,
			);
		await expect.poll(noticeBottom).toBeCloseTo(KBD_BAR + 16, 0);
		const noticeHeight = await notice.evaluate(
			(element) => element.getBoundingClientRect().height,
		);
		await expect
			.poll(() => stackHeight(page))
			.toBeCloseTo(KBD_BAR + 16 + noticeHeight, 0);

		await page.waitForTimeout(300);
		const writes = () =>
			page.evaluate(
				() => (window as unknown as { stackWrites: number }).stackWrites,
			);
		const settled = await writes();
		await page.waitForTimeout(1000);
		expect(await writes()).toBe(settled);
	});
});
