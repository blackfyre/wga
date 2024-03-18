import { test, expect } from "@playwright/test";

test("check feedback", async ({ page }) => {
  await page.goto("/");

  // Click the get started link.
  await page.getByRole("link", { name: "Feedback" }).click();

  // expect dialog #d to have text "Are doing good?"
  await expect(page.locator("#d")).toHaveText(/Are we doing good/);

  await page.getByLabel("Name").fill("Playwright Tester");
  await page.getByLabel("Email").fill("playwright.tester@local.host");
  await page.getByLabel("Message").fill("I am testing your site.");

  // Click the submit button.
  await page.getByRole("button", { name: "Send feedback" }).click();

  // expect notification popup: Thank you! Your feedback is valuable to us!
  await expect(page.locator(".is-success")).toBeVisible();
});
