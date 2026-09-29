// Recovers enhanced navigation from edge (Cloudflare) managed challenges. A
// challenge can only be solved by a full page load, so an HTMX request that
// receives one would otherwise silently leave the page unchanged. Challenged
// GET requests are reloaded as a full document at the requested URL; other
// methods are never converted into navigations, so submitted data is not
// repeated. The listener is document-level and bound once, so it survives swaps.

type ChallengeDetail = {
	xhr: Pick<XMLHttpRequest, "getResponseHeader">;
	requestConfig?: { verb?: string };
	pathInfo?: { finalRequestPath?: string; requestPath?: string };
	shouldSwap: boolean;
	isError: boolean;
};

type Navigator = { assign(url: string): void };

let initialised = false;

export const isEdgeChallenge = (detail: ChallengeDetail): boolean =>
	detail.xhr.getResponseHeader("cf-mitigated")?.toLowerCase() === "challenge";

// challengeNavigation returns the URL to load as a full page for a challenged
// response, or null when the response is not a challenge or is not a GET.
export const challengeNavigation = (detail: ChallengeDetail): string | null => {
	if (!isEdgeChallenge(detail)) {
		return null;
	}
	if ((detail.requestConfig?.verb ?? "").toLowerCase() !== "get") {
		return null;
	}
	return (
		detail.pathInfo?.finalRequestPath ?? detail.pathInfo?.requestPath ?? null
	);
};

export const handleEdgeChallenge = (
	detail: ChallengeDetail,
	location: Navigator,
): void => {
	if (!isEdgeChallenge(detail)) {
		return;
	}
	detail.shouldSwap = false;
	detail.isError = false;
	const url = challengeNavigation(detail);
	if (url) {
		location.assign(url);
	}
};

export const initEdgeChallengeRecovery = (): void => {
	if (initialised) {
		return;
	}
	initialised = true;

	document.addEventListener("htmx:beforeSwap", (event) => {
		handleEdgeChallenge(
			(event as CustomEvent<ChallengeDetail>).detail,
			window.location,
		);
	});
};
