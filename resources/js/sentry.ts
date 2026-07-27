import * as Sentry from "@sentry/browser";

export type SentryConfiguration = {
	dsn: string;
	environment: string;
};

type MetadataDocument = {
	querySelector: (selector: string) => {
		getAttribute: (name: string) => string | null;
	} | null;
};

export const loadSentryConfiguration = (
	document: MetadataDocument,
): SentryConfiguration => ({
	dsn:
		document
			.querySelector('meta[name="sentry-dsn"]')
			?.getAttribute("content") ?? "",
	environment:
		document
			.querySelector('meta[name="sentry-environment"]')
			?.getAttribute("content") ?? "",
});

export const initialiseSentry = (
	configuration: SentryConfiguration,
	initialise: typeof Sentry.init = Sentry.init,
): boolean => {
	if (configuration.dsn === "") {
		return false;
	}

	initialise({
		dsn: configuration.dsn,
		environment: configuration.environment,
	});
	return true;
};
