import { expect, test } from "@playwright/test";

test("the footer link returns to the page header", async ({ page }) => {
	await page.setViewportSize({ width: 1280, height: 720 });
	await page.goto("/artworks");
	const backToTop = page
		.locator("footer")
		.getByRole("link", { name: "↑ BACK TO TOP" });
	await backToTop.scrollIntoViewIfNeeded();
	await expect(page.locator("header#top")).not.toBeInViewport();

	await backToTop.click();
	await expect(page).toHaveURL(/#top$/);
	await expect(page.locator("header#top")).toBeInViewport();
	await expect(page.locator(".jump, .back-to-top")).toHaveCount(0);
});
