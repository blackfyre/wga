import Trix from "trix";
import Viewer from "viewerjs";
import "htmx.org";
import htmx from "htmx.org";
import warningSign from "../assets/warning-sign.svg";
import logger from "./logger";
import { initStatisticsChart } from "./statistics";

logger.setNamespace("WGA");
logger.setLevel("debug");

// const debugMode = import.meta.env.VITE_DEBUG_MODE === "true";

declare global {
	interface Window {
		wga: wgaWindow;
	}
}

type HtmxValidateUrlEvent = CustomEvent<{
	sameHost: boolean;
}>;

type popUpWindow = {
	url: string;
	title: string;
	position: {
		x: number;
		y: number;
	};
	size: {
		width: number;
		height: number;
	};
	scrollbars?: boolean;
};

type wgaWindow = {
	dialog: {
		open: () => void;
		close: () => void;
	};
	dual: {
		openLookup: (side: string) => void;
		setPaneTarget: (side: string, openInOtherPane: boolean) => void;
	};
	window: {
		historyBack: () => void;
		close: () => void;
		openPopUp: (w: popUpWindow) => void;
	};
	music: {
		openPopUp: () => void;
	};
	glossary: {
		closeAll: () => void;
	};
};

type wgaInternals = {
	els: {
		dialog: HTMLDialogElement | null;
		toastContainer: HTMLElement | null;
	};
	existingCloners: HTMLElement[];
	dialogDefaultContent: string;
	eventListeners: (() => void)[];
	func: {
		cloner: () => void;
		viewer: () => void;
		toast: (message: string, type: ToastEvent["detail"]["type"]) => void;
		artistSearchModal: () => void;
		glossary: () => void;
		init: () => void;
	};

	setup: {
		htmx: () => void;
		elements: () => void;
	};
};

type wgaComboBox = {
	input: {
		style: string;
		placeholder: string;
		type: string;
	};
	list: {
		style: string;
	};
	moreAvailable: {
		style: string;
		content: string;
	};
	noResults: {
		style: string;
		content: string;
	};
	item: {
		style: string;
	};
	options: object[];
	hooks: {
		onSelected: (v: string) => void;
	};
};

type wgaComboBoxConfig = {
	input?: {
		style?: string;
		placeholder?: string;
		type?: string;
	};
	list?: {
		style?: string;
	};
	moreAvailable?: {
		style?: string;
		content?: string;
	};
	noResults?: {
		style?: string;
		content?: string;
	};
	item?: {
		style?: string;
	};
	options?: {
		id: string;
		label: string;
	}[];
	hooks?: {
		onSelected?: (v: string) => void;
	};
};

interface ToastEvent extends Event {
	detail: {
		closeDialog: boolean;
		message: string;
		type: "info" | "alert" | "warning" | "error" | "success";
	};
}

const deepMerge = (target: object, source: object): object => {
	// Iterate over keys in the source object
	for (const key in source) {
		if (key === "__proto__" || key === "constructor") continue;
		if (Object.prototype.hasOwnProperty.call(source, key)) {
			// Check if the current key's value is an object and exists in the target
			if (typeof source[key] === "object" && source[key] !== null) {
				if (Array.isArray(source[key])) {
					// For arrays, merge (or you can decide to replace target[key] if needed)
					target[key] = (target[key] || []).concat(source[key]);
				} else {
					// For objects, initialize target[key] if it doesn't exist, then recourse
					target[key] = deepMerge(target[key] || {}, source[key]);
				}
			} else {
				// For primitives, directly assign the source value
				target[key] = source[key];
			}
		}
	}
	return target;
};

const paneToTarget = (side: string): string => {
	if (side === "left") {
		return "right";
	}

	return "left";
};

const normalizeDualPathInput = (value: string): string | null => {
	const normalizedValue = value.trim();

	if (!normalizedValue) {
		return null;
	}

	if (
		normalizedValue.startsWith("http://") ||
		normalizedValue.startsWith("https://")
	) {
		try {
			const parsed = new URL(normalizedValue);
			if (parsed.origin !== window.location.origin) {
				return null;
			}

			return parsed.pathname;
		} catch {
			return null;
		}
	}

	if (
		normalizedValue.startsWith("/artists/") ||
		normalizedValue.startsWith("/artworks/")
	) {
		return normalizedValue;
	}

	if (
		normalizedValue.startsWith("artists/") ||
		normalizedValue.startsWith("artworks/")
	) {
		return `/${normalizedValue}`;
	}

	return null;
};

const updateDualMode = (mutateUrl: (url: URL) => void) => {
	const nextUrl = new URL(window.location.href);
	nextUrl.pathname = "/dual-mode";

	mutateUrl(nextUrl);

	htmx.ajax("get", nextUrl.toString(), {
		target: "#dual-area",
		select: "#dual-area",
		swap: "outerHTML",
	});
};

const setDualPane = (side: string, value: string) => {
	updateDualMode((nextUrl) => {
		nextUrl.searchParams.set(side, value);
	});
};

const updatePaneTarget = (side: string, openInOtherPane: boolean) => {
	updateDualMode((nextUrl) => {
		const target = openInOtherPane ? paneToTarget(side) : side;
		nextUrl.searchParams.set(`${side}_render_to`, target);
	});
};

// Glossary popup state — hoisted to avoid event listener leaks
const glossaryState: {
	activePopup: HTMLElement | null;
	activeTerm: HTMLElement | null;
} = {
	activePopup: null,
	activeTerm: null,
};
const glossaryPopupID = "glossary-popup";

const glossaryClosePopup = (restoreFocus = false) => {
	const activeTerm = glossaryState.activeTerm;

	if (glossaryState.activePopup) {
		glossaryState.activePopup.remove();
		glossaryState.activePopup = null;
	}
	if (activeTerm) {
		activeTerm.removeAttribute("aria-controls");
		activeTerm.setAttribute("aria-expanded", "false");
		glossaryState.activeTerm = null;

		if (restoreFocus) {
			activeTerm.focus();
		}
	}
};

const glossaryShowPopup = (term: HTMLElement) => {
	glossaryClosePopup();

	const definitionTemplate = term.nextElementSibling;
	if (
		!(definitionTemplate instanceof HTMLTemplateElement) ||
		!definitionTemplate.classList.contains("glossary-definition")
	) {
		return;
	}

	const popup = document.createElement("div");
	popup.id = glossaryPopupID;
	popup.className = "glossary-popup";
	popup.setAttribute("role", "dialog");
	popup.setAttribute(
		"aria-label",
		`Definition of ${term.textContent || "glossary term"}`,
	);
	popup.tabIndex = -1;

	const closeButton = document.createElement("button");
	closeButton.type = "button";
	closeButton.className = "glossary-popup-close";
	closeButton.setAttribute("aria-label", "Close definition");
	closeButton.textContent = "×";
	closeButton.addEventListener("click", (e) => {
		e.stopPropagation();
		glossaryClosePopup(true);
	});

	const definition = document.createElement("div");
	definition.id = `${glossaryPopupID}-definition`;
	definition.append(definitionTemplate.content.cloneNode(true));
	popup.setAttribute("aria-describedby", definition.id);
	popup.append(closeButton, definition);

	popup.addEventListener("click", (e) => e.stopPropagation());

	document.body.appendChild(popup);
	glossaryState.activePopup = popup;
	glossaryState.activeTerm = term;
	term.setAttribute("aria-controls", popup.id);
	term.setAttribute("aria-expanded", "true");

	// Position relative to term
	const rect = term.getBoundingClientRect();
	const scrollY = window.scrollY;
	const scrollX = window.scrollX;

	popup.style.left = `${rect.left + scrollX}px`;
	popup.style.top = `${rect.bottom + scrollY + 8}px`;

	// Adjust if popup overflows right edge
	const popupRect = popup.getBoundingClientRect();
	if (popupRect.right > window.innerWidth - 16) {
		popup.style.left = `${window.innerWidth - popupRect.width - 16 + scrollX}px`;
	}

	// Adjust if popup overflows bottom — show above instead
	if (popupRect.bottom > window.innerHeight) {
		popup.style.top = `${rect.top + scrollY - popupRect.height - 8}px`;
	}

	popup.focus();
};

// One-time document-level listeners for glossary dismissal
document.addEventListener("click", glossaryClosePopup);
document.addEventListener("keydown", (e) => {
	if (e.key === "Escape") glossaryClosePopup(true);
});

let statisticsModule: typeof import("./statistics") | null = null;

const maybeInitStatisticsCharts = async () => {
	if (!document.getElementById("statistics")) {
		statisticsModule?.destroyStatisticsCharts();
		return;
	}

	try {
		if (statisticsModule) {
			statisticsModule.initStatisticsChart();
			return;
		}

		statisticsModule = await import("./statistics");
		statisticsModule.initStatisticsChart();
	} catch (error) {
		logger.error("Failed to initialise statistics charts", error);
	}
};

const wgaInternal: wgaInternals = {
	els: {
		dialog: null,
		toastContainer: null,
	},
	existingCloners: [],
	dialogDefaultContent: "",
	eventListeners: [
		() => {
			logger.debug("Setting up toast event listener");
			// Toast event listener
			document.body.addEventListener("notification:toast", (evt) => {
				const event = evt as ToastEvent;
				if (event.detail.closeDialog) {
					wgaInternal.els.dialog?.close();
				}

				wgaInternal.func.toast(event.detail.message, event.detail.type);
			});
		},
		// () => {
		//   document.addEventListener("DOMContentLoaded", function (event) {
		//     wgaInternal.func.viewer();
		//     wgaInternal.func.cloner();
		//     wgaInternal.func.augmentSelects();
		//   });
		// },
		() => {
			logger.debug("Setting up HTMX event listeners");
			// htmx event listeners
			document.body.addEventListener("htmx:load", () => {
				wgaInternal.func.viewer();
				wgaInternal.func.cloner();
				wgaInternal.func.artistSearchModal();
				wgaInternal.func.glossary();
				void maybeInitStatisticsCharts();
			});
			document.body.addEventListener("htmx:beforeSwap", () => {
				glossaryClosePopup();
				statisticsModule?.destroyStatisticsCharts();
			});

			document.body.addEventListener("htmx:swapError", (evt) => {
				logger.error("HTMX swap error", evt);
				wgaInternal.func.toast(
					"An error occurred while processing your request.",
					"error",
				);
			});

			document.body.addEventListener("htmx:targetError", (evt) => {
				logger.error("HTMX target error", evt);
				wgaInternal.func.toast(
					"An error occurred while processing your request.",
					"error",
				);
			});

			document.body.addEventListener("htmx:timeout", (evt) => {
				logger.error("HTMX request timed out", evt);
				wgaInternal.func.toast("Request timed out!", "error");
			});

			document.body.addEventListener("htmx:validateUrl", (evt) => {
				logger.debug("HTMX validateUrl event", evt);
				const event = evt as HtmxValidateUrlEvent;
				// Only allow requests to the current server
				if (!event.detail.sameHost) {
					evt.preventDefault();
				}
			});
		},
		() => {
			logger.debug("Setting up HTMX beforeHistorySave event listener");
			//! This is a workaround, has to be removed when htmx fixes the issue
			addEventListener("htmx:beforeHistorySave", () => {
				for (const el of document.querySelectorAll(":disabled")) {
					const e = el as HTMLInputElement;
					e.disabled = false;
				}
			});
		},
		() => {
			logger.debug("Setting up Trix event listeners");
			// Trix initialization
			document.addEventListener("trix-before-initialize", () => {
				Trix.config.toolbar.getDefaultHTML = () => {
					return `
        <div class="trix-button-row">
          <span class="trix-button-group trix-button-group--text-tools" data-trix-button-group="text-tools">
            <button type="button" class="trix-button trix-button--icon trix-button--icon-bold btn-neutral" data-trix-attribute="bold" data-trix-key="b" title="Bold" tabindex="-1">Bold</button>
            <button type="button" class="trix-button trix-button--icon trix-button--icon-italic btn-neutral" data-trix-attribute="italic" data-trix-key="i" title="Italic" tabindex="-1">Italic</button>
            <button type="button" class="trix-button trix-button--icon trix-button--icon-strike btn-neutral" data-trix-attribute="strike" title="Strike" tabindex="-1">Strike</button>
          </span>
    
          <span class="trix-button-group trix-button-group--block-tools" data-trix-button-group="block-tools">
            <button type="button" class="trix-button trix-button--icon trix-button--icon-heading-1 btn-neutral" data-trix-attribute="heading1" title="Heading 1" tabindex="-1">Heading 1</button>
            <button type="button" class="trix-button trix-button--icon trix-button--icon-quote btn-neutral" data-trix-attribute="quote" title="Quote" tabindex="-1">Quote</button>
          </span>
    
        </div>`;
				}; // Change `Trix.config` if you need
			});
		},
		() => {
			logger.debug("Setting up jumpToTop event listener");
			// Back to top button
			const jumpToTop = document.querySelector(".jump.back-to-top");
			if (jumpToTop) {
				jumpToTop.addEventListener("click", () => {
					window.scrollTo({ top: 0, behavior: "smooth" });
				});
			}
		},
	],
	func: {
		cloner: () => {
			logger.debug("Setting up cloner functionality");
			// Find all the elements with data-cloner-target attribute
			const cloners = document.querySelectorAll("[data-cloner-target]");

			// Remove not seen items from `wga.existingCloners`
			wgaInternal.existingCloners = wgaInternal.existingCloners.filter((el) =>
				document.body.contains(el),
			);

			for (const c of cloners) {
				const cloner = c as HTMLElement;
				logger.debug("Processing cloner", cloner);
				// Get the target element
				const target = document.querySelector(
					cloner.dataset.clonerTarget as string,
				);

				if (!target) {
					return;
				}

				// Get the target's innerHTML as the template
				const template = target.innerHTML;

				// If target not in `wga.existingCloners`

				if (!wgaInternal.existingCloners.includes(target as HTMLElement)) {
					// Add target to `wga.existingCloners`
					wgaInternal.existingCloners.push(target as HTMLElement);

					cloner.addEventListener("click", () => {
						// Append the template to the target
						target?.insertAdjacentHTML("beforeend", template);

						// Find all the elements with data-cloner-remove-me attribute
						const removeMe = target.querySelectorAll("[data-cloner-remove-me]");

						// Loop through each removeMe

						for (const el of removeMe) {
							const removeMe = () => {
								// Find the closest .field element
								const field = el.closest("label.input");

								// Remove the field
								field?.remove();
							};

							el.removeEventListener("click", removeMe);

							// Add click event listener
							el.addEventListener("click", removeMe);
						}
					});
				}
			}
		},
		viewer: () => {
			logger.debug("Setting up ViewerJS functionality");
			const elements = document.querySelectorAll("[data-viewer]");
			if (elements.length > 0) {
				for (const element of elements) {
					const e = element as HTMLElement;
					new Viewer(e, {
						toolbar: {
							zoomIn: 1,
							zoomOut: 1,
							oneToOne: 1,
							reset: 1,
							prev: 1,
							play: {
								show: 1,
								size: "large",
							},
							next: 1,
							rotateLeft: 1,
							rotateRight: 1,
							flipHorizontal: 0,
							flipVertical: 0,
						},
					});
				}
			}
		},
		toast: (message, type) => {
			logger.debug("Toast creation started", { message, type });
			// Define variants for color, title, and icon
			const colorVariants = {
				info: "alert alert-info cursor-pointer sm:alert-horizontal",
				alert: "alert cursor-pointer sm:alert-horizontal",
				warning: "alert alert-warning cursor-pointer sm:alert-horizontal",
				error: "alert alert-error cursor-pointer sm:alert-horizontal",
				success: "alert alert-success cursor-pointer sm:alert-horizontal",
			};

			const titleVariants = {
				info: "Info",
				alert: "Alert",
				warning: "Warning",
				error: "Error",
				success: "Success",
			};

			const iconVariants = {
				info: `<svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 shrink-0 stroke-current" fill="none" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>`,
				alert: `<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" class="stroke-info h-6 w-6 shrink-0">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
      </svg>`,
				warning: warningSign,
				error: `<svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 shrink-0 stroke-current" fill="none" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>`,
				success: `<svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 shrink-0 stroke-current" fill="none" viewBox="0 0 24 24">
    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
  </svg>`,
			};

			// Create elements
			const toast = document.createElement("div");
			const mb = document.createElement("div");
			const title = document.createElement("h3");
			const toastMessage = document.createElement("span");

			// Set icon and content
			toast.innerHTML = iconVariants[type] || iconVariants.info;
			title.textContent = titleVariants[type] || "Notification";
			title.className = "font-bold";
			toastMessage.textContent = message;

			// Set classes and attributes
			toast.className =
				colorVariants[type] ||
				"alert alert-info cursor-pointer sm:alert-horizontal";
			toast.setAttribute("role", "alert");

			// Compose toast content
			mb.appendChild(title);
			mb.appendChild(toastMessage);
			toast.appendChild(mb);

			// Add close on click
			toast.addEventListener("click", () => {
				toast.remove();
			});

			logger.debug("Toast presenting");
			// Show toast
			wgaInternal.els.toastContainer?.appendChild(toast);

			// Auto-remove after 5 seconds
			setTimeout(() => {
				logger.debug("Removing toast after 5 seconds");
				toast.remove();
			}, 5000);
		},
		init() {
			logger.debug("WGA Internal Functions Initialized");
			// Run all internal functions
			logger.debug("Setting up HTMX and elements");
			wgaInternal.setup.htmx();
			wgaInternal.setup.elements();
			wgaInternal.func.glossary();
			void maybeInitStatisticsCharts();

			// Run all event listeners
			logger.debug("Running event listeners");
			for (const listener of wgaInternal.eventListeners) {
				listener();
			}
		},
		glossary() {
			const terms = document.querySelectorAll(
				".glossary-term:not([data-glossary-bound])",
			);
			if (terms.length === 0) return;

			for (const el of terms) {
				const term = el as HTMLElement;
				term.setAttribute("data-glossary-bound", "true");
				const togglePopup = () => {
					if (glossaryState.activeTerm === term) {
						glossaryClosePopup();
						return;
					}

					glossaryShowPopup(term);
				};

				term.addEventListener("click", (e) => {
					e.preventDefault();
					e.stopPropagation();
					togglePopup();
				});

				term.addEventListener("keydown", (e) => {
					if (e.key === "Enter" || e.key === " ") {
						e.preventDefault();
						togglePopup();
					}
					if (e.key === "Escape") {
						glossaryClosePopup(true);
					}
				});
			}
		},
		artistSearchModal() {
			const artistSearchModal = document.getElementById("artist_lookup");

			if (!artistSearchModal) {
				return;
			}

			//find .modal-box in the modal
			const modalBox = artistSearchModal.querySelector(".modal-box");

			if (!modalBox) {
				logger.error("Modal box not found");
				return;
			}

			// get the contents of #artistList and parse it as json
			const artistList = document.getElementById("artistList");
			if (!artistList) {
				logger.error("Artist list not found");
				return;
			}

			const artists = JSON.parse(artistList.innerHTML);

			const side = artistSearchModal.getAttribute("data-side") || "left";

			const searchInput = document.createElement("input");
			searchInput.type = "search";
			searchInput.placeholder = "Filter artists";
			searchInput.className = "input input-bordered w-full";

			const pathInput = document.createElement("input");
			pathInput.type = "text";
			pathInput.placeholder = "/artists/... or /artworks/...";
			pathInput.className = "input input-bordered w-full";

			const pathButton = document.createElement("button");
			pathButton.type = "button";
			pathButton.className = "btn btn-secondary btn-sm";
			pathButton.textContent = `Load ${side} pane`;

			const resultList = document.createElement("div");
			resultList.className =
				"mt-4 max-h-80 overflow-y-auto rounded-box border border-base-300";

			const renderArtistButtons = (filterValue: string) => {
				resultList.innerHTML = "";

				const filteredArtists = artists.filter(
					(artist: { label: string; url: string }) =>
						artist.label.toLowerCase().includes(filterValue.toLowerCase()),
				);

				if (filteredArtists.length === 0) {
					const emptyState = document.createElement("p");
					emptyState.className = "p-4 text-sm text-base-content/70";
					emptyState.textContent = "No artists match that filter.";
					resultList.appendChild(emptyState);
					return;
				}

				for (const artist of filteredArtists) {
					const button = document.createElement("button");
					button.type = "button";
					button.className =
						"btn btn-ghost h-auto w-full justify-start rounded-none px-4 py-3 text-left";
					button.textContent = artist.label;
					button.addEventListener("click", () => {
						setDualPane(side, artist.url);
						(artistSearchModal as HTMLDialogElement).close();
					});
					resultList.appendChild(button);
				}
			};

			searchInput.addEventListener("input", () => {
				renderArtistButtons(searchInput.value);
			});

			pathButton.addEventListener("click", () => {
				const normalizedPath = normalizeDualPathInput(pathInput.value);

				if (!normalizedPath) {
					wgaInternal.func.toast(
						"Use a canonical /artists/... or /artworks/... path.",
						"warning",
					);
					return;
				}

				setDualPane(side, normalizedPath);
				(artistSearchModal as HTMLDialogElement).close();
			});

			// replace the contents of the dialog with the table
			modalBox.innerHTML = "";

			const modalTitle = document.createElement("h2");
			modalTitle.className = "mb-2 text-xl font-semibold";
			modalTitle.textContent = `Load ${side} pane`;
			const modalText = document.createElement("p");
			modalText.className = "mb-4 text-sm text-base-content/70";
			modalText.textContent =
				"Pick an artist from the list or paste a canonical artist or artwork path.";
			modalBox.appendChild(modalTitle);
			modalBox.appendChild(modalText);
			modalBox.appendChild(searchInput);
			modalBox.appendChild(document.createElement("div")).className = "h-3";
			modalBox.appendChild(pathInput);
			modalBox.appendChild(document.createElement("div")).className = "h-3";
			modalBox.appendChild(pathButton);
			modalBox.appendChild(resultList);

			renderArtistButtons("");
		},
	},

	setup: {
		htmx: () => {
			// Setup htmx
		},
		elements: () => {
			wgaInternal.els.dialog = document.getElementById(
				"d",
			) as HTMLDialogElement;
			wgaInternal.els.toastContainer = document.getElementById(
				"toast-container",
			) as HTMLElement;
			wgaInternal.dialogDefaultContent =
				wgaInternal.els.dialog?.innerHTML || "";
		},
	},
};

(() => {
	logger.debug("Initializing WGA");
	wgaInternal.func.init();
})();

window.wga = {
	dialog: {
		/**
		 * Opens the dialog.
		 */
		open() {
			// Open the dialog
			setTimeout(() => {
				logger.debug("Opening dialog");
				wgaInternal.els.dialog?.showModal();
			}, 500);
		},
		/**
		 * Closes the dialog and resets its content.
		 */
		close() {
			logger.debug("Closing dialog");
			// Close the dialog
			wgaInternal.els.dialog?.close();
			// Reset the dialog content
			setTimeout(() => {
				if (wgaInternal.els.dialog) {
					logger.debug("Resetting dialog content");
					wgaInternal.els.dialog.innerHTML = wgaInternal.dialogDefaultContent;
				}
			}, 500);
		},
	},
	dual: {
		openLookup(side: string) {
			const modal = document.getElementById(
				"artist_lookup",
			) as HTMLDialogElement | null;
			if (!modal) {
				return;
			}

			modal.setAttribute("data-side", side);
			wgaInternal.func.artistSearchModal();
			modal.showModal();
		},
		setPaneTarget(side: string, openInOtherPane: boolean) {
			updatePaneTarget(side, openInOtherPane);
		},
	},
	window: {
		historyBack() {
			window.history.back();
		},
		close() {
			window.close();
		},
		openPopUp(w: popUpWindow) {
			logger.debug("Opening pop-up window", w);
			const newWin = window.open(
				w.url,
				w.title,
				`width=${w.size.width},height=${w.size.height},left=${w.position.x},top=${w.position.y},scrollbars=${w.scrollbars ? "yes" : "no"}, resizable=yes, dependent=yes, toolbar=no, menubar=no, location=no, directories=no, status=no, popup=yes`,
			);

			if (!newWin) {
				return false;
			}

			newWin.opener = this;
			newWin.focus();
			return;
		},
	},
	glossary: {
		closeAll() {
			glossaryClosePopup();
		},
	},
	music: {
		openPopUp() {
			window.wga.window.openPopUp({
				url: "/music",
				title: "Music",
				position: {
					x: 0,
					y: 0,
				},
				size: {
					width: 300,
					height: 400,
				},
			});
		},
	},
};
