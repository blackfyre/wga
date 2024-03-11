import { test, expect } from "@playwright/test";

test("simple search", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("link", { name: "Artworks" }).click();
  // expect to find h1 "Artwork search" on page
  await expect(page.locator("h1")).toHaveText(/Artwork search/);
  await page.locator("#search-result-container").click();
  await page.getByRole("button", { name: "Search" }).click();
  // expect to at least 1 `.card` elements in #search-result-container
  await expect(page.locator("#search-result-container .card")).not.toHaveCount(
    0,
  );
});

test("artform search", async ({ page }) => {
  await page.goto("/artworks");
  await page.getByLabel("Artforms").selectOption("painting");
  await page.getByRole("button", { name: "Search" }).click();
  // expect to at least 1 `.card` elements in #search-result-container
  await expect(page.locator("#search-result-container .card")).not.toHaveCount(
    0,
  );
});

test("art type search", async ({ page }) => {
  await page.goto("/artworks");
  await page.getByLabel("Art types").selectOption("mythological");
  await page.getByRole("button", { name: "Search" }).click();
  // expect to at least 1 `.card` elements in #search-result-container
  await expect(page.locator("#search-result-container .card")).not.toHaveCount(
    0,
  );
});

test("art school search", async ({ page }) => {
  await page.goto("/artworks");
  await page.getByLabel("Art school").selectOption("hungarian");
  await page.getByRole("button", { name: "Search" }).click();
  // expect to at least 1 `.card` elements in #search-result-container
  await expect(page.locator("#search-result-container .card")).not.toHaveCount(
    0,
  );
});

test("art type and school combined search", async ({ page }) => {
  await page.goto("/artworks");
  await page.getByLabel("Art types").selectOption("mythological");
  await page.getByLabel("Art school").selectOption("hungarian");
  await page.getByRole("button", { name: "Search" }).click();
  // expect to at least 1 `.card` elements in #search-result-container
  await expect(page.locator("#search-result-container .card")).not.toHaveCount(
    0,
  );
});

test("title search", async ({ page }) => {
  await page.goto("http://localhost:8090/artworks");
  await page.getByPlaceholder("Artwork title").fill("Allegory");
  await page.getByRole("button", { name: "Search" }).click();
  await expect(page.locator("#search-result-container .card")).not.toHaveCount(
    0,
  );
});

test("artist name search", async ({ page }) => {
  await page.goto("http://localhost:8090/artworks");
  await page.getByPlaceholder("Artist name").fill("aachen");
  await page.getByRole("button", { name: "Search" }).click();
  await expect(page.locator("#search-result-container .card")).not.toHaveCount(
    0,
  );
});
