import { expect, test } from "@playwright/test";

const sessionCookieName = "wga_itinerary_dev";

async function hasSessionCookie(
	page: import("@playwright/test").Page,
): Promise<boolean> {
	return (await page.context().cookies()).some(
		(cookie) => cookie.name === sessionCookieName,
	);
}

test.describe("itinerary state on shared-cacheable documents", () => {
	test("a cookie-less document is shareable, the first add creates the session, and added state survives swaps and reloads", async ({
		page,
	}) => {
		await page.setViewportSize({ width: 1440, height: 900 });
		await page.goto("/inspire");
		const recordHref = await page
			.locator("section[aria-label='Shuffled artworks'] article")
			.first()
			.getByRole("link")
			.first()
			.getAttribute("href");
		expect(recordHref).toMatch(/^\/artists\//);

		const dualURL = `/dual-mode?right=${encodeURIComponent(recordHref ?? "")}&wide=1`;
		const documentResponse = await page.goto(dualURL);
		expect(documentResponse?.headers()["cache-control"]).toBe(
			"public, max-age=0, s-maxage=300",
		);
		expect(documentResponse?.headers()["set-cookie"]).toBeUndefined();
		expect(await documentResponse?.text()).not.toMatch(
			/data-itinerary-csrf="|name="_csrf" value="[^"]/,
		);
		await expect(page.locator("#itinerary-tray")).toBeEmpty();
		expect(await hasSessionCookie(page)).toBe(false);

		const right = page.locator("#dual-right");
		const add = right.getByRole("button", {
			name: "ADD TO AN ITINERARY +",
			exact: true,
		});
		await expect(add).toHaveCount(1);
		await add.click();
		await expect(page.locator("#itinerary-tray")).toContainText("1 OF 15");
		await expect(
			right.getByRole("button", { name: "IN YOUR ITINERARY ✓", exact: true }),
		).toBeDisabled();
		expect(await hasSessionCookie(page)).toBe(true);

		// The swapped panes are new markup; the tray state is reapplied after settle.
		await page
			.getByRole("link", { name: "⇄ SWAP WINDOWS", exact: true })
			.click();
		await expect(
			page
				.locator("#dual-left")
				.getByRole("button", { name: "IN YOUR ITINERARY ✓", exact: true }),
		).toBeDisabled();

		// A reload serves the shared document again; the session fragment restores
		// the tray and the added state.
		await page.reload();
		await expect(page.locator("#itinerary-tray")).toContainText("1 OF 15");
		await expect(
			page
				.locator("#dual-left")
				.getByRole("button", { name: "IN YOUR ITINERARY ✓", exact: true }),
		).toBeDisabled();

		// The wide Dual Mode table overlaps the fixed tray, so clear elsewhere.
		await page.goto("/artists");
		page.once("dialog", (dialog) => dialog.accept());
		await page.getByRole("button", { name: "CLEAR", exact: true }).click();
		await expect(page.locator("#itinerary-tray")).toBeEmpty();
		await page.goto(dualURL);
		await expect(page.locator("#itinerary-tray")).toBeEmpty();
		await expect(
			page.locator("#dual-area").getByRole("button", {
				name: "ADD TO AN ITINERARY +",
				exact: true,
			}),
		).toBeEnabled();
	});

	test("an HTMX fragment response stays private", async ({ page }) => {
		const response = await page.request.get("/artists", {
			headers: { "HX-Request": "true" },
		});
		expect(response.headers()["cache-control"]).toBe("private, no-store");
	});
});
