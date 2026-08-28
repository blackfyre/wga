import { expect, test } from "@playwright/test";

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
	await page.locator("[data-wga-preferences-open]").click();

	const dark = page.locator('[data-wga-scheme="dark"]');
	await dark.click();
	await expect(page.locator("html")).toHaveAttribute(
		"data-theme",
		"wga-rams-dark",
	);
	await expect(dark).toHaveAttribute("aria-pressed", "true");
	await expect(dark).toHaveClass(/bg-primary/);

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
	await page.locator("[data-wga-preferences-open]").click();
	const light = page.locator('[data-wga-scheme="light"]');
	await light.click();
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
	// The injected storage getter deliberately throws during initialisation.
	resetErrorCapture();

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

	await page.locator("[data-wga-preferences-open]").click();
	await page.locator('[data-wga-scheme="light"]').click();
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

	await expect(page.locator("html")).toHaveAttribute(
		"data-theme",
		"wga-classic-dark",
	);
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
	expect(await page.context().cookies()).toEqual(
		expect.arrayContaining([
			expect.objectContaining({ name: "wga_palette", value: "classic" }),
		]),
	);
});

test("restores a cookie-only palette independently from its scheme", async ({
	page,
}) => {
	await page.addInitScript(() => {
		localStorage.removeItem("wga-theme");
		localStorage.removeItem("wga-palette");
		document.cookie = "wga_theme=dark; path=/";
		document.cookie = "wga_palette=classical; path=/";
	});
	await page.goto("/");

	await expect(page.locator("html")).toHaveAttribute(
		"data-theme",
		"wga-classical-dark",
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

	await expect(page.locator("html")).toHaveAttribute(
		"data-theme",
		"wga-classic-dark",
	);
	expect(await page.context().cookies()).toEqual(
		expect.arrayContaining([
			expect.objectContaining({ name: "wga_theme", value: "dark" }),
			expect.objectContaining({ name: "wga_palette", value: "classic" }),
		]),
	);
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

	await expect(page.locator("html")).toHaveAttribute(
		"data-theme",
		"wga-baroque",
	);
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
	await expect(page.locator("html")).toHaveAttribute(
		"data-theme",
		"wga-classic",
	);
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
		"data-theme",
		"wga-verdigris",
	);

	await page.emulateMedia({ colorScheme: "dark" });
	await expect(page.locator("html")).toHaveAttribute(
		"data-theme",
		"wga-verdigris-dark",
	);
	await page.emulateMedia({ colorScheme: "light" });
	await expect(page.locator("html")).toHaveAttribute(
		"data-theme",
		"wga-verdigris",
	);
});

const FIRST_PAINT_PALETTES: Array<{
	key: string;
	light: string;
	dark: string;
}> = [
	{ key: "bone", light: "wga-rams", dark: "wga-rams-dark" },
	{ key: "classic", light: "wga-classic", dark: "wga-classic-dark" },
	{ key: "verdigris", light: "wga-verdigris", dark: "wga-verdigris-dark" },
	{ key: "gothic", light: "wga-gothic", dark: "wga-gothic-dark" },
	{
		key: "renaissance",
		light: "wga-renaissance",
		dark: "wga-renaissance-dark",
	},
	{ key: "baroque", light: "wga-baroque", dark: "wga-baroque" },
	{ key: "rococo", light: "wga-rococo", dark: "wga-rococo-dark" },
	{ key: "classical", light: "wga-classical", dark: "wga-classical-dark" },
	{
		key: "impressionist",
		light: "wga-impressionist",
		dark: "wga-impressionist-dark",
	},
	{
		key: "catppuccin",
		light: "wga-catppuccin",
		dark: "wga-catppuccin-dark",
	},
	{ key: "tokyo", light: "wga-tokyo", dark: "wga-tokyo" },
];

for (const palette of FIRST_PAINT_PALETTES) {
	for (const scheme of ["light", "dark"] as const) {
		test(`resolves ${palette.key} ${scheme} before the stylesheet`, async ({
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
			await page.addInitScript(
				({ key, schemeValue }) => {
					localStorage.removeItem("wga-palette");
					localStorage.removeItem("wga-theme");
					document.cookie = `wga_palette=${key}; path=/`;
					document.cookie = `wga_theme=${schemeValue}; path=/`;
				},
				{ key: palette.key, schemeValue: scheme },
			);
			await page.goto("/");

			const expected = scheme === "dark" ? palette.dark : palette.light;
			expect(themeAtStylesheetRequest).toBe(expected);
			await expect(page.locator("html")).toHaveAttribute(
				"data-theme",
				expected,
			);
		});
	}
}

test("uses a cookie-only palette before the stylesheet", async ({ page }) => {
	let themeAtStylesheetRequest = "";
	await page.route("**/assets/css/style.css", async (route) => {
		const theme = await page.locator("html").getAttribute("data-theme");
		if (theme) {
			themeAtStylesheetRequest = theme;
		}
		await route.continue();
	});
	await page.addInitScript(() => {
		localStorage.removeItem("wga-palette");
		localStorage.removeItem("wga-theme");
		document.cookie = "wga_palette=verdigris; path=/";
	});
	await page.emulateMedia({ colorScheme: "light" });
	await page.goto("/");

	expect(themeAtStylesheetRequest).toBe("wga-verdigris");
	await expect(page.locator("html")).toHaveAttribute(
		"data-theme",
		"wga-verdigris",
	);
});

test("falls back from an invalid stored palette to a valid palette cookie before the stylesheet", async ({
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
		localStorage.setItem("wga-palette", "neon");
		localStorage.removeItem("wga-theme");
		document.cookie = "wga_palette=classical; path=/";
	});
	await page.emulateMedia({ colorScheme: "dark" });
	await page.goto("/");

	expect(themeAtStylesheetRequest).toBe("wga-classical-dark");
	await expect(page.locator("html")).toHaveAttribute(
		"data-theme",
		"wga-classical-dark",
	);
});
