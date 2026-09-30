import { readdirSync } from "node:fs";
import { join, relative, sep } from "node:path";
import { expect, test } from "@playwright/test";
import {
	commentarySelection,
	curatedArtist,
	dualPath,
	musicSong,
	relationshipArtwork,
} from "./helpers/synthetic-fixture";

const artistPath = curatedArtist.path;
const artworkPath = relationshipArtwork.path;
const selectionPath = commentarySelection.path;
const musicPath = musicSong.path;

const releaseRoutes = [
	"/",
	"/artists",
	artistPath,
	selectionPath,
	"/artworks?q=Synthetic",
	artworkPath,
	dualPath(artistPath, artworkPath),
	"/search?q=Synthetic",
	"/timeline?from=1800&to=1900",
	"/inspire",
	"/tours",
	"/statistics",
	"/glossary?q=pala",
	"/pages/about",
	"/pages/privacy-policy",
	"/contributors",
	"/open-source-licences",
	"/itineraries",
	"/itineraries/new",
	"/postcard",
	`/postcard/send?awid=${relationshipArtwork.id}`,
	"/guestbook",
	musicPath,
] as const;

const publicRoot = join(process.cwd(), "internal/assets/public");

function publicAssetPaths(directory = publicRoot): string[] {
	return readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
		if (entry.name.startsWith(".")) return [];
		const path = join(directory, entry.name);
		if (entry.isDirectory()) return publicAssetPaths(path);
		return [`/assets/${relative(publicRoot, path).split(sep).join("/")}`];
	});
}

test.describe("release acceptance inventory", () => {
	test("serves every public release destination with fixture records", async ({
		request,
	}) => {
		test.setTimeout(120_000);

		for (const route of releaseRoutes) {
			const response = await request.get(route);
			expect(response.ok(), `${route} returned ${response.status()}`).toBe(
				true,
			);
			expect(response.headers()["content-type"], route).toContain("text/html");
			expect((await response.body()).byteLength, route).toBeGreaterThan(0);
		}
	});

	test("serves every embedded public asset", async ({ request }) => {
		for (const path of publicAssetPaths()) {
			const response = await request.get(path);
			expect(response.ok(), `${path} returned ${response.status()}`).toBe(true);
			expect((await response.body()).byteLength, path).toBeGreaterThan(0);
		}
	});

	test("serves artwork and music files used by release records", async ({
		page,
		request,
	}) => {
		const fileURLs = new Set<string>();
		for (const route of [artistPath, artworkPath, musicPath]) {
			const response = await page.goto(route);
			expect(response?.ok(), route).toBe(true);
			for (const url of await page
				.locator(
					"[src*='/api/files/'], [href*='/api/files/'], [data-zoom-url*='/api/files/']",
				)
				.evaluateAll((elements) =>
					elements.flatMap((element) =>
						["src", "href", "data-zoom-url"]
							.map((attribute) => element.getAttribute(attribute))
							.filter((value): value is string => value !== null),
					),
				)) {
				fileURLs.add(url);
			}
		}

		expect(fileURLs.size).toBeGreaterThanOrEqual(3);
		for (const url of fileURLs) {
			const response = await request.get(url);
			expect(response.ok(), `${url} returned ${response.status()}`).toBe(true);
			expect((await response.body()).byteLength, url).toBeGreaterThan(0);
		}
	});

	test("serves the generated discovery assets", async ({ request }) => {
		for (const path of ["/sitemap.xml", "/robots.txt", "/llms.txt"]) {
			const response = await request.get(path);
			expect(response.ok(), `${path} returned ${response.status()}`).toBe(true);
			expect((await response.body()).byteLength, path).toBeGreaterThan(0);
		}
	});
});
