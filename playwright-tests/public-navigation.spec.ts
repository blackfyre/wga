import { expect, test } from "@playwright/test";

test("desktop navigation opens the catalogue", async ({ page }) => {
  await page.goto("/");
  await page
    .getByRole("navigation", { name: "Primary navigation" })
    .getByRole("link", { name: "ARTWORKS", exact: true })
    .click();

  await expect(page).toHaveURL(/\/artworks$/);
  await expect(page.getByRole("heading", { name: "Artworks" })).toBeVisible();
});

test("desktop navigation underline sits on the navigation edge", async ({
  page,
}) => {
  await page.goto("/");
  const navigation = page.locator(
    "header > nav[aria-label='Primary navigation']",
  );
  const artworkLink = navigation.getByRole("link", {
    name: "ARTWORKS",
    exact: true,
  });

  await artworkLink.hover();
  await expect(artworkLink).toHaveCSS("border-bottom-color", "rgb(0, 51, 102)");

  const [navigationBottom, linkBottom] = await Promise.all([
    navigation.evaluate((element) => element.getBoundingClientRect().bottom),
    artworkLink.evaluate((element) => element.getBoundingClientRect().bottom),
  ]);
  expect(Math.abs(navigationBottom - linkBottom)).toBeLessThanOrEqual(1);
});

test("navigation matches the prototype type scale", async ({ page }) => {
  await page.goto("/");
  const artworkLink = page
    .locator("header > nav[aria-label='Primary navigation']")
    .getByRole("link", { name: "ARTWORKS", exact: true });

  await expect(artworkLink).toHaveCSS("font-size", "12px");
  await expect(artworkLink).toHaveCSS("letter-spacing", "1.5px");
  await expect(page.locator("header a[href='/'] span span").last()).toHaveCSS(
    "font-size",
    "10px",
  );
});

test("desktop header matches the prototype baseline", async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.goto("/");

  const bottom = await page
    .locator("header")
    .evaluate((element) => element.getBoundingClientRect().bottom);
	expect(Math.abs(bottom - 148)).toBeLessThanOrEqual(1);
});

test("header search submits an artwork title", async ({ page }) => {
  await page.goto("/");
  await page
    .getByRole("searchbox", { name: "Search artwork titles" })
    .fill("Synthetic Artwork 01-01");
  await page.getByRole("button", { name: "SEARCH", exact: true }).click();

  await expect(page).toHaveURL(/\/artworks\?title=Synthetic\+Artwork\+01-01/);
  await expect(page.locator("#search-result-container")).toContainText(
    "Synthetic Artwork 01-01",
  );
});

test("mobile navigation opens by keyboard", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/");

  const menu = page.locator("summary[aria-label='Open navigation']");
  await menu.focus();
  await expect(menu).toBeFocused();
  await menu.press("Enter");
  await expect(
    page.getByRole("navigation", { name: "Primary navigation" }),
  ).toBeVisible();
});

test("mobile navigation closes after following a link", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/");
  const menu = page.locator("header details");

  await page.locator("summary[aria-label='Open navigation']").click();
  await expect(menu).toHaveAttribute("open", "");
  await page
    .getByRole("navigation", { name: "Primary navigation" })
    .getByRole("link", { name: "ARTISTS", exact: true })
    .click();

  await expect(page).toHaveURL(/\/artists$/);
  await expect(menu).not.toHaveAttribute("open", "");
  const activeLink = page.locator(
    "[data-mobile-navigation] a[href='/artists']",
  );
  await expect(activeLink).toHaveAttribute("aria-current", "page");
  await expect(activeLink).toHaveCSS("background-color", "rgb(0, 51, 102)");
  await expect(activeLink).toHaveCSS("padding-left", "12px");
});

test("mobile navigation highlights the current page when opened", async ({
  page,
}) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/artists");
  await page.locator("summary[aria-label='Open navigation']").click();

  const activeLink = page.locator(
    "[data-mobile-navigation] a[href='/artists']",
  );
  await expect(activeLink).toHaveAttribute("aria-current", "page");
  await expect(activeLink).toHaveCSS("background-color", "rgb(0, 51, 102)");
});

test("reduced motion removes page-enter animation delay", async ({ page }) => {
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.goto("/");

  await expect(page.locator("#mc-area")).toHaveCSS(
    "animation-duration",
    "1e-05s",
  );
});

test("navigation works without JavaScript", async ({ browser }) => {
  const context = await browser.newContext({ javaScriptEnabled: false });
  const page = await context.newPage();
  await page.goto("/");
  await page
    .getByRole("navigation", { name: "Primary navigation" })
    .getByRole("link", { name: "ARTISTS", exact: true })
    .click();

  await expect(page).toHaveURL(/\/artists$/);
  await expect(page.getByRole("heading", { name: "Artists" })).toBeVisible();
  await context.close();
});

test("footer follows the public site map", async ({ page }) => {
  await page.goto("/");
  const footer = page.locator("footer");

  await expect(footer.getByText("BROWSE", { exact: true })).toBeVisible();
  await expect(footer.getByText("SERVICES", { exact: true })).toBeVisible();
  await expect(footer.getByText("ABOUT", { exact: true })).toBeVisible();
  await expect(footer.getByRole("link", { name: "Artists" })).toHaveAttribute(
    "href",
    "/artists",
  );
  await expect(page.getByRole("checkbox", { name: "Dark mode" })).toHaveCount(
    0,
  );
});
