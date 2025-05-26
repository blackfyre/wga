export type LogLevel = "trace" | "debug" | "info" | "warn" | "error" | "silent";

const levels: LogLevel[] = [
	"trace",
	"debug",
	"info",
	"warn",
	"error",
	"silent",
];

const levelColors: Record<Exclude<LogLevel, "silent">, string> = {
	trace: "color: #aaa",
	debug: "color: #0af",
	info: "color: #0a0",
	warn: "color: orange",
	error: "color: red",
};

export interface LoggerOptions {
	serverEndpoint?: string;
	bufferSize?: number;
}

export interface Logger {
	setLevel: (level: LogLevel) => void;
	setNamespace: (namespace: string) => void;
	trace: (...args: unknown[]) => void;
	debug: (...args: unknown[]) => void;
	info: (...args: unknown[]) => void;
	warn: (...args: unknown[]) => void;
	error: (...args: unknown[]) => void;
	setOptions: (opts: LoggerOptions) => void;
	flush: () => void;
}

const createLogger = (() => {
	let levelIndex = levels.indexOf("info");
	let namespace = "App";
	let options: LoggerOptions = {};
	let logBuffer: unknown[] = [];

	const localStorageKey = "AppLoggerLevel";

	// Initialize from localStorage if exists
	const savedLevel = localStorage.getItem(localStorageKey);
	if (savedLevel && levels.includes(savedLevel as LogLevel)) {
		levelIndex = levels.indexOf(savedLevel as LogLevel);
	}

	const setLevel = (newLevel: LogLevel) => {
		const index = levels.indexOf(newLevel);
		if (index !== -1) {
			levelIndex = index;
			localStorage.setItem(localStorageKey, newLevel);
		} else {
			console.warn(`[Logger] Invalid log level: ${newLevel}`);
		}
	};

	const setNamespace = (newNamespace: string) => {
		namespace = newNamespace;
	};

	const setOptions = (opts: LoggerOptions) => {
		options = opts;
	};

	const shouldLog = (msgLevel: LogLevel): boolean =>
		levels.indexOf(msgLevel) >= levelIndex && msgLevel !== "silent";

	const logFn =
		(msgLevel: Exclude<LogLevel, "silent">) =>
		(...args: unknown[]) => {
			if (shouldLog(msgLevel)) {
				const timestamp = new Date().toISOString();
				console.log(
					`%c[${namespace}] [${msgLevel}] ${timestamp}`,
					levelColors[msgLevel],
					...args,
				);

				if (
					options.serverEndpoint &&
					(msgLevel === "warn" || msgLevel === "error")
				) {
					queueServerLog({
						timestamp,
						namespace,
						level: msgLevel,
						message: args,
					});
				}
			}
		};

	const queueServerLog = (entry: unknown) => {
		logBuffer.push(entry);

		if (logBuffer.length >= (options.bufferSize ?? 5)) {
			flushServerLogs();
		}
	};

	const flushServerLogs = () => {
		if (!options.serverEndpoint || logBuffer.length === 0) return;

		const payload = JSON.stringify(logBuffer);
		logBuffer = [];

		if (navigator.sendBeacon) {
			navigator.sendBeacon(options.serverEndpoint, payload);
		} else {
			fetch(options.serverEndpoint, {
				method: "POST",
				headers: { "Content-Type": "application/json" },
				body: payload,
			}).catch((err) => console.error("[Logger] Log ship failed", err));
		}
	};

	const logger: Logger = {
		setLevel,
		setNamespace,
		trace: logFn("trace"),
		debug: logFn("debug"),
		info: logFn("info"),
		warn: logFn("warn"),
		error: logFn("error"),
		setOptions,
		flush: flushServerLogs,
	};

	return () => logger;
})();

// Export the singleton logger instance
const logger = createLogger();
export default logger;
