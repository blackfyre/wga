import { test, expect } from "@playwright/test";

const entry = {
  name: "Playwright",
  email: "playwright@local.host",
  location: "Testing Grounds",
  entry: "This is a test entry!",
  entryTest: /This is a test entry/,
};

test("test", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("link", { name: "Guestbook" }).click();
  await page.getByRole("link", { name: "Add Entry" }).click();
  await page.getByPlaceholder("Your name", { exact: true }).fill(entry.name);
  await page.getByPlaceholder("Your name", { exact: true }).press("Tab");
  await page.getByPlaceholder("Your email", { exact: true }).fill(entry.email);
  await page.getByPlaceholder("Your email", { exact: true }).press("Tab");
  await page.getByPlaceholder("Your location").fill(entry.location);
  await page.getByPlaceholder("Your location").press("Tab");
  await page.getByPlaceholder("Your entry").fill(entry.entry);
  await page.getByRole("button", { name: "Leave entry" }).click();

  await expect(page.locator(".toast")).toHaveText(/Message added successfully/);

  await expect(page.locator(".gb-entries .gb-entry").nth(0)).toHaveText(
    entry.entryTest,
  );
});
