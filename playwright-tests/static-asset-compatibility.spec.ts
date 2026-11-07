import { createServer } from "node:http";
import type { AddressInfo } from "node:net";
import { expect, type Page, test } from "@playwright/test";

// A browser holding an entry module from an earlier release can request a
// dynamic chunk that the current release no longer serves. The entry URL is
// versioned by content and revalidates, hashed chunks stay immutable, and a
// failed dynamic import reloads once before offering a manual retry.

const bootstrapChunk = /\/assets\/js\/bootstrap-[^/]+\.js(\?|$)/;
const recoveryKey = "wga.dynamic-module-recovery";

function countDocumentLoads(page: Page, pathname: string): () => number {
	let loads = 0;
	page.on("request", (request) => {
		if (
			request.resourceType() === "document" &&
			new URL(request.url()).pathname === pathname
		) {
			loads++;
		}
	});
	return () => loads;
}

test("the entry module revalidates and hashed chunks are immutable", async ({
	page,
}) => {
	const entry = page.waitForResponse(
		(response) => new URL(response.url()).pathname === "/assets/js/app.js",
	);
	const chunk = page.waitForResponse((response) =>
		bootstrapChunk.test(response.url()),
	);

	await page.goto("/artists");

	const entrySource = await page
		.locator('script[type="module"][src^="/assets/js/app.js"]')
		.getAttribute("src");
	expect(entrySource).toMatch(/^\/assets\/js\/app\.js\?v=[0-9a-f]{12}$/);
	expect(new URL((await entry).url()).search).toBe(
		new URL(entrySource ?? "", page.url()).search,
	);
	expect((await entry).headers()["cache-control"]).toBe("no-cache");
	expect((await chunk).headers()["cache-control"]).toBe(
		"public, max-age=31536000, immutable",
	);
});

test("a transient dynamic-module failure reloads once and keeps the URL", async ({
	page,
}) => {
	let failures = 0;
	await page.route(bootstrapChunk, async (route) => {
		if (failures === 0) {
			failures++;
			await route.abort();
			return;
		}
		await route.continue();
	});
	const documentLoads = countDocumentLoads(page, "/artists");
	const recovered = page.waitForResponse(
		(response) => bootstrapChunk.test(response.url()) && response.ok(),
	);

	await page.goto("/artists?letter=b");
	await recovered;

	await expect(page).toHaveURL(/\/artists\?letter=b$/);
	expect(documentLoads()).toBe(2);
	await expect
		.poll(() =>
			page.evaluate((key) => window.sessionStorage.getItem(key), recoveryKey),
		)
		.toBeNull();
	await expect(page.getByRole("button", { name: "Reload page" })).toHaveCount(
		0,
	);
});

test("a persistent dynamic-module failure offers a reload action instead of looping", async ({
	page,
}) => {
	await page.route(bootstrapChunk, (route) => route.abort());
	const documentLoads = countDocumentLoads(page, "/artists");

	await page.goto("/artists?letter=b");

	const alert = page.getByRole("alert").filter({ hasText: "assets changed" });
	await expect(alert).toBeVisible();
	await expect(page).toHaveURL(/\/artists\?letter=b$/);
	await page.waitForTimeout(500);
	expect(documentLoads()).toBe(2);

	const reload = page.waitForRequest(
		(request) =>
			request.resourceType() === "document" &&
			new URL(request.url()).pathname === "/artists",
	);
	await alert.getByRole("button", { name: "Reload page" }).click();
	await reload;
	expect(documentLoads()).toBe(3);
});

// Release A's catalogue HTML stays in the edge cache while the browser keeps
// release A's entry module for the edge's four-hour browser TTL. Release B no
// longer serves a chunk that entry imports. A proxy plays the edge and both
// origins; page.route is avoided because it disables the HTTP cache.
test("recovery after a release loads the current entry despite cached HTML and entry", async ({
	page,
	baseURL,
}) => {
	const origin = baseURL ?? "";
	const currentEntry = await (await fetch(`${origin}/assets/js/app.js`)).text();
	const currentChunk = currentEntry.match(/bootstrap-[^"]+\.js/)?.[0];
	expect(currentChunk).toBeTruthy();
	const removedChunk = "bootstrap-removed0.js";
	const releaseAEntry = currentEntry.replaceAll(
		currentChunk as string,
		removedChunk,
	);
	const cachedHTML = await (
		await fetch(`${origin}/artists`, { headers: { Accept: "text/html" } })
	).text();
	expect(cachedHTML).toMatch(/src="\/assets\/js\/app\.js\?v=[0-9a-f]{12}"/);

	let release: "A" | "B" = "A";
	const entryRequests: string[] = [];
	const currentChunkRequests: string[] = [];
	const edge = createServer(async (request, response) => {
		const url = new URL(request.url ?? "/", "http://edge.test");
		if (url.pathname === "/artists") {
			response.writeHead(200, {
				"content-type": "text/html; charset=utf-8",
				"cache-control": "no-cache",
			});
			response.end(cachedHTML);
			return;
		}
		if (url.pathname === "/assets/js/app.js") {
			entryRequests.push(release);
			response.writeHead(200, {
				"content-type": "text/javascript; charset=utf-8",
				"cache-control": "max-age=14400",
			});
			response.end(release === "A" ? releaseAEntry : currentEntry);
			return;
		}
		if (url.pathname === `/assets/js/${removedChunk}`) {
			if (release === "B") {
				response.writeHead(404, { "content-type": "text/plain" });
				response.end("not found");
				return;
			}
			url.pathname = `/assets/js/${currentChunk}`;
			const chunk = await fetch(`${origin}${url.pathname}`);
			response.writeHead(200, {
				"content-type": "text/javascript; charset=utf-8",
				"cache-control": "no-store",
			});
			response.end(await chunk.text());
			return;
		}
		if (url.pathname === `/assets/js/${currentChunk}`) {
			currentChunkRequests.push(release);
		}
		const upstream = await fetch(`${origin}${url.pathname}${url.search}`, {
			method: request.method,
			headers: { accept: request.headers.accept ?? "*/*" },
		});
		const headers: Record<string, string> = {};
		for (const name of ["content-type", "cache-control"]) {
			const value = upstream.headers.get(name);
			if (value) {
				headers[name] = value;
			}
		}
		response.writeHead(upstream.status, headers);
		response.end(Buffer.from(await upstream.arrayBuffer()));
	});
	await new Promise<void>((resolve) => edge.listen(0, "127.0.0.1", resolve));
	const { port } = edge.address() as AddressInfo;
	const pageURL = `http://127.0.0.1:${port}/artists`;

	try {
		await page.goto(pageURL);
		await expect
			.poll(() => page.evaluate(() => typeof (window as { wga?: unknown }).wga))
			.toBe("object");
		expect(entryRequests).toEqual(["A"]);

		release = "B";
		const documentLoads = countDocumentLoads(page, "/artists");
		await page.goto(pageURL);

		await expect.poll(() => currentChunkRequests).toContain("B");
		await expect
			.poll(() => page.evaluate(() => typeof (window as { wga?: unknown }).wga))
			.toBe("object");
		expect(documentLoads()).toBe(2);
		expect(entryRequests).toEqual(["A", "B"]);
		await expect(page).toHaveURL(pageURL);
		await expect(page.getByRole("button", { name: "Reload page" })).toHaveCount(
			0,
		);
	} finally {
		await new Promise((resolve) => edge.close(resolve));
	}
});
