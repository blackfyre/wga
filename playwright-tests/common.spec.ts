import { test, expect } from "@playwright/test";

test("has title", async ({ page }) => {
  await page.goto("/");

  // Expect a title "to contain" a substring.
  await expect(page).toHaveTitle(/WGA/);
});

test("check feedback", async ({ page }) => {
  await page.goto("/");

  // Click the get started link.
  await page.getByRole("link", { name: "Feedback" }).click();

  // expect dialog #d to be visible.
  await expect(page.locator("#d")).toBeVisible();

  // expect dialog #d to have text "Are doing good?"
  await expect(page.locator("#d")).toHaveText(/Are we doing good/);

  await page.getByLabel("Name").fill("Playwright Tester");
  await page.getByLabel("Email").fill("playwright.tester@local.host");
  await page.getByLabel("Message").fill("I am testing your site.");

  // Click the submit button.
  await page.getByRole("button", { name: "Send feedback" }).click();

  // expect dialog #d to be hidden.
  // await expect(page.locator("#d")).toBeHidden();

  // expect notification popup: Thank you! Your feedback is valuable to us!
  await expect(page.locator(".is-success")).toBeVisible();
});

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
