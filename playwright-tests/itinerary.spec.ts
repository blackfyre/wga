import { type Locator, type Page, expect, test } from "@playwright/test";

const viewports = [390, 834, 1440] as const;

// The listed itinerary's public URL, captured by the serial publication test
// and reused by the viewer tests (cookie-less) so they add no new draft.
let listedShareURL = "";

function uniquePublicationTitle(label: string): string {
	return `${label} ${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}

async function assertNoHorizontalOverflow(page: Page) {
	const dimensions = await page.evaluate(() => ({
		clientWidth: document.documentElement.clientWidth,
		scrollWidth: document.documentElement.scrollWidth,
	}));
	expect(dimensions.scrollWidth).toBeLessThanOrEqual(dimensions.clientWidth);
}

async function countItineraryPrefetch(page: Page): Promise<number> {
	return page.evaluate(() => {
		const links = Array.from(
			document.querySelectorAll<HTMLLinkElement>('link[rel="prefetch"]'),
		);
		return links.filter((link) => link.href.includes("/itineraries/")).length;
	});
}

// The fixed dark slideshow root must keep full-viewport geometry through every
// transition (direct, reload, next/previous, keyboard) rather than collapsing
// while the ordinary layout's entrance animation runs underneath it.
async function assertViewerGeometry(page: Page) {
	const viewer = page.locator(".wga-viewer");
	await expect(viewer).toBeVisible();
	const box = await viewer.boundingBox();
	expect(box).not.toBeNull();
	if (!box) {
		return;
	}
	const viewport = page.viewportSize();
	expect(viewport).not.toBeNull();
	if (!viewport) {
		return;
	}
	expect(Math.round(box.x)).toBe(0);
	expect(Math.round(box.y)).toBe(0);
	expect(Math.round(box.width)).toBe(viewport.width);
	expect(Math.round(box.height)).toBe(viewport.height);
}

// The viewer is intrinsically dark and fixed: the underlying shell (including
// the footer theme toggle) is covered, not themed.
async function assertViewerFixedDark(page: Page) {
	await expect(page.locator(".wga-viewer")).toHaveCSS("position", "fixed");
	await expect(page.locator(".wga-viewer")).toHaveCSS(
		"background-color",
		"rgb(19, 19, 17)",
	);
}

// The fixed itinerary tray overlays the bottom of the viewport. Playwright's
// default click scrolls a below-the-fold control to the bottom edge, where the
// tray then intercepts the pointer. Centring the control first keeps its click
// point clear of the tray without forcing a click through the overlay.
async function scrollIntoViewCentered(locator: Locator) {
	await locator.evaluate((element) =>
		element.scrollIntoView({ block: "center" }),
	);
}

// The visibility radios are visually hidden (sr-only) inside their labels, so
// the labelled text is the accessible click target. Toggle through the label
// and assert the radio state rather than force-clicking the 1px input.
async function checkVisibilityOption(page: Page, label: string) {
	const option = page.getByText(label, { exact: true });
	await scrollIntoViewCentered(option);
	await option.click();
	await expect(page.getByRole("radio", { name: label })).toBeChecked();
}

// Select a visibility radio with the keyboard: focus the labelled control and
// activate it, proving the control is keyboard-operable despite being sr-only.
async function checkVisibilityOptionByKeyboard(page: Page, label: string) {
	const radio = page.getByRole("radio", { name: label });
	await radio.focus();
	await page.keyboard.press("Space");
	await expect(radio).toBeChecked();
}

test.describe("itinerary index", () => {
	for (const width of viewports) {
		test(`composes CTA, rules, and tour distinction at ${width}px`, async ({
			page,
		}) => {
			await page.setViewportSize({ width, height: 900 });
			await page.goto("/itineraries");

			await expect(
				page.getByText("16 — VISITOR ITINERARIES", { exact: true }),
			).toBeVisible();
			await expect(
				page.getByRole("link", { name: /BUILD AN ITINERARY/ }),
			).toBeVisible();
			for (const rule of ["LENGTH", "NARRATION", "ACCOUNT", "KEPT FOR"]) {
				await expect(page.getByText(rule, { exact: true })).toBeVisible();
			}
			await expect(page.getByText(/Separate from the/)).toBeVisible();
			await expect(
				page.getByRole("link", { name: "Guided Tours", exact: true }),
			).toHaveAttribute("href", "/tours");
			await expect(page.getByText("UNLISTED ITINERARIES")).toBeVisible();
			await expect(page.getByText("NEWEST FIRST")).toBeVisible();
			await assertNoHorizontalOverflow(page);
		});
	}

	test("keeps the index composed at 200 percent text", async ({ page }) => {
		await page.setViewportSize({ width: 390, height: 844 });
		await page.goto("/itineraries");
		await page.evaluate(() => {
			document.documentElement.style.fontSize = "2em";
		});
		await assertNoHorizontalOverflow(page);
		await expect(
			page.getByRole("link", { name: /BUILD AN ITINERARY/ }),
		).toBeVisible();
	});

	test("applies the dark theme from stored preference", async ({ page }) => {
		await page.addInitScript(() => {
			localStorage.setItem("wga-theme", "dark");
		});
		await page.setViewportSize({ width: 1440, height: 900 });
		await page.goto("/itineraries");
		await expect(page.locator("html")).toHaveAttribute(
			"data-theme",
			"wga-rams-dark",
		);
		await expect(
			page.getByText("16 — VISITOR ITINERARIES", { exact: true }),
		).toBeVisible();
		await assertNoHorizontalOverflow(page);
	});

	test("remains composed with reduced motion", async ({ page }) => {
		await page.setViewportSize({ width: 1440, height: 900 });
		await page.emulateMedia({ reducedMotion: "reduce" });
		await page.goto("/itineraries");
		await expect(
			page.getByText("16 — VISITOR ITINERARIES", { exact: true }),
		).toBeVisible();
		await assertNoHorizontalOverflow(page);
	});
});

test.describe("builder", () => {
	test("searches the picker by artist name", async ({ page }) => {
		await page.setViewportSize({ width: 1440, height: 900 });
		await page.goto("/inspire");
		const sourceCard = page
			.locator("section[aria-label='Shuffled artworks'] article")
			.nth(0);
		const artist = await sourceCard.locator("p").innerText();
		expect(artist).not.toBe("");
		const artistQuery = artist.split(/\s+/).at(-1);
		expect(artistQuery).toBeTruthy();
		await page.goto("/itineraries/new?picker=1");
		// The visible search input is the only search-typed pq control; the
		// builder-state forms carry hidden pq inputs.
		const query = page.locator('input[type="search"][name="pq"]');
		await expect(query).toBeVisible();

		await query.fill(artistQuery ?? artist);
		await expect(query).toHaveValue(artistQuery ?? artist);
		// The picker returns works whose artist actually matches, not just a
		// non-zero result count.
		const matchingWorks = page.locator("ul li").filter({
			hasText: artist,
		});
		await expect(matchingWorks).not.toHaveCount(0);
		await expect(
			matchingWorks.nth(0).locator("span span").nth(0),
		).toBeVisible();
		await expect(
			matchingWorks.nth(0).locator("span span").nth(1),
		).toContainText(artist);
		await expect(page.getByText(/SHOWN$/)).toBeVisible();
	});
});

test.describe("publication and slideshow", () => {
	test.describe.configure({ mode: "serial" });

	test("builds, orders, narrates, and publishes a listed itinerary", async ({
		page,
	}) => {
		await page.setViewportSize({ width: 1440, height: 900 });

		// Two distinct works, addressed by artwork identity rather than index.
		await page.goto("/inspire");
		const cards = page.locator(
			"section[aria-label='Shuffled artworks'] article",
		);
		const firstTitle = await cards.nth(0).locator("h2").innerText();
		const secondTitle = await cards.nth(1).locator("h2").innerText();
		expect(secondTitle).not.toBe(firstTitle);

		await page
			.locator("article", {
				has: page.getByRole("heading", { name: firstTitle, exact: true }),
			})
			.getByRole("button", { name: "ADD TO ITINERARY" })
			.click();
		await expect(page.locator("#itinerary-tray")).toContainText("1 STOP");
		await page
			.locator("article", {
				has: page.getByRole("heading", { name: secondTitle, exact: true }),
			})
			.getByRole("button", { name: "ADD TO ITINERARY" })
			.click();
		await expect(page.locator("#itinerary-tray")).toContainText("2 STOPS");

		// The builder shows the running-order filmstrip and empty narrations.
		await page.goto("/itineraries/new");
		await expect(page.getByText("ORDER OF PRESENTATION")).toBeVisible();
		await expect(page.getByText("NO NARRATION YET")).toHaveCount(2);
		await expect(page.getByText(/STOP 01 OF 2/)).toBeVisible();

		// Ordering: move the second work up so it becomes the first stop.
		await page
			.getByRole("button", { name: `Move ${secondTitle} earlier` })
			.click();
		await expect(page.locator("aside ol li").nth(0)).toContainText(secondTitle);

		// Reload recovery: the reorder persists.
		await page.reload();
		await expect(page.locator("aside ol li").nth(0)).toContainText(secondTitle);

		// Removal: drop the second work (now the moved-up stop's neighbour).
		await page.getByRole("button", { name: `Remove ${secondTitle}` }).click();
		await expect(page.getByText(/STOP 01 OF 1/)).toBeVisible();

		// Clear: empty the draft, then rebuild it for publication.
		page.once("dialog", (dialog) => dialog.accept());
		await page.getByRole("button", { name: "CLEAR ALL" }).click();
		await expect(page.getByText("No works yet.")).toBeVisible();

		// Rebuild with two fresh distinct works, again by artwork identity.
		await page.goto("/inspire");
		const rebCards = page.locator(
			"section[aria-label='Shuffled artworks'] article",
		);
		const reFirstTitle = await rebCards.nth(0).locator("h2").innerText();
		const reSecondTitle = await rebCards.nth(1).locator("h2").innerText();
		expect(reSecondTitle).not.toBe(reFirstTitle);
		await page
			.locator("article", {
				has: page.getByRole("heading", { name: reFirstTitle, exact: true }),
			})
			.getByRole("button", { name: "ADD TO ITINERARY" })
			.click();
		await expect(page.locator("#itinerary-tray")).toContainText("1 STOP");
		await page
			.locator("article", {
				has: page.getByRole("heading", { name: reSecondTitle, exact: true }),
			})
			.getByRole("button", { name: "ADD TO ITINERARY" })
			.click();
		await expect(page.locator("#itinerary-tray")).toContainText("2 STOPS");

		await page.goto("/itineraries/new");
		await expect(page.getByText(/STOP 01 OF 2/)).toBeVisible();

		// Narration auto-saves on change (blur).
		const narration = page.locator('textarea[name="narration"]').nth(0);
		await narration.fill("A narrated first stop.");
		await narration.press("Tab");
		await expect(page.getByText(/CHARACTERS USED$/)).toBeVisible();

		// Title auto-saves on change.
		const title = page.locator('input[name="title"]');
		const publicationTitle = uniquePublicationTitle(
			"Playwright listed journey",
		);
		await title.fill(publicationTitle);
		await title.press("Tab");
		await expect(
			page.getByRole("button", { name: "PUBLISH — THIS IS FINAL →" }),
		).toBeVisible();

		// Choose listed-publicly by keyboard, then publish.
		await checkVisibilityOptionByKeyboard(page, "LISTED PUBLICLY");
		await page
			.getByRole("button", { name: "PUBLISH — THIS IS FINAL →" })
			.click();
		await expect(page).toHaveURL(/\/itineraries\/published$/);

		await expect(page.getByText("COPY LINK")).toBeVisible();
		await expect(page.getByText("MAKER")).toBeVisible();
		await expect(page.getByText("Listed publicly")).toBeVisible();
		await expect(page.getByText("AVAILABLE UNTIL")).toBeVisible();

		const shareURL = await page.locator("#itinerary-url").innerText();
		expect(shareURL).toContain("/itineraries/");
		listedShareURL = shareURL;

		// The listed itinerary appears on the public index.
		await page.goto("/itineraries");
		const listedPath = new URL(shareURL).pathname;
		const listedEntry = page.locator("ul[data-kbd-list] > li").filter({
			has: page.getByText(publicationTitle, { exact: true }),
		});
		await expect(listedEntry).toHaveCount(1);
		await expect(listedEntry.getByRole("link")).toHaveAttribute(
			"href",
			listedPath,
		);
	});

	test.describe("keeps a link-only itinerary off the index", () => {
		test.use({ hasTouch: true });

		test("with a touch-selected visibility choice", async ({ page }) => {
			await page.setViewportSize({ width: 1440, height: 900 });
			await page.goto("/inspire");
			const cards = page.locator(
				"section[aria-label='Shuffled artworks'] article",
			);
			const title = await cards.nth(0).locator("h2").innerText();
			await page
				.locator("article", { hasText: title })
				.getByRole("button", { name: "ADD TO ITINERARY" })
				.click();
			await expect(page.locator("#itinerary-tray")).toContainText("1 STOP");

			await page.goto("/itineraries/new");
			const titleInput = page.locator('input[name="title"]');
			const linkOnlyTitle = uniquePublicationTitle(
				"Playwright link-only journey",
			);
			await titleInput.fill(linkOnlyTitle);
			await titleInput.press("Tab");
			await expect(
				page.getByRole("button", { name: "PUBLISH — THIS IS FINAL →" }),
			).toBeVisible();
			// Choose link-only by touch.
			const linkOnly = page.getByText("LINK ONLY", { exact: true });
			await scrollIntoViewCentered(linkOnly);
			await linkOnly.tap();
			await expect(
				page.getByRole("radio", { name: "LINK ONLY" }),
			).toBeChecked();
			await page
				.getByRole("button", { name: "PUBLISH — THIS IS FINAL →" })
				.click();
			await expect(page).toHaveURL(/\/itineraries\/published$/);
			await expect(page.getByText("Link only")).toBeVisible();

			await page.goto("/itineraries");
			await expect(page.getByText(linkOnlyTitle, { exact: true })).toHaveCount(
				0,
			);
		});
	});

	test("viewer is a fixed dark root across keyboard, link, and reload transitions", async ({
		page,
	}) => {
		await page.setViewportSize({ width: 1440, height: 900 });
		await page.goto(listedShareURL);

		await expect(page.locator(".wga-viewer")).toBeVisible();
		await assertViewerFixedDark(page);
		await expect(page.locator('ol[aria-label="Progress"] a')).toHaveCount(2);
		await expect(page.getByText(/STOP 01 OF 02/)).toBeVisible();
		await expect(page.getByText(/NARRATION BY/)).toBeVisible();
		await expect(page.getByText(/USE ← AND → TO MOVE/)).toBeVisible();
		await expect(page.locator(".wga-viewer-plate img")).toHaveAttribute(
			"data-zoom-url",
			/thumb=2000x0|\.jpg/,
		);
		await expect(
			page.locator('a[data-viewer][href*="/api/files/artworks/"]').nth(0),
		).toBeVisible();
		expect(await countItineraryPrefetch(page)).toBeLessThanOrEqual(2);

		// Direct load, keyboard, link, and reload transitions keep geometry.
		await assertViewerGeometry(page);
		await page.keyboard.press("ArrowRight");
		await page.waitForURL(/stop=1/);
		await assertViewerGeometry(page);
		await page.keyboard.press("ArrowLeft");
		await page.waitForURL(/stop=0/);
		await assertViewerGeometry(page);
		await page.locator('a[data-itinerary-nav="next"]').click();
		await page.waitForURL(/stop=1/);
		await assertViewerGeometry(page);
		await page.reload();
		await assertViewerGeometry(page);

		// A direct-stop URL works and honours its position.
		await page.goto(`${listedShareURL}?stop=1`);
		await assertViewerGeometry(page);
		await expect(page.getByText(/STOP 02 OF 02/)).toBeVisible();
		await page.keyboard.press("ArrowLeft");
		await page.waitForURL(/stop=0/);

		// Arrow keys stay guarded while focus sits inside an editable control,
		// so the slideshow never navigates over a typing cursor.
		await page.evaluate(() => {
			const field = document.createElement("textarea");
			document.body.appendChild(field);
			field.focus();
		});
		const beforeGuarded = page.url();
		await page.keyboard.press("ArrowRight");
		await page.waitForTimeout(300);
		expect(page.url()).toBe(beforeGuarded);
		await page.evaluate(() => {
			if (document.activeElement instanceof HTMLElement) {
				document.activeElement.blur();
			}
		});

		// Escape closes the overlay and lands on the public index.
		await page.keyboard.press("Escape");
		await page.waitForURL(/\/itineraries$/);
	});

	test("viewer stays fixed-dark and overflow-free across viewports and 200 percent text", async ({
		page,
	}) => {
		for (const width of viewports) {
			await page.setViewportSize({ width, height: 900 });
			await page.goto(listedShareURL);
			await assertViewerFixedDark(page);
			await assertViewerGeometry(page);
			await assertNoHorizontalOverflow(page);

			// Scrolling the underlying document must not shift the fixed root.
			await page.evaluate(() => window.scrollTo(0, 400));
			await assertViewerGeometry(page);
		}

		await page.setViewportSize({ width: 390, height: 844 });
		await page.goto(listedShareURL);
		await page.evaluate(() => {
			document.documentElement.style.fontSize = "2em";
		});
		await assertViewerFixedDark(page);
		await assertViewerGeometry(page);
		await assertNoHorizontalOverflow(page);
	});

	test("viewer keeps the theme and reduced motion independent of keyboard navigation", async ({
		page,
	}) => {
		// A light user theme must not lighten the intrinsically dark viewer.
		await page.addInitScript(() => {
			localStorage.setItem("wga-theme", "light");
		});
		await page.emulateMedia({ reducedMotion: "reduce" });
		await page.setViewportSize({ width: 1440, height: 900 });
		await page.goto(listedShareURL);
		await assertViewerFixedDark(page);
		await assertViewerGeometry(page);

		await page.keyboard.press("ArrowRight");
		await page.waitForURL(/stop=1/);
		await assertViewerGeometry(page);
		await page.keyboard.press("Escape");
		await page.waitForURL(/\/itineraries$/);
	});
});

test.describe("without JavaScript", () => {
	test.use({ javaScriptEnabled: false });

	test("adds, rebuilds, and publishes a listed draft through ordinary forms", async ({
		page,
	}) => {
		// Reduced motion keeps the decorative rise animation from holding the
		// ordinary add control unstable while JavaScript is switched off.
		await page.emulateMedia({ reducedMotion: "reduce" });
		await page.setViewportSize({ width: 390, height: 844 });
		await page.goto("/inspire");
		await page.getByRole("button", { name: "ADD TO ITINERARY" }).nth(0).click();
		// The ordinary 303 redirect lands back on the builder.
		await expect(page).toHaveURL(/\/itineraries\/new/);
		await expect(page.getByText(/STOP 01 OF 1/)).toBeVisible();

		const noJSTitle = uniquePublicationTitle("No-JS draft");
		await page.locator('input[name="title"]').fill(noJSTitle);
		const saveDetails = page.getByRole("button", { name: "SAVE DETAILS" });
		await scrollIntoViewCentered(saveDetails);
		await saveDetails.click();
		await page.locator('textarea[name="narration"]').fill("Plain narration");
		const saveNarration = page.getByRole("button", { name: "SAVE NARRATION" });
		await scrollIntoViewCentered(saveNarration);
		await saveNarration.click();

		await page.reload();
		await expect(page.locator('input[name="title"]')).toHaveValue(noJSTitle);
		await expect(page.getByText(/STOP 01 OF 1/)).toBeVisible();

		// Publish as listed publicly through the ordinary form.
		await checkVisibilityOption(page, "LISTED PUBLICLY");
		const publish = page.getByRole("button", {
			name: "PUBLISH — THIS IS FINAL →",
		});
		await scrollIntoViewCentered(publish);
		await publish.click();
		await expect(page).toHaveURL(/\/itineraries\/published$/);
		await expect(page.getByText("Listed publicly")).toBeVisible();

		const noJSShareURL = await page.locator("#itinerary-url").innerText();
		await page.goto("/itineraries");
		const noJSEntry = page.locator("ul[data-kbd-list] > li").filter({
			has: page.getByText(noJSTitle, { exact: true }),
		});
		await expect(noJSEntry).toHaveCount(1);
		await expect(noJSEntry.getByRole("link")).toHaveAttribute(
			"href",
			new URL(noJSShareURL).pathname,
		);
	});
});
