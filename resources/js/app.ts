import Viewer from "viewerjs";
import Trix from "trix";
import "htmx.org";
import htmx from "htmx.org";
import warningSign from "../assets/warning-sign.svg";
import logger from "./logger";

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
	window: {
		historyBack: () => void;
		close: () => void;
		openPopUp: (w: popUpWindow) => void;
	};
	music: {
		openPopUp: () => void;
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

const dualNavAction = (side: string, url: string) => {
	logger.debug("Side", side);
	logger.debug("URL", url);
	logger.debug("Base URL", window.location.href.split("?")[0]);
	// Get the current url base
	const baseUrl = window.location.href.split("?")[0];

	const newUrl = new URL("/dual-mode", baseUrl);

	// Inherit the search parameters
	const searchParams = new URLSearchParams(window.location.search);
	searchParams.forEach((value, key) => {
		newUrl.searchParams.set(key, value);
	});

	// Set the search parameters
	newUrl.searchParams.set(side, url);

	// Create an anchor element
	htmx.ajax("get", newUrl.href, {
		target: `#${paneToTarget(side)}`,
		select: `#${paneToTarget(side)}`,
		swap: "outerHTML",
	});
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

			// Run all event listeners
			logger.debug("Running event listeners");
			for (const listener of wgaInternal.eventListeners) {
				listener();
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

			const label = document.createElement("label");

			const searchInput = document.createElement("input");

			searchInput.type = "search";

			label.appendChild(searchInput);

			// Create a table within the modal from the json which has label and url keys
			const tableContainer = document.createElement("div");
			tableContainer.className = "overflow-x-auto";
			const table = document.createElement("table");
			table.className = "table";

			tableContainer.appendChild(table);

			// artists.forEach((artist: { label: string; url: string }) => {
			//   const tr = document.createElement("tr");
			//   const td = document.createElement("td");
			//   const a = document.createElement("a");

			//   a.href = artist.url;
			//   a.textContent = artist.label;

			//   td.appendChild(a);
			//   tr.appendChild(td);
			//   table.appendChild(tr);
			// });

			// replace the contents of the dialog with the table
			modalBox.innerHTML = "";

			const modalTitle = document.createElement("h2");
			modalTitle.textContent = "Artist Lookup";
			modalBox.appendChild(modalTitle);
			modalBox.appendChild(document.createElement("hr"));

			modalBox.appendChild(label);
			modalBox.appendChild(tableContainer);
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
