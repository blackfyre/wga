import { expect, test } from "bun:test";
import {
	initialiseSentry,
	loadSentryConfiguration,
	scrubSentryEvent,
} from "./sentry";

test("loads browser Sentry configuration from page metadata", () => {
	const configuration = loadSentryConfiguration({
		querySelector(selector) {
			if (selector === 'meta[name="sentry-dsn"]') {
				return {
					getAttribute: () => "https://public@example.ingest.sentry.io/1",
				};
			}
			if (selector === 'meta[name="sentry-environment"]') {
				return { getAttribute: () => "production" };
			}
			return null;
		},
	});

	expect(configuration).toEqual({
		dsn: "https://public@example.ingest.sentry.io/1",
		environment: "production",
	});
});

test("initialises Sentry only when a DSN is configured", () => {
	const calls: unknown[] = [];
	const initialise = ((options: unknown) => {
		calls.push(options);
		return {};
	}) as typeof import("@sentry/browser").init;

	expect(
		initialiseSentry({ dsn: "", environment: "development" }, initialise),
	).toBe(false);
	expect(calls).toEqual([]);
	expect(
		initialiseSentry(
			{
				dsn: "https://public@example.ingest.sentry.io/1",
				environment: "production",
			},
			initialise,
		),
	).toBe(true);
	expect(calls).toEqual([
		{
			dsn: "https://public@example.ingest.sentry.io/1",
			environment: "production",
			beforeSend: scrubSentryEvent,
		},
	]);
});

test("does not report successful initialisation when the SDK returns no client", () => {
	const originalError = console.error;
	console.error = () => {};

	try {
		expect(
			initialiseSentry(
				{
					dsn: "https://public@example.ingest.sentry.io/1",
					environment: "production",
				},
				(() => undefined) as typeof import("@sentry/browser").init,
			),
		).toBe(false);
	} finally {
		console.error = originalError;
	}
});

test("continues when Sentry initialisation throws", () => {
	const originalError = console.error;
	console.error = () => {};

	try {
		expect(
			initialiseSentry(
				{
					dsn: "https://public@example.ingest.sentry.io/1",
					environment: "production",
				},
				(() => {
					throw new Error("initialisation failed");
				}) as typeof import("@sentry/browser").init,
			),
		).toBe(false);
	} finally {
		console.error = originalError;
	}
});

test("scrubs query parameters from Sentry event URLs", () => {
	const event = scrubSentryEvent({
		request: { url: "https://wga.example/postcard?p=secret#fragment" },
		breadcrumbs: [
			{
				data: {
					url: "https://wga.example/postcard?p=secret",
					from: "https://wga.example/postcard?p=secret",
					to: "https://wga.example/artworks?filter=secret",
				},
			},
		],
	});

	expect(event.request?.url).toBe("https://wga.example/postcard");
	expect(event.breadcrumbs?.[0]?.data?.url).toBe(
		"https://wga.example/postcard",
	);
	expect(event.breadcrumbs?.[0]?.data?.from).toBe(
		"https://wga.example/postcard",
	);
	expect(event.breadcrumbs?.[0]?.data?.to).toBe("https://wga.example/artworks");
});

test("scrubs query parameters from relative breadcrumb URLs", () => {
	const event = scrubSentryEvent({
		breadcrumbs: [
			{
				data: { url: "/dual-mode/lookup?q=secret#fragment" },
			},
		],
	});

	expect(event.breadcrumbs?.[0]?.data?.url).toBe("/dual-mode/lookup");
});

test("scrubs query parameters from path-relative breadcrumb URLs", () => {
	const event = scrubSentryEvent({
		breadcrumbs: [
			{
				data: { url: "lookup?q=secret" },
			},
		],
	});

	expect(event.breadcrumbs?.[0]?.data?.url).toBe("/lookup");
});
