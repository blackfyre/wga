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
	const acceptEssential = consentModal.getByRole("button", {
		name: "ACCEPT ESSENTIAL COOKIES",
	});
	await acceptEssential.scrollIntoViewIfNeeded();
	await expect(acceptEssential).toBeInViewport();
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
		document.documentElement.dataset.theme = "wga-rams-dark";
	});
	await expect(consentModal).toHaveCSS("background-color", "rgb(26, 24, 20)");
	await page.emulateMedia({ reducedMotion: "reduce" });
	await acceptEssential.click();
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
