import { expect, test } from "@playwright/test";

test("statistics charts provide text summaries", async ({ page }) => {
  await page.goto("/statistics");
  await expect(page.getByRole("heading", { name: "Statistics" })).toBeVisible();

  const chartSummaries = {
    "art-form-chart": "art-form-summary",
    "artworks-by-period-chart": "artworks-period-summary",
    "artists-by-period-chart": "artists-period-summary",
  };

  for (const [chartID, summaryID] of Object.entries(chartSummaries)) {
    await expect(page.locator(`#${chartID}`)).toHaveAttribute(
      "aria-describedby",
      summaryID,
    );
    await expect(page.locator(`#${summaryID}`)).toBeVisible();
  }
});

test("art form chart and summary use the reference desktop layout", async ({
  page,
}) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.goto("/statistics");
  await page.waitForFunction(
    () =>
      document.getElementById("art-form-chart")?.getBoundingClientRect()
        .width === 240,
  );

  const summary = await page.locator("#art-form-summary").boundingBox();
  expect(summary?.x).toBeCloseTo(436, 0);
  await expect(page.locator(".art-form-label").first()).toBeVisible();
	const periodChart = page.locator("#artworks-by-period-chart");
	await page.waitForFunction(() => {
		const box = document
			.getElementById("artworks-by-period-chart")
			?.getBoundingClientRect();
		return box !== undefined && box.width > 0 && box.width / box.height === 2;
	});
	const chartBox = await periodChart.boundingBox();
	expect(chartBox?.width || 0).toBeCloseTo((chartBox?.height || 0) * 2, 0);
});
