import * as Sentry from "@sentry/browser";

export type SentryConfiguration = {
	dsn: string;
	environment: string;
	release: string;
};

const scrubURL = (value: string): string => {
	try {
		const isRelative =
			!value.startsWith("//") && !/^[a-zA-Z][a-zA-Z\d+.-]*:/.test(value);
		const url = new URL(value, "https://wga.invalid");
		url.search = "";
		url.hash = "";
		if (isRelative) {
			return url.pathname;
		}
		return url.toString();
	} catch {
		return "[invalid-url]";
	}
};

const scrubBreadcrumb = (breadcrumb: Sentry.Breadcrumb): Sentry.Breadcrumb => {
	let data = { ...breadcrumb.data };
	if (breadcrumb.category === "console") {
		data = Object.fromEntries(
			Object.entries(data).filter(([key]) => key !== "arguments"),
		);
		return { ...breadcrumb, message: undefined, data };
	}
	for (const key of ["url", "from", "to"]) {
		if (typeof data[key] === "string") {
			data[key] = scrubURL(data[key]);
		}
	}

	return { ...breadcrumb, data };
};

const scrubStackFrame = (frame: Sentry.StackFrame): Sentry.StackFrame => {
	const scrubbed = { ...frame };
	if (scrubbed.filename) {
		scrubbed.filename = scrubURL(scrubbed.filename);
	}
	if (scrubbed.abs_path) {
		scrubbed.abs_path = scrubURL(scrubbed.abs_path);
	}

	return scrubbed;
};

const scrubException = (exception: Sentry.Exception): Sentry.Exception => {
	if (!exception.stacktrace?.frames) {
		return exception;
	}

	return {
		...exception,
		stacktrace: {
			...exception.stacktrace,
			frames: exception.stacktrace.frames.map(scrubStackFrame),
		},
	};
};

const scrubEventException = (exception: Sentry.Event["exception"]) => {
	if (!exception?.values) {
		return exception;
	}

	return { ...exception, values: exception.values.map(scrubException) };
};

export const scrubSentryEvent = (event: Sentry.Event): Sentry.Event => ({
	...event,
	exception: scrubEventException(event.exception),
	request: event.request?.url
		? { ...event.request, url: scrubURL(event.request.url) }
		: event.request,
	breadcrumbs: event.breadcrumbs?.map(scrubBreadcrumb),
});

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
		release:
			document.querySelector('meta[name="sentry-release"]')?.getAttribute("content") ??
			"",
});

export const initialiseSentry = (
	configuration: SentryConfiguration,
	initialise: typeof Sentry.init = Sentry.init,
): boolean => {
	if (configuration.dsn === "") {
		return false;
	}

	try {
		const client = initialise({
			dsn: configuration.dsn,
			environment: configuration.environment,
			release: configuration.release,
			beforeBreadcrumb: scrubBreadcrumb,
			beforeSend: scrubSentryEvent,
		});
		if (!client) {
			console.error("Sentry browser initialisation failed");
			return false;
		}
		return true;
	} catch (error) {
		console.error("Sentry browser initialisation failed", error);
		return false;
	}
};

export const captureTestMessage = (
	capture: typeof Sentry.captureMessage = Sentry.captureMessage,
) => {
	capture("It works!");
};
