import { expect, test } from "@playwright/test";

test("artwork viewer hides its thumbnail navigation bar", async ({ page }) => {
	await page.goto(
		"/artists/synthetic-artist-01-ad32608c6e36b2e/synthetic-artwork-01-01-2225c982be1af02",
	);

	await page.locator("[data-viewer-no-navbar] img").click();

	await expect(page.locator(".viewer-container")).toBeVisible();
	await expect(page.locator(".viewer-navbar")).toBeHidden();
});

test("initialises the artwork BibTeX copy helper", async ({ page }) => {
	await page.goto(
		"/artists/synthetic-artist-01-ad32608c6e36b2e/synthetic-artwork-01-01-2225c982be1af02",
	);

	const copyButton = page.getByRole("button", { name: "COPY BIBTEX" });
	await expect(copyButton).toHaveAttribute("data-copy-bound", "true");
});
