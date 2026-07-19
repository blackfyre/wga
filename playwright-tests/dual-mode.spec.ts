import { expect, test } from "@playwright/test";

const aachenPath = "/artists/aachen-hans-von-139ac2dff50d65c";
const koedijckPath = "/artists/koedijck-isaack-3ed9e200b9e8252";

const dualModeURL = (left: string, right: string) =>
	`/dual-mode?left=${encodeURIComponent(left)}&right=${encodeURIComponent(right)}&left_render_to=right&right_render_to=left`;

const expectDualModeState = async (page, left: string, right: string) => {
	await expect(page).toHaveURL(
		(url) =>
			url.pathname === "/dual-mode" &&
			url.searchParams.get("left") === left &&
			url.searchParams.get("right") === right &&
			url.searchParams.get("left_render_to") === "right" &&
			url.searchParams.get("right_render_to") === "left",
	);
};

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

test("updates pane state through enhanced operation links", async ({
	page,
}) => {
	const operations = page.getByRole("navigation", { name: "Pane operations" });

	await page.goto(dualModeURL(aachenPath, koedijckPath));
	const reverseResponse = page.waitForResponse(
		(response) =>
			response.url().includes("/dual-mode") &&
			response.request().headers()["hx-request"] === "true",
	);
	await operations
		.getByRole("link", { name: "Reverse panes", exact: true })
		.click();
	await reverseResponse;
	await expect(page.locator("#left h1")).toContainText("KOEDIJCK, Isaack");
	await expect(page.locator("#right h1")).toContainText("AACHEN, Hans von");
	await expectDualModeState(page, koedijckPath, aachenPath);

	await page.goto(dualModeURL(aachenPath, koedijckPath));
	await operations
		.getByRole("link", { name: "Copy left to right", exact: true })
		.click();
	await expectDualModeState(page, aachenPath, aachenPath);
	await expect(page.locator("#right h1")).toContainText("AACHEN, Hans von");

	await page.goto(dualModeURL(aachenPath, koedijckPath));
	await operations
		.getByRole("link", { name: "Copy right to left", exact: true })
		.click();
	await expectDualModeState(page, koedijckPath, koedijckPath);
	await expect(page.locator("#left h1")).toContainText("KOEDIJCK, Isaack");

	await page.goto(dualModeURL(aachenPath, koedijckPath));
	await operations
		.getByRole("link", { name: "Clear left", exact: true })
		.click();
	await expectDualModeState(page, "default", koedijckPath);
	await expect(page.locator("#left")).toContainText(
		"Choose content for comparison",
	);

	await page.goto(dualModeURL(aachenPath, koedijckPath));
	await operations
		.getByRole("link", { name: "Clear right", exact: true })
		.click();
	await expectDualModeState(page, aachenPath, "default");
	await expect(page.locator("#right")).toContainText(
		"Choose content for comparison",
	);

	await page.goto(dualModeURL(aachenPath, koedijckPath));
	await operations
		.getByRole("link", { name: "Standard view", exact: true })
		.click();
	await expect(page).toHaveURL(/\/$/);
});

test.describe("Dual Mode operations without JavaScript", () => {
	test.use({ javaScriptEnabled: false });

	test("falls back to the reverse-pane href", async ({ page }) => {
		await page.goto(dualModeURL(aachenPath, koedijckPath));
		await page
			.getByRole("navigation", { name: "Pane operations" })
			.getByRole("link", { name: "Reverse panes", exact: true })
			.click();

		await expectDualModeState(page, koedijckPath, aachenPath);
		await expect(page.locator("#left h1")).toContainText("KOEDIJCK, Isaack");
		await expect(page.locator("#right h1")).toContainText("AACHEN, Hans von");
	});
});
