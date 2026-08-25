import { expect, test } from "@playwright/test";

for (const colorScheme of ["light", "dark"] as const) {
	test(`uses the browser ${colorScheme} preference by default`, async ({
		page,
	}) => {
		await page.emulateMedia({ colorScheme });
		await page.goto("/");

		await expect(page.locator("html")).toHaveCSS("color-scheme", colorScheme);
	});
}

test.describe("without JavaScript", () => {
	test.use({ javaScriptEnabled: false });

	test("uses the browser dark preference by default", async ({ page }) => {
		await page.emulateMedia({ colorScheme: "dark" });
		await page.goto("/");

		await expect(page.locator("html")).toHaveCSS("color-scheme", "dark");
	});
});

test("switches and remembers the selected colour scheme", async ({ page }) => {
	await page.goto("/");

	await page.getByRole("button", { name: "DARK" }).first().click();
	await expect(page.locator("html")).toHaveAttribute(
		"data-theme",
		"wga-rams-dark",
	);
	await expect(
		page.getByRole("button", { name: "DARK" }).first(),
	).toHaveAttribute("aria-pressed", "true");
	await expect(page.getByRole("button", { name: "DARK" }).first()).toHaveClass(
		/bg-primary/,
	);

	await expect(page).toHaveURL(/\/$/);
	expect(await page.evaluate(() => localStorage.getItem("wga-theme"))).toBe(
		"dark",
	);
	expect(await page.context().cookies()).toEqual(
		expect.arrayContaining([
			expect.objectContaining({ name: "wga_theme", value: "dark" }),
		]),
	);

	await page.reload();
	await expect(page.locator("html")).toHaveAttribute(
		"data-theme",
		"wga-rams-dark",
	);
	await page.getByRole("button", { name: "LIGHT" }).first().click();
	await expect(page.locator("html")).toHaveAttribute("data-theme", "wga-rams");
	expect(await page.evaluate(() => localStorage.getItem("wga-theme"))).toBe(
		"light",
	);
});

test("uses a legacy stored theme without changing the applied Rams name", async ({
	page,
}) => {
	await page.addInitScript(() => {
		localStorage.setItem("wga-theme", "wga_dark");
	});
	await page.goto("/");

	await expect(page.locator("html")).toHaveAttribute(
		"data-theme",
		"wga-rams-dark",
	);
});

test("uses a cookie-only dark preference before the stylesheet loads", async ({
	page,
}) => {
	let themeAtStylesheetRequest = "";
	await page.route("**/assets/css/style.css", async (route) => {
		const theme = await page.locator("html").getAttribute("data-theme");
		if (theme) {
			themeAtStylesheetRequest = theme;
		}
		await route.continue();
	});
	await page.addInitScript(() => {
		localStorage.removeItem("wga-theme");
		document.cookie = "wga_theme=dark; path=/";
	});
	await page.emulateMedia({ colorScheme: "light" });
	await page.goto("/");

	expect(themeAtStylesheetRequest).toBe("wga-rams-dark");
	await expect(page.locator("html")).toHaveAttribute(
		"data-theme",
		"wga-rams-dark",
	);
});

test("uses a cookie-only light preference over a dark operating system", async ({
	page,
}) => {
	await page.addInitScript(() => {
		localStorage.removeItem("wga-theme");
		document.cookie = "wga_theme=light; path=/";
	});
	await page.emulateMedia({ colorScheme: "dark" });
	await page.goto("/");

	await expect(page.locator("html")).toHaveAttribute("data-theme", "wga-rams");
	expect(
		await page.evaluate(() => {
			const application = window as unknown as {
				wga: { theme: { current(): string } };
			};
			return application.wga.theme.current();
		}),
	).toBe("light");
});

test("uses a valid cookie when localStorage is unavailable", async ({
	page,
}) => {
	await page.addInitScript(() => {
		Object.defineProperty(window, "localStorage", {
			get() {
				throw new Error("storage unavailable");
			},
		});
		document.cookie = "wga_theme=dark; path=/";
	});
	await page.emulateMedia({ colorScheme: "light" });
	await page.goto("/");

	await expect(page.locator("html")).toHaveAttribute(
		"data-theme",
		"wga-rams-dark",
	);
});

test("returns to operating system tracking after clearing the preference", async ({
	page,
}) => {
	await page.emulateMedia({ colorScheme: "dark" });
	await page.goto("/");
	await expect(page.locator("html")).toHaveAttribute(
		"data-theme",
		"wga-rams-dark",
	);

	await page.getByRole("button", { name: "LIGHT" }).click();
	await page.emulateMedia({ colorScheme: "light" });
	await page.emulateMedia({ colorScheme: "dark" });
	await expect(page.locator("html")).toHaveAttribute("data-theme", "wga-rams");

	await page.evaluate(() => {
		const application = window as unknown as {
			wga: { theme: { clear(): void } };
		};
		application.wga.theme.clear();
	});
	await expect(page.locator("html")).toHaveAttribute(
		"data-theme",
		"wga-rams-dark",
	);
	expect(
		await page.evaluate(() => localStorage.getItem("wga-theme")),
	).toBeNull();
	expect(await page.context().cookies()).not.toEqual(
		expect.arrayContaining([expect.objectContaining({ name: "wga_theme" })]),
	);

	await page.emulateMedia({ colorScheme: "light" });
	await expect(page.locator("html")).toHaveAttribute("data-theme", "wga-rams");

	await page.evaluate(() => {
		for (const toggle of document.querySelectorAll("[data-wga-theme]")) {
			toggle.setAttribute("aria-pressed", "false");
		}
		document.dispatchEvent(new Event("htmx:afterSwap"));
	});
	await expect(page.getByRole("button", { name: "LIGHT" })).toHaveAttribute(
		"aria-pressed",
		"true",
	);
});
