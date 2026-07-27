import {
	captureTestMessage,
	initialiseSentry,
	loadSentryConfiguration,
} from "./sentry";

const sentryReady = initialiseSentry(loadSentryConfiguration(document));
if (window.location.pathname === "/sentry-test") {
	if (sentryReady) {
		captureTestMessage();
	}
} else {
	void import("./bootstrap");
}
