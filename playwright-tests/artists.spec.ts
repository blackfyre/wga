import { test, expect } from "@playwright/test";

test("check artists page", async ({ page }) => {
  await page.goto("/");

  // Click the get started link.
  await page.getByRole("link", { name: "Artists", exact: true }).click();

  // expect to find "AACHEN, Hans von" on the page, in a table.

  await expect(page.locator("table")).toHaveText(/AACHEN, Hans von/);

  // use the search box to find "KOEDIJCK"
  await page
    .getByPlaceholder("Find an artist")
    .pressSequentially("KOEDIJCK", { delay: 100 });

  // expect to find "KOEDIJCK, Isaack" on the page, in a table.
  await expect(page.locator("table")).toHaveText(/KOEDIJCK, Isaack/);

  // follow the link KOEDIJCK, Isaack
  await page.getByRole("link", { name: "KOEDIJCK, Isaack" }).click();

  // expect to find "KOEDIJCK, Isaack" in the title.
  await expect(page).toHaveTitle(/KOEDIJCK, Isaack/);
});
