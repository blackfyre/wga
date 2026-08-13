import { expect, test } from "@playwright/test";

for (const colorScheme of ["light", "dark"] as const) {
	test(`uses the browser ${colorScheme} preference by default`, async ({
		page,
	}) => {
		await page.emulateMedia({ colorScheme });
		await page.goto("/");

		await expect(page.locator("html")).toHaveCSS("color-scheme", colorScheme);
	});
}

test.describe("without JavaScript", () => {
	test.use({ javaScriptEnabled: false });

	test("uses the browser dark preference by default", async ({ page }) => {
		await page.emulateMedia({ colorScheme: "dark" });
		await page.goto("/");

		await expect(page.locator("html")).toHaveCSS("color-scheme", "dark");
	});
});

test("switches and remembers the selected colour scheme", async ({ page }) => {
	await page.goto("/");

	await page.getByRole("button", { name: "DARK" }).first().click();
	await expect(page.locator("html")).toHaveAttribute("data-theme", "wga_dark");
	await expect(
		page.getByRole("button", { name: "DARK" }).first(),
	).toHaveAttribute("aria-pressed", "true");
	await expect(page.getByRole("button", { name: "DARK" }).first()).toHaveClass(
		/bg-primary/,
	);

	await page.reload();
	await expect(page.locator("html")).toHaveAttribute("data-theme", "wga_dark");
	await page.getByRole("button", { name: "LIGHT" }).first().click();
	await expect(page.locator("html")).toHaveAttribute("data-theme", "wga_light");
});
