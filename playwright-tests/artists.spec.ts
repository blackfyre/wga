import { test, expect } from "@playwright/test";

test("check artists page", async ({ page }) => {
  await page.goto("/");

  // Click the get started link.
  await page.getByRole("link", { name: "Artists", exact: true }).click();

  // expect to find "Synthetic Artist 01" on the page, in a table.

  await expect(page.locator("table")).toHaveText(/Synthetic Artist 01/);

  // use the search box to find "Synthetic Artist 02"
  await page
    .getByPlaceholder("Find an artist")
    .pressSequentially("Synthetic Artist 02", { delay: 100 });

  // expect to find "Synthetic Artist 02" on the page, in a table.
  await expect(page.locator("table")).toHaveText(/Synthetic Artist 02/);

  // follow the link Synthetic Artist 02
  await page.getByRole("link", { name: "Synthetic Artist 02" }).click();

  // expect to find "Synthetic Artist 02" in the title.
  await expect(page).toHaveTitle(/Synthetic Artist 02/);
});
