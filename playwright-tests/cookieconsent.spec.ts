import { expect, test } from "@playwright/test";

test.beforeEach(async ({ context, page }) => {
	await context.clearCookies();
	await page.addInitScript(() => {
		window.localStorage.clear();
	});
});

test("renders a truthful necessary-only notice and reopens preferences", async ({
	page,
}) => {
	await page.addInitScript(() => {
		Object.defineProperty(navigator, "webdriver", { get: () => false });
	});
	await page.goto("/");

	const consentModal = page.locator("#cc-main .cm");
	await expect(consentModal).toBeVisible();
	await expect(consentModal).toContainText("Analytics cookies are not in use.");
	await expect(
		consentModal.getByRole("link", { name: "privacy policy" }),
	).toHaveAttribute("href", "/pages/privacy-policy");
	await expect(
		page.getByRole("link", { name: "Cookie settings" }),
	).toBeVisible();
	for (const width of [390, 834, 1440]) {
		await page.setViewportSize({ width, height: 900 });
		const feedback = page.getByRole("link", { name: "FEEDBACK" });
		const [noticeBox, feedbackBox] = await Promise.all([
			consentModal.boundingBox(),
			feedback.boundingBox(),
		]);
		expect(noticeBox).not.toBeNull();
		expect(feedbackBox).not.toBeNull();
		if (!noticeBox || !feedbackBox) {
			return;
		}
		expect(noticeBox.x).toBeGreaterThanOrEqual(0);
		expect(noticeBox.x + noticeBox.width).toBeLessThanOrEqual(width);
		expect(
			noticeBox.x + noticeBox.width <= feedbackBox.x ||
				noticeBox.y + noticeBox.height <= feedbackBox.y ||
				feedbackBox.y + feedbackBox.height <= noticeBox.y,
		).toBeTruthy();
	}
	await page.setViewportSize({ width: 390, height: 844 });
	await page.evaluate(() => {
		document.documentElement.style.fontSize = "200%";
	});
	const enlargedNotice = await consentModal.boundingBox();
	expect(enlargedNotice).not.toBeNull();
	if (!enlargedNotice) {
		return;
	}
	expect(enlargedNotice.x).toBeGreaterThanOrEqual(0);
	expect(enlargedNotice.x + enlargedNotice.width).toBeLessThanOrEqual(390);
	expect(
		await consentModal.evaluate((notice) => notice.scrollWidth),
	).toBeLessThanOrEqual(enlargedNotice.width);
	const deny = consentModal.getByRole("button", {
		name: "DENY",
	});
	await deny.scrollIntoViewIfNeeded();
	await expect(deny).toBeInViewport();
	const enlargedFeedback = await page
		.getByRole("link", { name: "FEEDBACK" })
		.boundingBox();
	expect(enlargedFeedback).not.toBeNull();
	if (!enlargedFeedback) {
		return;
	}
	expect(
		enlargedNotice.x + enlargedNotice.width <= enlargedFeedback.x ||
			enlargedNotice.y + enlargedNotice.height <= enlargedFeedback.y ||
			enlargedFeedback.y + enlargedFeedback.height <= enlargedNotice.y,
	).toBeTruthy();
	await page.evaluate(() => {
		document.documentElement.dataset.palette = "bone";
		document.documentElement.dataset.theme = "dark";
	});
	await expect(consentModal).toHaveCSS("background-color", "rgb(26, 24, 20)");
	await page.emulateMedia({ reducedMotion: "reduce" });
	await deny.click();
	await expect(consentModal).toBeHidden();
	await page.reload();
	await expect(consentModal).toBeHidden();

	const cookieSettings = page.getByRole("link", { name: "Cookie settings" });
	await expect(cookieSettings).toBeVisible();
	await cookieSettings.click();
	const preferences = page.locator("#cc-main .pm");
	await expect(preferences).toBeVisible();
	await expect(preferences).toContainText("Strictly necessary cookies");
	await expect(preferences).not.toContainText("Performance and Analytics");
	await expect(
		preferences.getByRole("button", { name: "Close cookie preferences" }),
	).toBeVisible();
});

test("keeps the client-only settings control unavailable without JavaScript", async ({
	browser,
}) => {
	const context = await browser.newContext({ javaScriptEnabled: false });
	const page = await context.newPage();
	await page.goto("/");

	expect(
		await page.evaluate(() => document.querySelector("#cc-main .cm") === null),
	).toBeTruthy();
	await expect(
		page.getByRole("link", { name: "Cookie settings" }),
	).toBeHidden();
	await context.close();
});

test("keeps settings unavailable when CookieConsent cannot generate preferences UI", async ({
	page,
}) => {
	await page.addInitScript(() => {
		const querySelector = Document.prototype.querySelector;
		Document.prototype.querySelector = function (selector) {
			if (selector === "#cc-main .cm") {
				return null;
			}
			return querySelector.call(this, selector);
		};
	});
	await page.goto("/");

	await expect(
		page.getByRole("link", { name: "Cookie settings" }),
	).toBeHidden();
	await expect(page.getByRole("link", { name: "FEEDBACK" })).toBeVisible();
});

async function consentCategories(page): Promise<string[] | null> {
	const cookie = (await page.context().cookies()).find(
		(entry) => entry.name === "cc_cookie",
	);
	if (!cookie) {
		return null;
	}
	return JSON.parse(decodeURIComponent(cookie.value)).categories;
}

test.describe("consent actions", () => {
	test.beforeEach(async ({ page }) => {
		await page.addInitScript(() => {
			Object.defineProperty(navigator, "webdriver", { get: () => false });
		});
	});

	test("the notice offers ACCEPT ALL, DENY and PREFERENCES in reading and focus order", async ({
		page,
	}) => {
		await page.setViewportSize({ width: 1440, height: 900 });
		await page.goto("/");
		const notice = page.locator("#cc-main .cm");
		await expect(notice).toBeVisible();
		const buttons = notice.getByRole("button");
		await expect(buttons).toHaveText(["ACCEPT ALL", "DENY", "PREFERENCES"]);

		const boxes = await buttons.evaluateAll((elements) =>
			elements.map((element) => element.getBoundingClientRect().left),
		);
		expect(boxes).toEqual([...boxes].sort((a, b) => a - b));
		for (const button of await buttons.all()) {
			await expect(button).toHaveCSS("text-transform", "none");
			await expect(button).toHaveCSS("font-family", /mono/i);
		}

		await buttons.first().focus();
		await page.keyboard.press("Tab");
		await expect(notice.getByRole("button", { name: "DENY" })).toBeFocused();
		await page.keyboard.press("Tab");
		await expect(
			notice.getByRole("button", { name: "PREFERENCES", exact: true }),
		).toBeFocused();
		await page.keyboard.press("Enter");
		await expect(page.locator("#cc-main .pm")).toBeVisible();
	});

	test("ACCEPT ALL grants preference storage", async ({ page }) => {
		await page.goto("/");
		await page
			.locator("#cc-main .cm")
			.getByRole("button", { name: "ACCEPT ALL" })
			.click();
		await expect(page.locator("#cc-main .cm")).toBeHidden();
		expect(await consentCategories(page)).toEqual(
			expect.arrayContaining(["necessary", "preferences"]),
		);
	});

	test("DENY keeps only necessary cookies and deletes stored preferences", async ({
		page,
		baseURL,
	}) => {
		await page
			.context()
			.addCookies([
				{
					name: "wga_aw_prefs",
					value: "%7B%22actions%22%3Atrue%7D",
					url: baseURL,
				},
			]);
		await page.addInitScript(() => {
			window.localStorage.setItem("wga-aw-prefs", '{"actions":true}');
		});
		await page.goto("/");
		await page
			.locator("#cc-main .cm")
			.getByRole("button", { name: "DENY" })
			.press("Enter");
		await expect(page.locator("#cc-main .cm")).toBeHidden();
		expect(await consentCategories(page)).toEqual(["necessary"]);
		const cookies = await page.context().cookies();
		expect(
			cookies.find((entry) => entry.name === "wga_aw_prefs"),
		).toBeUndefined();
	});

	test("the preferences dialog offers accept all, deny and save with a category toggle", async ({
		page,
	}) => {
		await page.goto("/");
		await page
			.locator("#cc-main .cm")
			.getByRole("button", { name: "PREFERENCES", exact: true })
			.click();
		const dialog = page.locator("#cc-main .pm");
		await expect(dialog).toBeVisible();
		for (const name of ["ACCEPT ALL", "DENY", "SAVE PREFERENCES"]) {
			await expect(
				dialog.getByRole("button", { name, exact: true }),
			).toBeVisible();
		}
		const toggle = dialog.locator('input[value="preferences"]');
		await expect(toggle).not.toBeChecked();
		await toggle.focus();
		await page.keyboard.press("Space");
		await expect(toggle).toBeChecked();
		await dialog
			.getByRole("button", { name: "SAVE PREFERENCES", exact: true })
			.click();
		await expect(dialog).toBeHidden();
		expect(await consentCategories(page)).toEqual(
			expect.arrayContaining(["necessary", "preferences"]),
		);
	});
});
