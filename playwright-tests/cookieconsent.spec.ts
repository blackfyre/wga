import { expect, test } from "@playwright/test";

test("allows visitors to reopen cookie preferences", async ({ page }) => {
	await page.addInitScript(() => {
		Object.defineProperty(navigator, "webdriver", { get: () => false });
	});
	await page.goto("/");

	const consentModal = page.locator("#cc-main .cm");
	await expect(consentModal).toBeVisible();
	await consentModal
		.getByRole("button", { name: "Accept essential cookies" })
		.click();
	await expect(consentModal).toBeHidden();
	await page.reload();
	await expect(consentModal).toBeHidden();

	await page.getByRole("link", { name: "Manage cookie settings" }).click();
	await expect(page.locator("#cc-main .pm")).toBeVisible();
});
