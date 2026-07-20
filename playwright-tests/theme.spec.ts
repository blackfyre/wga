import { expect, test } from "@playwright/test";

for (const colorScheme of ["light", "dark"] as const) {
	test(`uses the browser ${colorScheme} preference by default`, async ({
		page,
	}) => {
		await page.emulateMedia({ colorScheme });
		await page.goto("/");

		await expect(page.locator("html")).toHaveCSS("color-scheme", colorScheme);

		const themeToggle = page.getByRole("checkbox", { name: "Dark mode" });
		if (colorScheme === "dark") {
			await expect(themeToggle).toBeChecked();
			return;
		}

		await expect(themeToggle).not.toBeChecked();
	});
}

test("persists explicitly selected themes", async ({ page }) => {
	await page.emulateMedia({ colorScheme: "dark" });
	await page.goto("/");

	const themeToggle = page.getByRole("checkbox", { name: "Dark mode" });
	await expect(themeToggle).toBeChecked();
	await themeToggle.focus();
	await page.keyboard.press("Space");

	await expect(page.locator("html")).toHaveAttribute("data-theme", "wga_light");
	await expect(page.locator("html")).toHaveCSS("color-scheme", "light");

	await page.reload();

	await expect(themeToggle).not.toBeChecked();
	await expect(page.locator("html")).toHaveAttribute("data-theme", "wga_light");

	await themeToggle.focus();
	await page.keyboard.press("Space");

	await expect(page.locator("html")).toHaveAttribute("data-theme", "wga_dark");
	await expect(page.locator("html")).toHaveCSS("color-scheme", "dark");

	await page.reload();

	await expect(themeToggle).toBeChecked();
	await expect(page.locator("html")).toHaveAttribute("data-theme", "wga_dark");
});

test.describe("without JavaScript", () => {
	test.use({ javaScriptEnabled: false });

	test("uses the browser dark preference by default", async ({ page }) => {
		await page.emulateMedia({ colorScheme: "dark" });
		await page.goto("/");

		await expect(page.locator("html")).toHaveCSS("color-scheme", "dark");
	});
});
