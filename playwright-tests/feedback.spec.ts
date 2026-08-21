import { expect, test } from "@playwright/test";

const feedbackURL =
	"https://github.com/blackfyre/wga/issues?q=sort%3Aupdated-desc+is%3Aissue+state%3Aopen+";

test("renders feedback as a styled ordinary external link", async ({
	page,
}) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await page.goto("/");

	const feedback = page.locator("a.wga-feedback-anchor");
	await expect(feedback).toHaveAttribute("href", feedbackURL);
	await expect(feedback).not.toHaveAttribute("hx-get");
	await expect(feedback).not.toHaveAttribute("hx-on:click");
	await expect(feedback).not.toHaveAttribute("hx-target");
	await expect(feedback).not.toHaveAttribute("hx-select");
	await expect(feedback).not.toHaveAttribute("hx-swap");
	const bottom = await feedback.evaluate(
		(element) => window.innerHeight - element.getBoundingClientRect().bottom,
	);
	expect(bottom).toBeCloseTo(16, 0);
	await expect(feedback).toHaveCSS("padding-left", "18px");
	await expect(feedback).toHaveCSS("padding-top", "13px");
});

test("navigates to the exact feedback URL without JavaScript", async ({
	browser,
}) => {
	const context = await browser.newContext({ javaScriptEnabled: false });
	const page = await context.newPage();
	await page.route(feedbackURL, async (route) =>
		route.fulfill({
			status: 200,
			contentType: "text/html",
			body: "GitHub issues",
		}),
	);
	await page.goto("/");
	await page.getByRole("link", { name: "FEEDBACK" }).click();
	await expect(page).toHaveURL(feedbackURL);
	await expect(page.locator("body")).toContainText("GitHub issues");
	await context.close();
});
