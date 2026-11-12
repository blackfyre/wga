import { expect, type Page, test } from "@playwright/test";

// findPerson returns the page's Person node, searching each JSON-LD block's
// @graph as well as top-level nodes.
async function findPerson(
  page: Page,
): Promise<Record<string, unknown> | undefined> {
  const scripts = await page
    .locator('script[type="application/ld+json"]')
    .allTextContents();
  return scripts
    .flatMap((text) => {
      const data = JSON.parse(text);
      return Array.isArray(data["@graph"]) ? data["@graph"] : [data];
    })
    .find((node) => node["@type"] === "Person");
}

test("artist without a portrait retains the labelled fallback and artwork metadata", async ({
  page,
}) => {
  await page.goto("/artists/synthetic-artist-01-ad32608c6e36b2e");
  const portrait = page.locator("article > figure").first();
  await expect(portrait).toHaveText("PORTRAIT — SYNTHETIC ARTIST 01");
  await expect(portrait.locator("img")).toHaveCount(0);
  const person = await findPerson(page);
  expect(person).toBeDefined();
  expect(person).not.toHaveProperty("image");
  await expect(page.locator('meta[property="og:image"]')).toHaveAttribute(
    "content",
    /\/api\/files\/artworks\//,
  );
});

// Run against an artist whose file bytes have been staged independently of
// its database reference; this also supports read-only staging verification.
test("staged artist portrait loads and matches identity metadata", async ({
  page,
}) => {
  const path = process.env.WGA_PORTRAIT_ARTIST_PATH;
  test.skip(!path, "Set WGA_PORTRAIT_ARTIST_PATH to a portrait-bearing artist");
  await page.goto(path!);
  const image = page.locator("article > figure img").first();
  await expect(image).toBeVisible();
  await expect(image).toHaveAttribute(
    "alt",
    (await page.locator("h1").innerText()).trim(),
  );
  await expect
    .poll(() =>
      image.evaluate((element: HTMLImageElement) => element.naturalWidth),
    )
    .toBeGreaterThan(0);
  const source = await image.evaluate(
    (element: HTMLImageElement) => element.src,
  );
  expect(source).toContain("/api/files/artists/");
  await expect(page.locator('meta[property="og:image"]')).toHaveAttribute(
    "content",
    source,
  );
  const person = await findPerson(page);
  expect(person?.image).toBe(source);
});
