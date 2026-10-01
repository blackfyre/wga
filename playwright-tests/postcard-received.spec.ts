import { type Browser, expect, type Page, test } from "@playwright/test";

type MailpitSearchResponse = {
	messages: { ID: string; Subject: string }[];
};

type MailpitMessage = {
	HTML: string;
};

const artworkID = "2225c982be1af02";
const artworkTitle = "Synthetic Artwork 01-01";
const artworkPath =
	"/artists/synthetic-artist-01-ad32608c6e36b2e/synthetic-artwork-01-01-2225c982be1af02";
const composePath = `/postcard/send?awid=${artworkID}`;
const subject = "You got a postcard from Received Tester!";

// One postcard is sent for the whole file: the submission throttle allows only
// a few successful sends per client, and every test reuses the shared link.
test.describe.configure({ mode: "serial" });

let postcardPath = "";
let mailpitMessageIDs: string[] = [];

function mailpitURL(): string {
	const url = process.env.MAILPIT_URL;
	if (!url) throw new Error("MAILPIT_URL environment variable is not set.");
	return url;
}

async function sendPostcard(page: Page, recipient: string): Promise<void> {
	await page.goto(composePath);
	await page.getByLabel("YOUR NAME").fill("Received Tester");
	await page.getByLabel("YOUR EMAIL").fill("received.tester@local.host");
	await page.getByLabel("Recipient email 1").fill(recipient);
	await page.locator("[data-rte-surface]").fill("A postcard to follow back.");
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
}

test.beforeAll(async ({ browser }) => {
	test.setTimeout(150000);
	const recipient = `received.${Date.now()}@local.host`;
	const context = await browser.newContext({
		baseURL: test.info().project.use.baseURL,
	});
	const page = await context.newPage();
	try {
		await sendPostcard(page, recipient);
		const searchURL = `${mailpitURL()}/api/v1/search?${new URLSearchParams({
			query: `to:${recipient}`,
		})}`;
		await expect
			.poll(
				async () => {
					const response = await context.request.get(searchURL);
					if (!response.ok()) return 0;
					const found = (await response.json()) as MailpitSearchResponse;
					mailpitMessageIDs = found.messages
						.filter((message) => message.Subject === subject)
						.map((message) => message.ID);
					return mailpitMessageIDs.length;
				},
				{ intervals: [1000, 2000, 5000], timeout: 120000 },
			)
			.toBe(1);
		const messageResponse = await context.request.get(
			`${mailpitURL()}/api/v1/message/${mailpitMessageIDs[0]}`,
		);
		const message = (await messageResponse.json()) as MailpitMessage;
		const link = message.HTML.match(
			/<a\b[^>]*\bhref=["']([^"']+)["'][^>]*>\s*OPEN YOUR POSTCARD/i,
		)?.[1];
		if (!link) throw new Error("Postcard link not found");
		const url = new URL(link.replaceAll("&amp;", "&"));
		postcardPath = `${url.pathname}${url.search}`;
	} finally {
		await context.close();
	}
});

test.afterAll(async ({ request }) => {
	if (mailpitMessageIDs.length === 0) return;
	await request.delete(`${mailpitURL()}/api/v1/messages`, {
		data: { ids: mailpitMessageIDs },
	});
});

test("received card shows the work's dimensions and holding location", async ({
	page,
}) => {
	await page.goto(postcardPath);
	const card = page.locator("#postcard-view article");
	await expect(card.locator("[data-postcard-material]")).toHaveText(
		"Oil on canvas, 101 x 201 cm",
	);
	await expect(card.locator("[data-postcard-location]")).toHaveText(
		"Synthetic Museum of Fine Arts",
	);
});

for (const viewport of [
	{ width: 390, height: 844 },
	{ width: 834, height: 900 },
	{ width: 1440, height: 1000 },
]) {
	test(`received postcard links back into the gallery at ${viewport.width}px`, async ({
		page,
	}) => {
		await page.setViewportSize(viewport);
		await page.goto(postcardPath);

		const view = page.locator("#postcard-view");
		const gallery = view.getByRole("link", { name: "VIEW IN GALLERY →" });
		const title = view.getByRole("link", { name: artworkTitle });
		const sendOwn = view.getByRole("link", { name: "SEND YOUR OWN →" });
		const browse = view.getByRole("link", { name: "BROWSE THE ARCHIVE →" });
		for (const link of [gallery, title, sendOwn, browse]) {
			await expect(link).toBeVisible();
		}
		await expect(title).toHaveAttribute("href", artworkPath);
		expect(
			await page.evaluate(
				() => document.documentElement.scrollWidth <= window.innerWidth,
			),
		).toBeTruthy();

		await gallery.click();
		await expect(page).toHaveURL(new RegExp(`${artworkPath}$`));
		await expect(page.getByRole("heading", { level: 1 })).toContainText(
			artworkTitle,
		);

		await page.goto(postcardPath);
		await view.getByRole("link", { name: "SEND YOUR OWN →" }).click();
		await expect(page).toHaveURL(
			new RegExp(`/postcard/send\\?awid=${artworkID}$`),
		);
		await expect(page.locator("#postcard-compose")).toContainText(artworkTitle);
		await expect(page.getByLabel("YOUR NAME")).toHaveValue("");
		await expect(page.getByLabel("Recipient email 1")).toHaveValue("");
		await expect(page.locator("[data-rte-count]")).toHaveText(
			"300 CHARACTERS LEFT",
		);
	});
}

async function withoutJavaScript(
	browser: Browser,
	run: (page: Page) => Promise<void>,
): Promise<void> {
	const context = await browser.newContext({
		baseURL: test.info().project.use.baseURL,
		javaScriptEnabled: false,
	});
	try {
		await run(await context.newPage());
	} finally {
		await context.close();
	}
}

test("received postcard links navigate without JavaScript", async ({
	browser,
}) => {
	await withoutJavaScript(browser, async (page) => {
		await page.goto(postcardPath);
		await page
			.locator("#postcard-view")
			.getByRole("link", { name: artworkTitle })
			.click();
		await expect(page).toHaveURL(new RegExp(`${artworkPath}$`));

		await page.goto(postcardPath);
		await page
			.locator("#postcard-view")
			.getByRole("link", { name: "VIEW IN GALLERY →" })
			.click();
		await expect(page).toHaveURL(new RegExp(`${artworkPath}$`));

		await page.goto(postcardPath);
		await page.getByRole("link", { name: "SEND YOUR OWN →" }).click();
		await expect(page).toHaveURL(
			new RegExp(`/postcard/send\\?awid=${artworkID}$`),
		);
		await expect(page.locator("#postcard_create")).toBeVisible();
		await expect(page.locator("[data-rte-count]")).toHaveText(
			"300 CHARACTERS AT MOST",
		);

		await page.goto(postcardPath);
		await page.getByRole("link", { name: "BROWSE THE ARCHIVE →" }).click();
		await expect(page).toHaveURL(/\/artists$/);
	});
});
