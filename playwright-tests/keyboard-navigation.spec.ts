import { type Page, expect, test } from "@playwright/test";

const waitForKeyboard = (page: Page) =>
	page.waitForFunction(
		() => document.documentElement.dataset.keyboardNavigationReady === "true",
	);

test("opens keyboard help and command palette", async ({ page }) => {
	await page.goto("/");
	await waitForKeyboard(page);

	const helpButton = page.getByRole("button", { name: "Keyboard shortcuts" });
	await helpButton.click();
	await expect(page.locator("#keyboard-help")).toBeVisible();
	await expect(
		page.getByRole("button", { name: "Close shortcuts" }),
	).toBeFocused();

	await page.keyboard.press("Escape");
	await expect(helpButton).toBeFocused();
	await page.keyboard.press("?");
	await expect(
		page.getByRole("dialog", { name: "Moving without the mouse" }),
	).toBeVisible();
	await page.keyboard.press("?");
	await expect(page.locator("#keyboard-help")).not.toBeVisible();
	await page.keyboard.press("Control+k");
	await expect(page.getByRole("dialog", { name: "Go to" })).toBeVisible();
	await expect(
		page.getByRole("searchbox", { name: "Search sections, artists and works" }),
	).toBeFocused();
});

test("centres keyboard dialogs on desktop", async ({ page }) => {
	await page.setViewportSize({ width: 1440, height: 900 });
	await page.goto("/");
	await waitForKeyboard(page);

	for (const { open, panel } of [
		{
			open: () =>
				page.getByRole("button", { name: "Keyboard shortcuts" }).click(),
			panel: ".wga-kbd-help",
		},
		{
			open: () => page.keyboard.press("Control+k"),
			panel: ".wga-kbd-palette",
		},
	]) {
		await open();
		const box = await page.locator(panel).boundingBox();
		expect(
			Math.abs((box?.x || 0) + (box?.width || 0) / 2 - 720),
		).toBeLessThanOrEqual(1);
		expect(
			Math.abs((box?.y || 0) + (box?.height || 0) / 2 - 450),
		).toBeLessThanOrEqual(1);
		await page.keyboard.press("Escape");
	}
});

test("palette filters sections and bounds record suggestions", async ({
	page,
}) => {
	await page.goto("/");
	await waitForKeyboard(page);
	await page.keyboard.press("Control+k");

	const input = page.getByRole("searchbox", {
		name: "Search sections, artists and works",
	});
	await input.fill("artist index");
	await expect(
		page.getByRole("link", { name: /artist index/i }).first(),
	).toBeVisible();

	const response = page.waitForResponse((candidate) => {
		const url = new URL(candidate.url());
		return (
			url.pathname === "/keyboard/suggestions" &&
			url.searchParams.get("q") === "art"
		);
	});
	await input.fill("art");
	const suggestionURL = new URL((await response).url());
	expect(suggestionURL.searchParams.get("limit")).toBe("7");
	await page.keyboard.press("Enter");
	await expect(page).toHaveURL(/\/artists/);
});

test("section shortcuts navigate by letter and catalogue number", async ({
	page,
}) => {
	await page.goto("/");
	await waitForKeyboard(page);
	await expect(page.locator("#kbd-screens")).toHaveAttribute(
		"data-json",
		/"key":"A","num":"01","label":"ARTIST INDEX","href":"\/artists"/,
	);
	await page.keyboard.press("a");
	await expect(page).toHaveURL(/\/artists/);

	await page.goto("/");
	await waitForKeyboard(page);
	await page.keyboard.press("0");
	await page.keyboard.press("1");
	await expect(page).toHaveURL(/\/artists/);
});

test("deferred destinations remain available in the screen registry and palette", async ({
	page,
}) => {
	await page.goto("/");
	await waitForKeyboard(page);
	await expect(page.locator("#kbd-screens")).toHaveAttribute(
		"data-json",
		/"key":"M","num":"19","label":"TIMELINE EXPLORER","href":"\/timeline"/,
	);
	await page.keyboard.press("Control+k");
	await expect(
		page.locator('[data-kbd-item="section"][data-kbd-href="/timeline"]'),
	).toHaveAttribute("href", "/timeline");
});

test("palette sections and help routes match the server registry", async ({
	page,
}) => {
	await page.goto("/");
	await waitForKeyboard(page);

	const screens = await page.locator("#kbd-screens").getAttribute("data-json");
	if (!screens) {
		throw new Error("Expected the server keyboard registry payload");
	}
	const registry = JSON.parse(screens) as Array<{
		key: string;
		num: string;
		label: string;
		href: string;
	}>;

	await page.keyboard.press("Control+k");
	const sectionRows = page.locator('[data-kbd-item="section"]');
	await expect(sectionRows).toHaveCount(registry.length);
	for (const screen of registry) {
		const row = page.locator(
			`[data-kbd-item="section"][data-kbd-href="${screen.href}"]`,
		);
		await expect(row).toHaveCount(1);
		await expect(row).toHaveAttribute("data-kbd-key", screen.key);
		await expect(row).toHaveAttribute("data-kbd-num", screen.num);
		await expect(row).toHaveAttribute("data-kbd-label", screen.label);
	}

	await page.keyboard.press("Escape");
	await page.keyboard.press("?");
	const helpRows = page
		.locator("#keyboard-help p")
		.filter({ has: page.locator("kbd") });
	for (const screen of registry) {
		await expect(
			helpRows.filter({
				hasText: `${screen.key} ${screen.num} · ${screen.label}`,
			}),
		).toHaveCount(1);
	}
});

test("search and palette shortcuts respect editing contexts", async ({
	page,
}) => {
	await page.goto("/");
	await waitForKeyboard(page);
	const search = page.locator("[data-kbd-search]").filter({ visible: true });

	await search.fill("Vermeer");
	await page.getByRole("heading", { level: 1 }).click();
	await page.keyboard.press("/");
	await expect(search).toBeFocused();
	await expect(search).toHaveJSProperty("selectionStart", 0);
	await page.keyboard.press("Escape");
	await expect(search).not.toBeFocused();

	await search.focus();
	await search.press("?");
	await expect(page.locator("#keyboard-help")).not.toBeVisible();
	await search.press("a");
	await expect(page).toHaveURL(/\/$/);
	await search.press("Control+k");
	await expect(page.getByRole("dialog", { name: "Go to" })).toBeVisible();
});

test("global shortcuts stay isolated from every editable control", async ({
	page,
}) => {
	await page.goto("/");
	await waitForKeyboard(page);
	await page.evaluate(() => {
		const fixture = document.createElement("div");
		fixture.innerHTML =
			'<input id="kbd-test-input"><textarea id="kbd-test-textarea"></textarea><select id="kbd-test-select"><option>One</option><option>Two</option></select><div id="kbd-test-editable" contenteditable="true"></div>';
		document.body.append(fixture);
	});

	for (const selector of [
		"#kbd-test-input",
		"#kbd-test-textarea",
		"#kbd-test-select",
		"#kbd-test-editable",
	]) {
		const control = page.locator(selector);
		await control.focus();
		for (const key of ["a", "0", "ArrowDown", "h", "j", "k", "l", "/", "?"]) {
			await page.keyboard.press(key);
		}
		await expect(page.locator("#keyboard-help")).not.toBeVisible();
		await expect(page.locator("#keyboard-palette")).not.toBeVisible();
		await expect(page.locator("[data-kbd-caret]")).toHaveCount(0);
		await expect(page).toHaveURL(/\/$/);

		await page.keyboard.press("Escape");
		await expect(control).not.toBeFocused();
		await control.focus();
		await page.keyboard.press("Control+k");
		await expect(page.getByRole("dialog", { name: "Go to" })).toBeVisible();
		await page.keyboard.press("Escape");
		await expect(control).toBeFocused();
	}
});

test("mobile search shortcut opens the navigation disclosure", async ({
	page,
}) => {
	await page.setViewportSize({ width: 600, height: 900 });
	await page.goto("/");
	await waitForKeyboard(page);
	await page.keyboard.press("/");
	await expect(page.locator("[data-kbd-mobile-navigation]")).toHaveAttribute(
		"open",
		"",
	);
	await expect(
		page.locator("[data-kbd-mobile-navigation] [data-kbd-search]"),
	).toBeFocused();
	await page.getByRole("heading", { level: 1 }).click();
	await expect(
		page.locator("[data-kbd-mobile-navigation]"),
	).not.toHaveAttribute("open");

	await page.keyboard.press("/");
	await page.keyboard.press("Escape");
	await expect(
		page.locator("[data-kbd-mobile-navigation]"),
	).not.toHaveAttribute("open");
});

test("search shortcut respects the 45rem navigation breakpoint", async ({
	page,
}) => {
	await page.setViewportSize({ width: 720, height: 900 });
	await page.goto("/");
	await waitForKeyboard(page);
	await page.keyboard.press("/");
	await expect(
		page.locator("[data-kbd-mobile-navigation]"),
	).not.toHaveAttribute("open");
	await expect(
		page.locator("header [data-kbd-search]").filter({ visible: true }),
	).toBeFocused();

	await page.setViewportSize({ width: 719, height: 900 });
	await page.keyboard.press("/");
	await expect(page.locator("[data-kbd-mobile-navigation]")).toHaveAttribute(
		"open",
		"",
	);
	await expect(
		page.locator("[data-kbd-mobile-navigation] [data-kbd-search]"),
	).toBeFocused();
});

test("palette selection wraps between its first and last result", async ({
	page,
}) => {
	await page.goto("/");
	await waitForKeyboard(page);
	await page.keyboard.press("Control+k");
	await page.keyboard.press("ArrowUp");
	await page.keyboard.press("Enter");
	await expect(page).toHaveURL(/\/contributors$/);
});

test("palette requests only remaining capacity and contains at most nine rows", async ({
	page,
}) => {
	await page.goto("/");
	await waitForKeyboard(page);
	await page.keyboard.press("Control+k");

	const input = page.getByRole("searchbox", {
		name: "Search sections, artists and works",
	});
	const items = page.locator("#kbd-palette-results [data-kbd-item]");
	const visibleItemCount = () =>
		items.evaluateAll(
			(elements) =>
				elements.filter(
					(element) => !element.hidden && element.getClientRects().length > 0,
				).length,
		);
	await expect.poll(visibleItemCount).toBe(9);

	const requests: string[] = [];
	const requestListener = (request: import("@playwright/test").Request) => {
		if (new URL(request.url()).pathname === "/keyboard/suggestions") {
			requests.push(request.url());
		}
	};
	page.on("request", requestListener);
	await input.fill("a");
	await expect.poll(() => requests.length, { timeout: 500 }).toBe(0);
	const response = page.waitForResponse((candidate) => {
		const url = new URL(candidate.url());
		return (
			url.pathname === "/keyboard/suggestions" &&
			url.searchParams.get("q") === "art"
		);
	});
	await input.fill("art");
	const suggestionURL = new URL((await response).url());
	expect(suggestionURL.searchParams.get("limit")).toBe("7");
	await expect.poll(visibleItemCount).toBeLessThanOrEqual(9);
	page.off("request", requestListener);
});

test("suggestion failure does not disable section navigation", async ({
	page,
}) => {
	await page.route("**/keyboard/suggestions**", (route) =>
		route.fulfill({ status: 503 }),
	);
	await page.goto("/");
	await waitForKeyboard(page);
	await page.keyboard.press("Control+k");
	const input = page.getByRole("searchbox", {
		name: "Search sections, artists and works",
	});
	const failedSuggestion = page.waitForResponse(
		(candidate) =>
			new URL(candidate.url()).pathname === "/keyboard/suggestions" &&
			candidate.status() === 503,
	);
	await input.fill("art");
	await failedSuggestion;
	await expect(
		page
			.locator('[data-kbd-item="section"]')
			.filter({ hasText: "ARTIST INDEX" }),
	).toBeVisible();
	await page
		.getByRole("link", { name: /artist index/i })
		.first()
		.click();
	await expect(page).toHaveURL(/\/artists$/);
});

test("palette close paths restore focus and reopen clean", async ({ page }) => {
	await page.goto("/");
	await waitForKeyboard(page);
	const opener = page.getByRole("button", { name: "Open Go to" });
	await opener.click();
	const query = page.getByRole("searchbox", {
		name: "Search sections, artists and works",
	});
	await query.fill("artist");
	await page.getByRole("button", { name: "Close Go to" }).click();
	await expect(opener).toBeFocused();
	await opener.click();
	await expect(query).toHaveValue("");
	await page.keyboard.press("Escape");
	await expect(opener).toBeFocused();
});

test("mobile disclosure closes on Escape and outside click", async ({
	page,
}) => {
	await page.setViewportSize({ width: 600, height: 900 });
	await page.goto("/");
	await waitForKeyboard(page);
	await page.keyboard.press("/");
	await page.keyboard.press("Escape");
	await expect(
		page.locator("[data-kbd-mobile-navigation]"),
	).not.toHaveAttribute("open");
	await page.keyboard.press("/");
	await page.getByRole("heading", { level: 1 }).click();
	await expect(
		page.locator("[data-kbd-mobile-navigation]"),
	).not.toHaveAttribute("open");
});

test("platform keycaps follow the visitor platform after HTMX settlement", async ({
	page,
}) => {
	await page.goto("/");
	await waitForKeyboard(page);
	await expect(page.locator("[data-kbd-modifier]").first()).toHaveText(
		"CTRL K",
	);
	await page.evaluate(() => {
		Object.defineProperty(navigator, "platform", {
			configurable: true,
			value: "MacIntel",
		});
		document.dispatchEvent(new Event("htmx:afterSettle"));
	});
	await expect(page.locator("[data-kbd-modifier]").first()).toHaveText("⌘K");
});

test("artwork grid traversal uses declared responsive rows and clamps", async ({
	page,
}) => {
	await page.setViewportSize({ width: 700, height: 900 });
	await page.goto("/artworks");
	await waitForKeyboard(page);

	await page.keyboard.press("ArrowDown");
	await expect(page.locator("[data-kbd-caret]")).toHaveAttribute(
		"data-kbd-idx",
		"0",
	);
	await page.keyboard.press("j");
	await expect(page.locator("[data-kbd-caret]")).toHaveAttribute(
		"data-kbd-idx",
		"2",
	);
	await page.keyboard.press("K");
	await expect(page.locator("[data-kbd-caret]")).toHaveAttribute(
		"data-kbd-idx",
		"0",
	);
	for (let index = 0; index < 20; index += 1) {
		await page.keyboard.press("ArrowUp");
	}
	await expect(page.locator("[data-kbd-caret]")).toHaveAttribute(
		"data-kbd-idx",
		"0",
	);
});

test("artwork traversal moves by record and Enter opens the marked link", async ({
	page,
}) => {
	await page.goto("/artworks");
	await waitForKeyboard(page);

	await page.keyboard.press("ArrowRight");
	await page.keyboard.press("l");
	await expect(page.locator("[data-kbd-caret]")).toHaveAttribute(
		"data-kbd-idx",
		"1",
	);
	await page.keyboard.press("H");
	await page.keyboard.press("ArrowLeft");
	await expect(page.locator("[data-kbd-caret]")).toHaveAttribute(
		"data-kbd-idx",
		"0",
	);
	const href = await page
		.locator("[data-kbd-list] [data-kbd-idx='0']")
		.getAttribute("data-kbd-href");
	if (!href) {
		throw new Error("Expected the marked artwork to declare a destination");
	}
	await page.keyboard.press("Enter");
	await expect(page).toHaveURL(href);
});

test("artwork list traversal uses one-row movement and resets after HTMX replacement", async ({
	page,
}) => {
	await page.goto("/artworks");
	await waitForKeyboard(page);
	await page.getByRole("link", { name: "LIST" }).click();
	await expect(page.locator("[data-kbd-list]")).toHaveAttribute(
		"data-view",
		"list",
	);

	await page.keyboard.press("ArrowDown");
	await page.keyboard.press("ArrowDown");
	await expect(page.locator("[data-kbd-caret]")).toHaveAttribute(
		"data-kbd-idx",
		"1",
	);
	await page.keyboard.press("Escape");
	await expect(page.locator("[data-kbd-caret]")).toHaveCount(0);

	await page.keyboard.press("ArrowDown");
	const replacement = page.waitForResponse((candidate) => {
		const url = new URL(candidate.url());
		return (
			url.pathname === "/artworks/results" &&
			url.searchParams.get("q") === "Synthetic Artwork 01-01"
		);
	});
	await page
		.locator("[data-keyboard-page-search]")
		.fill("Synthetic Artwork 01-01");
	await replacement;
	await expect(page.locator("[data-kbd-list] [data-kbd-idx]")).toHaveCount(1);
	await expect(page.locator("[data-kbd-caret]")).toHaveCount(0);
	await page.getByRole("heading", { level: 1 }).click();
	await page.keyboard.press("ArrowDown");
	await expect(page.locator("[data-kbd-caret]")).toHaveAttribute(
		"data-kbd-idx",
		"0",
	);
});

test("guestbook entries can be marked without creating navigation", async ({
	page,
}) => {
	await page.goto("/guestbook");
	await waitForKeyboard(page);
	await page.evaluate(() => {
		const list = document.createElement("div");
		list.dataset.kbdList = "";
		list.dataset.kbdCols = "1";
		const entry = document.createElement("article");
		entry.dataset.kbdIdx = "0";
		entry.textContent = "Guestbook entry";
		list.append(entry);
		document.querySelector("#guestbook")?.append(list);
	});
	await expect(
		page.locator("[data-kbd-list] [data-kbd-idx]").first(),
	).toBeVisible();
	const location = page.url();

	await page.keyboard.press("ArrowDown");
	await expect(page.locator("[data-kbd-caret]")).toHaveAttribute(
		"data-kbd-idx",
		"0",
	);
	await page.keyboard.press("Enter");
	await expect(page).toHaveURL(location);
});

test("non-opted-in pages preserve browser keys and reduced motion is immediate", async ({
	page,
}) => {
	await page.goto("/");
	await waitForKeyboard(page);
	await page.evaluate(() => {
		window.addEventListener("keydown", (event) => {
			if (event.key === "ArrowDown" || event.key === "Enter") {
				document.documentElement.dataset.kbdDefaultPrevented = String(
					event.defaultPrevented,
				);
			}
		});
	});
	await page.keyboard.press("ArrowDown");
	await expect(page.locator("html")).toHaveAttribute(
		"data-kbd-default-prevented",
		"false",
	);
	await page.keyboard.press("Enter");
	await expect(page.locator("html")).toHaveAttribute(
		"data-kbd-default-prevented",
		"false",
	);

	await page.emulateMedia({ reducedMotion: "reduce" });
	await page.goto("/artworks");
	await waitForKeyboard(page);
	await page.evaluate(() => {
		HTMLElement.prototype.scrollIntoView = (options) => {
			document.documentElement.dataset.kbdScroll = JSON.stringify(options);
		};
	});
	await page.keyboard.press("ArrowDown");
	await expect(page.locator("html")).toHaveAttribute(
		"data-kbd-scroll",
		'{"block":"nearest","behavior":"auto"}',
	);
});
