export const PERIOD_MUSIC_WINDOW_NAME = "wga-period-music";

const PLAYER_FEATURES =
	"popup=yes,width=420,height=380,resizable=yes,scrollbars=yes,menubar=no,toolbar=no,location=no,status=no";

type PlayerMessage = {
	wgaPlayer?: "playing" | "paused" | "closed";
	piece?: string;
};

type MusicMessage = {
	wgaMusic: "play" | "pause";
	href?: string;
	piece?: string;
};

const validSongID = (value: unknown): value is string =>
	typeof value === "string" && /^[a-z0-9]{15}$/.test(value);

const hasOnlyKeys = (value: object, allowed: string[]): boolean =>
	Object.keys(value).every((key) => allowed.includes(key));

const playerMessage = (value: unknown): PlayerMessage | null => {
	if (!value || typeof value !== "object" || Array.isArray(value)) {
		return null;
	}
	const message = value as PlayerMessage;
	if (!hasOnlyKeys(value, ["wgaPlayer", "piece"])) {
		return null;
	}
	if (message.wgaPlayer === "playing" && validSongID(message.piece)) {
		return message;
	}
	if (message.wgaPlayer === "paused" || message.wgaPlayer === "closed") {
		if (message.piece === undefined || validSongID(message.piece)) {
			return message;
		}
	}
	return null;
};

const musicMessage = (value: unknown): MusicMessage | null => {
	if (!value || typeof value !== "object" || Array.isArray(value)) {
		return null;
	}
	const message = value as MusicMessage;
	if (message.wgaMusic === "pause" && hasOnlyKeys(value, ["wgaMusic"])) {
		return message;
	}
	if (!hasOnlyKeys(value, ["wgaMusic", "href", "piece"])) {
		return null;
	}
	if (
		message.wgaMusic !== "play" ||
		!validSongID(message.piece) ||
		typeof message.href !== "string"
	) {
		return null;
	}
	if (!validPlayerURL(message.href)) {
		return null;
	}
	const target = new URL(message.href, window.location.href);
	if (target.searchParams.get("song") !== message.piece) {
		return null;
	}
	return message;
};

let playerWindow: Window | null = null;
let soundingPiece: string | null = null;
let initialised = false;

const musicCards = (): HTMLAnchorElement[] =>
	Array.from(document.querySelectorAll<HTMLAnchorElement>("a[data-wga-music]"));

const updateCards = (piece: string | null): void => {
	soundingPiece = piece;
	for (const card of musicCards()) {
		const active = piece !== null && card.dataset.wgaMusic === piece;
		card.dataset.wgaMusicState = "idle";
		const control = card.querySelector<HTMLElement>("[data-wga-music-control]");
		if (control) {
			control.textContent = "▶";
		}
		let action = "Open";
		if (active) {
			card.dataset.wgaMusicState = "playing";
			if (control) {
				control.textContent = "■";
			}
			action = "Stop";
		}
		const label = card
			.closest<HTMLElement>("[data-wga-music-card]")
			?.querySelector<HTMLElement>("[data-wga-music-label]")
			?.textContent?.trim();
		card.setAttribute(
			"aria-label",
			`${action} ${label || "period music"} in the period music player`,
		);
	}
};

const showBlockedNotice = (card: HTMLAnchorElement): void => {
	const container = card.closest<HTMLElement>("[data-wga-music-card]");
	const notice = container?.querySelector<HTMLElement>(
		"[data-wga-music-blocked]",
	);
	if (!notice) {
		return;
	}
	notice.hidden = false;
	notice.querySelector<HTMLButtonElement>("[data-wga-music-dismiss]")?.focus();
};

export const validPlayerURL = (href: string): boolean => {
	try {
		const parsed = new URL(href, window.location.href);
		if (
			parsed.origin !== window.location.origin ||
			parsed.pathname !== "/player"
		) {
			return false;
		}
		if (
			parsed.username !== "" ||
			parsed.password !== "" ||
			parsed.hash !== ""
		) {
			return false;
		}
		const keys = Array.from(parsed.searchParams.keys());
		return (
			keys.length === 1 &&
			keys[0] === "song" &&
			validSongID(parsed.searchParams.get("song"))
		);
	} catch {
		return false;
	}
};

const playerGeometry = (): string => {
	const availableWidth = window.screen?.availWidth || window.innerWidth;
	const left = Math.max(0, availableWidth - 470);
	return `${PLAYER_FEATURES},left=${left},top=96`;
};

const postToPlayer = (message: MusicMessage): void => {
	if (!playerWindow || playerWindow.closed) {
		return;
	}
	playerWindow.postMessage(message, window.location.origin);
};

const handleCardActivation = (
	event: MouseEvent,
	card: HTMLAnchorElement,
): void => {
	if (
		event.defaultPrevented ||
		event.button !== 0 ||
		event.metaKey ||
		event.ctrlKey ||
		event.shiftKey ||
		event.altKey
	) {
		return;
	}
	if (card.target !== PERIOD_MUSIC_WINDOW_NAME || !validPlayerURL(card.href)) {
		return;
	}

	event.preventDefault();
	const piece = card.dataset.wgaMusic || "";
	if (playerWindow && !playerWindow.closed) {
		if (soundingPiece === piece) {
			postToPlayer({ wgaMusic: "pause" });
			return;
		}
		postToPlayer({ wgaMusic: "play", href: card.href, piece });
		updateCards(null);
		try {
			playerWindow.focus();
		} catch {
			// A browser may refuse programmatic focus without affecting playback.
		}
		return;
	}

	playerWindow = window.open(
		card.href,
		PERIOD_MUSIC_WINDOW_NAME,
		playerGeometry(),
	);
	if (!playerWindow || playerWindow.closed) {
		playerWindow = null;
		showBlockedNotice(card);
	}
};

const bindCardEvents = (): void => {
	document.addEventListener("click", (event) => {
		if (!(event.target instanceof Element)) {
			return;
		}
		const card = event.target.closest<HTMLAnchorElement>("a[data-wga-music]");
		if (!card) {
			return;
		}
		handleCardActivation(event, card);
	});

	document.addEventListener("click", (event) => {
		if (!(event.target instanceof Element)) {
			return;
		}
		const dismiss = event.target.closest<HTMLButtonElement>(
			"[data-wga-music-dismiss]",
		);
		const notice = dismiss?.closest<HTMLElement>("[data-wga-music-blocked]");
		if (notice) {
			notice.hidden = true;
		}
	});

	document.body.addEventListener("htmx:afterSwap", () =>
		updateCards(soundingPiece),
	);
	window.addEventListener("message", (event: MessageEvent<unknown>) => {
		if (
			event.origin !== window.location.origin ||
			event.source !== playerWindow
		) {
			return;
		}
		const message = playerMessage(event.data);
		if (!message) {
			return;
		}
		if (message.wgaPlayer === "playing") {
			updateCards(message.piece || null);
			return;
		}
		if (message.wgaPlayer === "paused") {
			updateCards(null);
			return;
		}
		playerWindow = null;
		updateCards(null);
	});
};

const notifyOpener = (message: PlayerMessage): void => {
	if (!window.opener) {
		return;
	}
	window.opener.postMessage(message, window.location.origin);
};

const bindPlayerEvents = (player: HTMLElement): void => {
	const audio = player.querySelector<HTMLAudioElement>(
		"[data-wga-music-audio]",
	);
	if (!audio) {
		return;
	}
	const piece = player.dataset.wgaMusicSong || "";
	let navigating = false;
	audio.addEventListener("play", () =>
		notifyOpener({ wgaPlayer: "playing", piece }),
	);
	audio.addEventListener("pause", () =>
		notifyOpener({ wgaPlayer: "paused", piece }),
	);
	window.addEventListener("message", (event: MessageEvent<unknown>) => {
		if (
			event.origin !== window.location.origin ||
			event.source !== window.opener
		) {
			return;
		}
		const message = musicMessage(event.data);
		if (!message) {
			return;
		}
		if (message.wgaMusic === "pause") {
			audio.pause();
			return;
		}
		navigating = true;
		window.location.assign(message.href || "");
	});
	player
		.querySelector<HTMLButtonElement>("[data-wga-music-close]")
		?.addEventListener("click", () => window.close());
	window.addEventListener("pagehide", () => {
		if (!navigating) {
			notifyOpener({ wgaPlayer: "closed", piece });
		}
	});
};

export const initPeriodMusic = (): void => {
	if (initialised) {
		return;
	}
	initialised = true;
	bindCardEvents();
	const player = document.querySelector<HTMLElement>("[data-wga-music-player]");
	if (player) {
		bindPlayerEvents(player);
	}
};

// Auto-initialise in browser contexts. Non-DOM environments (unit tests) import
// this module without a document and call initPeriodMusic explicitly after
// installing their own DOM stub.
if (typeof document !== "undefined") {
	initPeriodMusic();
}
