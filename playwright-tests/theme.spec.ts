import { type BrowserContext, expect, test } from "@playwright/test";

import {
	expectNoPageErrors,
	guardPageErrors,
	resetErrorCapture,
} from "./helpers/page-errors";

test.beforeEach(async ({ page }) => {
	resetErrorCapture();
	guardPageErrors(page);
});

test.afterEach(() => {
	expectNoPageErrors();
});

async function addLegacyAppearanceCookies(
	context: BrowserContext,
	baseURL: string | undefined,
	cookies: Array<{ name: string; value: string }>,
): Promise<void> {
	if (!baseURL) {
		throw new Error("Playwright baseURL is required");
	}
	await context.addCookies(
		cookies.map((cookie) => ({ ...cookie, url: baseURL })),
	);
}

for (const colorScheme of ["light", "dark"] as const) {
	test(`uses the browser ${colorScheme} preference by default`, async ({
		page,
	}) => {
		await page.emulateMedia({ colorScheme });
		await page.goto("/");

		await expect(page.locator("html")).toHaveAttribute("data-palette", "bone");
		await expect(page.locator("html")).toHaveAttribute(
			"data-theme",
			colorScheme,
		);
		await expect(page.locator("html")).toHaveCSS("color-scheme", colorScheme);
	});
}

test.describe("without JavaScript", () => {
	test.use({ javaScriptEnabled: false });

	for (const preference of [
		{ scheme: "light", background: "#f4f2ed" },
		{ scheme: "dark", background: "#1a1814" },
	] as const) {
		test(`uses the browser ${preference.scheme} preference for native roles`, async ({
			page,
		}) => {
			await page.emulateMedia({ colorScheme: preference.scheme });
			await page.goto("/");

			await expect(page.locator("html")).not.toHaveAttribute("data-palette");
			await expect(page.locator("html")).not.toHaveAttribute("data-theme");
			await expect(page.locator("html")).toHaveCSS(
				"color-scheme",
				preference.scheme,
			);
			expect(
				await page
					.locator("html")
					.evaluate((root) =>
						getComputedStyle(root)
							.getPropertyValue("--wga-bg")
							.trim()
							.toLowerCase(),
					),
			).toBe(preference.background);
		});
	}
});

test("switches and remembers the selected colour scheme", async ({ page }) => {
	await page.goto("/");
	await page.locator("[data-wga-preferences-open]").click();

	const dark = page.locator('[data-wga-scheme="dark"]');
	await dark.click();
	await expect(page.locator("html")).toHaveAttribute("data-palette", "bone");
	await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
	await expect(dark).toHaveAttribute("aria-pressed", "true");
	await expect(dark).toHaveClass(/bg-wga-accent-bg/);

	await expect(page).toHaveURL(/\/$/);
	expect(await page.evaluate(() => localStorage.getItem("wga-theme"))).toBe(
		"dark",
	);

	await page.reload();
	await expect(page.locator("html")).toHaveAttribute("data-palette", "bone");
	await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
	await page.locator("[data-wga-preferences-open]").click();
	const light = page.locator('[data-wga-scheme="light"]');
	await light.click();
	await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
	expect(await page.evaluate(() => localStorage.getItem("wga-theme"))).toBe(
		"light",
	);
});

test("normalises a legacy stored scheme into native root state", async ({
	page,
}) => {
	await page.addInitScript(() => {
		localStorage.setItem("wga-theme", "wga_dark");
	});
	await page.goto("/");

	await expect(page.locator("html")).toHaveAttribute("data-palette", "bone");
	await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
});

test("ignores a cookie-only dark preference before the stylesheet loads", async ({
	page,
	context,
	baseURL,
}) => {
	let appearanceAtStylesheetRequest = { palette: "", theme: "" };
	await page.route("**/assets/css/style.css", async (route) => {
		appearanceAtStylesheetRequest = {
			palette: (await page.locator("html").getAttribute("data-palette")) ?? "",
			theme: (await page.locator("html").getAttribute("data-theme")) ?? "",
		};
		await route.continue();
	});
	await addLegacyAppearanceCookies(context, baseURL, [
		{ name: "wga_theme", value: "dark" },
	]);
	await page.addInitScript(() => localStorage.removeItem("wga-theme"));
	await page.emulateMedia({ colorScheme: "light" });
	await page.goto("/");

	expect(appearanceAtStylesheetRequest).toEqual({
		palette: "bone",
		theme: "light",
	});
	await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
});

test("ignores a cookie-only light preference and follows the operating system", async ({
	page,
	context,
	baseURL,
}) => {
	await addLegacyAppearanceCookies(context, baseURL, [
		{ name: "wga_theme", value: "light" },
	]);
	await page.addInitScript(() => localStorage.removeItem("wga-theme"));
	await page.emulateMedia({ colorScheme: "dark" });
	await page.goto("/");

	await expect(page.locator("html")).toHaveAttribute("data-palette", "bone");
	await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
	expect(
		await page.evaluate(() => {
			const application = window as unknown as {
				wga: { theme: { current(): string } };
			};
			return application.wga.theme.current();
		}),
	).toBe("dark");
});

test("uses operating-system defaults when localStorage is unavailable", async ({
	page,
	context,
	baseURL,
}) => {
	await addLegacyAppearanceCookies(context, baseURL, [
		{ name: "wga_theme", value: "dark" },
	]);
	await page.addInitScript(() => {
		Object.defineProperty(window, "localStorage", {
			get() {
				throw new Error("storage unavailable");
			},
		});
	});
	await page.emulateMedia({ colorScheme: "light" });
	await page.goto("/");
	// The injected storage getter deliberately throws during initialisation.
	resetErrorCapture();

	await expect(page.locator("html")).toHaveAttribute("data-palette", "bone");
	await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
});

test("returns to operating system tracking after clearing the preference", async ({
	page,
}) => {
	await page.emulateMedia({ colorScheme: "dark" });
	await page.goto("/");
	await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");

	await page.locator("[data-wga-preferences-open]").click();
	await page.locator('[data-wga-scheme="light"]').click();
	await page.emulateMedia({ colorScheme: "light" });
	await page.emulateMedia({ colorScheme: "dark" });
	await expect(page.locator("html")).toHaveAttribute("data-theme", "light");

	await page.evaluate(() => {
		const application = window as unknown as {
			wga: { theme: { clear(): void } };
		};
		application.wga.theme.clear();
	});
	await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
	expect(
		await page.evaluate(() => localStorage.getItem("wga-theme")),
	).toBeNull();

	await page.emulateMedia({ colorScheme: "light" });
	await expect(page.locator("html")).toHaveAttribute("data-theme", "light");

	await page.evaluate(() => {
		for (const toggle of document.querySelectorAll("[data-wga-scheme]")) {
			toggle.setAttribute("aria-pressed", "false");
		}
		document.dispatchEvent(new Event("htmx:afterSwap"));
	});
	await expect(page.locator('[data-wga-scheme="light"]')).toHaveAttribute(
		"aria-pressed",
		"true",
	);
});

test("changes palette without changing the explicit scheme", async ({
	page,
}) => {
	await page.goto("/");
	await page.locator("[data-wga-preferences-open]").click();
	await page.locator('[data-wga-scheme="dark"]').click();
	await page.locator('[data-wga-palette="classic"]').click();
	await page.reload();

	await expect(page.locator("html")).toHaveAttribute("data-palette", "classic");
	await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
	await expect(page.locator('[data-wga-scheme="dark"]')).toHaveAttribute(
		"aria-pressed",
		"true",
	);
	await expect(page.locator('[data-wga-palette="classic"]')).toHaveAttribute(
		"aria-checked",
		"true",
	);
	expect(await page.evaluate(() => localStorage.getItem("wga-theme"))).toBe(
		"dark",
	);
	expect(await page.evaluate(() => localStorage.getItem("wga-palette"))).toBe(
		"classic",
	);
});

test("ignores cookie-only palette and scheme preferences", async ({
	page,
	context,
	baseURL,
}) => {
	await addLegacyAppearanceCookies(context, baseURL, [
		{ name: "wga_theme", value: "dark" },
		{ name: "wga_palette", value: "classical" },
	]);
	await page.addInitScript(() => {
		localStorage.removeItem("wga-theme");
		localStorage.removeItem("wga-palette");
	});
	await page.goto("/");

	await expect(page.locator("html")).toHaveAttribute("data-palette", "bone");
	await expect(page.locator("html")).toHaveAttribute(
		"data-theme",
		await page.evaluate(() =>
			matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light",
		),
	);
	expect(
		await page.evaluate(() => localStorage.getItem("wga-theme")),
	).toBeNull();
	expect(
		await page.evaluate(() => localStorage.getItem("wga-palette")),
	).toBeNull();
});

test("keeps session palette and scheme choices when storage is blocked", async ({
	page,
}) => {
	await page.addInitScript(() => {
		Object.defineProperty(window, "localStorage", {
			value: {
				getItem: () => {
					throw new Error("blocked");
				},
				setItem: () => {
					throw new Error("blocked");
				},
				removeItem: () => {
					throw new Error("blocked");
				},
			},
		});
	});
	await page.goto("/");
	await page.waitForFunction(() => "wga" in window);
	// The injected storage methods deliberately throw during initialisation.
	resetErrorCapture();
	await page.evaluate(() => {
		const application = window as unknown as {
			wga: {
				palette: { set(value: "classic"): void };
				theme: { set(value: "dark"): void };
			};
		};
		application.wga.theme.set("dark");
		application.wga.palette.set("classic");
	});

	await expect(page.locator("html")).toHaveAttribute("data-palette", "classic");
	await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
	const appearanceCookies = (await page.context().cookies()).filter(
		({ name }) => ["wga_theme", "wga_palette"].includes(name),
	);
	expect(appearanceCookies).toEqual([]);
});

test("dark-only palettes preserve the stored light scheme", async ({
	page,
}) => {
	await page.goto("/");
	await page.locator("[data-wga-preferences-open]").click();
	const light = page.locator('[data-wga-scheme="light"]');
	await light.click();
	await page.evaluate(() => {
		const application = window as unknown as {
			wga: { palette: { set(value: "baroque" | "classic"): void } };
		};
		application.wga.palette.set("baroque");
	});

	await expect(page.locator("html")).toHaveAttribute("data-palette", "baroque");
	await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
	await expect(light).toBeDisabled();
	await expect(light).toHaveAttribute(
		"title",
		"BAROQUE is a dark-only palette",
	);
	expect(await page.evaluate(() => localStorage.getItem("wga-theme"))).toBe(
		"light",
	);

	await page.evaluate(() => {
		const application = window as unknown as {
			wga: { palette: { set(value: "classic"): void } };
		};
		application.wga.palette.set("classic");
	});
	await expect(page.locator("html")).toHaveAttribute("data-palette", "classic");
	await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
	await expect(light).toBeEnabled();
});

test("an unset scheme continues following live operating-system changes", async ({
	page,
}) => {
	await page.emulateMedia({ colorScheme: "light" });
	await page.goto("/");
	await page.waitForFunction(() => "wga" in window);
	await page.evaluate(() => {
		const application = window as unknown as {
			wga: {
				palette: { set(value: "verdigris"): void };
				theme: { clear(): void };
			};
		};
		application.wga.palette.set("verdigris");
		application.wga.theme.clear();
	});
	await expect(page.locator("html")).toHaveAttribute(
		"data-palette",
		"verdigris",
	);
	await expect(page.locator("html")).toHaveAttribute("data-theme", "light");

	await page.emulateMedia({ colorScheme: "dark" });
	await expect(page.locator("html")).toHaveAttribute(
		"data-palette",
		"verdigris",
	);
	await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
	await page.emulateMedia({ colorScheme: "light" });
	await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
});

const FIRST_PAINT_PALETTES: Array<{ key: string; darkOnly?: boolean }> = [
	{ key: "bone" },
	{ key: "classic" },
	{ key: "verdigris" },
	{ key: "gothic" },
	{ key: "renaissance" },
	{ key: "baroque", darkOnly: true },
	{ key: "rococo" },
	{ key: "classical" },
	{ key: "impressionist" },
	{ key: "catppuccin" },
	{ key: "tokyo", darkOnly: true },
];

for (const palette of FIRST_PAINT_PALETTES) {
	for (const scheme of ["light", "dark"] as const) {
		test(`resolves ${palette.key} ${scheme} before the stylesheet`, async ({
			page,
		}) => {
			let appearanceAtStylesheetRequest = { palette: "", theme: "" };
			await page.route("**/assets/css/style.css", async (route) => {
				appearanceAtStylesheetRequest = {
					palette:
						(await page.locator("html").getAttribute("data-palette")) ?? "",
					theme: (await page.locator("html").getAttribute("data-theme")) ?? "",
				};
				await route.continue();
			});
			await page.addInitScript(
				({ key, schemeValue }) => {
					localStorage.setItem("wga-palette", key);
					localStorage.setItem("wga-theme", schemeValue);
				},
				{ key: palette.key, schemeValue: scheme },
			);
			await page.goto("/");

			const expectedTheme = palette.darkOnly ? "dark" : scheme;
			expect(appearanceAtStylesheetRequest).toEqual({
				palette: palette.key,
				theme: expectedTheme,
			});
			await expect(page.locator("html")).toHaveAttribute(
				"data-palette",
				palette.key,
			);
			await expect(page.locator("html")).toHaveAttribute(
				"data-theme",
				expectedTheme,
			);
		});
	}
}

test("ignores a cookie-only palette before the stylesheet", async ({
	page,
	context,
	baseURL,
}) => {
	let appearanceAtStylesheetRequest = { palette: "", theme: "" };
	await page.route("**/assets/css/style.css", async (route) => {
		appearanceAtStylesheetRequest = {
			palette: (await page.locator("html").getAttribute("data-palette")) ?? "",
			theme: (await page.locator("html").getAttribute("data-theme")) ?? "",
		};
		await route.continue();
	});
	await addLegacyAppearanceCookies(context, baseURL, [
		{ name: "wga_palette", value: "verdigris" },
	]);
	await page.addInitScript(() => {
		localStorage.removeItem("wga-palette");
		localStorage.removeItem("wga-theme");
	});
	await page.emulateMedia({ colorScheme: "light" });
	await page.goto("/");

	expect(appearanceAtStylesheetRequest).toEqual({
		palette: "bone",
		theme: "light",
	});
	await expect(page.locator("html")).toHaveAttribute("data-palette", "bone");
	await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
});

test("falls back from an invalid stored palette to bone before the stylesheet", async ({
	page,
	context,
	baseURL,
}) => {
	let appearanceAtStylesheetRequest = { palette: "", theme: "" };
	await page.route("**/assets/css/style.css", async (route) => {
		appearanceAtStylesheetRequest = {
			palette: (await page.locator("html").getAttribute("data-palette")) ?? "",
			theme: (await page.locator("html").getAttribute("data-theme")) ?? "",
		};
		await route.continue();
	});
	await addLegacyAppearanceCookies(context, baseURL, [
		{ name: "wga_palette", value: "classical" },
	]);
	await page.addInitScript(() => {
		localStorage.setItem("wga-palette", "neon");
		localStorage.removeItem("wga-theme");
	});
	await page.emulateMedia({ colorScheme: "dark" });
	await page.goto("/");

	expect(appearanceAtStylesheetRequest).toEqual({
		palette: "bone",
		theme: "dark",
	});
	await expect(page.locator("html")).toHaveAttribute("data-palette", "bone");
	await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
});
