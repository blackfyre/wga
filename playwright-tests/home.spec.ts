import { expect, test } from "@playwright/test";

test("home hero uses the reference desktop geometry", async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.goto("/");

  const hero = page.locator(".home-page > section").first();
  await expect(hero).toHaveCSS("max-width", "1160px");

  const featured = hero.locator("figure");
  const box = await featured.boundingBox();
  expect(box?.width).toBeCloseTo(300, 0);
  expect(box?.height).toBeCloseTo(342, 0);
});

test("home hero uses the reference mobile inset", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/");

  const box = await page
    .getByRole("heading", { name: /Explore artists/ })
    .boundingBox();
  expect(box?.x).toBeCloseTo(40, 0);
});
