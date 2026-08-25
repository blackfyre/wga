import { expect, test } from "@playwright/test";

const fixture = `
	<main style="min-height: 40rem; padding: 4rem 2rem; overflow: visible">
		<p style="margin: 12rem auto; max-width: 20rem">
			<dfn id="term" class="wga-term" role="note" tabindex="0" aria-label="Triptych: A three-panel work painted on a folding support." data-bionic="off">Triptych<span id="term-tip" class="wga-term__tooltip" aria-hidden="true"><span class="wga-tooltip__meta">GLOSSARY</span><span class="wga-tooltip__category">OBJECT TYPE</span><span class="wga-tooltip__body">A three-panel work painted on a folding support.</span></span></dfn>
			<span id="help" class="wga-help" role="note" tabindex="0" aria-label="Selection: Choose up to three works for comparison." data-bionic="off">?<span id="help-tip" class="wga-help__tooltip" aria-hidden="true"><span class="wga-tooltip__meta">Selection</span><span class="wga-tooltip__body">Choose up to three works for comparison.</span></span></span>
		</p>
		<p style="display: flex; justify-content: flex-end; gap: 1rem; margin: 12rem 0 0">
			<dfn id="right-term" class="wga-term" role="note" tabindex="0" aria-label="Triptych: A three-panel work painted on a folding support." data-bionic="off">Triptych<span id="right-term-tip" class="wga-term__tooltip" aria-hidden="true"><span class="wga-tooltip__meta">GLOSSARY</span><span class="wga-tooltip__category">OBJECT TYPE</span><span class="wga-tooltip__body">A three-panel work painted on a folding support.</span></span></dfn>
			<span id="right-help" class="wga-help" role="note" tabindex="0" aria-label="Selection: Choose up to three works for comparison." data-bionic="off">?<span id="right-help-tip" class="wga-help__tooltip" aria-hidden="true"><span class="wga-tooltip__meta">Selection</span><span class="wga-tooltip__body">Choose up to three works for comparison.</span></span></span>
		</p>
		<a id="after" href="#after">After</a>
	</main>`;

async function mountFixture(page: import("@playwright/test").Page) {
	await page.goto("/");
	await page.setContent(
		`<link rel="stylesheet" href="/assets/css/style.css">${fixture}`,
	);
	await expect(page.locator("#term")).toHaveCSS("cursor", "help");
}

async function expectViewportContainment(
	page: import("@playwright/test").Page,
	tooltip: ReturnType<import("@playwright/test").Page["locator"]>,
	viewportWidth: number,
) {
	await expect(tooltip).toBeVisible();
	const box = await tooltip.boundingBox();
	if (!box) {
		throw new Error("Expected tooltip geometry");
	}
	expect(box.x).toBeGreaterThanOrEqual(0);
	expect(box.x + box.width).toBeLessThanOrEqual(viewportWidth);
	expect(
		await tooltip.evaluate(
			(element) => element.scrollWidth === element.clientWidth,
		),
	).toBe(true);
	expect(
		await page.evaluate(
			() =>
				document.documentElement.scrollWidth ===
				document.documentElement.clientWidth,
		),
	).toBe(true);
}

function contrastRatio(foreground: string, background: string): number {
	const channels = (colour: string) =>
		colour
			.match(/\d+(?:\.\d+)?/g)
			?.slice(0, 3)
			.map(Number) || [];
	const luminance = (colour: string) => {
		const values = channels(colour).map((channel) => {
			const value = channel / 255;
			if (value <= 0.04045) {
				return value / 12.92;
			}
			return ((value + 0.055) / 1.055) ** 2.4;
		});
		return values[0] * 0.2126 + values[1] * 0.7152 + values[2] * 0.0722;
	};
	const first = luminance(foreground);
	const second = luminance(background);
	if (first > second) {
		return (first + 0.05) / (second + 0.05);
	}
	return (second + 0.05) / (first + 0.05);
}

test("shared help surfaces expose matching hover and keyboard definitions", async ({
	page,
}) => {
	await page.setViewportSize({ width: 800, height: 844 });
	await mountFixture(page);

	const term = page.locator("#term");
	const termTip = page.locator("#term-tip");
	const help = page.locator("#help");
	const helpTip = page.locator("#help-tip");
	await expect(term).toHaveAccessibleName(
		"Triptych: A three-panel work painted on a folding support.",
	);
	await expect(help).toHaveAccessibleName(
		"Selection: Choose up to three works for comparison.",
	);
	await expect(termTip).toHaveAttribute("aria-hidden", "true");
	await expect(helpTip).toHaveAttribute("aria-hidden", "true");
	await expect(termTip).toHaveCSS("visibility", "hidden");

	await term.hover();
	await expect(termTip).toHaveCSS("visibility", "visible");
	await expect(termTip).toContainText("GLOSSARY");
	await expect(termTip).toContainText("OBJECT TYPE");
	await expect(termTip).toContainText(
		"A three-panel work painted on a folding support.",
	);
	await page.mouse.move(380, 800);
	await page.getByRole("link", { name: "After" }).focus();
	await expect(termTip).toHaveCSS("visibility", "hidden");

	await term.focus();
	await expect(termTip).toHaveCSS("visibility", "visible");
	const termBox = await term.boundingBox();
	const termTipBox = await termTip.boundingBox();
	if (!termBox || !termTipBox) {
		throw new Error("Expected glossary tooltip geometry");
	}
	expect(termTipBox.y + termTipBox.height).toBeLessThanOrEqual(termBox.y);
	expect(termTipBox.x).toBeGreaterThanOrEqual(0);
	expect(termTipBox.x + termTipBox.width).toBeLessThanOrEqual(800);

	await help.focus();
	await expect(helpTip).toHaveCSS("visibility", "visible");
	const helpBox = await help.boundingBox();
	const helpTipBox = await helpTip.boundingBox();
	if (!helpBox || !helpTipBox) {
		throw new Error("Expected help tooltip geometry");
	}
	expect(helpTipBox.y).toBeGreaterThanOrEqual(helpBox.y + helpBox.height);
});

test("keyboard help exposes the complete registry and accessible lifecycle", async ({
	page,
}) => {
	await page.goto("/");
	await page.waitForFunction(
		() => document.documentElement.dataset.keyboardNavigationReady === "true",
	);
	const payload = await page.locator("#kbd-screens").getAttribute("data-json");
	if (!payload) {
		throw new Error("Expected the server keyboard registry payload");
	}
	const registry = JSON.parse(payload) as Array<{
		key: string;
		num: string;
		label: string;
	}>;

	const opener = page.getByRole("button", { name: "Keyboard shortcuts" });
	await opener.focus();
	await page.keyboard.press("?");
	const dialog = page.getByRole("dialog", { name: "Moving without the mouse" });
	await expect(dialog).toBeVisible();
	await expect(
		page.getByRole("button", { name: "Close shortcuts" }),
	).toBeFocused();
	const helpRows = dialog.locator("p");
	for (const screen of registry) {
		await expect(
			helpRows
				.filter({ hasText: screen.num })
				.filter({ hasText: screen.label }),
		).toHaveCount(1);
	}
	for (const action of [
		"BROWSING",
		"FINDING",
		"CONTEXTUAL",
		"Focus visible search; opens mobile navigation first",
		"Open Go to",
		"Open or close this help",
	]) {
		await expect(dialog).toContainText(action);
	}
	await expect(dialog).toContainText("CTRL K");
	await page.keyboard.press("Escape");
	await expect(dialog).not.toBeVisible();
	await expect(opener).toBeFocused();
});

test("right-edge help surfaces remain contained at normal and enlarged text", async ({
	page,
}) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await mountFixture(page);

	for (const rootFontSize of ["100%", "200%"]) {
		await page.locator("html").evaluate((element, value) => {
			element.style.fontSize = value;
		}, rootFontSize);
		await page.locator("#right-term").focus();
		await expectViewportContainment(page, page.locator("#right-term-tip"), 390);
		await expect(page.locator("#right-term-tip")).toContainText("GLOSSARY");

		await page.locator("#right-help").focus();
		await expectViewportContainment(page, page.locator("#right-help-tip"), 390);
		await expect(page.locator("#right-help-tip")).toContainText(
			"Choose up to three works for comparison.",
		);
	}
});

test("shared help surfaces remain readable across themes, enlarged text, and reduced motion", async ({
	page,
}) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await mountFixture(page);

	for (const theme of ["wga-rams", "wga-rams-dark"]) {
		await page.locator("html").evaluate((element, value) => {
			element.setAttribute("data-theme", value);
		}, theme);
		await page.locator("#help").focus();
		const colours = await page.locator("#help-tip").evaluate((element) => {
			const style = getComputedStyle(element);
			return { background: style.backgroundColor, colour: style.color };
		});
		expect(contrastRatio(colours.colour, colours.background)).toBeGreaterThan(
			4.5,
		);
	}

	await expect(page.locator("#term-tip .wga-tooltip__body")).toHaveCSS(
		"overflow-wrap",
		"anywhere",
	);

	await page.emulateMedia({ reducedMotion: "reduce" });
	const transitionDuration = await page
		.locator("#term-tip")
		.evaluate((element) => getComputedStyle(element).transitionDuration);
	expect(Number.parseFloat(transitionDuration)).toBeLessThanOrEqual(0.01);
});

test.describe("without JavaScript", () => {
	test.use({ javaScriptEnabled: false });

	test("shared help surfaces reveal on hover and keyboard focus", async ({
		page,
	}) => {
		await page.setViewportSize({ width: 390, height: 844 });
		await mountFixture(page);
		await page.locator("#right-term").hover();
		await expect(page.locator("#right-term-tip")).toHaveCSS(
			"visibility",
			"visible",
		);
		await page.locator("#right-help").focus();
		await expect(page.locator("#right-help-tip")).toHaveCSS(
			"visibility",
			"visible",
		);
	});
});
