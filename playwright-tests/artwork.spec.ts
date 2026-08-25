import { expect, test } from "@playwright/test";

test("artwork viewer hides its thumbnail navigation bar", async ({ page }) => {
	await page.goto(
		"/artists/synthetic-artist-01-ad32608c6e36b2e/synthetic-artwork-01-01-2225c982be1af02",
	);

	await page.locator("[data-viewer-no-navbar] img").click();

	await expect(page.locator(".viewer-container")).toBeVisible();
	const zoomURL = await page
		.locator("[data-viewer-no-navbar] img")
		.getAttribute("data-zoom-url");
	if (!zoomURL) {
		throw new Error("artwork viewer must receive a zoom URL");
	}
	await expect(page.locator(".viewer-canvas img").first()).toHaveAttribute(
		"src",
		zoomURL,
	);
	await expect(page.locator(".viewer-navbar")).toBeHidden();
	await expect(page.locator(".viewer-container")).toHaveAttribute(
		"role",
		"dialog",
	);
	await expect(page.locator("body > main")).toHaveJSProperty("inert", true);
	await expect(
		page.getByRole("button", { name: "Close artwork viewer" }),
	).toBeFocused();
	await expect(
		page.getByRole("button", { name: "Close artwork viewer" }),
	).toBeVisible();
	await page.keyboard.press("Tab");
	expect(
		await page.evaluate(
			() =>
				document.activeElement?.closest(".viewer-container") instanceof
				HTMLElement,
		),
	).toBe(true);

	await page.keyboard.press("Escape");
	await expect(page.locator(".viewer-container")).toBeHidden();
	await expect(page.locator("[data-viewer-no-navbar]")).toBeFocused();
	await expect(page.locator("body > main")).toHaveJSProperty("inert", false);

	await page.locator("[data-viewer-no-navbar]").click();
	await expect(page.locator(".viewer-container")).toBeVisible();
	await page
		.locator("[data-viewer-no-navbar]")
		.evaluate((element) => element.remove());
	await page.getByRole("button", { name: "Close artwork viewer" }).click();
	await expect(page.locator(".viewer-container")).toBeHidden();
	await expect(page.locator("body > main")).toHaveJSProperty("inert", false);
});

test("artwork viewer disables transitions for reduced motion", async ({
	page,
}) => {
	await page.emulateMedia({ reducedMotion: "reduce" });
	await page.goto(
		"/artists/synthetic-artist-01-ad32608c6e36b2e/synthetic-artwork-01-01-2225c982be1af02",
	);

	await page.locator("[data-viewer-no-navbar]").focus();
	await page.keyboard.press("Enter");
	await expect(page.locator(".viewer-container")).toBeVisible();
	await expect(page.locator(".viewer-container")).not.toHaveClass(
		/viewer-transition/,
	);
});

test("artwork viewer retains an ordinary zoom link without JavaScript", async ({
	browser,
}) => {
	const context = await browser.newContext({ javaScriptEnabled: false });
	const page = await context.newPage();
	await page.goto(
		"/artists/synthetic-artist-01-ad32608c6e36b2e/synthetic-artwork-01-01-2225c982be1af02",
	);

	const image = page.locator("[data-viewer-no-navbar] img");
	const zoomURL = await image.getAttribute("data-zoom-url");
	if (!zoomURL) {
		throw new Error("artwork viewer must receive a zoom URL");
	}
	await expect(page.locator("[data-viewer-no-navbar] a")).toHaveAttribute(
		"href",
		zoomURL,
	);
	await context.close();
});

test("initialises the artwork BibTeX copy helper", async ({ page }) => {
	await page.goto(
		"/artists/synthetic-artist-01-ad32608c6e36b2e/synthetic-artwork-01-01-2225c982be1af02",
	);

	const copyButton = page.getByRole("button", { name: "COPY BIBTEX" });
	await expect(copyButton).toHaveAttribute("data-copy-bound", "true");
});
