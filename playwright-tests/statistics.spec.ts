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
