import { expect, test } from "@playwright/test";

const entry = {
	name: "Playwright visitor",
	location: "Testing Grounds",
	message: "This note must remain private until moderation.",
};

test("places the guestbook note rail on the desktop right edge", async ({
	page,
}) => {
	await page.setViewportSize({ width: 1440, height: 900 });
	await page.goto("/guestbook?year=all");

	const box = await page.locator("aside").boundingBox();
	expect(box?.width).toBeCloseTo(320, 0);
	const archive = await page.locator('form[hx-get="/guestbook"]').boundingBox();
	expect((box?.x || 0) + (box?.width || 0)).toBeGreaterThan(
		(archive?.x || 0) + (archive?.width || 0),
	);
});

test("uses the reference guestbook title scale on mobile", async ({ page }) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await page.goto("/guestbook");

	await expect(page.getByRole("heading", { name: "Guestbook" })).toHaveCSS(
		"font-size",
		"44px",
	);
});

test("places the archive before the note form on mobile", async ({ page }) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await page.goto("/guestbook?year=all");

	const form = await page.locator("aside").boundingBox();
	const archive = await page.locator('form[hx-get="/guestbook"]').boundingBox();
	expect(archive?.y).toBeLessThan(form?.y || 0);
});

test("preserves text and year archive state in the URL", async ({ page }) => {
	await page.goto("/guestbook?year=all");

	const search = page.getByRole("searchbox", { name: "Search entries" });
	await search.fill("chapel");
	await expect(page).toHaveURL(/q=chapel/);
	await expect(page).toHaveURL(/year=all/);
	await expect(page.getByText(/FOUND$/)).toBeVisible();
});

test("queues a note without collecting email or publishing it", async ({
	page,
}) => {
	await page.goto("/guestbook?year=all");
	await expect(page.locator('input[name="sender_email"]')).toHaveCount(0);

	await page.getByLabel("NAME").fill(entry.name);
	await page.getByLabel("LOCATION").fill(entry.location);
	await page.getByLabel(/NOTE/).fill(entry.message);
	await page.getByRole("button", { name: "SIGN THE GUESTBOOK →" }).click();

	await expect(page.getByRole("status")).toContainText("your note is queued");
	await expect(
		page.locator(".gb-entry", { hasText: entry.message }),
	).toHaveCount(0);
});

test("supports keyboard-only form completion", async ({ page }) => {
	await page.goto("/guestbook?year=all");
	await page.getByLabel("NAME").focus();
	await page.keyboard.type("Keyboard visitor");
	await page.keyboard.press("Tab");
	await expect(page.getByLabel("LOCATION")).toBeFocused();
	await page.keyboard.type("Keyboard location");
	await page.keyboard.press("Tab");
	await expect(page.getByLabel(/NOTE/)).toBeFocused();
	await page.keyboard.type("Keyboard-only private note");
	await page.keyboard.press("Tab");
	await page.keyboard.press("Enter");
	await expect(page.getByRole("status")).toContainText("your note is queued");
});

test.describe("without JavaScript", () => {
	test.use({ javaScriptEnabled: false });

	test("submits through the ordinary form and keeps the note private", async ({
		page,
	}) => {
		await page.goto("/guestbook?year=all");
		const privateMessage = "No-JavaScript private note";
		await page.getByLabel("NAME").fill("No-JavaScript visitor");
		await page.getByLabel("LOCATION").fill("No-JavaScript location");
		await page.getByLabel(/NOTE/).fill(privateMessage);
		await page.getByRole("button", { name: "SIGN THE GUESTBOOK →" }).click();

		await expect(page).toHaveURL(/\/guestbook\?submitted=1/);
		await expect(page.getByRole("status")).toContainText("your note is queued");
		await page.goto("/guestbook?year=all");
		await expect(
			page.locator(".gb-entry", { hasText: privateMessage }),
		).toHaveCount(0);
	});
});
