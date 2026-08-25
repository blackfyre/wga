import { expect, test } from "bun:test";
import { viewerImageURL } from "./viewer";

// The ViewerJS `url` option is handed each <img> element inside the viewer
// target. The shared plate carries the deliberate zoom source on that image as
// `data-zoom-url`, so the resolver must prefer it over the display `src`.
test("returns the image zoom URL when data-zoom-url is present", () => {
	const image = {
		dataset: {
			zoomUrl: "/api/files/artworks/aw0000000000001/work.jpg?thumb=2000x0",
		},
		src: "/api/files/artworks/aw0000000000001/work.jpg?thumb=1400x0",
	};

	expect(viewerImageURL(image as unknown as HTMLImageElement)).toBe(
		"/api/files/artworks/aw0000000000001/work.jpg?thumb=2000x0",
	);
});

test("falls back to the display src when no zoom URL is carried", () => {
	const image = {
		dataset: {},
		src: "/api/files/artworks/aw0000000000001/work.jpg?thumb=1400x0",
	};

	expect(viewerImageURL(image as unknown as HTMLImageElement)).toBe(
		"/api/files/artworks/aw0000000000001/work.jpg?thumb=1400x0",
	);
});
