import { expect, test } from "@playwright/test";

test("renders the XML sitemap with the site stylesheet", async ({ page }) => {
	await page.goto("/sitemap.xml");

	await expect(page).toHaveTitle("Web Gallery of Art sitemap");
	await expect(page.getByRole("heading", { name: "Sitemap" })).toBeVisible();
	await expect(page.locator("link[rel='stylesheet']")).toHaveAttribute(
		"href",
		/\/assets\/css\/style\.css$/,
	);

	const sitemapLinks = page.locator("main ol a");
	expect(await sitemapLinks.count()).toBeGreaterThan(0);
	await expect(sitemapLinks.first()).toHaveAttribute("href", /\/sitemap\//);
});
