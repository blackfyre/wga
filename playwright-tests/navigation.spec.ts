import { expect, test } from "@playwright/test";

test("desktop header exposes supplemental navigation", async ({ page }) => {
  await page.goto("/");
  await page.locator(".site-header__more summary").click();

  await expect(
    page.locator(".site-header__more a", { hasText: "Inspiration" }),
  ).toHaveAttribute("href", "/inspire");
});

test("mobile header exposes the collection navigation", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/");
  await page.locator(".site-header__mobile-menu summary").click();

  await expect(
    page.locator(".site-header__mobile-menu a", { hasText: "Artworks" }),
  ).toHaveAttribute("href", "/artworks");
});

test("environment status confirms before opening GitHub", async ({ page }) => {
  await page.goto("/");
  await expect(page.locator(".environment-status")).toHaveAttribute(
    "href",
    "https://github.com/blackfyre/wga",
  );

  page.once("dialog", async (dialog) => {
    expect(dialog.message()).toContain("not intended for public use");
    await dialog.dismiss();
  });
  await page.locator(".environment-status").click();
});

test("footer exposes utility navigation", async ({ page }) => {
  await page.goto("/");
  const footer = page.getByRole("contentinfo");

  await expect(
    footer.getByRole("link", { name: "Open-source licences" }),
  ).toHaveAttribute("href", "/open-source-licences");
  await expect(
    footer.getByRole("link", { name: "GitHub", exact: true }),
  ).toHaveAttribute("href", "https://github.com/blackfyre/wga");
});
