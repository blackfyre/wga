import { test, expect } from "@playwright/test";

const entry = {
  name: "Playwright",
  email: "playwright@local.host",
  location: "Testing Grounds",
  entry: "This is a test entry!",
  entryTest: /This is a test entry/,
};

test("places the guestbook note rail on the desktop left edge", async ({
  page,
}) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.goto("/guestbook");

  const box = await page.locator("aside").boundingBox();
	expect(box?.x).toBeCloseTo(140, 0);
  expect(box?.width).toBeCloseTo(340, 0);
});

test("uses the reference guestbook title scale on mobile", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/guestbook");

  await expect(page.getByRole("heading", { name: "Guestbook" })).toHaveCSS(
    "font-size",
    "44px",
  );
});

test("test", async ({ page }) => {
  await page.goto("/");
  await page
    .getByRole("navigation", { name: "Primary navigation" })
    .getByRole("link", { name: "GUESTBOOK", exact: true })
    .click();
  await expect(
    page.getByRole("navigation", { name: "Filter guestbook entries by year" }),
  ).toBeVisible();
  await page.getByPlaceholder("Your name", { exact: true }).fill(entry.name);
  await page.getByPlaceholder("Your name", { exact: true }).press("Tab");
  await page.getByPlaceholder("Your email", { exact: true }).fill(entry.email);
  await page.getByPlaceholder("Your email", { exact: true }).press("Tab");
  await page.getByPlaceholder("Your location").fill(entry.location);
  await page.getByPlaceholder("Your location").press("Tab");
  await page
    .getByPlaceholder("What did you come here to find?")
    .fill(entry.entry);
  await page.getByRole("button", { name: "SIGN THE GUESTBOOK →" }).click();

  await expect(page.locator(".toast")).toHaveText(/Message added successfully/);

  await expect(page.locator(".gb-entries .gb-entry").nth(0)).toHaveText(
    entry.entryTest,
  );
});
