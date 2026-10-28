import { expect, type Page, test } from "@playwright/test";

// Embedded synthetic identities make this journey executable in a fresh default
// application without an external producer database.
const artistRecordPath = "/artists/synthetic-artist-02-2236bdd57f7492e";

const suppliedCommentaryPath =
	"/artists/synthetic-artist-02-2236bdd57f7492e/selections/5ef9746d522357d";

const missingCommentaryPath =
	"/artists/synthetic-artist-02-2236bdd57f7492e/selections/ff3b947b8303fbd";

const suppliedCommentaryTitle = "Synthetic selection with commentary";
const missingCommentaryTitle = "Synthetic selection without commentary";

// The `main#mc-area` element runs a 280ms entry animation that releases its
// transform once complete. Wait for that before clicking so the target is
// stable while JavaScript is disabled.
async function settleEntryAnimation(page: Page) {
	await expect(page.locator("main#mc-area")).toHaveCSS("transform", "none");
}

test.describe("selection journey without JavaScript", () => {
	test.use({ javaScriptEnabled: false });

	test("artist record renders curated selection previews as ordinary links", async ({
		page,
	}) => {
		await page.goto(artistRecordPath);

		await expect(
			page.getByRole("heading", { name: "CURATED SELECTIONS" }),
		).toBeVisible();

		const firstPreview = page
			.locator("article", {
				hasText: suppliedCommentaryTitle,
			})
			.first();
		await expect(
			firstPreview.getByRole("heading", {
				name: suppliedCommentaryTitle,
			}),
		).toBeVisible();
		await expect(
			firstPreview.getByText(/SELECTED · .* CATALOGUED/),
		).toBeVisible();
		await expect(
			firstPreview.getByText("2 SELECTED · 4 CATALOGUED"),
		).toBeVisible();

		const openLink = firstPreview.getByRole("link", {
			name: /OPEN SELECTION/,
		});
		await expect(openLink).toHaveAttribute("href", suppliedCommentaryPath);

		const toc = page.getByRole("navigation", { name: "On this page" });
		await expect(toc.getByRole("link", { name: "Biography" })).toHaveAttribute(
			"href",
			"#biography",
		);
		const selectionLink = toc.getByRole("link", {
			name: /Synthetic selection with commentary · 2 works/,
		});
		await selectionLink.focus();
		await expect(selectionLink).toBeFocused();
		await page.keyboard.press("Enter");
		expect(new URL(page.url()).hash).toMatch(/^#selection-/);
		await expect(
			toc.getByRole("link", { name: "Cite This Record" }),
		).toHaveAttribute("href", "#cite-this-record");
	});

	test("selection preview opens the dedicated selection page", async ({
		page,
	}) => {
		await page.goto(artistRecordPath);
		await settleEntryAnimation(page);

		const preview = page
			.locator("article", {
				hasText: suppliedCommentaryTitle,
			})
			.first();
		await preview.getByRole("link", { name: /OPEN SELECTION/ }).click();

		await expect(page).toHaveURL(suppliedCommentaryPath);
		await expect(page.locator("h1")).toHaveText(suppliedCommentaryTitle);
		await expect(page.getByText("21 — SELECTION")).toBeVisible();
		await expect(
			page.locator("nav[aria-label='Breadcrumb'] a").last(),
		).toHaveText("Synthetic Artist 02");
	});

	test("dedicated selection page renders supplied commentary, citation, and ordinary holding link", async ({
		page,
	}) => {
		await page.goto(suppliedCommentaryPath);

		await expect(
			page.getByRole("heading", { name: "COMMENTARY", exact: true }),
		).toBeVisible();
		await expect(
			page.getByText("Synthetic fixture commentary for browser coverage."),
		).toBeVisible();

		await expect(
			page.getByRole("heading", { name: "CITE THIS RECORD — BIBTEX" }),
		).toBeVisible();
		await expect(page.locator("pre#bibtex-wga-5ef9746d522357d")).toContainText(
			"@online{wga-5ef9746d522357d,",
		);

		const holdingLink = page.getByRole("link", {
			name: /VIEW FULL HOLDING/,
		});
		await expect(holdingLink).toHaveAttribute(
			"href",
			"/artworks?artist=SYNTHETIC+ARTIST+02",
		);
		await expect(holdingLink).not.toHaveAttribute("hx-get", /.*/);

		await expect(page.getByText("OTHER SELECTIONS")).toBeVisible();
		await expect(page.getByText(missingCommentaryTitle)).toBeVisible();
		await expect(page.locator("ul[data-selection-plate-grid]")).toHaveCount(1);
	});

	test("dedicated selection page states missing commentary honestly", async ({
		page,
	}) => {
		await page.goto(missingCommentaryPath);

		await expect(page.locator("h1")).toHaveText(missingCommentaryTitle);
		await expect(
			page.getByText("Commentary is unavailable for this selection."),
		).toBeVisible();
		await expect(
			page.getByRole("heading", { name: "CITE THIS RECORD — BIBTEX" }),
		).toBeVisible();
	});
});

for (const width of [390, 834, 1440]) {
	test(`artist navigation rail composes with the measured stack at ${width}px`, async ({
		page,
	}) => {
		await page.setViewportSize({ width, height: 900 });
		await page.goto(artistRecordPath);
		const toc = page.getByRole("navigation", { name: "On this page" });
		const rail = toc.locator("xpath=ancestor::aside");
		await expect(rail).toHaveCSS(
			"position",
			width >= 834 ? "sticky" : "static",
		);

		await page.evaluate(() => {
			const tray = document.createElement("div");
			tray.className = "wga-bottom-stack-item";
			tray.dataset.wgaBottomStackItem = "test";
			tray.dataset.wgaBottomStackOrder = "10";
			tray.style.cssText =
				"position:fixed;inset-inline:0;height:88px;background:black";
			document.body.append(tray);
		});
		await expect
			.poll(() =>
				page.evaluate(() =>
					Number.parseFloat(
						getComputedStyle(document.body).getPropertyValue(
							"--wga-bottom-stack-height",
						),
					),
				),
			)
			.toBeGreaterThanOrEqual(88);
		if (width >= 834) {
			const maxHeight = await rail.evaluate((element) =>
				Number.parseFloat(getComputedStyle(element).maxHeight),
			);
			expect(maxHeight).toBeLessThanOrEqual(900 - 88 - 48);
		}

		await toc.getByRole("link", { name: "Cite This Record" }).click();
		expect(new URL(page.url()).hash).toBe("#cite-this-record");
	});
}

type PlateBox = {
	left: number;
	top: number;
	imageWidth: number;
	imageHeight: number;
	viewWorkTop: number;
	itineraryTop: number;
};

// Measures the plates of the first selection plate grid on the page.
async function measureSelectionPlates(page: Page): Promise<PlateBox[]> {
	const grid = page.locator("ul[data-selection-plate-grid]").first();
	await expect(grid.locator("article").first()).toBeVisible();
	return grid.locator("article").evaluateAll((plates) =>
		plates.map((plate) => {
			const box = plate.getBoundingClientRect();
			const image = plate.querySelector("figure")?.getBoundingClientRect();
			const viewWork = plate
				.querySelector("[data-plate-view-work]")
				?.getBoundingClientRect();
			const itinerary = plate
				.querySelector("[data-plate-itinerary] button")
				?.getBoundingClientRect();
			return {
				left: Math.round(box.left),
				top: Math.round(box.top + window.scrollY),
				imageWidth: Math.round(image?.width ?? -1),
				imageHeight: Math.round(image?.height ?? -1),
				viewWorkTop: Math.round((viewWork?.top ?? -1) + window.scrollY),
				itineraryTop: Math.round((itinerary?.top ?? -1) + window.scrollY),
			};
		}),
	);
}

for (const width of [390, 834, 1440]) {
	test(`selection plates match and align at ${width}px`, async ({ page }) => {
		await page.setViewportSize({ width, height: 900 });

		// Make the first row's plates differ in text height and in itinerary
		// action height, as a long title or a mixed itinerary state would.
		const unevenFirstRow = async () => {
			const plates = page.locator("ul[data-selection-plate-grid] article");
			await plates
				.nth(0)
				.locator("h2")
				.evaluate((title) => {
					title.textContent =
						"A deliberately long synthetic title that wraps over several lines";
				});
			await plates
				.nth(1)
				.locator("[data-plate-itinerary] button")
				.first()
				.evaluate((button) => {
					button.textContent =
						"A deliberately long itinerary label that wraps over several lines";
				});
		};

		await page.goto(artistRecordPath);
		const previewPlates = await measureSelectionPlates(page);
		await page.goto(suppliedCommentaryPath);
		await unevenFirstRow();
		const pagePlates = await measureSelectionPlates(page);

		expect(pagePlates.length).toBeGreaterThan(1);
		for (const plate of [...previewPlates, ...pagePlates]) {
			expect(plate.imageWidth).toBe(pagePlates[0].imageWidth);
			expect(plate.imageHeight).toBe(pagePlates[0].imageHeight);
		}

		const firstRow = pagePlates.filter(
			(plate) => plate.top === pagePlates[0].top,
		);
		expect(firstRow.length).toBeGreaterThan(1);
		for (const plate of firstRow) {
			expect(plate.viewWorkTop).toBe(firstRow[0].viewWorkTop);
			expect(plate.itineraryTop).toBe(firstRow[0].itineraryTop);
		}
	});
}

test("selection plates show the stored holding only in selections", async ({
	page,
}) => {
	await page.goto(suppliedCommentaryPath);
	const plates = page.locator("ul[data-selection-plate-grid] article");
	await expect(
		plates
			.filter({ hasText: "Synthetic Artwork 02-01" })
			.locator("[data-plate-location]"),
	).toHaveText("Synthetic Civic Gallery");
	await expect(
		plates
			.filter({ hasText: "Synthetic Artwork 02-02" })
			.locator("[data-plate-location]"),
	).toHaveText("Synthetic Harbour Collection");

	await page.goto(artistRecordPath);
	await expect(
		page.locator("ul[data-selection-plate-grid] [data-plate-location]").first(),
	).toBeVisible();
});
