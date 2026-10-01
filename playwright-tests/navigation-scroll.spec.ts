import { expect, type Locator, type Page, test } from "@playwright/test";

// Scroll position contract: smooth in-page anchors, an instant top-of-page start
// after a main-area navigation, and no movement for fragment updates.

const curatedArtistPath = "/artists/synthetic-artist-02-2236bdd57f7492e";

async function settleEntryAnimation(page: Page) {
	await expect(page.locator("main#mc-area")).toHaveCSS("transform", "none");
}

const scrollY = (page: Page) => page.evaluate(() => window.scrollY);

async function scrollInstantly(page: Page, top: number) {
	await page.evaluate((y) => {
		window.scrollTo({ top: y, left: 0, behavior: "instant" });
	}, top);
	await expect.poll(() => scrollY(page)).toBe(top);
}

// Records the window's (or `scroller`'s) scroll offset on every animation frame
// while `action` runs, until the position has been stable for half a second.
async function sampleScroll(
	page: Page,
	action: () => Promise<void>,
	scroller?: string,
): Promise<number[]> {
	await page.evaluate((selector) => {
		const read = () =>
			selector
				? (document.querySelector(selector)?.scrollTop ?? -1)
				: window.scrollY;
		const samples: number[] = [read()];
		(window as unknown as { __scrollSamples: number[] }).__scrollSamples =
			samples;
		const tick = () => {
			samples.push(read());
			if (samples.length < 600) {
				requestAnimationFrame(tick);
			}
		};
		requestAnimationFrame(tick);
	}, scroller);
	await action();
	await expect
		.poll(() =>
			page.evaluate(() => {
				const samples = (window as unknown as { __scrollSamples: number[] })
					.__scrollSamples;
				const tail = samples.slice(-30);
				return tail.length === 30 && tail.every((value) => value === tail[0]);
			}),
		)
		.toBe(true);
	return page.evaluate(
		() => (window as unknown as { __scrollSamples: number[] }).__scrollSamples,
	);
}

const intermediates = (samples: number[]) => {
	const start = samples[0];
	const end = samples[samples.length - 1];
	return samples.filter((value) => value !== start && value !== end);
};

// Positions the window so the control stays visible below the top edge while
// the page is scrolled, then returns the position.
async function scrollWithVisible(page: Page, control: Locator) {
	const top = await control.evaluate(
		(element) => element.getBoundingClientRect().top + window.scrollY,
	);
	const target = Math.max(40, Math.floor(top - 120));
	await scrollInstantly(page, target);
	return target;
}

test.use({ viewport: { width: 1280, height: 640 } });

test.describe("in-page anchors", () => {
	test("scroll smoothly through intermediate positions", async ({ page }) => {
		await page.goto(curatedArtistPath);
		await settleEntryAnimation(page);
		const toc = page.getByRole("navigation", { name: "On this page" });
		const samples = await sampleScroll(page, () =>
			toc.getByRole("link", { name: "Cite This Record" }).click(),
		);
		expect(new URL(page.url()).hash).toBe("#cite-this-record");
		expect(samples[samples.length - 1]).toBeGreaterThan(0);
		expect(intermediates(samples).length).toBeGreaterThan(1);
		// Smooth behaviour is scoped to the jump; scripted scrolling stays instant.
		await expect(page.locator("html")).not.toHaveClass(/wga-smooth-scroll/);
		await expect(page.locator("html")).toHaveCSS("scroll-behavior", "auto");
	});

	test.describe("with reduced motion", () => {
		test.use({ reducedMotion: "reduce" });

		test("jump immediately", async ({ page }) => {
			await page.goto(curatedArtistPath);
			await settleEntryAnimation(page);
			const toc = page.getByRole("navigation", { name: "On this page" });
			const samples = await sampleScroll(page, () =>
				toc.getByRole("link", { name: "Cite This Record" }).click(),
			);
			expect(new URL(page.url()).hash).toBe("#cite-this-record");
			expect(samples[samples.length - 1]).toBeGreaterThan(0);
			expect(intermediates(samples)).toEqual([]);
		});
	});

	test("a Dual Mode pane scrolls its own contents smoothly", async ({
		page,
	}) => {
		await page.goto(
			`/dual-mode?wide=1&left=${encodeURIComponent(curatedArtistPath)}`,
		);
		const contents = page
			.locator("#dual-left")
			.getByRole("navigation", { name: "On this page" });
		const samples = await sampleScroll(
			page,
			() => contents.getByRole("link", { name: "Cite This Record" }).click(),
			"#dual-left .wga-dual-body",
		);
		expect(new URL(page.url()).hash).toBe("#dual-left-citation");
		expect(samples[samples.length - 1]).toBeGreaterThan(0);
		expect(intermediates(samples).length).toBeGreaterThan(1);
	});
});

for (const reducedMotion of ["no-preference", "reduce"] as const) {
	test.describe(`main-area navigation (${reducedMotion})`, () => {
		test.use({ reducedMotion });

		test("starts the new page at the top instantly and Back restores the position", async ({
			page,
		}) => {
			await page.goto(curatedArtistPath);
			await settleEntryAnimation(page);
			const footerLink = page
				.locator("footer")
				.getByRole("link", { name: "Artworks", exact: true });
			const bottom = await page.evaluate(
				() => document.documentElement.scrollHeight - window.innerHeight,
			);
			expect(bottom).toBeGreaterThan(200);
			await scrollInstantly(page, bottom);
			await expect(footerLink).toBeInViewport();

			const samples = await sampleScroll(page, async () => {
				await footerLink.click();
				await expect(page).toHaveURL(/\/artworks$/);
				await expect(page.locator("h1")).toHaveText("Artworks");
			});
			expect(samples[samples.length - 1]).toBe(0);
			expect(intermediates(samples)).toEqual([]);

			const restored = await sampleScroll(page, async () => {
				await page.goBack();
				await expect(page).toHaveURL(new RegExp(`${curatedArtistPath}$`));
			});
			expect(restored[restored.length - 1]).toBe(bottom);
			expect(intermediates(restored)).toEqual([]);
		});
	});
}

// Resolves once HTMX has settled a swap whose new element carries one of `ids`.
function settledSwap(page: Page, ids: string[]) {
	return page.evaluate(
		(targetIds) =>
			new Promise<void>((resolve) => {
				const listener = (event: Event) => {
					const id = (event.target as Element | null)?.id ?? "";
					if (targetIds.includes(id)) {
						document.removeEventListener("htmx:afterSettle", listener);
						resolve();
					}
				};
				document.addEventListener("htmx:afterSettle", listener);
			}),
		ids,
	);
}

test.describe("fragment updates keep the scroll position", () => {
	test("global search results", async ({ page }) => {
		await page.goto("/search?q=Synthetic");
		await expect(page.locator("#global-search-results")).toContainText(
			"SYNTHETIC ARTIST 01",
		);
		const searchbox = page.locator("#search").getByRole("searchbox");
		const position = await scrollWithVisible(page, searchbox);
		const settled = settledSwap(page, ["global-search-results"]);
		await searchbox.evaluate((element: HTMLElement) =>
			element.focus({ preventScroll: true }),
		);
		await page.keyboard.press("End");
		await page.keyboard.type(" Artist");
		await settled;
		await expect(page.locator("#global-search-results")).toContainText(
			"SYNTHETIC ARTIST",
		);
		expect(await scrollY(page)).toBe(position);
	});

	test("artwork search filter change", async ({ page }) => {
		await page.goto("/artworks");
		await expect(
			page.locator("#artwork-search-results [data-view='grid']"),
		).toBeVisible();
		const searchbox = page.locator("#artwork-filters").getByRole("searchbox");
		const position = await scrollWithVisible(page, searchbox);
		const settled = settledSwap(page, ["artwork-search"]);
		await searchbox.evaluate((element: HTMLElement) =>
			element.focus({ preventScroll: true }),
		);
		await page.keyboard.type("Synthetic Artwork 01");
		await settled;
		await expect(page).toHaveURL(/q=Synthetic/);
		expect(await scrollY(page)).toBe(position);
	});

	test("Dual Mode pane swap", async ({ page }) => {
		await page.goto("/dual-mode?wide=1");
		const leftBody = page.locator("#dual-left .wga-dual-body");
		await expect(leftBody).toContainText("SYNTHETIC ARTIST 01");
		await leftBody.evaluate((element) => {
			element.scrollTo({ top: 60, behavior: "instant" });
		});
		await expect
			.poll(() => leftBody.evaluate((element) => element.scrollTop))
			.toBeGreaterThan(0);
		const leftTop = await leftBody.evaluate((element) => element.scrollTop);
		const windowRange = await page.evaluate(
			() => document.documentElement.scrollHeight - window.innerHeight,
		);
		if (windowRange > 0) {
			await scrollInstantly(page, Math.min(40, windowRange));
		}
		const windowTop = await scrollY(page);

		// The index routes to the right pane. Dispatch the click directly so the
		// test itself does not scroll the link into view.
		const settled = settledSwap(page, ["dual-right", "dual-area"]);
		await leftBody
			.getByRole("link", { name: /SYNTHETIC ARTIST/i })
			.first()
			.dispatchEvent("click");
		await settled;
		await expect(page).toHaveURL(/right=/);
		expect(await scrollY(page)).toBe(windowTop);
		expect(
			await page
				.locator("#dual-left .wga-dual-body")
				.evaluate((element) => element.scrollTop),
		).toBe(leftTop);
	});
});

test("a URL with a section fragment lands on the section", async ({ page }) => {
	await page.goto(`${curatedArtistPath}#cite-this-record`);
	const section = page.locator("#cite-this-record");
	await expect(section).toBeVisible();
	await expect
		.poll(() =>
			section.evaluate((element) =>
				Math.round(element.getBoundingClientRect().top),
			),
		)
		.toBeLessThan(80);
	expect(await scrollY(page)).toBeGreaterThan(0);
});
