import { test, expect } from "@playwright/test";

test("check artists page", async ({ page }) => {
  await page.goto("/");

  // Click the get started link.
  await page
    .getByRole("navigation", { name: "Primary navigation" })
    .getByRole("link", { name: "ARTISTS", exact: true })
    .click();

  await expect(page.locator("#search-results")).toHaveText(/Synthetic Artist 01/);

  // use the search box to find "Synthetic Artist 02"
  await page
    .getByPlaceholder("Find an artist")
    .pressSequentially("Synthetic Artist 02", { delay: 100 });

  await expect(page.locator("#search-results")).toHaveText(/Synthetic Artist 02/);

  // follow the link Synthetic Artist 02
  await page.getByRole("link", { name: "Synthetic Artist 02" }).click();

  // expect to find "Synthetic Artist 02" in the title.
	await expect(page).toHaveTitle(/Synthetic Artist 02/);
});

test("filters artists by school, profession, period, and letter", async ({ page }) => {
	await page.goto("/artists");

	await expect(page.getByRole("combobox", { name: "SCHOOL" })).toBeVisible();
	await expect(page.getByRole("combobox", { name: "PERIOD" })).toBeVisible();
	await expect(page.getByRole("combobox", { name: "PROFESSION" })).toBeVisible();
	await page.getByRole("combobox", { name: "school" }).selectOption("bohemian");
	await page
		.getByRole("combobox", { name: "profession" })
		.selectOption("Synthetic test profession");
	await page.getByRole("button", { name: "S", exact: true }).click();

	await expect(page).toHaveURL(/school=bohemian/);
	await expect(page).toHaveURL(/profession=Synthetic\+test\+profession/);
	await expect(page).toHaveURL(/letter=S/);
});
