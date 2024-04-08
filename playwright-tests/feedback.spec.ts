import { test, expect } from "@playwright/test";

test("check feedback", async ({ page }) => {
  await page.goto("/");

  // Click the get started link.
  await page.getByRole("link", { name: "Feedback" }).click();

  // expect dialog #d to have text "Are doing good?"
  await expect(page.locator("#d")).toHaveText(/Are we doing good/);

  await page
    .getByPlaceholder("Name", { exact: true })
    .fill("Playwright Tester");
  await page
    .getByPlaceholder("Email", { exact: true })
    .fill("playwright.tester@local.host");
  await page.getByPlaceholder("Your message").fill("I am testing your site.");

  // Click the submit button.
  await page.getByRole("button", { name: "Send feedback" }).click();

  // expect notification popup: Thank you! Your feedback is valuable to us!
  await expect(page.locator(".toast")).toHaveText(
    "Thank you! Your feedback is valuable to us!",
  );
});
