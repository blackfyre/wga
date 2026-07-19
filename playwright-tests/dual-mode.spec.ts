import { type Route, expect, test } from "@playwright/test";

const aachenPath = "/artists/aachen-hans-von-139ac2dff50d65c";
const aachenArtworkPath =
	"/artists/aachen-hans-von-139ac2dff50d65c/a-couple-in-a-tavern-4035847eedfacc4";
const koedijckPath = "/artists/koedijck-isaack-3ed9e200b9e8252";

const dualModeURL = (
	left: string,
	right: string,
	leftRenderTo = "right",
	rightRenderTo = "left",
) =>
	`/dual-mode?left=${encodeURIComponent(left)}&right=${encodeURIComponent(right)}&left_render_to=${leftRenderTo}&right_render_to=${rightRenderTo}`;

const expectDualModeState = async (
	page,
	left: string,
	right: string,
	leftRenderTo = "right",
	rightRenderTo = "left",
) => {
	await expect(page).toHaveURL(
		(url) =>
			url.pathname === "/dual-mode" &&
			url.searchParams.get("left") === left &&
			url.searchParams.get("right") === right &&
			url.searchParams.get("left_render_to") === leftRenderTo &&
			url.searchParams.get("right_render_to") === rightRenderTo,
	);
};

const paneTargetControls = (page, side: string) =>
	page.getByRole("navigation", { name: `Open ${side} links in` });

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
		paneTargetControls(page, "left").getByRole("link", {
			name: "Other pane",
		}),
	).toHaveAttribute("aria-current", "true");
	await expect(page.locator("#artistList")).toHaveCount(0);

	await page.getByRole("button", { name: "Choose left" }).click();
	const lookupResponse = page.waitForResponse((response) =>
		response.url().includes("/dual-mode/lookup"),
	);
	await page.getByLabel("Search collection").fill("AACHEN");
	await lookupResponse;
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

test("loads an artwork through the chooser and preserves Dual Mode state", async ({
	page,
}) => {
	await page.goto(dualModeURL(aachenPath, "default", "left", "right"));
	await page.getByRole("button", { name: "Choose right" }).click();
	await page.getByLabel("Search for").selectOption("artwork");

	const lookupResponse = page.waitForResponse((response) =>
		response.url().includes("/dual-mode/lookup"),
	);
	await page.getByLabel("Search collection").fill("A Couple");
	await lookupResponse;

	const artworkResult = page.getByRole("button", {
		name: /A Couple in a Tavern/,
	});
	await expect(artworkResult).toContainText("AACHEN, Hans von");
	await artworkResult.click();

	await expectDualModeState(
		page,
		aachenPath,
		aachenArtworkPath,
		"left",
		"right",
	);
	await expect(page.locator("#right h1")).toContainText("A Couple in a Tavern");
});

test("loads an artwork search result into the selected pane", async ({
	page,
}) => {
	await page.goto(dualModeURL(aachenPath, koedijckPath, "left", "right"));
	await page.getByRole("link", { name: "Search left artworks" }).click();

	await expect(page).toHaveURL(
		(url) =>
			url.pathname === "/artworks" &&
			url.searchParams.get("dual_left") === aachenPath &&
			url.searchParams.get("dual_right") === koedijckPath &&
			url.searchParams.get("dual_target") === "left",
	);

	await page.getByLabel("Title").fill("A Couple in a Tavern");
	const resultsResponse = page.waitForResponse(
		(response) =>
			response.url().includes("/artworks/results") &&
			response.request().headers()["hx-request"] === "true",
	);
	await page.getByRole("button", { name: "Search" }).click();
	await resultsResponse;

	const clear = page.getByRole("link", { name: "Clear" });
	await expect(clear).toHaveAttribute("href", /dual_target=left/);
	const clearResponse = page.waitForResponse(
		(response) =>
			response.url().includes("/artworks?") &&
			response.request().headers()["hx-request"] === "true",
	);
	await clear.click();
	await clearResponse;
	await expect(page).toHaveURL(
		(url) =>
			url.pathname === "/artworks" &&
			url.searchParams.get("dual_left") === aachenPath &&
			url.searchParams.get("dual_right") === koedijckPath &&
			url.searchParams.get("dual_target") === "left",
	);

	await page.getByLabel("Title").fill("A Couple in a Tavern");
	await page.getByRole("button", { name: "Search" }).click();

	await page
		.getByRole("link", {
			name: "Open A Couple in a Tavern in left pane",
		})
		.click();

	await expectDualModeState(
		page,
		aachenArtworkPath,
		koedijckPath,
		"left",
		"right",
	);
	await expect(page.locator("#left h1")).toContainText("A Couple in a Tavern");
});

test("shows accessible lookup query states", async ({ page }) => {
	await page.goto("/dual-mode");
	await page.getByRole("button", { name: "Choose left" }).click();

	const results = page.locator("#dual-lookup-results");
	await expect(results).toContainText("Start typing to search artists.");

	const shortQueryResponse = page.waitForResponse((response) =>
		response.url().includes("/dual-mode/lookup"),
	);
	await page.getByLabel("Search collection").fill("a");
	await shortQueryResponse;
	await expect(results).toContainText(
		"Enter at least two characters to search.",
	);

	const noResultResponse = page.waitForResponse((response) =>
		response.url().includes("/dual-mode/lookup"),
	);
	await page.getByLabel("Search collection").fill("wga-no-match");
	await noResultResponse;
	await expect(results).toContainText("No artists match");
});

test("cancels an active lookup when the chooser closes", async ({ page }) => {
	let delayedRoute: Route | null = null;
	await page.route("**/dual-mode/lookup**", (route) => {
		delayedRoute = route;
	});

	await page.goto("/dual-mode");
	await page.getByRole("button", { name: "Choose left" }).click();

	const lookupRequest = page.waitForRequest((request) =>
		request.url().includes("/dual-mode/lookup"),
	);
	await page.getByLabel("Search collection").fill("AACHEN");
	await lookupRequest;
	await expect.poll(() => delayedRoute).not.toBeNull();
	await page.keyboard.press("Escape");

	if (!delayedRoute) {
		throw new Error("Expected delayed lookup route");
	}

	await delayedRoute.fulfill({
		body: "<p>Delayed lookup result</p>",
		contentType: "text/html",
	});
	await page.waitForTimeout(100);

	await expect(page.locator("#dual-lookup-results")).toContainText(
		"Start typing to search artists.",
	);
});

test("persists a same-pane target choice", async ({ page }) => {
	await page.goto(`/dual-mode?left=${aachenPath}&right=default`);

	const leftTargets = paneTargetControls(page, "left");
	await expect(
		leftTargets.getByRole("link", { name: "Other pane" }),
	).toHaveAttribute("aria-current", "true");
	await leftTargets.getByRole("link", { name: "This pane" }).click();

	await expect(page).toHaveURL(/left_render_to=left/);
	await page.reload();
	await expect(
		leftTargets.getByRole("link", { name: "This pane" }),
	).toHaveAttribute("aria-current", "true");

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

test("loads pane paths through enhanced forms", async ({ page }) => {
	const leftForm = page.getByRole("form", { name: "Load left pane" });
	const rightForm = page.getByRole("form", { name: "Load right pane" });

	await page.goto(dualModeURL(aachenPath, koedijckPath));
	await expect(leftForm.getByLabel("Load left pane path")).toHaveValue(
		aachenPath,
	);
	const leftResponse = page.waitForResponse(
		(response) =>
			response.url().includes("/dual-mode") &&
			response.request().headers()["hx-request"] === "true",
	);
	await leftForm.getByLabel("Load left pane path").fill(koedijckPath);
	await leftForm
		.getByRole("button", { name: "Load left", exact: true })
		.click();
	await leftResponse;
	await expectDualModeState(page, koedijckPath, koedijckPath);
	await expect(page.locator("#left h1")).toContainText("KOEDIJCK, Isaack");

	await page.goto(dualModeURL(aachenPath, koedijckPath, "left", "right"));
	await rightForm.getByLabel("Load right pane path").fill(aachenPath);
	await rightForm
		.getByRole("button", { name: "Load right", exact: true })
		.click();
	await expectDualModeState(page, aachenPath, aachenPath, "left", "right");
	await expect(page.locator("#right h1")).toContainText("AACHEN, Hans von");

	await page.goto(dualModeURL(aachenPath, koedijckPath, "left", "right"));
	await leftForm.getByLabel("Load left pane path").fill(aachenArtworkPath);
	await leftForm
		.getByRole("button", { name: "Load left", exact: true })
		.click();
	await expectDualModeState(
		page,
		aachenArtworkPath,
		koedijckPath,
		"left",
		"right",
	);
	await expect(page.locator("#left h1")).toContainText("A Couple in a Tavern");

	await page.goto(dualModeURL(aachenPath, koedijckPath));
	await leftForm
		.getByLabel("Load left pane path")
		.fill("/pages/privacy-policy");
	await leftForm
		.getByRole("button", { name: "Load left", exact: true })
		.click();
	await expectDualModeState(page, "default", koedijckPath);
	await expect(page.locator("#left")).toContainText(
		"Choose content for comparison",
	);
});

test("updates pane targets through enhanced links", async ({ page }) => {
	await page.goto(dualModeURL(aachenPath, koedijckPath));
	const leftTargets = paneTargetControls(page, "left");
	const targetResponse = page.waitForResponse(
		(response) =>
			response.url().includes("/dual-mode") &&
			response.request().headers()["hx-request"] === "true",
	);
	await leftTargets.getByRole("link", { name: "This pane" }).click();
	await targetResponse;
	await expectDualModeState(page, aachenPath, koedijckPath, "left", "left");
	await expect(
		leftTargets.getByRole("link", { name: "This pane" }),
	).toHaveAttribute("aria-current", "true");

	await page.goto(dualModeURL(aachenPath, koedijckPath));
	const rightTargets = paneTargetControls(page, "right");
	await rightTargets.getByRole("link", { name: "This pane" }).click();
	await expectDualModeState(page, aachenPath, koedijckPath, "right", "right");

	await rightTargets.getByRole("link", { name: "Other pane" }).click();
	await expectDualModeState(page, aachenPath, koedijckPath, "right", "left");
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

	test("changes pane targets through standard hrefs", async ({ page }) => {
		await page.goto(dualModeURL(aachenPath, koedijckPath));
		await paneTargetControls(page, "left")
			.getByRole("link", { name: "This pane" })
			.click();
		await expectDualModeState(page, aachenPath, koedijckPath, "left", "left");

		await page
			.locator("#left")
			.getByRole("link", { name: "Learn More" })
			.first()
			.click();
		await expect(page.locator("#right h1")).toContainText("KOEDIJCK, Isaack");

		await page.goto(dualModeURL(aachenPath, koedijckPath));
		await paneTargetControls(page, "right")
			.getByRole("link", { name: "This pane" })
			.click();
		await expectDualModeState(page, aachenPath, koedijckPath, "right", "right");
	});

	test("loads a right pane path through a standard GET form", async ({
		page,
	}) => {
		await page.goto(dualModeURL(aachenPath, koedijckPath));
		const rightForm = page.getByRole("form", { name: "Load right pane" });
		await rightForm.getByLabel("Load right pane path").fill(aachenPath);
		await rightForm
			.getByRole("button", { name: "Load right", exact: true })
			.click();

		await expectDualModeState(page, aachenPath, aachenPath);
		await expect(page.locator("#right h1")).toContainText("AACHEN, Hans von");
	});

	test("loads a left pane path through a standard GET form", async ({
		page,
	}) => {
		await page.goto(dualModeURL(aachenPath, koedijckPath));
		const leftForm = page.getByRole("form", { name: "Load left pane" });
		await leftForm.getByLabel("Load left pane path").fill(koedijckPath);
		await leftForm
			.getByRole("button", { name: "Load left", exact: true })
			.click();

		await expectDualModeState(page, koedijckPath, koedijckPath);
		await expect(page.locator("#left h1")).toContainText("KOEDIJCK, Isaack");
	});

	test("loads an artwork search result through standard links", async ({
		page,
	}) => {
		await page.goto(dualModeURL(aachenPath, koedijckPath, "left", "right"));
		await page.getByRole("link", { name: "Search right artworks" }).click();

		await page.getByLabel("Title").fill("A Couple in a Tavern");
		await page.getByRole("button", { name: "Search" }).click();
		await page
			.getByRole("link", {
				name: "Open A Couple in a Tavern in right pane",
			})
			.click();

		await expectDualModeState(
			page,
			aachenPath,
			aachenArtworkPath,
			"left",
			"right",
		);
		await expect(page.locator("#right h1")).toContainText(
			"A Couple in a Tavern",
		);
	});
});
