import { expect, test } from "@playwright/test";

test("keeps the static page contents list sticky on desktop", async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.goto("/pages/about");

  const contents = page.getByRole("navigation", { name: "Contents" });
	await expect(contents).toHaveCSS("position", "sticky");

	await page.evaluate(() => window.scrollTo(0, 500));
	const box = await contents.boundingBox();
	expect(box?.y).toBeGreaterThanOrEqual(32);
	expect(box?.y).toBeLessThan(40);
});
