import { test, expect } from "@playwright/test";

const entry = {
  name: "Playwright",
  email: "playwright@local.host",
  location: "Testing Grounds",
  entry: "This is a test entry!",
  entryTest: /This is a test entry/,
};

test("places the guestbook note rail on the desktop right edge", async ({
  page,
}) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.goto("/guestbook");

	const box = await page.locator("aside").boundingBox();
	expect(box?.width).toBeCloseTo(320, 0);
	const entries = await page.locator(".gb-entries").boundingBox();
	expect((box?.x || 0) + (box?.width || 0)).toBeGreaterThan(
		(entries?.x || 0) + (entries?.width || 0),
	);
});

test("uses the reference guestbook title scale on mobile", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/guestbook");

  await expect(page.getByRole("heading", { name: "Guestbook" })).toHaveCSS(
    "font-size",
    "44px",
  );
});

test("places guestbook entries before the note form on mobile", async ({
	page,
}) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await page.goto("/guestbook");

	const form = await page.locator("aside").boundingBox();
	const entries = await page.locator(".gb-entries").boundingBox();
	expect(entries?.y).toBeLessThan(form?.y || 0);
});

test("filters and sorts guestbook entries without leaving the page", async ({
	page,
}) => {
	await page.goto("/guestbook");

	const filters = page.locator('form[hx-get="/guestbook"]');
	await expect(filters.locator('input[type="search"]')).toBeVisible();
	await expect(filters.getByText("ALL", { exact: true })).toBeVisible();

	await page.getByRole("link", { name: "NEWEST ↕" }).click();
	await expect(page).toHaveURL(/sort=oldest/);
	await expect(page.getByRole("link", { name: "OLDEST ↕" })).toBeVisible();
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
