import { expect, test } from "@playwright/test";

const stackHeight = (page: import("@playwright/test").Page) =>
	page.evaluate(() =>
		Number.parseFloat(
			getComputedStyle(document.body).getPropertyValue(
				"--wga-bottom-stack-height",
			),
		),
	);

for (const width of [390, 834, 1440]) {
	test(`measures and coordinates fixed bottom surfaces at ${width}px`, async ({
		page,
	}) => {
		await page.setViewportSize({ width, height: 844 });
		await page.goto("/");
		await page.evaluate(() => {
			document.querySelector("#cc-main")?.remove();
			const tray = document.createElement("div");
			tray.className = "wga-bottom-stack-item";
			tray.dataset.wgaBottomStackItem = "itinerary";
			tray.dataset.wgaBottomStackOrder = "10";
			tray.style.cssText =
				"position:fixed;inset-inline:0;height:72px;background:black";
			document.body.append(tray);

			const consent = document.createElement("div");
			consent.id = "cc-main";
			consent.innerHTML =
				'<div class="cm cm--bottom" style="display:block!important;visibility:visible;transform:none!important;opacity:1!important;position:fixed;height:120px;width:280px;background:white"></div>';
			document.body.append(consent);
		});
		const noticeOnlyHeight = 136;

		await expect
			.poll(() => stackHeight(page))
			.toBeCloseTo(noticeOnlyHeight + 72, 0);
		const geometry = await page.evaluate(() => {
			const tray = document.querySelector<HTMLElement>(
				'[data-wga-bottom-stack-item="itinerary"]',
			);
			const notice = document.querySelector<HTMLElement>("#cc-main .cm");
			const main = document.querySelector<HTMLElement>("#mc-area");
			const toast = document.querySelector<HTMLElement>("#toast-container");
			if (!main || !toast) {
				throw new Error("shared bottom-stack mounts are missing");
			}
			return {
				trayBottom: tray
					? innerHeight - tray.getBoundingClientRect().bottom
					: -1,
				noticeBottom: notice
					? innerHeight - notice.getBoundingClientRect().bottom
					: -1,
				mainPadding: Number.parseFloat(getComputedStyle(main).paddingBottom),
				toastBottom: Number.parseFloat(getComputedStyle(toast).bottom),
			};
		});
		expect(geometry.trayBottom).toBeCloseTo(0, 0);
		expect(geometry.noticeBottom).toBeCloseTo(88, 0);
		expect(geometry.mainPadding).toBeCloseTo(noticeOnlyHeight + 72, 0);
		expect(geometry.toastBottom).toBeCloseTo(noticeOnlyHeight + 88, 0);

		await page.evaluate(() =>
			document
				.querySelector('[data-wga-bottom-stack-item="itinerary"]')
				?.remove(),
		);
		await expect.poll(() => stackHeight(page)).toBeCloseTo(noticeOnlyHeight, 0);
	});
}

test("footer exposes labelled ordinary community links without JavaScript", async ({
	browser,
}) => {
	const context = await browser.newContext({ javaScriptEnabled: false });
	const page = await context.newPage();
	await page.goto("/");
	for (const link of [
		{
			name: "WGA on GitHub (opens in a new tab)",
			href: "https://github.com/blackfyre/wga/",
		},
		{
			name: "Web Gallery of Art on Mastodon (opens in a new tab)",
			href: "https://mastodon.social/@webgalleryofart",
		},
	]) {
		const destination = page.getByRole("link", { name: link.name });
		await expect(destination).toHaveAttribute("href", link.href);
		await expect(destination).toHaveAttribute("target", "_blank");
		await expect(destination).toHaveAttribute("rel", "noopener");
	}
	await context.close();
});
