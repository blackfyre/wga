import { type Page, expect, test } from "@playwright/test";

import {
	expectNoPageErrors,
	guardPageErrors,
	resetErrorCapture,
} from "./helpers/page-errors";

const PALETTE_KEYS = [
	"bone",
	"classic",
	"verdigris",
	"gothic",
	"renaissance",
	"baroque",
	"rococo",
	"classical",
	"impressionist",
	"catppuccin",
	"tokyo",
] as const;

const DARK_ONLY: Array<{ key: string; label: string; theme: string }> = [
	{ key: "baroque", label: "BAROQUE", theme: "wga-baroque" },
	{ key: "tokyo", label: "TOKYO NIGHT", theme: "wga-tokyo" },
];

function openPreferences(page: Page) {
	return page.locator("[data-wga-preferences-open]").click();
}

test.beforeEach(async ({ page }) => {
	resetErrorCapture();
	guardPageErrors(page);
});

test.afterEach(() => {
	expectNoPageErrors();
});

test("trigger summary states palette, effective scheme, and bionic reading", async ({
	page,
}) => {
	await page.goto("/");
	await openPreferences(page);

	await page.locator('[data-wga-palette="verdigris"]').click();
	await page.locator('[data-wga-scheme="dark"]').click();
	await page.getByRole("switch", { name: "Bionic reading" }).click();

	await expect(page.locator("[data-wga-preferences-summary]")).toHaveText(
		"VERDIGRIS · DARK · BIONIC",
	);
});

test("groups palette choices by provenance", async ({ page }) => {
	await page.goto("/");
	await openPreferences(page);

	await expect(page.locator("[data-wga-preferences-panel] h3")).toHaveText([
		"THIS ARCHIVE",
		"FROM THE COLLECTION",
		"BORROWED",
	]);
});

test("presents each palette with text and a split swatch", async ({ page }) => {
	await page.goto("/");
	await openPreferences(page);

	const rows = page.locator("[data-wga-preferences-panel] [data-wga-palette]");
	await expect(rows).toHaveCount(11);
	for (const key of PALETTE_KEYS) {
		const row = page.locator(`[data-wga-palette="${key}"]`);
		await expect(row.locator("[data-wga-palette-name]")).not.toHaveText("");
		await expect(row.locator("span[style]")).toHaveAttribute(
			"style",
			/background:linear-gradient/,
		);
	}
});

test("moves the IN USE marker and label highlight on palette selection", async ({
	page,
}) => {
	await page.goto("/");
	await openPreferences(page);

	await expect(
		page.locator('[data-wga-palette="bone"] [data-wga-palette-in-use]'),
	).toHaveCount(1);
	await expect(
		page.locator('[data-wga-palette="classic"] [data-wga-palette-in-use]'),
	).toHaveCount(0);

	await page.locator('[data-wga-palette="classic"]').click();

	await expect(
		page.locator('[data-wga-palette="bone"] [data-wga-palette-in-use]'),
	).toHaveCount(0);
	await expect(
		page.locator('[data-wga-palette="classic"] [data-wga-palette-in-use]'),
	).toHaveCount(1);
	await expect(page.locator('[data-wga-palette="classic"]')).toHaveClass(
		/bg-primary\/10/,
	);
	await expect(page.locator('[data-wga-palette="bone"]')).not.toHaveClass(
		/bg-primary\/10/,
	);
	await expect(
		page.locator('[data-wga-palette="classic"] [data-wga-palette-name]'),
	).toHaveClass(/text-primary/);
	await expect(
		page.locator('[data-wga-palette="bone"] [data-wga-palette-name]'),
	).toHaveClass(/text-base-content/);
});

for (const palette of DARK_ONLY) {
	test(`${palette.key} disables LIGHT with a reason and preserves the stored scheme`, async ({
		page,
	}) => {
		await page.goto("/");
		await openPreferences(page);

		const light = page.locator('[data-wga-scheme="light"]');
		await light.click();
		await expect(light).toHaveAttribute("aria-pressed", "true");

		await page.locator(`[data-wga-palette="${palette.key}"]`).click();

		await expect(page.locator("html")).toHaveAttribute(
			"data-theme",
			palette.theme,
		);
		await expect(light).toBeDisabled();
		await expect(light).toHaveAttribute(
			"title",
			`${palette.label} is a dark-only palette`,
		);
		await expect(page.locator("[data-wga-scheme-explanation]")).toHaveText(
			`${palette.label} has no light build, so light is unavailable while it is chosen.`,
		);
		expect(await page.evaluate(() => localStorage.getItem("wga-theme"))).toBe(
			"light",
		);

		await page.locator('[data-wga-palette="classic"]').click();
		await expect(light).toBeEnabled();
		await expect(page.locator("html")).toHaveAttribute(
			"data-theme",
			"wga-classic",
		);
	});
}

test("opens a labelled modal with initial focus and restores the trigger on Escape", async ({
	page,
}) => {
	await page.goto("/");

	const trigger = page.locator("[data-wga-preferences-open]");
	await trigger.focus();
	await trigger.click();

	const panel = page.locator("#wga-preferences");
	await expect(panel).toHaveJSProperty("open", true);
	await expect(panel).toHaveAttribute("aria-label", "Preferences");
	await expect(page.locator("[data-wga-preferences-close]")).toBeFocused();

	await page.keyboard.press("Escape");
	await expect(panel).toHaveJSProperty("open", false);
	await expect(trigger).toBeFocused();
});

test("remains functional after repeated footer replacement", async ({
	page,
}) => {
	await page.goto("/tmp/visual-overhaul/footer");
	await page.waitForFunction(() => "wga" in window);

	await page.evaluate(() => {
		document.documentElement.dataset.footerSwaps = "0";
		document.addEventListener("htmx:afterSwap", (event) => {
			if (
				(event as CustomEvent<{ target?: Element }>).detail?.target?.matches(
					"footer",
				)
			) {
				document.documentElement.dataset.footerSwaps = String(
					Number(document.documentElement.dataset.footerSwaps) + 1,
				);
			}
		});
	});
	for (let index = 1; index <= 2; index += 1) {
		await page.getByRole("button", { name: "Replace footer" }).click();
		await page.waitForFunction(
			(count) => document.documentElement.dataset.footerSwaps === String(count),
			index,
		);
	}

	await expect(page.locator("[data-wga-preferences-control]")).toHaveClass(
		/flex/,
	);
	await expect(page.locator("[data-wga-bionic-control]")).toHaveClass(/flex/);
	const trigger = page.locator("[data-wga-preferences-open]");
	await trigger.click();
	await expect(page.locator("#wga-preferences")).toHaveJSProperty("open", true);
	await expect(page.locator("[data-wga-preferences-close]")).toBeFocused();
	await page.keyboard.press("Escape");
	await expect(page.locator("#wga-preferences")).toHaveJSProperty(
		"open",
		false,
	);
	await expect(trigger).toBeFocused();
	await trigger.click();
	await page.locator("[data-wga-bionic-toggle]").click();
	await expect(page.locator("html")).toHaveAttribute(
		"data-bionic-reading",
		"true",
	);
});

test.describe("without JavaScript", () => {
	test.use({ javaScriptEnabled: false });

	test("hides manual controls and follows the operating system", async ({
		page,
	}) => {
		await page.emulateMedia({ colorScheme: "dark" });
		await page.goto("/");

		await expect(page.locator("[data-wga-preferences-control]")).toHaveClass(
			/hidden/,
		);
		await expect(page.locator("[data-wga-bionic-control]")).toHaveClass(
			/hidden/,
		);
		await expect(page.locator("html")).toHaveCSS("color-scheme", "dark");
	});
});

for (const viewport of [
	{ width: 390, height: 844 },
	{ width: 834, height: 1112 },
	{ width: 1440, height: 900 },
]) {
	test(`preferences panel is composed at ${viewport.width}px`, async ({
		page,
	}) => {
		await page.setViewportSize({
			width: viewport.width,
			height: viewport.height,
		});
		await page.goto("/");
		await openPreferences(page);

		const panel = page.locator("#wga-preferences");
		await expect(panel).toHaveJSProperty("open", true);

		const dimensions = await page.evaluate(() => ({
			clientWidth: document.documentElement.clientWidth,
			scrollWidth: document.documentElement.scrollWidth,
		}));
		expect(dimensions.scrollWidth).toBeLessThanOrEqual(dimensions.clientWidth);

		const box = await panel.boundingBox();
		expect(box).not.toBeNull();
		if (box) {
			expect(box.x).toBeGreaterThanOrEqual(0);
			expect(box.x + box.width).toBeLessThanOrEqual(viewport.width + 1);
		}
	});
}

test("preferences panel reflows at 200% text without overflow", async ({
	page,
}) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await page.goto("/");
	await page.evaluate(() => {
		document.documentElement.style.fontSize = "2em";
	});
	await openPreferences(page);

	const panel = page.locator("#wga-preferences");
	await expect(panel).toHaveJSProperty("open", true);
	const dimensions = await page.evaluate(() => ({
		clientWidth: document.documentElement.clientWidth,
		scrollWidth: document.documentElement.scrollWidth,
	}));
	expect(dimensions.scrollWidth).toBeLessThanOrEqual(dimensions.clientWidth);
});

test("preferences panel opens and closes under reduced motion", async ({
	page,
}) => {
	await page.emulateMedia({ reducedMotion: "reduce" });
	await page.goto("/");

	const trigger = page.locator("[data-wga-preferences-open]");
	await trigger.click();
	await expect(page.locator("#wga-preferences")).toHaveJSProperty("open", true);
	await expect(page.locator("[data-wga-preferences-close]")).toBeFocused();
	await page.keyboard.press("Escape");
	await expect(page.locator("#wga-preferences")).toHaveJSProperty(
		"open",
		false,
	);
});
