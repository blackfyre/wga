import { test, expect } from "@playwright/test";

test("send postcard", async ({ page }) => {
  await page.goto("/artists/koedijck-isaack-3ed9e200b9e8252");

  // Click the get started link.
  await page.getByRole("link", { name: "Send postcard" }).click();

  // expect dialog #d to be visible.
  await expect(page.locator("#d")).toBeVisible();

  // expect dialog #d to have text "Write a postcard"
  await expect(page.locator("#d")).toHaveText(/Write a postcard/);

  await page.getByLabel("Name").fill("Playwright Tester");
  await page.getByLabel("Email").fill("playwright.tester@local.host");
  await page
    .getByPlaceholder("Email address", { exact: true })
    .fill("playwright.tester@local.host");
  await page.locator("trix-editor").fill("I am testing your site.");

  // Click the submit button.
  await page.getByRole("button", { name: "Send postcard" }).click();

  // expect dialog #d to be hidden.
  await expect(page.locator(".is-success")).toBeVisible();

  test.slow();
  // wait for a minute
  await page.waitForTimeout(60000);

  await page.goto("http://localhost:8025/");
  await page
    .getByRole("link", { name: "WGA playwright.tester@local." })
    .click();
  const page1Promise = page.waitForEvent("popup");
  await page
    .frameLocator("#preview-html")
    .getByRole("link", { name: "Pickup my Postcard!" })
    .click();
  const page1 = await page1Promise;

  // expect to find "I am testing your site." on the page.
  await expect(page1.locator("#mc-area")).toHaveText(/I am testing your site/);
});
