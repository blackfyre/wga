import { test, expect } from "@playwright/test";

test("places feedback at the mobile bottom edge", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/");

  const feedback = page.locator("a[hx-get='/feedback']");
  const bottom = await feedback.evaluate(
    (element) => window.innerHeight - element.getBoundingClientRect().bottom,
  );
  expect(bottom).toBeCloseTo(16, 0);
  await expect(feedback).toHaveCSS("padding-left", "18px");
  await expect(feedback).toHaveCSS("padding-top", "13px");
});

test("cancel closes the feedback dialog", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("link", { name: "Feedback" }).click();
  await expect(page.locator("#d")).toHaveAttribute("open", "");

  await page.getByRole("button", { name: "CANCEL" }).click();
  await expect(page.locator("#d")).not.toHaveAttribute("open", "");
});

test("check feedback", async ({ page }) => {
  await page.goto("/");

  // Click the get started link.
  await page.getByRole("link", { name: "Feedback" }).click();

  await expect(page.locator("#d")).toContainText("Tell us what is wrong");
  await expect(page.locator("#d .modal-box")).toHaveCSS("opacity", "1");
  await expect(
    page.locator("#d").getByLabel("Close dialog").first(),
  ).toBeVisible();
  await expect(page.locator("#d")).toContainText("SENT WITH THIS REPORT");
  await expect(page.locator("#d")).toContainText("Home");
  await expect(page.locator("#d")).toContainText("BUILD");
  await expect(page.locator("#d")).toContainText(/(development|test) · dev/);
  await page.getByText("CORRECTION", { exact: true }).click();
  await expect(page.getByRole("radio", { name: "CORRECTION" })).toBeChecked();

  await page
    .getByRole("textbox", { name: /YOUR MESSAGE/ })
    .fill("I am testing your site.");
  await page
    .getByRole("textbox", { name: /EMAIL — OPTIONAL/ })
    .fill("playwright.tester@local.host");

  // Click the submit button.
  await page.getByRole("button", { name: "SEND REPORT →" }).click();

  await expect(page.locator("#d")).toContainText("Thank you — report received");
});
