import { expect, test } from "@playwright/test";

test.describe("bionic reading", () => {
	test.describe("without JavaScript", () => {
		test.use({ javaScriptEnabled: false });

		test("is unavailable without JavaScript", async ({ page }) => {
			await page.goto("/");

			await expect(page.locator("[data-wga-bionic-control]")).toHaveClass(
				/hidden/,
			);
		});
	});

	test("switches, restores, and remembers eligible prose", async ({ page }) => {
		await page.goto("/");

		const toggle = page.getByRole("switch", { name: "Bionic reading" });
		const prose = page.locator("main p:not(.font-mono)").first();
		const originalText = await prose.textContent();

		await expect(toggle).toHaveAttribute("aria-checked", "false");
		await toggle.click();
		await expect(page.locator("html")).toHaveAttribute(
			"data-bionic-reading",
			"true",
		);
		await expect(toggle).toHaveAttribute("aria-checked", "true");
		await expect(prose.locator("[data-bionic-mark]")).not.toHaveCount(0);
		expect(await prose.textContent()).toBe(originalText);
		expect(await page.evaluate(() => localStorage.getItem("wga-bionic"))).toBe(
			"on",
		);
		await page.reload();
		await expect(toggle).toHaveAttribute("aria-checked", "true");
		await expect(prose.locator("[data-bionic-mark]")).not.toHaveCount(0);

		await toggle.click();
		await expect(page.locator("html")).toHaveAttribute(
			"data-bionic-reading",
			"false",
		);
		await expect(prose.locator("[data-bionic-mark]")).toHaveCount(0);
		expect(await prose.textContent()).toBe(originalText);
	});

	test("limits the transformation to eligible prose", async ({ page }) => {
		await page.goto("/");

		await page.evaluate(() => {
			const fixture = document.createElement("section");
			fixture.innerHTML = `<p id="bionic-new">Fresh prose <strong id="bionic-strong">stays bold</strong>.</p><div data-bionic id="bionic-marked">Marked prose.</div><p data-bionic="off" id="bionic-off">Excluded prose.</p>`;
			document.querySelector("main")?.append(fixture);
		});
		await page.getByRole("switch", { name: "Bionic reading" }).click();

		await expect(
			page.locator("#bionic-new [data-bionic-mark]"),
		).not.toHaveCount(0);
		await expect(
			page.locator("#bionic-marked [data-bionic-mark]"),
		).not.toHaveCount(0);
		await expect(page.locator("#bionic-strong [data-bionic-mark]")).toHaveCount(
			0,
		);
		await expect(page.locator("#bionic-off [data-bionic-mark]")).toHaveCount(0);
		await expect(page.locator("footer [data-bionic-mark]")).toHaveCount(0);
	});

	test("applies the enabled preference to HTMX-loaded prose", async ({
		page,
	}) => {
		await page.addInitScript(() => localStorage.setItem("wga-bionic", "on"));
		await page.goto("/");

		await page.evaluate(() => {
			const target = document.createElement("section");
			target.innerHTML = `<p id="bionic-loaded">Loaded prose.</p>`;
			document.querySelector("main")?.append(target);
			document.body.dispatchEvent(
				new CustomEvent("htmx:load", {
					bubbles: true,
					detail: { elt: target },
				}),
			);
		});

		await expect(page.locator("#bionic-loaded")).toHaveText("Loaded prose.");
		await expect(
			page.locator("#bionic-loaded [data-bionic-mark]"),
		).not.toHaveCount(0);
	});
});
