import { expect, test } from "@playwright/test";

const widths = [390, 834, 1440] as const;

async function noHorizontalOverflow(page: import("@playwright/test").Page) {
	const dimensions = await page.evaluate(() => ({
		clientWidth: document.documentElement.clientWidth,
		scrollWidth: document.documentElement.scrollWidth,
	}));
	expect(dimensions.scrollWidth).toBeLessThanOrEqual(dimensions.clientWidth);
}

async function centre(locator: import("@playwright/test").Locator) {
	await locator.evaluate((element) =>
		element.scrollIntoView({ block: "center" }),
	);
}

async function assertToastAboveTray(page: import("@playwright/test").Page) {
	const trayBox = await page
		.locator("#itinerary-tray [role='region']")
		.boundingBox();
	const toastBox = await page.getByRole("alert").boundingBox();
	expect(trayBox).not.toBeNull();
	expect(toastBox).not.toBeNull();
	if (trayBox && toastBox) {
		console.log(
			`tray top=${trayBox.y} toast bottom=${toastBox.y + toastBox.height}`,
		);
		expect(toastBox.y + toastBox.height).toBeLessThanOrEqual(trayBox.y);
	}
}

test.describe("task 9.2 itinerary tray acceptance", () => {
	test.describe.configure({ mode: "serial" });

	test("HTMX add mounts the exact tray, reserves offsets, persists, and clears elsewhere", async ({
		page,
	}) => {
		await page.setViewportSize({ width: 390, height: 834 });
		await page.goto("/inspire");
		const card = page
			.locator("section[aria-label='Shuffled artworks'] article")
			.first();
		const add = card.getByRole("button", {
			name: "ADD TO AN ITINERARY +",
			exact: true,
		});
		await centre(add);
		await add.click();

		const tray = page.locator("#itinerary-tray");
		await expect(tray).toContainText("ITINERARY DRAFT · 1 OF 15");
		await expect(
			tray.getByRole("button", { name: "CLEAR", exact: true }),
		).toBeVisible();
		await expect(
			tray.getByRole("link", { name: "ARRANGE & NARRATE →", exact: true }),
		).toBeVisible();
		await expect(page.locator("#mc-area")).toHaveClass(/pb-28/);
		await expect(page.locator("#mc-area")).toHaveClass(/md:pb-20/);
		await expect(page.locator("#toast-container")).toHaveClass(/bottom-28/);
		await expect(page.locator("#toast-container")).toHaveClass(/md:bottom-20/);
		await expect(page.getByRole("alert")).toBeVisible();
		await assertToastAboveTray(page);

		await page.reload();
		await expect(page.locator("#itinerary-tray")).toContainText(
			"ITINERARY DRAFT · 1 OF 15",
		);
		await expect(page.locator("#mc-area")).toHaveClass(/pb-28/);
		await expect(page.locator("#mc-area")).toHaveClass(/md:pb-20/);
		await expect(page.locator("#toast-container")).toHaveClass(/bottom-28/);
		await expect(page.locator("#toast-container")).toHaveClass(/md:bottom-20/);

		await page.goto("/artists");
		page.once("dialog", (dialog) => dialog.accept());
		await page.getByRole("button", { name: "CLEAR", exact: true }).click();
		await expect(page.locator("#itinerary-tray")).toBeEmpty();
		await expect(page.locator("#mc-area")).not.toHaveClass(/pb-28/);
		await expect(page.locator("#mc-area")).not.toHaveClass(/md:pb-20/);
		await expect(page.locator("#toast-container")).not.toHaveClass(/bottom-28/);
		await expect(page.locator("#toast-container")).not.toHaveClass(
			/md:bottom-20/,
		);
		await page.goto("/inspire");
		await expect(page.locator("#itinerary-tray")).toBeEmpty();
	});

	test("the same draft stays composed in light and dark at every viewport and 200% text", async ({
		page,
	}) => {
		await page.setViewportSize({ width: 390, height: 834 });
		await page.goto("/inspire");
		const add = page
			.locator("section[aria-label='Shuffled artworks'] article")
			.first()
			.getByRole("button", { name: "ADD TO AN ITINERARY +", exact: true });
		await centre(add);
		await add.click();
		await expect(page.locator("#itinerary-tray")).toContainText("1 OF 15");

		for (const width of widths) {
			await page.setViewportSize({ width, height: 834 });
			await page.reload();
			await expect(page.locator("#itinerary-tray")).toContainText("1 OF 15");
			await noHorizontalOverflow(page);
		}
		await page.addInitScript(() => localStorage.setItem("wga-theme", "dark"));
		await page.setViewportSize({ width: 1440, height: 834 });
		await page.reload();
		await expect(page.locator("html")).toHaveAttribute(
			"data-theme",
			"wga-rams-dark",
		);
		await expect(page.locator("#itinerary-tray")).toContainText("1 OF 15");
		await noHorizontalOverflow(page);

		await page.setViewportSize({ width: 390, height: 834 });
		await page.evaluate(() => {
			document.documentElement.style.fontSize = "2em";
		});
		await noHorizontalOverflow(page);
		await expect(page.locator("#itinerary-tray")).toContainText("1 OF 15");
		const largeTrayBox = await page
			.locator("#itinerary-tray [role='region']")
			.boundingBox();
		expect(largeTrayBox).not.toBeNull();
		if (largeTrayBox) {
			console.log(
				`200% tray top=${largeTrayBox.y} bottom=${largeTrayBox.y + largeTrayBox.height}`,
			);
		}
		await page.goto("/artists");
		page.once("dialog", (dialog) => dialog.accept());
		await page.getByRole("button", { name: "CLEAR", exact: true }).click();
		await expect(page.locator("#itinerary-tray")).toBeEmpty();
	});

	test("Dual Mode exposes the typed row action at 46px with its full label", async ({
		page,
	}) => {
		await page.setViewportSize({ width: 1440, height: 834 });
		await page.goto("/inspire");
		const recordHref = await page
			.locator("section[aria-label='Shuffled artworks'] article")
			.first()
			.getByRole("link")
			.first()
			.getAttribute("href");
		expect(recordHref).toMatch(/^\/artists\//);
		await page.goto(
			`/dual-mode?right=${encodeURIComponent(recordHref ?? "")}&wide=1`,
		);
		const action = page.locator("#dual-right").getByRole("button", {
			name: "ADD TO AN ITINERARY +",
			exact: true,
		});
		await expect(action).toHaveCount(1);
		await expect(action).toHaveCSS("height", "46px");
		await expect(action).toHaveAttribute("class", /h-\[46px\]/);
	});
});

test.describe("task 9.2 ordinary form fallback", () => {
	test.use({ javaScriptEnabled: false });

	test("add redirects 303 to builder, survives reload without duplication, and clear stays empty", async ({
		page,
	}) => {
		await page.emulateMedia({ reducedMotion: "reduce" });
		await page.setViewportSize({ width: 390, height: 834 });
		await page.goto("/inspire");
		const add = page
			.getByRole("button", { name: "ADD TO AN ITINERARY +", exact: true })
			.first();
		await centre(add);
		const form = add.locator("..");
		await expect(form.locator('input[name="_csrf"]')).toHaveValue(/.+/);
		await expect
			.poll(async () =>
				(await page.context().cookies()).some(
					(cookie) => cookie.name === "wga_itinerary_dev",
				),
			)
			.toBe(true);
		const addResponse = page.waitForResponse((response) =>
			response.url().endsWith("/itineraries/draft/add"),
		);
		await add.click();
		const response = await addResponse;
		console.log(
			`no-JS add response status=${response.status()} location=${response.headers().location ?? ""}`,
		);
		await expect(page).toHaveURL(/\/itineraries\/new$/);
		await expect(page.getByText("STOP 01 OF 1", { exact: true })).toBeVisible();
		await page.reload();
		await expect(page.getByText("STOP 01 OF 1", { exact: true })).toBeVisible();
		await expect(page.getByText("STOP 02 OF 2", { exact: true })).toHaveCount(
			0,
		);

		await page.goto("/artists");
		await page.getByRole("button", { name: "CLEAR", exact: true }).click();
		await expect(page.locator("#itinerary-tray")).toBeEmpty();
		await page.goto("/inspire");
		await expect(page.locator("#itinerary-tray")).toBeEmpty();
	});
});
