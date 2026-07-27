import { initialiseSentry, loadSentryConfiguration } from "./sentry";

initialiseSentry(loadSentryConfiguration(document));
void import("./bootstrap");
