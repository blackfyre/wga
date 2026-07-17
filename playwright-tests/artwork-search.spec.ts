import { test, expect } from "@playwright/test";

test.describe.configure({ mode: "serial" });
test.setTimeout(60000);

async function expectArtworkResults(page) {
  await expect(page.locator("#search-result-container .card").first()).toBeVisible({
    timeout: 30000,
  });
}

test("artwork search", async ({ page }) => {
  await page.goto("/artworks");
  await expect(page.locator("h1")).toHaveText(/Artwork search/);
  await page.locator("[name='title']").fill("Allegory");
  await page.getByRole("button", { name: "Search" }).click();
  await expectArtworkResults(page);
  await expect(page.locator("#search-result-container")).toContainText(
    /artworks found\./i,
  );
});

test("artform search", async ({ page }) => {
  await page.goto("/artworks");
  await page.locator("[name='art_form']").selectOption("painting");
  await page.getByRole("button", { name: "Search" }).click();
  await expectArtworkResults(page);
});

test("art type search", async ({ page }) => {
  await page.goto("/artworks");
  await page.locator("[name='art_type']").selectOption("mythological");
  await page.getByRole("button", { name: "Search" }).click();
  await expectArtworkResults(page);
});

test("art school search", async ({ page }) => {
  await page.goto("/artworks");
  await page.locator("[name='art_school']").selectOption("hungarian");
  await page.getByRole("button", { name: "Search" }).click();
  await expectArtworkResults(page);
});

test("art type and school combined search", async ({ page }) => {
  await page.goto("/artworks");
  await page.locator("[name='art_type']").selectOption("mythological");
  await page.locator("[name='art_school']").selectOption("hungarian");
  await page.getByRole("button", { name: "Search" }).click();
  await expectArtworkResults(page);
});

test("title search", async ({ page }) => {
  await page.goto("http://localhost:8090/artworks");
  await page.locator("[name='title']").fill("Allegory");
  await page.getByRole("button", { name: "Search" }).click();
  await expectArtworkResults(page);
});

test("artist name search", async ({ page }) => {
  await page.goto("http://localhost:8090/artworks");
  await page.locator("[name='artist']").fill("aachen");
  await page.getByRole("button", { name: "Search" }).click();
  await expectArtworkResults(page);
});

test("clear resets the artwork search form", async ({ page }) => {
  await page.goto("http://localhost:8090/artworks");
  await page.locator("[name='title']").fill("Allegory");
  await page.locator("[name='art_school']").selectOption("hungarian");
  await page.getByRole("link", { name: "Clear" }).click();

  await expect(page).toHaveURL(/\/artworks$/);
  await expect(page.locator("[name='title']").first()).toHaveValue("");
  await expect(page.locator("[name='art_school']").first()).toHaveValue("");
  await expect(page.locator("#search-result-container")).toContainText(
    /combine filters, then press search/i,
  );
});
