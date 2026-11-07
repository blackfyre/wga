import { expect, type Page, test } from "@playwright/test";

type MailpitAddress = {
	Address: string;
};

type MailpitMessageSummary = {
	ID: string;
	Subject: string;
};

type MailpitSearchResponse = {
	messages: MailpitMessageSummary[];
};

type MailpitMessage = {
	From: MailpitAddress;
	To: MailpitAddress[];
	Subject: string;
	HTML: string;
};

const syntheticArtworkID = "2225c982be1af02";
const syntheticArtworkPath =
	"/artists/synthetic-artist-01-ad32608c6e36b2e/synthetic-artwork-01-01-2225c982be1af02";

async function fillValidComposer(page: Page, recipient: string) {
	await page.getByLabel("YOUR NAME").fill("Playwright Sender");
	await page.getByLabel("YOUR EMAIL").fill("sender@example.test");
	await page
		.getByRole("textbox", { name: "Recipient email 1" })
		.fill(recipient);
	await page.locator("[data-rte-surface]").fill("A message worth keeping.");
	// The CI handler skips remote verification but still requires a token.
	await page.locator("#postcard_create").evaluate((form) => {
		const token = document.createElement("input");
		token.name = "g-recaptcha-response";
		token.type = "hidden";
		token.value = "playwright-test-token";
		form.append(token);
	});
}

test.describe("without JavaScript", () => {
	test.use({ javaScriptEnabled: false });

	test("postcard composition remains keyboard operable", async ({ page }) => {
		const response = await page.goto(
			`/postcard/send?awid=${syntheticArtworkID}`,
		);
		expect(response?.status()).toBe(200);

		const form = page.locator("#postcard_create");
		await expect(form).toHaveAttribute("action", "/postcard");
		await expect(form).toHaveAttribute("method", "post");
		await expect(page.getByLabel("YOUR NAME")).toBeVisible();
		await expect(page.getByLabel("YOUR EMAIL")).toBeVisible();
		await expect(
			page.getByRole("textbox", { name: "RECIPIENT EMAIL 1" }),
		).toBeVisible();
		await expect(page.locator("textarea[name='message']")).toBeVisible();
		await page.getByRole("button", { name: "+ ADD ANOTHER RECIPIENT" }).focus();
		await page.keyboard.press("Enter");
		await expect(page.locator("[name='recipients[]']")).toHaveCount(2);
		await page.getByRole("button", { name: "Remove recipient 2" }).focus();
		await page.keyboard.press("Enter");
		await expect(page.locator("[name='recipients[]']")).toHaveCount(1);

		let reachedSenderName = false;
		for (let index = 0; index < 30; index += 1) {
			await page.keyboard.press("Tab");
			if (
				await page
					.getByLabel("YOUR NAME")
					.evaluate((field) => field === document.activeElement)
			) {
				reachedSenderName = true;
				break;
			}
		}
		expect(reachedSenderName).toBeTruthy();
		await page.keyboard.type("Keyboard Sender");
		await expect(page.getByLabel("YOUR NAME")).toHaveValue("Keyboard Sender");
	});

	test("public entry leads through artwork selection to the composer", async ({
		page,
	}) => {
		await page.goto("/postcard");
		await page
			.locator("#postcard-landing")
			.getByRole("link", { name: "BROWSE ARTWORKS →" })
			.click();
		await expect(page).toHaveURL(/\/artworks(?:\?|$)/);

		await page.goto(syntheticArtworkPath);
		await page.getByRole("link", { name: "SEND AS POSTCARD →" }).click();
		await expect(page).toHaveURL(
			new RegExp(`/postcard/send\\?awid=${syntheticArtworkID}$`),
		);
		await expect(page.locator("#postcard-compose")).toContainText(
			"Send a postcard",
		);
		await expect(
			page.locator("#postcard_create input[name='image_id']"),
		).toHaveValue(syntheticArtworkID);

		await page.getByRole("link", { name: "CHANGE ARTWORK →" }).click();
		await expect(page).toHaveURL(/\/artworks(?:\?|$)/);
	});

	test("postcard correction retains entered fields", async ({ page }) => {
		await page.goto(`/postcard/send?awid=${syntheticArtworkID}`);
		await page.getByLabel("YOUR NAME").fill("Keyboard Sender");
		await page.getByLabel("YOUR EMAIL").fill("sender@example.test");
		await page
			.getByRole("textbox", { name: "RECIPIENT EMAIL 1" })
			.fill("recipient@example.test");
		await page.locator("textarea[name='message']").fill("   ");
		await page.getByRole("button", { name: "SEND POSTCARD →" }).focus();
		await page.keyboard.press("Enter");
		await expect(page.getByRole("alert")).toContainText(
			"Check the required fields",
		);
		await expect(page.getByLabel("YOUR NAME")).toHaveValue("Keyboard Sender");
	});
});

test("CAPTCHA rejection swaps an actionable composer error", async ({
	page,
}) => {
	await page.goto(`/postcard/send?awid=${syntheticArtworkID}`);
	await page.getByLabel("YOUR NAME").fill("Playwright Sender");
	await page.getByLabel("YOUR EMAIL").fill("sender@example.test");
	await page
		.getByRole("textbox", { name: "RECIPIENT EMAIL 1" })
		.fill("recipient@example.test");
	const message = page.locator("[data-rte-surface]");
	await expect(message).toBeVisible();
	await expect(page.locator("textarea[name='message']")).toBeHidden();
	await message.fill("Hello world");
	await message.evaluate((surface) => {
		const text = surface.querySelector("p")?.firstChild;
		if (!(text instanceof Text)) throw new Error("Message text node not found");
		const range = document.createRange();
		range.setStart(text, 6);
		range.setEnd(text, 11);
		const selection = window.getSelection();
		selection?.removeAllRanges();
		selection?.addRange(range);
	});
	await expect
		.poll(() => message.evaluate(() => window.getSelection()?.toString()))
		.toBe("world");
	await page.getByRole("button", { name: "Bold" }).click();
	await expect(page.locator("textarea[name='message']")).toHaveValue(
		/<p>Hello <b>world<\/b><\/p>/,
	);
	await page.locator("[name='g-recaptcha-response']").evaluate((element) => {
		(element as HTMLInputElement).value = "";
	});
	await page.getByRole("button", { name: "SEND POSTCARD →" }).click();
	await expect(page.locator("#postcard-compose")).toContainText(
		"Complete the CAPTCHA",
	);
	await expect(page.getByLabel("YOUR NAME")).toHaveValue("Playwright Sender");
	await expect(page.locator("[data-rte-surface]")).toContainText("Hello world");
	await expect(page.locator("[data-rte-surface] b")).toHaveText("world");
});

test("unavailable artwork swaps an actionable composer error", async ({
	page,
}) => {
	await page.goto(`/postcard/send?awid=${syntheticArtworkID}`);
	await fillValidComposer(page, "unavailable@example.test");
	await page
		.locator("#postcard_create input[name='image_id']")
		.evaluate((input) => {
			(input as HTMLInputElement).value = "unavailable0000";
		});
	await page.getByRole("button", { name: "SEND POSTCARD →" }).click();

	const alert = page.getByRole("alert");
	await expect(alert).toContainText("no longer available");
	await expect(alert).toBeFocused();
	await expect(page.locator("#postcard_create")).toHaveAttribute(
		"aria-describedby",
		"postcard-error",
	);
	await expect(page.getByLabel("YOUR NAME")).toHaveValue("Playwright Sender");
	await expect(
		page.getByRole("textbox", { name: "Recipient email 1" }),
	).toHaveValue("unavailable@example.test");
	await expect(page.locator("[data-rte-surface]")).toContainText(
		"A message worth keeping.",
	);
	await expect(
		page.getByRole("link", { name: "CHOOSE ANOTHER ARTWORK →" }),
	).toHaveAttribute("href", "/artworks");
});

// Throttling and CAPTCHA-provider outages cannot be produced on demand against
// a shared test server, so these cases fulfil the server's response shape to
// verify the HTMX swap contract; handler tests cover the server responses.
for (const { status, error } of [
	{
		status: 429,
		error: "Too many postcard submissions. Please try again later.",
	},
	{
		status: 503,
		error:
			"CAPTCHA verification is unavailable right now, so your postcard was not queued. Please try again shortly.",
	},
]) {
	test(`HTMX ${status} composer response replaces the form with its error`, async ({
		page,
	}) => {
		await page.goto(`/postcard/send?awid=${syntheticArtworkID}`);
		await fillValidComposer(page, "swap@example.test");
		await page.route("**/postcard", async (route) => {
			if (route.request().method() !== "POST") return route.continue();
			await route.fulfill({
				status,
				contentType: "text/html",
				body: `<section id="postcard-compose"><form id="postcard_create" aria-describedby="postcard-error"><p id="postcard-error" role="alert" tabindex="-1">${error}</p><input name="sender_name" aria-label="YOUR NAME" value="Playwright Sender"/></form></section>`,
			});
		});
		await page.getByRole("button", { name: "SEND POSTCARD →" }).click();
		await expect(page.getByRole("alert")).toHaveText(error);
		await expect(page.getByRole("alert")).toBeFocused();
		await expect(page.getByLabel("YOUR NAME")).toHaveValue("Playwright Sender");
	});
}

test("HTMX error page without a composer leaves the form in place", async ({
	page,
}) => {
	await page.goto(`/postcard/send?awid=${syntheticArtworkID}`);
	await fillValidComposer(page, "fault@example.test");
	await page.route("**/postcard", async (route) => {
		if (route.request().method() !== "POST") return route.continue();
		await route.fulfill({
			status: 500,
			contentType: "text/html",
			body: "<main><h1>Something went wrong</h1></main>",
		});
	});
	await page.getByRole("button", { name: "SEND POSTCARD →" }).click();
	await expect(page.getByLabel("YOUR NAME")).toHaveValue("Playwright Sender");
	await expect(page.locator("#postcard_create")).toBeVisible();
	await expect(
		page.getByRole("button", { name: "SEND POSTCARD →" }),
	).toBeEnabled();
});

test("toolbar formats an empty caret and starts a list", async ({ page }) => {
	await page.goto(`/postcard/send?awid=${syntheticArtworkID}`);
	const message = page.locator("[data-rte-surface]");
	const source = page.locator("textarea[name='message']");

	await message.click();
	await page.getByRole("button", { name: "Bold" }).click();
	await page.keyboard.type("Fresh text");
	await expect(source).toHaveValue(/<p><b>Fresh text<\/b>(?:<br>)?<\/p>/);

	await page.reload();
	await message.click();
	await page.getByRole("button", { name: "Bulleted list" }).click();
	await page.keyboard.type("First item");
	await expect(source).toHaveValue(/<ul><li>First item<\/li><\/ul>/);
});

test("editor preserves soft breaks and announces an empty message", async ({
	page,
}) => {
	await page.goto(`/postcard/send?awid=${syntheticArtworkID}`);
	const message = page.locator("[data-rte-surface]");
	const source = page.locator("textarea[name='message']");

	await page.getByRole("button", { name: "ADD ANOTHER RECIPIENT" }).click();
	await expect(page.getByLabel("Recipient email 2")).toBeVisible();
	await page.reload();

	await message.fill("First line");
	await message.press("Shift+Enter");
	await message.pressSequentially("Second line");
	await expect(source).toHaveValue(
		/<p>First line<\/p><p>Second line(?:<br>)?<\/p>/,
	);

	await page.reload();
	await page.getByLabel("YOUR NAME").fill("Playwright Sender");
	await page.getByLabel("YOUR EMAIL").fill("sender@example.com");
	await page.getByLabel("Recipient email 1").fill("recipient@example.com");
	await page.getByRole("button", { name: "SEND POSTCARD" }).click();
	await expect(message).toHaveAttribute("aria-invalid", "true");
	await expect(page.locator("[data-rte-count]")).toHaveText("Enter a message");
});

test.describe("without JavaScript message limit", () => {
	test.use({ javaScriptEnabled: false });

	test("leaves visible Unicode length enforcement to the server", async ({
		page,
	}) => {
		await page.goto(`/postcard/send?awid=${syntheticArtworkID}`);
		await expect(page.locator("textarea[name='message']")).not.toHaveAttribute(
			"maxlength",
		);
	});
});

for (const viewport of [
	{ width: 390, height: 844 },
	{ width: 834, height: 900 },
	{ width: 1440, height: 1000 },
]) {
	test(`postcard composer remains usable at ${viewport.width}px`, async ({
		page,
	}) => {
		await page.setViewportSize(viewport);
		await page.goto(`/postcard/send?awid=${syntheticArtworkID}`);
		await expect(page.locator("#postcard-compose")).toBeVisible();
		await expect(page.locator("#postcard_create")).toBeVisible();
		await expect(page.locator(".wga-dialog-panel")).toHaveCount(0);
		expect(
			await page.evaluate(
				() =>
					document.documentElement.scrollWidth <=
					document.documentElement.clientWidth,
			),
		).toBeTruthy();

		const sender = page.getByLabel("YOUR NAME");
		let reachedSender = false;
		for (let index = 0; index < 40 && !reachedSender; index += 1) {
			await page.keyboard.press("Tab");
			reachedSender = await sender.evaluate(
				(field) => field === document.activeElement,
			);
		}
		expect(reachedSender).toBeTruthy();
		await page.keyboard.type("Keyboard Sender");
		await expect(sender).toHaveValue("Keyboard Sender");

		const send = page.getByRole("button", { name: "SEND POSTCARD →" });
		await send.focus();
		await expect(send).toBeFocused();
		await expect(send).toBeInViewport();
		for (const field of [
			sender,
			page.getByLabel("YOUR EMAIL"),
			page.getByRole("textbox", { name: "Recipient email 1" }),
			send,
		]) {
			const box = await field.boundingBox();
			expect(box).not.toBeNull();
			expect((box?.x ?? -1) >= 0).toBeTruthy();
			expect((box?.x ?? 0) + (box?.width ?? 0) <= viewport.width).toBeTruthy();
		}
	});
}

test("send postcard", async ({ page, request }) => {
	test.setTimeout(150000);

	const mailpitUrl = process.env.MAILPIT_URL;
	if (!mailpitUrl) {
		throw new Error("MAILPIT_URL environment variable is not set.");
	}

	const recipients = [
		"playwright.first@local.host",
		"playwright.second@local.host",
	];
	const subject = "You got a postcard from Playwright Tester!";
	const searchUrls = recipients.map(
		(recipient) =>
			`${mailpitUrl}/api/v1/search?${new URLSearchParams({
				query: `to:${recipient}`,
			})}`,
	);
	const existingMessageIDs: string[] = [];
	for (const searchUrl of searchUrls) {
		const existingMessagesResponse = await request.get(searchUrl);
		expect(existingMessagesResponse.ok()).toBeTruthy();
		const existingMessages =
			(await existingMessagesResponse.json()) as MailpitSearchResponse;
		existingMessageIDs.push(
			...existingMessages.messages
				.filter((message) => message.Subject === subject)
				.map((message) => message.ID),
		);
	}

	if (existingMessageIDs.length > 0) {
		const deleteResponse = await request.delete(
			`${mailpitUrl}/api/v1/messages`,
			{ data: { ids: existingMessageIDs } },
		);
		expect(deleteResponse.ok()).toBeTruthy();
	}

	await page.goto(
		"/artists/synthetic-artist-01-ad32608c6e36b2e/synthetic-artwork-01-01-2225c982be1af02",
	);
	await page.getByRole("link", { name: "SEND AS POSTCARD →" }).click();

	await expect(page).toHaveURL(/\/postcard\/send\?awid=/);
	await expect(page.locator("#postcard-compose")).toHaveText(/Send a postcard/);
	await expect(page.locator("[name='recipients[]']")).toHaveCount(1);
	await page.getByRole("button", { name: "+ ADD ANOTHER RECIPIENT" }).click();
	await expect(page.locator("[name='recipients[]']")).toHaveCount(2);

	await page.locator("[name='sender_name']").fill("Playwright Tester");
	await page
		.locator("[name='sender_email']")
		.fill("playwright.tester@local.host"); // this is the postcard sender's email
	await page.locator("[name='recipients[]']").nth(0).fill(recipients[0]);
	await page.locator("[name='recipients[]']").nth(1).fill(recipients[1]);
	const message = page.locator("[data-rte-surface]");
	await message.fill("I am testing your site.");
	await message.selectText();
	await page.getByRole("button", { name: "Italic" }).click();
	// The CI handler skips remote verification but still requires a token.
	await page.locator("#postcard_create").evaluate((form) => {
		const token = document.createElement("input");
		token.name = "g-recaptcha-response";
		token.type = "hidden";
		token.value = "playwright-test-token";
		form.append(token);
	});

	const submissions: string[] = [];
	page.on("request", (sent) => {
		if (
			sent.method() === "POST" &&
			new URL(sent.url()).pathname === "/postcard"
		) {
			submissions.push(sent.postData() ?? "");
		}
	});
	// A double activation is one send intent: the in-flight form drops the
	// repeat and disables its submit controls.
	await page.getByRole("button", { name: "SEND POSTCARD →" }).dblclick();

	await expect(page.locator("#postcard-compose")).toContainText(
		"Postcard queued",
	);
	expect(submissions).toHaveLength(1);

	// Retrying the same send intent after an interrupted response returns the
	// existing outcome instead of queueing duplicate delivery work.
	const retry = await request.post("/postcard", {
		headers: { "Content-Type": "application/x-www-form-urlencoded" },
		data: submissions[0],
	});
	expect(retry.status()).toBe(202);
	expect(await retry.text()).toContain("Postcard queued");
	await expect(page.locator("#postcard-compose")).toContainText(
		"p••••@local.host, p••••@local.host",
	);
	await expect(page.locator("#postcard-confirmation-title")).toBeFocused();
	await expect(page.locator("#postcard-compose")).toContainText(
		"has not been sent yet",
	);
	await expect(
		page.getByRole("link", { name: "SEND ANOTHER POSTCARD →" }),
	).toHaveAttribute("href", `/postcard/send?awid=${syntheticArtworkID}`);

	let messageIDs: string[] = [];
	try {
		await expect
			.poll(
				async () => {
					const found: string[] = [];
					for (const searchUrl of searchUrls) {
						const response = await request.get(searchUrl);
						if (!response.ok()) return 0;
						const messages = (await response.json()) as MailpitSearchResponse;
						const matching = messages.messages.filter(
							(message) => message.Subject === subject,
						);
						if (matching.length === 0) return -1;
						found.push(...matching.map((message) => message.ID));
					}
					messageIDs = found;
					return found.length;
				},
				{ intervals: [1000, 2000, 5000], timeout: 120000 },
			)
			// Exactly one delivery per recipient despite the repeated intent.
			.toBe(recipients.length);

		const postcardLinks: string[] = [];
		for (const [index, messageID] of messageIDs.entries()) {
			const messageResponse = await request.get(
				`${mailpitUrl}/api/v1/message/${messageID}`,
			);
			expect(messageResponse.ok()).toBeTruthy();
			const message = (await messageResponse.json()) as MailpitMessage;
			expect(message.From.Address).toBe("do-not-reply@wga.hu");
			expect(message.To.map((address) => address.Address)).toContain(
				recipients[index],
			);
			expect(message.Subject).toBe(subject);
			expect(message.HTML).toContain("FROM PLAYWRIGHT TESTER");
			expect(message.HTML).toContain("A postcard is waiting for you");
			expect(message.HTML).toContain("<i>I am testing your site.</i>");
			const postcardLink = message.HTML.match(
				/<a\b[^>]*\bhref=["']([^"']+)["'][^>]*>\s*OPEN YOUR POSTCARD(?:&nbsp;|\s)*&rarr;\s*<\/a>/i,
			)?.[1];
			if (!postcardLink) throw new Error("Postcard link not found");
			postcardLinks.push(postcardLink);
		}
		expect(new Set(postcardLinks).size).toBe(1);
		const postcardLink = postcardLinks[0];
		expect(postcardLink).toContain("/postcard?token=");
		expect(postcardLink).not.toContain("?p=");

		const postcardURL = new URL(postcardLink);
		await page.goto(`${postcardURL.pathname}${postcardURL.search}`);
		await expect(page.locator("#postcard-view")).toContainText(
			"I am testing your site",
		);
		await expect(page.locator("#postcard-view i")).toHaveText(
			"I am testing your site.",
		);
	} finally {
		if (messageIDs.length > 0) {
			const deleteResponse = await request.delete(
				`${mailpitUrl}/api/v1/messages`,
				{ data: { ids: messageIDs } },
			);
			expect(deleteResponse.ok()).toBeTruthy();
		}
	}
});

test("postcard recipient route denies arbitrary identifiers", async ({
	page,
}) => {
	const response = await page.goto("/postcard?token=0123456789abcde");
	expect(response?.status()).toBe(404);
	await expect(page).toHaveURL(/token=0123456789abcde/);
});
