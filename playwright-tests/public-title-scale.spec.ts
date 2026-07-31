import { expect, test } from "@playwright/test";

for (const [path, title] of [
  ["/artworks", "Artworks"],
  ["/statistics", "Statistics"],
  ["/pages/about", "About"],
] as const) {
  test(`${title} uses the reference title scale on mobile`, async ({
    page,
  }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto(path);

    await expect(page.getByRole("heading", { name: title })).toHaveCSS(
      "font-size",
      "44px",
    );
  });
}
