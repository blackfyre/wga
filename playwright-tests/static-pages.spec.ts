import { expect, test } from "@playwright/test";

test("keeps the static page contents list sticky on desktop", async ({
	page,
}) => {
	await page.setViewportSize({ width: 1440, height: 900 });
	await page.goto("/pages/about");

	const contents = page.getByRole("navigation", { name: "Contents" });
	await expect(contents).toHaveCSS("position", "sticky");

	await page.evaluate(() => window.scrollTo(0, 500));
	const box = await contents.boundingBox();
	expect(box?.y).toBeGreaterThanOrEqual(32);
	expect(box?.y).toBeLessThan(40);
});

test("the privacy policy describes cookie and storage use under the consent model", async ({
	page,
}) => {
	await page.goto("/pages/privacy-policy");
	const article = page.locator("main");
	const heading = article.getByRole("heading", {
		name: "Cookies and browser storage",
	});
	await expect(heading).toBeVisible();
	await expect(article).toContainText(
		"They hold your session and the record of your cookie choice",
	);
	await expect(article).toContainText("Reading preferences are optional.");
	await expect(article).toContainText(
		"No advertising or cross-site tracking cookies are set",
	);
	await expect(
		article.getByRole("heading", { name: "Cookies and Web Beacons" }),
	).toHaveCount(0);
	await expect(
		article.getByRole("heading", { name: "Log Files" }),
	).toBeVisible();
});
