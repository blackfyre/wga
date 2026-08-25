import { expect, test } from "@playwright/test";

test.describe("bionic reading", () => {
	test.describe("without JavaScript", () => {
		test.use({ javaScriptEnabled: false });

		test("is unavailable without JavaScript", async ({ page }) => {
			await page.goto("/");

			await expect(page.locator("[data-wga-bionic-control]")).toHaveClass(
				/hidden/,
			);
			await expect(page.locator("main p").first()).toHaveCount(1);
			await expect(
				page.locator("main p").first().locator("[data-bionic-mark]"),
			).toHaveCount(0);
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
		expect(await page.evaluate(() => document.cookie)).toContain(
			"wga_bionic=on",
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
		expect(await page.evaluate(() => localStorage.getItem("wga-bionic"))).toBe(
			"off",
		);
		expect(await page.evaluate(() => document.cookie)).toContain(
			"wga_bionic=off",
		);
	});

	test("defaults off for missing, inaccessible, and invalid storage", async ({
		page,
	}) => {
		await page.addInitScript(() => {
			Object.defineProperty(window, "localStorage", {
				value: { getItem: () => "invalid", setItem: () => undefined },
			});
		});
		await page.goto("/");
		await expect(
			page.getByRole("switch", { name: "Bionic reading" }),
		).toHaveAttribute("aria-checked", "false");
	});

	test("limits the transformation to eligible prose", async ({ page }) => {
		await page.goto("/");

		await page.evaluate(() => {
			const fixture = document.createElement("section");
			fixture.innerHTML = `<p id="bionic-new">Fresh prose <strong id="bionic-strong">stays bold</strong> <em id="bionic-em">and emphasis</em> <mark id="bionic-highlight">and highlighting</mark>.</p><div data-bionic id="bionic-marked">Marked prose.</div><p data-bionic="off" id="bionic-off">Excluded prose.</p><nav id="bionic-nav"><p>Navigation prose.</p></nav><figure id="bionic-figure"><p>Figure prose.</p></figure><p class="font-mono" id="bionic-mono">Mono prose.</p><pre id="bionic-pre">Pre prose.</pre><form id="bionic-form"><p>Form prose.</p><input value="Input prose."/></form>`;
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
		await expect(page.locator("#bionic-em [data-bionic-mark]")).toHaveCount(0);
		await expect(
			page.locator("#bionic-highlight [data-bionic-mark]"),
		).toHaveCount(0);
		await expect(page.locator("#bionic-off [data-bionic-mark]")).toHaveCount(0);
		await expect(page.locator("#bionic-nav [data-bionic-mark]")).toHaveCount(0);
		await expect(page.locator("#bionic-figure [data-bionic-mark]")).toHaveCount(
			0,
		);
		await expect(page.locator("#bionic-mono [data-bionic-mark]")).toHaveCount(
			0,
		);
		await expect(page.locator("#bionic-pre [data-bionic-mark]")).toHaveCount(0);
		await expect(page.locator("#bionic-form [data-bionic-mark]")).toHaveCount(
			0,
		);
		await expect(page.locator("footer [data-bionic-mark]")).toHaveCount(0);
	});

	test("applies the enabled preference only to HTMX-swapped prose", async ({
		page,
	}) => {
		await page.addInitScript(() => localStorage.setItem("wga-bionic", "on"));
		await page.goto("/");

		await page.evaluate(() => {
			const target = document.createElement("section");
			target.innerHTML = `<p id="bionic-loaded">Loaded prose.</p><div data-wga-bionic-control class="hidden items-center gap-2"><button type="button" role="switch" aria-label="Bionic reading" aria-checked="false" data-wga-bionic-toggle class="border border-base-content/20 bg-base-100">BIONIC</button></div>`;
			document.querySelector("main")?.append(target);
			document.dispatchEvent(
				new CustomEvent("htmx:afterSwap", {
					bubbles: true,
					detail: { target },
				}),
			);
		});

		await expect(page.locator("#bionic-loaded")).toHaveText("Loaded prose.");
		await expect(
			page.locator("#bionic-loaded [data-bionic-mark]"),
		).not.toHaveCount(0);
		await expect(page.locator("[data-wga-bionic-control]").last()).toHaveClass(
			/flex/,
		);
		await expect(
			page.locator("[data-wga-bionic-toggle]").last(),
		).toHaveAttribute("aria-checked", "true");
		await expect(page.locator("[data-wga-bionic-toggle]").last()).toHaveClass(
			/bg-primary/,
		);

		const markCount = await page
			.locator("#bionic-loaded [data-bionic-mark]")
			.count();
		await page.evaluate(() => {
			const target = document.querySelector("#bionic-loaded")?.parentElement;
			document.dispatchEvent(
				new CustomEvent("htmx:afterSwap", {
					bubbles: true,
					detail: { target },
				}),
			);
		});
		expect(
			await page.locator("#bionic-loaded [data-bionic-mark]").count(),
		).toBe(markCount);
	});

	test("does not transform HTMX-swapped prose while disabled", async ({
		page,
	}) => {
		await page.goto("/");
		await page.evaluate(() => {
			const target = document.createElement("section");
			target.innerHTML = `<p id="bionic-disabled-swap">Untouched prose.</p>`;
			document.querySelector("main")?.append(target);
			document.dispatchEvent(
				new CustomEvent("htmx:afterSwap", {
					bubbles: true,
					detail: { target },
				}),
			);
		});

		await expect(
			page.locator("#bionic-disabled-swap [data-bionic-mark]"),
		).toHaveCount(0);
	});

	test("continues bionic initialisation and session toggles when Cookie Consent storage fails", async ({
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
				},
			});
		});
		await page.goto("/");
		const toggle = page.getByRole("switch", { name: "Bionic reading" });
		await expect(toggle).toHaveAttribute("aria-checked", "false");
		await expect(page.locator("[data-wga-bionic-control]")).toHaveClass(/flex/);
		await toggle.click();
		await expect(toggle).toHaveAttribute("aria-checked", "true");
		await toggle.click();
		await expect(toggle).toHaveAttribute("aria-checked", "false");
	});
});
