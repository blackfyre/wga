import { expect, test } from "@playwright/test";

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
		await expect(page.getByLabel(/MESSAGE/)).toBeVisible();

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

	test("postcard correction retains entered fields", async ({ page }) => {
		await page.goto(`/postcard/send?awid=${syntheticArtworkID}`);
		await page.getByLabel("YOUR NAME").fill("Keyboard Sender");
		await page.getByLabel("YOUR EMAIL").fill("sender@example.test");
		await page
			.getByRole("textbox", { name: "RECIPIENT EMAIL 1" })
			.fill("recipient@example.test");
		await page.getByLabel(/MESSAGE/).fill("A postcard message");
		await page.getByRole("button", { name: "SEND POSTCARD →" }).click();
		await expect(page.getByRole("alert")).toContainText("Complete the CAPTCHA");
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
	await page.getByLabel(/MESSAGE/).fill("A postcard message");
	await page.getByRole("button", { name: "SEND POSTCARD →" }).click();
	await expect(page.locator("#postcard-compose")).toContainText(
		"Complete the CAPTCHA",
	);
	await expect(page.getByLabel("YOUR NAME")).toHaveValue("Playwright Sender");
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
		await expect(page.locator(".modal-box")).toHaveCount(0);
	});
}

test("send postcard", async ({ page, request }) => {
	test.setTimeout(150000);

	const mailpitUrl = process.env.MAILPIT_URL;
	if (!mailpitUrl) {
		throw new Error("MAILPIT_URL environment variable is not set.");
	}

	const recipient = "playwright.tester@local.host";
	const subject = "You got a postcard from Playwright Tester!";
	const searchUrl = `${mailpitUrl}/api/v1/search?${new URLSearchParams({
		query: `to:${recipient}`,
	})}`;
	const existingMessagesResponse = await request.get(searchUrl);
	expect(existingMessagesResponse.ok()).toBeTruthy();
	const existingMessages =
		(await existingMessagesResponse.json()) as MailpitSearchResponse;
	const existingMessageIDs = existingMessages.messages
		.filter((message) => message.Subject === subject)
		.map((message) => message.ID);

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
	const recipientField = page.locator("[name='recipients[]']").first();
	await expect(page.locator("[name='recipients[]']")).toHaveCount(5);

	await page.locator("[name='sender_name']").fill("Playwright Tester");
	await page
		.locator("[name='sender_email']")
		.fill("playwright.tester@local.host"); // this is the postcard sender's email
	await recipientField.fill(recipient);
	await page
		.locator("textarea[name='message']")
		.fill("I am testing your site.");
	// The CI handler skips remote verification but still requires a token.
	await page.locator("#postcard_create").evaluate((form) => {
		const token = document.createElement("input");
		token.name = "g-recaptcha-response";
		token.type = "hidden";
		token.value = "playwright-test-token";
		form.append(token);
	});

	await page.getByRole("button", { name: "SEND POSTCARD →" }).click();

	await expect(page.locator("#postcard-compose")).toContainText(
		"Postcard queued",
	);
	await expect(page.locator("#postcard-compose")).toContainText(
		"p••••@local.host",
	);
	await expect(page.locator("#postcard-confirmation-title")).toBeFocused();

	let messageID = "";
	try {
		await expect
			.poll(
				async () => {
					const response = await request.get(searchUrl);
					if (!response.ok()) {
						return "";
					}

					const messages = (await response.json()) as MailpitSearchResponse;
					messageID =
						messages.messages.find((message) => message.Subject === subject)
							?.ID ?? "";
					return messageID;
				},
				{ intervals: [1000, 2000, 5000], timeout: 120000 },
			)
			.toBeTruthy();

		const messageResponse = await request.get(
			`${mailpitUrl}/api/v1/message/${messageID}`,
		);
		expect(messageResponse.ok()).toBeTruthy();
		const message = (await messageResponse.json()) as MailpitMessage;
		expect(message.From.Address).toBe("do-not-reply@wga.hu");
		expect(message.To.map((address) => address.Address)).toContain(recipient);
		expect(message.Subject).toBe(subject);
		expect(message.HTML).toContain(
			"Playwright Tester has left postcard for you to pick up",
		);

		const postcardLink = message.HTML.match(
			/<a\b[^>]*\bhref=["']([^"']+)["'][^>]*>\s*Pickup my Postcard!\s*<\/a>/i,
		)?.[1];
		if (!postcardLink) {
			throw new Error("Postcard link not found");
		}
		expect(postcardLink).toContain("/postcard?token=");
		expect(postcardLink).not.toContain("?p=");

		const postcardURL = new URL(postcardLink);
		await page.goto(`${postcardURL.pathname}${postcardURL.search}`);
		await expect(page.locator("#postcard-view")).toContainText(
			"I am testing your site",
		);
	} finally {
		if (messageID) {
			const deleteResponse = await request.delete(
				`${mailpitUrl}/api/v1/messages`,
				{ data: { ids: [messageID] } },
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
