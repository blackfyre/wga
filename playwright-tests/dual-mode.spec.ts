import { expect, test } from "@playwright/test";

const aachenPath = "/artists/aachen-hans-von-139ac2dff50d65c";

test("loads an artist through the chooser and opens its artwork in the other pane", async ({
	page,
}) => {
	await page.goto("/dual-mode");

	await expect(page.locator("#left")).toContainText(
		"Choose content for comparison",
	);
	await expect(page.locator("#right")).toContainText(
		"Choose content for comparison",
	);
	await expect(
		page.getByLabel("Open left links in the other pane"),
	).toBeChecked();

	await page.getByRole("button", { name: "Choose left" }).click();
	await page.getByPlaceholder("Filter artists").fill("AACHEN");
	await page
		.getByRole("button", { name: "AACHEN, Hans von", exact: true })
		.click();

	await expect(page).toHaveURL(/\/dual-mode\?.*left=/);
	await expect(page.locator("#left h1")).toContainText("AACHEN, Hans von");

	await page
		.locator("#left")
		.getByRole("link", { name: "Learn More" })
		.first()
		.click();

	await expect(page.locator("#dual-area")).toHaveCount(1);
	await expect(page.locator("#left h1")).toContainText("AACHEN, Hans von");
	await expect(page.locator("#right")).not.toContainText(
		"Choose content for comparison",
	);
});

test("persists a same-pane target choice", async ({ page }) => {
	await page.goto(`/dual-mode?left=${aachenPath}&right=default`);

	const targetToggle = page.getByLabel("Open left links in the other pane");
	await expect(targetToggle).toBeChecked();
	await targetToggle.uncheck();

	await expect(page).toHaveURL(/left_render_to=left/);
	await page.reload();
	await expect(targetToggle).not.toBeChecked();

	await page
		.locator("#left")
		.getByRole("link", { name: "Learn More" })
		.first()
		.click();

	await expect(page.locator("#dual-area")).toHaveCount(1);
	await expect(page.locator("#left .card-actions")).toHaveCount(0);
	await expect(page.locator("#right")).toContainText(
		"Choose content for comparison",
	);
});

test("keeps a valid pane when the other selected record is missing", async ({
	page,
}) => {
	await page.goto(
		`/dual-mode?left=/artists/missing-000000000000000&right=${aachenPath}`,
	);

	await expect(page.locator("#left")).toContainText(
		"Choose content for comparison",
	);
	await expect(page.locator("#right h1")).toContainText("AACHEN, Hans von");
});

test("shows the desktop-only message on a small screen", async ({ page }) => {
	await page.setViewportSize({ width: 767, height: 900 });
	await page.goto("/dual-mode");

	await expect(page.getByRole("alert")).toContainText(
		"Dual Mode is desktop-only",
	);
	await expect(page.locator("#dual-area")).toBeHidden();
});
