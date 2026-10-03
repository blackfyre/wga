import { expect, type Page, test } from "@playwright/test";

// Design parity for artwork search results (align-artwork-search-results):
// the PAGE n OF m pagination row, the counted SCHOOL/FORM text rows, and the
// filtered empty state. The synthetic fixture holds more than one 16-work page,
// so the unfiltered catalogue always paginates.

const viewports = [390, 834, 1440];

function paginationRow(page: Page) {
  return page.locator("#artwork-search-results [data-artwork-pagination]");
}

async function expectNoLinkWithoutHref(page: Page) {
  await expect(
    page.locator("#artwork-search-results a:not([href])"),
  ).toHaveCount(0);
}

for (const width of viewports) {
  test.describe(`artwork search results design at ${width}px`, () => {
    test.use({ viewport: { width, height: 900 } });
    test.setTimeout(60000);

    test("pagination row uses real links and swaps only the results", async ({
      page,
    }) => {
      await page.goto("/artworks");
      const row = paginationRow(page);
      await expect(row).toBeVisible();
      await expect(row).toContainText(/^PAGE 1 OF \d+/);
      const pageCount = Number(
        (await row.locator("p").innerText()).match(/OF (\d+)/)?.[1],
      );
      expect(pageCount).toBeGreaterThan(1);

      const disabledPrev = row.locator("span[aria-disabled='true']", {
        hasText: "← PREV",
      });
      await expect(disabledPrev).toBeVisible();
      await expect(row.getByRole("link", { name: "← PREV" })).toHaveCount(0);
      const next = row.getByRole("link", { name: "NEXT →" });
      await expect(next).toHaveAttribute("href", /^\/artworks\?.*page=2/);
      await expect(next).toHaveAttribute(
        "hx-get",
        /^\/artworks\/results\?.*page=2/,
      );
      await expectNoLinkWithoutHref(page);

      const filters = await page.locator("#artwork-filters").elementHandle();
      const response = page.waitForResponse(
        (item) =>
          new URL(item.url()).pathname === "/artworks/results" &&
          item.request().headers()["hx-request"] === "true",
      );
      await next.click();
      await response;
      await expect(page).toHaveURL(
        (url) => url.searchParams.get("page") === "2",
      );
      await expect(paginationRow(page)).toContainText(`PAGE 2 OF ${pageCount}`);
      expect(await filters?.evaluate((element) => element.isConnected)).toBe(
        true,
      );

      const prev = paginationRow(page).getByRole("link", { name: "← PREV" });
      await expect(prev).toHaveAttribute("href", /^\/artworks/);
      await expect(prev).not.toHaveAttribute("href", /page=/);
      if (pageCount === 2) {
        await expect(
          paginationRow(page).locator("span[aria-disabled='true']", {
            hasText: "NEXT →",
          }),
        ).toBeVisible();
      }
      await expectNoLinkWithoutHref(page);
    });

    test("pagination follows ordinary links without JavaScript", async ({
      browser,
    }) => {
      const context = await browser.newContext({
        javaScriptEnabled: false,
        viewport: { width, height: 900 },
      });
      const page = await context.newPage();
      await page.goto("/artworks?view=list");
      await paginationRow(page).getByRole("link", { name: "NEXT →" }).click();
      await expect(page).toHaveURL(
        (url) =>
          url.pathname === "/artworks" &&
          url.searchParams.get("page") === "2" &&
          url.searchParams.get("view") === "list",
      );
      await expect(paginationRow(page)).toContainText(/^PAGE 2 OF/);
      await expect(page.locator("[data-view='list']")).toBeVisible();
      await paginationRow(page).getByRole("link", { name: "← PREV" }).click();
      await expect(page).toHaveURL(
        (url) =>
          url.pathname === "/artworks" &&
          !url.searchParams.has("page") &&
          url.searchParams.get("view") === "list",
      );
      await expect(paginationRow(page)).toContainText(/^PAGE 1 OF/);
      await context.close();
    });

    test("school facet renders count-ordered text rows", async ({ page }) => {
      await page.goto("/artworks");
      const rows = page.locator(
        "[data-artwork-facet-options='art_school'] [data-artwork-facet-option]",
      );
      await expect(rows.first()).toBeVisible();
      const counts = (
        await rows.evaluateAll((labels) =>
          labels.map((label) => label.lastElementChild?.textContent ?? ""),
        )
      ).map((text) => Number(text.replace(/\D/g, "")));
      expect(counts.length).toBeGreaterThan(1);
      expect(counts).toEqual([...counts].sort((a, b) => b - a));

      const checkbox = rows.first().locator("input[type='checkbox']");
      const box = await checkbox.boundingBox();
      expect(box === null || (box.width <= 1 && box.height <= 1)).toBe(true);
      await expect(rows.first()).not.toContainText("×");

      const value = await checkbox.getAttribute("value");
      const response = page.waitForResponse(
        (item) =>
          new URL(item.url()).pathname === "/artworks" &&
          new URL(item.url()).searchParams
            .getAll("art_school")
            .includes(value as string),
      );
      await rows.first().click();
      await response;
      const picked = page.locator(
        `[data-artwork-facet-option]:has(input[name='art_school'][value='${value}'])`,
      );
      await expect(picked.locator("input")).toBeChecked();
      await expect(picked).toContainText("×");
    });

    test("filtered empty state uses the reference copy", async ({ page }) => {
      await page.goto("/artworks?q=zzzz-no-such-artwork");
      const results = page.locator("#artwork-search-results");
      await expect(
        results.getByRole("heading", { name: "No works match these filters." }),
      ).toBeVisible();
      await expect(results).toContainText(
        "Widen the year range or clear the school filter — the collection is uneven across periods.",
      );
      await expect(
        results.getByRole("link", { name: "RESET FILTERS →" }),
      ).toHaveAttribute("href", "/artworks");
      await expect(paginationRow(page)).toHaveCount(0);
    });
  });
}
