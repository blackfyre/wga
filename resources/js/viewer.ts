// Shared ViewerJS image URL resolution.
//
// ViewerJS 1.11.7 resolves each image's URL through its `url` option. We point
// the viewer at the deliberate zoom source carried on the <img> element as
// `data-zoom-url`, and fall back to the ordinary display `src` when that
// attribute is absent (for example on a no-zoom plate). The zoom source must
// live on the <img> itself: ViewerJS hands the image element, not the anchor,
// to the resolver.
export function viewerImageURL(image: HTMLImageElement): string {
	return image.dataset.zoomUrl || image.src;
}
