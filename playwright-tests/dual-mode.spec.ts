import { type Route, expect, test } from "@playwright/test";

const artistOnePath = "/artists/synthetic-artist-01-ad32608c6e36b2e";
const artistOneArtworkPath =
	"/artists/synthetic-artist-01-ad32608c6e36b2e/synthetic-artwork-01-01-2225c982be1af02";
const artistTwoPath = "/artists/synthetic-artist-02-2236bdd57f7492e";

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
	await page.getByLabel("Search collection").fill("Synthetic Artist 01");
	await lookupResponse;
	await page
		.getByRole("button", { name: "Synthetic Artist 01", exact: true })
		.click();

	await expect(page).toHaveURL(/\/dual-mode\?.*left=/);
	await expect(page.locator("#left h1")).toContainText("Synthetic Artist 01");

	await page
		.locator("#left")
		.getByRole("link", { name: "Learn More" })
		.first()
		.click();

	await expect(page.locator("#dual-area")).toHaveCount(1);
	await expect(page.locator("#left h1")).toContainText("Synthetic Artist 01");
	await expect(page.locator("#right")).not.toContainText(
		"Choose content for comparison",
	);
});

test("loads an artwork through the chooser and preserves Dual Mode state", async ({
	page,
}) => {
	await page.goto(dualModeURL(artistOnePath, "default", "left", "right"));
	await page.getByRole("button", { name: "Choose right" }).click();
	await page.getByLabel("Search for").selectOption("artwork");

	const lookupResponse = page.waitForResponse((response) =>
		response.url().includes("/dual-mode/lookup"),
	);
	await page.getByLabel("Search collection").fill("Synthetic Artwork");
	await lookupResponse;

	const artworkResult = page.getByRole("button", {
		name: /Synthetic Artwork 01-01/,
	});
	await expect(artworkResult).toContainText("Synthetic Artist 01");
	await artworkResult.click();

	await expectDualModeState(
		page,
		artistOnePath,
		artistOneArtworkPath,
		"left",
		"right",
	);
	await expect(page.locator("#right h1")).toContainText(
		"Synthetic Artwork 01-01",
	);
});

test("loads an artwork search result into the selected pane", async ({
	page,
}) => {
	await page.goto(dualModeURL(artistOnePath, artistTwoPath, "left", "right"));
	await page.getByRole("link", { name: "Search left artworks" }).click();

	await expect(page).toHaveURL(
		(url) =>
			url.pathname === "/artworks" &&
			url.searchParams.get("dual_left") === artistOnePath &&
			url.searchParams.get("dual_right") === artistTwoPath &&
			url.searchParams.get("dual_target") === "left",
	);

	await page.getByLabel("Title").fill("Synthetic Artwork 01-01");
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
			url.searchParams.get("dual_left") === artistOnePath &&
			url.searchParams.get("dual_right") === artistTwoPath &&
			url.searchParams.get("dual_target") === "left",
	);

	await page.getByLabel("Title").fill("Synthetic Artwork 01-01");
	await page.getByRole("button", { name: "Search" }).click();

	await page
		.getByRole("link", {
			name: "Open Synthetic Artwork 01-01 in left pane",
		})
		.click();

	await expectDualModeState(
		page,
		artistOneArtworkPath,
		artistTwoPath,
		"left",
		"right",
	);
	await expect(page.locator("#left h1")).toContainText(
		"Synthetic Artwork 01-01",
	);
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
	await page.getByLabel("Search collection").fill("Synthetic Artist 01");
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
	await page.goto(`/dual-mode?left=${artistOnePath}&right=default`);

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
		`/dual-mode?left=/artists/missing-000000000000000&right=${artistOnePath}`,
	);

	await expect(page.locator("#left")).toContainText(
		"Choose content for comparison",
	);
	await expect(page.locator("#right h1")).toContainText("Synthetic Artist 01");
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

	await page.goto(dualModeURL(artistOnePath, artistTwoPath));
	const reverseResponse = page.waitForResponse(
		(response) =>
			response.url().includes("/dual-mode") &&
			response.request().headers()["hx-request"] === "true",
	);
	await operations
		.getByRole("link", { name: "Reverse panes", exact: true })
		.click();
	await reverseResponse;
	await expect(page.locator("#left h1")).toContainText("Synthetic Artist 02");
	await expect(page.locator("#right h1")).toContainText("Synthetic Artist 01");
	await expectDualModeState(page, artistTwoPath, artistOnePath);

	await page.goto(dualModeURL(artistOnePath, artistTwoPath));
	await operations
		.getByRole("link", { name: "Copy left to right", exact: true })
		.click();
	await expectDualModeState(page, artistOnePath, artistOnePath);
	await expect(page.locator("#right h1")).toContainText("Synthetic Artist 01");

	await page.goto(dualModeURL(artistOnePath, artistTwoPath));
	await operations
		.getByRole("link", { name: "Copy right to left", exact: true })
		.click();
	await expectDualModeState(page, artistTwoPath, artistTwoPath);
	await expect(page.locator("#left h1")).toContainText("Synthetic Artist 02");

	await page.goto(dualModeURL(artistOnePath, artistTwoPath));
	await operations
		.getByRole("link", { name: "Clear left", exact: true })
		.click();
	await expectDualModeState(page, "default", artistTwoPath);
	await expect(page.locator("#left")).toContainText(
		"Choose content for comparison",
	);

	await page.goto(dualModeURL(artistOnePath, artistTwoPath));
	await operations
		.getByRole("link", { name: "Clear right", exact: true })
		.click();
	await expectDualModeState(page, artistOnePath, "default");
	await expect(page.locator("#right")).toContainText(
		"Choose content for comparison",
	);

	await page.goto(dualModeURL(artistOnePath, artistTwoPath));
	await operations
		.getByRole("link", { name: "Standard view", exact: true })
		.click();
	await expect(page).toHaveURL(/\/$/);
});

test("loads pane paths through enhanced forms", async ({ page }) => {
	const leftForm = page.getByRole("form", { name: "Load left pane" });
	const rightForm = page.getByRole("form", { name: "Load right pane" });

	await page.goto(dualModeURL(artistOnePath, artistTwoPath));
	await expect(leftForm.getByLabel("Load left pane path")).toHaveValue(
		artistOnePath,
	);
	const leftResponse = page.waitForResponse(
		(response) =>
			response.url().includes("/dual-mode") &&
			response.request().headers()["hx-request"] === "true",
	);
	await leftForm.getByLabel("Load left pane path").fill(artistTwoPath);
	await leftForm
		.getByRole("button", { name: "Load left", exact: true })
		.click();
	await leftResponse;
	await expectDualModeState(page, artistTwoPath, artistTwoPath);
	await expect(page.locator("#left h1")).toContainText("Synthetic Artist 02");

	await page.goto(dualModeURL(artistOnePath, artistTwoPath, "left", "right"));
	await rightForm.getByLabel("Load right pane path").fill(artistOnePath);
	await rightForm
		.getByRole("button", { name: "Load right", exact: true })
		.click();
	await expectDualModeState(
		page,
		artistOnePath,
		artistOnePath,
		"left",
		"right",
	);
	await expect(page.locator("#right h1")).toContainText("Synthetic Artist 01");

	await page.goto(dualModeURL(artistOnePath, artistTwoPath, "left", "right"));
	await leftForm.getByLabel("Load left pane path").fill(artistOneArtworkPath);
	await leftForm
		.getByRole("button", { name: "Load left", exact: true })
		.click();
	await expectDualModeState(
		page,
		artistOneArtworkPath,
		artistTwoPath,
		"left",
		"right",
	);
	await expect(page.locator("#left h1")).toContainText(
		"Synthetic Artwork 01-01",
	);

	await page.goto(dualModeURL(artistOnePath, artistTwoPath));
	await leftForm
		.getByLabel("Load left pane path")
		.fill("/pages/privacy-policy");
	await leftForm
		.getByRole("button", { name: "Load left", exact: true })
		.click();
	await expectDualModeState(page, "default", artistTwoPath);
	await expect(page.locator("#left")).toContainText(
		"Choose content for comparison",
	);
});

test("updates pane targets through enhanced links", async ({ page }) => {
	await page.goto(dualModeURL(artistOnePath, artistTwoPath));
	const leftTargets = paneTargetControls(page, "left");
	const targetResponse = page.waitForResponse(
		(response) =>
			response.url().includes("/dual-mode") &&
			response.request().headers()["hx-request"] === "true",
	);
	await leftTargets.getByRole("link", { name: "This pane" }).click();
	await targetResponse;
	await expectDualModeState(page, artistOnePath, artistTwoPath, "left", "left");
	await expect(
		leftTargets.getByRole("link", { name: "This pane" }),
	).toHaveAttribute("aria-current", "true");

	await page.goto(dualModeURL(artistOnePath, artistTwoPath));
	const rightTargets = paneTargetControls(page, "right");
	await rightTargets.getByRole("link", { name: "This pane" }).click();
	await expectDualModeState(
		page,
		artistOnePath,
		artistTwoPath,
		"right",
		"right",
	);

	await rightTargets.getByRole("link", { name: "Other pane" }).click();
	await expectDualModeState(
		page,
		artistOnePath,
		artistTwoPath,
		"right",
		"left",
	);
});

test.describe("Dual Mode operations without JavaScript", () => {
	test.use({ javaScriptEnabled: false });

	test("falls back to the reverse-pane href", async ({ page }) => {
		await page.goto(dualModeURL(artistOnePath, artistTwoPath));
		await page
			.getByRole("navigation", { name: "Pane operations" })
			.getByRole("link", { name: "Reverse panes", exact: true })
			.click();

		await expectDualModeState(page, artistTwoPath, artistOnePath);
		await expect(page.locator("#left h1")).toContainText("Synthetic Artist 02");
		await expect(page.locator("#right h1")).toContainText(
			"Synthetic Artist 01",
		);
	});

	test("changes pane targets through standard hrefs", async ({ page }) => {
		await page.goto(dualModeURL(artistOnePath, artistTwoPath));
		await paneTargetControls(page, "left")
			.getByRole("link", { name: "This pane" })
			.click();
		await expectDualModeState(
			page,
			artistOnePath,
			artistTwoPath,
			"left",
			"left",
		);

		await page
			.locator("#left")
			.getByRole("link", { name: "Learn More" })
			.first()
			.click();
		await expect(page.locator("#right h1")).toContainText(
			"Synthetic Artist 02",
		);

		await page.goto(dualModeURL(artistOnePath, artistTwoPath));
		await paneTargetControls(page, "right")
			.getByRole("link", { name: "This pane" })
			.click();
		await expectDualModeState(
			page,
			artistOnePath,
			artistTwoPath,
			"right",
			"right",
		);
	});

	test("loads a right pane path through a standard GET form", async ({
		page,
	}) => {
		await page.goto(dualModeURL(artistOnePath, artistTwoPath));
		const rightForm = page.getByRole("form", { name: "Load right pane" });
		await rightForm.getByLabel("Load right pane path").fill(artistOnePath);
		await rightForm
			.getByRole("button", { name: "Load right", exact: true })
			.click();

		await expectDualModeState(page, artistOnePath, artistOnePath);
		await expect(page.locator("#right h1")).toContainText(
			"Synthetic Artist 01",
		);
	});

	test("loads a left pane path through a standard GET form", async ({
		page,
	}) => {
		await page.goto(dualModeURL(artistOnePath, artistTwoPath));
		const leftForm = page.getByRole("form", { name: "Load left pane" });
		await leftForm.getByLabel("Load left pane path").fill(artistTwoPath);
		await leftForm
			.getByRole("button", { name: "Load left", exact: true })
			.click();

		await expectDualModeState(page, artistTwoPath, artistTwoPath);
		await expect(page.locator("#left h1")).toContainText("Synthetic Artist 02");
	});

	test("loads an artwork search result through standard links", async ({
		page,
	}) => {
		await page.goto(dualModeURL(artistOnePath, artistTwoPath, "left", "right"));
		await page.getByRole("link", { name: "Search right artworks" }).click();

		await page.getByLabel("Title").fill("Synthetic Artwork 01-01");
		await page.getByRole("button", { name: "Search" }).click();
		await page
			.getByRole("link", {
				name: "Open Synthetic Artwork 01-01 in right pane",
			})
			.click();

		await expectDualModeState(
			page,
			artistOnePath,
			artistOneArtworkPath,
			"left",
			"right",
		);
		await expect(page.locator("#right h1")).toContainText(
			"Synthetic Artwork 01-01",
		);
	});
});
