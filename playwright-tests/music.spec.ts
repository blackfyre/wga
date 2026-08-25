import { readFileSync } from "node:fs";
import { type Page, expect, test } from "@playwright/test";
import { transformSync } from "esbuild";

const musicSource = readFileSync("resources/js/music.ts", "utf8");
const musicScript = transformSync(musicSource, {
	format: "esm",
	loader: "ts",
	target: "es2022",
}).code;

const cardMarkup = (piece: string, suffix: string) => `
	<div data-wga-music-card>
		<a href="/player?song=${suffix.padEnd(15, "a")}"
			target="wga-period-music" data-wga-music="${suffix.padEnd(15, "a")}"
			data-wga-music-state="idle"
			aria-label="Open ${piece} in the period music player"><span data-wga-music-control>▶</span></a>
		<span data-wga-music-label>${piece}</span>
		<div role="status" data-wga-music-blocked hidden>
			<p>The period-music player could not be opened.</p>
			<button type="button" data-wga-music-dismiss>Dismiss</button>
		</div>
	</div>`;

const loadFixture = async (page: Page) => {
	await page.route("https://wga.test/record", async (route) => {
		await route.fulfill({
			contentType: "text/html",
			body: `${cardMarkup("Fantasia", "one")}${cardMarkup("Toccata", "two")}`,
		});
	});
	await page.goto("https://wga.test/record");
	await page.addScriptTag({ content: musicScript, type: "module" });
};

test("reuses one fixed named player and synchronises honest card state", async ({
	page,
}) => {
	await loadFixture(page);
	await page.evaluate(() => {
		const calls: unknown[] = [];
		const messages: unknown[] = [];
		const frame = document.createElement("iframe");
		document.body.append(frame);
		const fakePlayer = frame.contentWindow;
		if (!fakePlayer) {
			throw new Error("Missing music player fixture window");
		}
		fakePlayer.addEventListener("message", (event) => {
			messages.push(event.data);
		});
		Object.assign(window, {
			musicOpenCalls: calls,
			musicMessages: messages,
			musicFakePlayer: fakePlayer,
		});
		window.open = ((url: string | URL, name: string, features: string) => {
			calls.push({ url: String(url), name, features });
			return fakePlayer as unknown as Window;
		}) as typeof window.open;
	});

	await page.getByRole("link", { name: /Fantasia/ }).click();
	await page.getByRole("link", { name: /Toccata/ }).click();
	await page.waitForFunction(
		() =>
			(window as unknown as { musicMessages: unknown[] }).musicMessages
				.length === 1,
	);
	const result = await page.evaluate(() => ({
		calls: (window as unknown as { musicOpenCalls: unknown[] }).musicOpenCalls,
		messages: (window as unknown as { musicMessages: unknown[] }).musicMessages,
	}));
	expect(result.calls).toHaveLength(1);
	expect(result.calls[0]).toMatchObject({ name: "wga-period-music" });
	expect(result.messages).toEqual([
		expect.objectContaining({ wgaMusic: "play", piece: "twoaaaaaaaaaaaa" }),
	]);

	await page.evaluate(() => {
		const fakePlayer = (window as unknown as { musicFakePlayer: Window })
			.musicFakePlayer;
		window.dispatchEvent(
			new MessageEvent("message", {
				origin: window.location.origin,
				source: fakePlayer,
				data: { wgaPlayer: "playing", piece: "twoaaaaaaaaaaaa" },
			}),
		);
	});
	await expect(
		page.getByRole("link", { name: /Stop Toccata/ }),
	).toHaveAttribute("data-wga-music-state", "playing");
	await expect(
		page.getByRole("link", { name: /Open Fantasia/ }),
	).toHaveAttribute("data-wga-music-state", "idle");

	await page.getByRole("link", { name: /Stop Toccata/ }).click();
	await page.waitForFunction(() =>
		(window as unknown as { musicMessages: unknown[] }).musicMessages.some(
			(message) => (message as { wgaMusic?: string }).wgaMusic === "pause",
		),
	);
});

test("rejects same-origin player state from the wrong window", async ({
	page,
}) => {
	await loadFixture(page);
	await page.evaluate(() => {
		const frame = document.createElement("iframe");
		document.body.append(frame);
		const fakePlayer = frame.contentWindow;
		if (!fakePlayer) {
			throw new Error("Missing music player fixture window");
		}
		Object.assign(window, { musicFakePlayer: fakePlayer });
		window.open = (() => fakePlayer) as typeof window.open;
	});
	await page.getByRole("link", { name: /Fantasia/ }).click();
	await page.evaluate(() =>
		window.dispatchEvent(
			new MessageEvent("message", {
				origin: window.location.origin,
				source: window,
				data: { wgaPlayer: "playing", piece: "oneaaaaaaaaaaaa" },
			}),
		),
	);
	await expect(
		page.getByRole("link", { name: /Open Fantasia/ }),
	).toHaveAttribute("data-wga-music-state", "idle");
});

test("player rejects same-origin commands from a window other than its opener", async ({
	page,
}) => {
	await page.route("https://wga.test/record", async (route) => {
		await route.fulfill({ contentType: "text/html", body: "<p>Record</p>" });
	});
	await page
		.context()
		.route("https://wga.test/player?song=oneaaaaaaaaaaaa", async (route) => {
			await route.fulfill({
				contentType: "text/html",
				body: '<main data-wga-music-player data-wga-music-song="oneaaaaaaaaaaaa"><audio data-wga-music-audio></audio></main>',
			});
		});
	await page.goto("https://wga.test/record");
	const popupPromise = page.waitForEvent("popup");
	await page.evaluate(() =>
		window.open("/player?song=oneaaaaaaaaaaaa", "wga-period-music"),
	);
	const popup = await popupPromise;
	await popup.addScriptTag({ content: musicScript, type: "module" });
	await popup.evaluate(() => {
		let pauses = 0;
		const audio = document.querySelector("audio");
		if (audio) {
			audio.pause = () => {
				pauses += 1;
			};
		}
		Object.assign(window, { musicPauseCalls: () => pauses });
		window.dispatchEvent(
			new MessageEvent("message", {
				origin: window.location.origin,
				source: window,
				data: { wgaMusic: "pause" },
			}),
		);
	});
	expect(
		await popup.evaluate(() =>
			(
				window as unknown as { musicPauseCalls: () => number }
			).musicPauseCalls(),
		),
	).toBe(0);
	await popup.evaluate(() =>
		window.dispatchEvent(
			new MessageEvent("message", {
				origin: window.location.origin,
				source: window.opener,
				data: { wgaMusic: "pause" },
			}),
		),
	);
	expect(
		await popup.evaluate(() =>
			(
				window as unknown as { musicPauseCalls: () => number }
			).musicPauseCalls(),
		),
	).toBe(1);
});

test("reports a blocked pop-up in a dismissible status notice", async ({
	page,
}) => {
	await loadFixture(page);
	await page.evaluate(() => {
		window.open = (() => null) as typeof window.open;
	});
	await page.getByRole("link", { name: /Fantasia/ }).click();
	await expect(page.getByRole("status")).toBeVisible();
	await expect(page.getByRole("button", { name: "Dismiss" })).toBeFocused();
	await page.getByRole("button", { name: "Dismiss" }).click();
	await expect(page.getByRole("status")).toBeHidden();
});

test("keyboard Enter activates the named-window enhancement", async ({
	page,
}) => {
	await loadFixture(page);
	await page.evaluate(() => {
		let calls = 0;
		Object.assign(window, { musicKeyboardCalls: () => calls });
		window.open = (() => {
			calls += 1;
			return {
				closed: false,
				focus() {},
				postMessage() {},
			} as unknown as Window;
		}) as typeof window.open;
	});
	const card = page.getByRole("link", { name: /Fantasia/ });
	await card.focus();
	await page.keyboard.press("Enter");
	const calls = await page.evaluate(() =>
		(
			window as unknown as { musicKeyboardCalls: () => number }
		).musicKeyboardCalls(),
	);
	expect(calls).toBe(1);
});

test("ordinary named link works with JavaScript disabled and player does not autoplay", async ({
	browser,
}) => {
	const context = await browser.newContext({ javaScriptEnabled: false });
	const page = await context.newPage();
	await page.route("https://wga.test/record", async (route) => {
		await route.fulfill({
			contentType: "text/html",
			body: cardMarkup("Fantasia", "one"),
		});
	});
	await context.route(/https:\/\/wga\.test\/player.*/, async (route) => {
		await route.fulfill({
			contentType: "text/html",
			body: '<audio controls preload="metadata" src="/song.mp3"></audio>',
		});
	});
	await page.goto("https://wga.test/record");
	const popupPromise = page.waitForEvent("popup");
	await page.getByRole("link", { name: /Fantasia/ }).focus();
	await page.keyboard.press("Enter");
	const popup = await popupPromise;
	const audio = popup.locator("audio");
	await expect(audio).toHaveAttribute("controls", "");
	await expect(audio).not.toHaveAttribute("autoplay", "");
	expect(
		await audio.evaluate((element: HTMLAudioElement) => element.paused),
	).toBe(true);
	await context.close();
});
