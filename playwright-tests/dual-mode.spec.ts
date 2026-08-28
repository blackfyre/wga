import { type Page, expect, test } from "@playwright/test";

// Dual mode markup contract. The 1080px narrow gate and the `[data-wide]`
// override are CSS-driven (.wga-dual-narrow / .wga-dual-split / .wga-dual-bar);
// the owning serial integration supplies that stylesheet. Until then these
// checks assert the deterministic markup the CSS keys on: two independently
// addressable panes, the narrow notice and its override link, and ordinary
// no-JavaScript links and forms for the per-pane index controls.

const artistOnePath = "/artists/synthetic-artist-01-ad32608c6e36b2e";
const artistOneArtworkPath =
	"/artists/synthetic-artist-01-ad32608c6e36b2e/synthetic-artwork-01-01-2225c982be1af02";

async function settleEntryAnimation(page: Page) {
	await expect(page.locator("main#mc-area")).toHaveCSS("transform", "none");
}

test("dual mode renders two independent panes and the narrow notice", async ({
	page,
}) => {
	await page.setViewportSize({ width: 834, height: 900 });
	await page.goto("/dual-mode");

	const root = page.locator("#dual-area");
	await expect(root).toHaveAttribute("class", /wga-dual/);
	await expect(root).not.toHaveAttribute("data-wide", "");

	await expect(page.locator("#dual-left")).toHaveCount(1);
	await expect(page.locator("#dual-right")).toHaveCount(1);

	const narrow = page.locator(".wga-dual-narrow");
	await expect(narrow).toHaveCount(1);
	await expect(
		narrow.getByRole("heading", { name: "This mode needs a wide screen" }),
	).toBeVisible();
	await expect(
		narrow.locator("a", { hasText: "OPEN IT ANYWAY" }),
	).toHaveAttribute("href", "/dual-mode?wide=1");
	await expect(
		narrow.locator("a", { hasText: "LEAVE DUAL MODE" }),
	).toHaveAttribute("href", "/artists");
});

test("the wide override is explicit and reversible in the URL", async ({
	page,
}) => {
	await page.goto("/dual-mode?wide=1");
	await expect(page.locator("#dual-area")).toHaveAttribute("data-wide", "");

	// Leaving dual mode is an ordinary link back to the collection.
	await expect(
		page.locator(".wga-dual-bar a", { hasText: "EXIT" }),
	).toHaveAttribute("href", "/artists");
});

test("both panes default to the artist index with independent filters", async ({
	page,
}) => {
	await page.goto("/dual-mode?wide=1");

	await expect(
		page.locator("#dual-left").getByRole("heading", { name: "Artists" }),
	).toBeVisible();
	await expect(
		page.locator("#dual-right").getByRole("heading", { name: "Artists" }),
	).toBeVisible();

	// The per-pane index form is a plain GET form so filtering works without
	// JavaScript; htmx only enhances it.
	for (const side of ["left", "right"]) {
		const form = page.locator(`form#dual-filters-${side}`);
		await expect(form).toHaveAttribute("action", "/dual-mode");
		await expect(form).toHaveAttribute("method", "GET");
	}
});

test("an artist and its work render as complete records with citations", async ({
	page,
}) => {
	await page.goto(
		`/dual-mode?left=${encodeURIComponent(artistOnePath)}&right=${encodeURIComponent(artistOneArtworkPath)}&wide=1`,
	);

	await expect(
		page.locator("#dual-left").getByRole("heading", {
			name: "Synthetic Artist 01",
		}),
	).toBeVisible();
	await expect(page.locator("#dual-left")).toContainText("CITE THIS RECORD");
	await expect(page.locator("#dual-left")).toContainText("BIOGRAPHY");

	await expect(
		page.locator("#dual-right").getByRole("heading", {
			name: "Synthetic Artwork 01-01",
		}),
	).toBeVisible();
	await expect(page.locator("#dual-right")).toContainText("IMAGE SIZE");
	await expect(page.locator("#dual-right")).toContainText("CITE THIS RECORD");
});

test("each pane keeps its own routing toggle and the bar swaps windows", async ({
	page,
}) => {
	await page.goto(
		`/dual-mode?left=${encodeURIComponent(artistOnePath)}&wide=1`,
	);

	// Link routing is a per-pane state toggle rendered as an ordinary link with
	// selected-link semantics (aria-current), not a toggle button (aria-pressed).
	await expect(page.locator("#dual-left a[aria-pressed]")).toHaveCount(0);
	await expect(
		page
			.locator("#dual-left", { hasText: "LINKS OPEN IN" })
			.locator("a[aria-current]"),
	).toHaveCount(1);

	// The top bar exposes the global swap action as a link carrying full state.
	await expect(
		page.locator(".wga-dual-bar a", { hasText: "SWAP WINDOWS" }),
	).toHaveAttribute("href", /^\/dual-mode/);
});

test("dual share URL restores independent pane paths, sizes, and routing", async ({
	page,
}) => {
	const url = `/dual-mode?wide=1&left=${encodeURIComponent(artistOnePath)}&right=${encodeURIComponent(artistOneArtworkPath)}&l_size=small&r_size=large&left_render_to=left`;
	await page.goto(url);
	await expect(page).toHaveURL(/wide=1/);
	await expect(page.locator("#dual-left")).toContainText("SYNTHETIC ARTIST 01");
	await expect(page.locator("#dual-right")).toContainText(
		"Synthetic Artwork 01-01",
	);
	await expect(page.locator("#dual-left a[aria-current]")).toHaveCount(1);
	expect(
		await page.locator("#dual-right a[aria-current]").count(),
	).toBeGreaterThan(0);
	await page.reload();
	await expect(page.locator("#dual-left")).toContainText("SYNTHETIC ARTIST 01");
	await expect(page.locator("#dual-right")).toContainText(
		"Synthetic Artwork 01-01",
	);
});

test.describe("dual mode without JavaScript", () => {
	test.use({ javaScriptEnabled: false });

	test("keeps ordinary index links and a GET filter form", async ({ page }) => {
		await page.goto("/dual-mode");

		await expect(page.locator("#dual-left")).toHaveCount(1);
		await expect(page.locator("#dual-right")).toHaveCount(1);

		await expect(page.locator("form#dual-filters-left")).toHaveAttribute(
			"action",
			"/dual-mode",
		);
		await expect(
			page.locator(".wga-dual-bar a", { hasText: "SWAP WINDOWS" }),
		).toHaveAttribute("href", /^\/dual-mode/);

		const link = page
			.locator("#dual-left a", { hasText: "SYNTHETIC ARTIST 01" })
			.first();
		await expect(link).toHaveAttribute(
			"href",
			`/dual-mode?right=${encodeURIComponent(artistOnePath)}`,
		);
		await settleEntryAnimation(page);
		await link.click();
		await expect(page).toHaveURL(
			`/dual-mode?right=${encodeURIComponent(artistOnePath)}`,
		);
		await expect(
			page.locator("#dual-right").getByRole("heading", {
				name: "Synthetic Artist 01",
			}),
		).toBeVisible();
	});
});

for (const width of [390, 834, 1440]) {
	test(`dual mode has no document overflow at ${width}px`, async ({ page }) => {
		await page.setViewportSize({ width, height: 900 });
		await page.goto("/dual-mode?wide=1");
		const overflow = await page.evaluate(
			() => document.documentElement.scrollWidth > window.innerWidth,
		);
		expect(overflow).toBe(false);
	});

	test(`dual mode has no 200% text overflow at ${width}px`, async ({
		page,
	}) => {
		await page.setViewportSize({ width, height: 900 });
		await page.goto("/dual-mode?wide=1");
		await page.evaluate(() => {
			document.documentElement.style.fontSize = "2em";
		});
		expect(
			await page.evaluate(
				() =>
					document.documentElement.scrollWidth <=
					document.documentElement.clientWidth,
			),
		).toBe(true);
	});
}

test("dual pane route target changes preserve the opposite pane through history", async ({
	page,
}) => {
	await page.goto(
		`/dual-mode?wide=1&left=${encodeURIComponent(artistOnePath)}&right=${encodeURIComponent(artistOneArtworkPath)}`,
	);
	const opposite = page.locator("#dual-right");
	const oppositeHeading = opposite.getByRole("heading", {
		name: "Synthetic Artwork 01-01",
	});
	await expect(oppositeHeading).toBeVisible();
	await expect(opposite).toContainText("IMAGE SIZE");
	const initialURL = page.url();
	const target = page.locator("#dual-left a[aria-current]").first();
	const targetHref = await target.getAttribute("href");
	expect(targetHref).toMatch(/left_render_to=/);
	await target.click();
	await expect(page).toHaveURL(/left_render_to=/);
	expect(page.url()).toContain(
		`right=${encodeURIComponent(artistOneArtworkPath)}`,
	);
	await expect(oppositeHeading).toBeVisible();
	await expect(opposite).toContainText("IMAGE SIZE");
	await page.goBack();
	await expect(page).toHaveURL(initialURL);
	await expect(oppositeHeading).toBeVisible();
	await expect(opposite).toContainText("IMAGE SIZE");
});
