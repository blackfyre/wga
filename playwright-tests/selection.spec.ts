import { type Page, expect, test } from "@playwright/test";

// Embedded synthetic identities make this journey executable in a fresh default
// application without an external producer database.
const artistRecordPath = "/artists/synthetic-artist-02-2236bdd57f7492e";

const suppliedCommentaryPath =
	"/artists/synthetic-artist-02-2236bdd57f7492e/selections/ra01b4fda068382";

const missingCommentaryPath =
	"/artists/synthetic-artist-02-2236bdd57f7492e/selections/r71ee5b06e7865f";

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

		const openLink = firstPreview.getByRole("link", {
			name: /OPEN SELECTION/,
		});
		await expect(openLink).toHaveAttribute("href", suppliedCommentaryPath);
		await expect(openLink).not.toHaveAttribute("hx-get", /.*/);
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
		await expect(page.getByText("03 — SELECTION")).toBeVisible();
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
		await expect(page.locator("pre#bibtex-wga-ra01b4fda068382")).toContainText(
			"@online{wga-ra01b4fda068382,",
		);

		const holdingLink = page.getByRole("link", {
			name: /VIEW FULL HOLDING/,
		});
		await expect(holdingLink).toHaveAttribute(
			"href",
			"/artworks?artist=Synthetic+Artist+02",
		);
		await expect(holdingLink).not.toHaveAttribute("hx-get", /.*/);

		await expect(page.getByText("OTHER SELECTIONS")).toBeVisible();
		await expect(page.getByText(missingCommentaryTitle)).toBeVisible();
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
