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
    "@online{wga-artist-",
  );
  await expect(page.locator(".artist-biography__citation pre")).toContainText(
    "url = {http",
  );
  await expect(
    page.getByRole("navigation", { name: "Breadcrumb" }),
  ).toBeVisible();

  await page.locator(".artist-biography__work-grid a").first().click();

  await expect(
    page.getByRole("heading", { name: "Core metadata" }),
  ).toBeVisible();
  await expect(
    page.getByRole("heading", { name: "Scholarly commentary" }),
  ).toBeVisible();
  await expect(page.locator(".artwork-detail__citation pre")).toContainText(
    "@online{wga-artwork-",
  );
  await expect(page.locator(".artwork-detail__citation pre")).toContainText(
    "url = {http",
  );
  await expect(
    page.getByRole("navigation", { name: "Breadcrumb" }),
  ).toBeVisible();
});

test("filters artists by initial letter", async ({ page }) => {
  await page.goto("/artists");

  await expect(page.getByText("A", { exact: true })).toHaveAttribute(
    "aria-disabled",
    "true",
  );
  await page.goto("/artists?letter=A");

  await expect(page).toHaveURL(/\/artists\?letter=A/);
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
