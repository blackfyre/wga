import { expect, test } from "@playwright/test";

test("glossary supports alphabetic filtering, delayed search, and reset", async ({
  page,
}) => {
  await page.goto("/glossary");
  await expect(page.locator("#glossary")).toHaveCSS("padding-top", "40px");
  await expect(page.getByRole("heading", { name: "Glossary" })).toBeVisible();
  await expect(
    page.getByRole("navigation", { name: "Filter terms by letter" }),
  ).toBeVisible();
  await expect(
    page.getByRole("searchbox", { name: "Search terms and definitions" }),
  ).toBeVisible();
  await expect(
    page.getByRole("link", { name: "ALL", exact: true }),
  ).toHaveAttribute("aria-current", "page");

  const firstTerm = await page.locator("#glossary dt").first().textContent();
  if (!firstTerm) {
    throw new Error("Expected seeded glossary terms");
  }
  const letter = firstTerm.trim().at(0)?.toUpperCase();
  if (!letter) {
    throw new Error("Expected a glossary term beginning with a letter");
  }

  await page.getByRole("link", { name: letter, exact: true }).click();
  await expect(page).toHaveURL(new RegExp(`/glossary\\?letter=${letter}`));
  await expect(
    page.getByRole("link", { name: letter, exact: true }),
  ).toHaveAttribute("aria-current", "page");
  await expect(
    page.getByRole("link", { name: "ALL", exact: true }),
  ).toHaveAttribute("aria-current", "false");
  await expect(page.locator("#glossary dt").first()).toHaveText(
    new RegExp(`^${letter}`, "i"),
  );

  await page
    .getByRole("searchbox", { name: "Search terms and definitions" })
    .fill(firstTerm.trim());
  await expect(page).toHaveURL(/q=/);
  await expect(page.locator("#glossary dt")).toHaveCount(1);

  await page
    .getByRole("searchbox", { name: "Search terms and definitions" })
    .fill("no matching glossary term");
  await expect(page).toHaveURL(/q=no(?:%20|\+)matching/);
  await expect(page.getByText("No terms match.")).toBeVisible();
  await page.getByRole("link", { name: "CLEAR SEARCH →" }).click();
  await expect(page).toHaveURL(/\/glossary$/);
  await expect(page.locator("#glossary dt").first()).toBeVisible();
});

test("glossary uses the reference title scale on mobile", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/glossary");

  await expect(page.getByRole("heading", { name: "Glossary" })).toHaveCSS(
    "font-size",
    "44px",
  );
});

test("glossary search form works without JavaScript", async ({ browser }) => {
  const source = await browser.newPage();
  await source.goto("/glossary");
  const term = await source.locator("#glossary dt").first().textContent();
  await source.close();
  if (!term) {
    throw new Error("Expected seeded glossary terms");
  }

  const context = await browser.newContext({ javaScriptEnabled: false });
  const page = await context.newPage();
  await page.goto("/glossary");
  await page
    .getByRole("searchbox", { name: "Search terms and definitions" })
    .fill(term.trim());
  await page
    .getByRole("searchbox", { name: "Search terms and definitions" })
    .press("Enter");

  await expect(page).toHaveURL(/\/glossary\?q=/);
  await expect(page.locator("#glossary dt")).toHaveCount(1);
  await context.close();
});
