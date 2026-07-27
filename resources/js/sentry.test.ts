import { expect, test } from "bun:test";
import { initialiseSentry, loadSentryConfiguration } from "./sentry";

test("loads browser Sentry configuration from page metadata", () => {
	const configuration = loadSentryConfiguration({
		querySelector(selector) {
			if (selector === 'meta[name="sentry-dsn"]') {
				return {
					getAttribute: () => "https://public@example.ingest.sentry.io/1",
				};
			}
			return { getAttribute: () => "production" };
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
		},
	]);
});
