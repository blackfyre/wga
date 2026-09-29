import { expect, test } from "bun:test";
import { handleEdgeChallenge } from "./edge-challenge";

const detail = (verb: string, headers: Record<string, string>) => ({
	xhr: {
		getResponseHeader: (name: string) => headers[name.toLowerCase()] ?? null,
	},
	requestConfig: { verb },
	pathInfo: {
		requestPath: "/artworks",
		finalRequestPath: "/artworks?q=milkmaid",
	},
	shouldSwap: false,
	isError: true,
});

const recordingLocation = () => {
	const assigned: string[] = [];
	return { assigned, assign: (url: string) => assigned.push(url) };
};

test("a challenged GET loads the requested URL as a full page", () => {
	const location = recordingLocation();
	const challenged = detail("get", { "cf-mitigated": "challenge" });

	handleEdgeChallenge(challenged, location);

	expect(location.assigned).toEqual(["/artworks?q=milkmaid"]);
	expect(challenged.shouldSwap).toBe(false);
	expect(challenged.isError).toBe(false);
});

test("a challenged POST is neither swapped nor resubmitted", () => {
	const location = recordingLocation();
	const challenged = detail("post", { "cf-mitigated": "Challenge" });

	handleEdgeChallenge(challenged, location);

	expect(location.assigned).toEqual([]);
	expect(challenged.shouldSwap).toBe(false);
});

test("an ordinary 403 keeps the existing error handling", () => {
	const location = recordingLocation();
	const forbidden = detail("get", {});

	handleEdgeChallenge(forbidden, location);

	expect(location.assigned).toEqual([]);
	expect(forbidden.isError).toBe(true);
});
