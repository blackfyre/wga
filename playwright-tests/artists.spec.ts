import { test, expect } from "@playwright/test";

test("check artists page", async ({ page }) => {
  await page.goto("/artists");

  await expect(page.getByRole("heading", { name: "Artist index" })).toBeVisible();
  await expect(page.getByRole("navigation", { name: "Filter artists by first letter" })).toBeVisible();

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
  await expect(
    page.getByRole("heading", { name: "Biography", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("heading", { name: "Works in the collection" }),
  ).toBeVisible();
  await expect(page.locator(".artist-biography__citation pre")).toContainText(
    "@misc{wga_artist_",
  );
});

test("filters artists by initial letter", async ({ page }) => {
  await page.goto("/artists");

  await page.getByRole("link", { name: "A", exact: true }).click();

  await expect(page).toHaveURL(/\/artists\?letter=A/);
  await expect(page.getByRole("link", { name: "A", exact: true })).toHaveAttribute("aria-current", "page");
  await expect(page.getByText("No artists match this search.")).toBeVisible();
});

test("search preserves the selected initial letter", async ({ page }) => {
  await page.goto("/artists?letter=S");

  await page
    .getByPlaceholder("Find an artist")
    .pressSequentially("Synthetic Artist 02", { delay: 100 });

  await expect(page.getByRole("link", { name: "S", exact: true })).toHaveAttribute(
    "aria-current",
    "page",
  );
  await expect(page.locator("table")).toHaveText(/Synthetic Artist 02/);
});
