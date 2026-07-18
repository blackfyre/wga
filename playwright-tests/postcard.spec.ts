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

	await page.goto("/artists/koedijck-isaack-3ed9e200b9e8252");
	await page.getByRole("link", { name: "Send postcard" }).click();

	await expect(page.locator("#d")).toBeVisible();

	await expect(page.locator("#d")).toHaveText(/Write a postcard/);

	await page.locator("[name='sender_name']").fill("Playwright Tester");
	await page
		.locator("[name='sender_email']")
		.fill("playwright.tester@local.host"); // this is the postcard sender's email
	await page.locator("[name='recipients[]']").fill(recipient); // this is the postcard recipient's email
	await page.locator("trix-editor").fill("I am testing your site.");
	// The CI handler skips remote verification but still requires a token.
	await page.locator("#postcard_create").evaluate((form) => {
		const token = document.createElement("input");
		token.name = "g-recaptcha-response";
		token.type = "hidden";
		token.value = "playwright-test-token";
		form.append(token);
	});

	await page.getByRole("button", { name: "Send postcard" }).click();

	await expect(page.locator(".toast")).toHaveText(
		/Thank you! Your postcard has been queued for sending!/,
	);

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

		await page.goto(postcardLink);
		await expect(page.locator("#mc-area")).toContainText([
			"I am testing your site",
		]);
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
