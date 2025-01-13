import Viewer from "viewerjs";
import Trix from "trix";
import Htmx from "htmx.org";

declare global {
  interface Window {
    wga: wgaWindow;
  }
}

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
    openPopUp: (w: popUpWindow) => boolean | void;
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
    toast: (message: string, type: string) => void;
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
    type: string;
  };
}

const deepMerge = (target: object, source: object): object => {
  // Iterate over keys in the source object
  for (const key in source) {
    if (key === "__proto__" || key === "constructor") continue;
    if (source.hasOwnProperty(key)) {
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
  console.log("Side", side);
  console.log("URL", url);
  console.log("Base URL", window.location.href.split("?")[0]);
  // Get the current url base
  const baseUrl = window.location.href.split("?")[0];

  let newUrl = new URL("/dual-mode", baseUrl);

  // Inherit the search parameters
  const searchParams = new URLSearchParams(window.location.search);
  searchParams.forEach((value, key) => {
    newUrl.searchParams.set(key, value);
  });

  // Set the search parameters
  newUrl.searchParams.set(side, url);

  // Create an anchor element
  Htmx.ajax("get", newUrl.href, {
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
      // Toast event listener
      document.body.addEventListener("notification:toast", function (evt) {
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
      // htmx event listeners
      document.body.addEventListener("htmx:load", function (evt) {
        wgaInternal.func.viewer();
        wgaInternal.func.cloner();
        wgaInternal.func.artistSearchModal();
      });

      document.body.addEventListener("htmx:swapError", function (evt) {
        console.error(evt);
        wgaInternal.func.toast(
          "An error occurred while processing your request.",
          "danger",
        );
      });

      document.body.addEventListener("htmx:targetError", function (evt) {
        console.error(evt);
        wgaInternal.func.toast(
          "An error occurred while processing your request.",
          "danger",
        );
      });

      document.body.addEventListener("htmx:timeout", function (evt) {
        console.error(evt);
        wgaInternal.func.toast("Request timed out!", "danger");
      });
    },
    () => {
      //! This is a workaround, has to be removed when htmx fixes the issue
      addEventListener("htmx:beforeHistorySave", () => {
        document.querySelectorAll(":disabled").forEach((el) => {
          const e = el as HTMLInputElement;
          e.disabled = false;
        });
      });
    },
    () => {
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
      // Find all the elements with data-cloner-target attribute
      const cloners = document.querySelectorAll("[data-cloner-target]");

      // Remove not seen items from `wga.existingCloners`
      wgaInternal.existingCloners = wgaInternal.existingCloners.filter((el) =>
        document.body.contains(el),
      );

      cloners.forEach((c) => {
        const cloner = c as HTMLElement;
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

            removeMe.forEach((el) => {
              const removeMe = () => {
                // Find the closest .field element
                const field = el.closest("label.input");

                // Remove the field
                field?.remove();
              };

              el.removeEventListener("click", removeMe);

              // Add click event listener
              el.addEventListener("click", removeMe);
            });
          });
        }
      });
    },
    viewer: () => {
      const elements = document.querySelectorAll("[data-viewer]");
      if (elements.length > 0) {
        elements.forEach((element) => {
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
        });
      }
    },
    toast: (message: string, type: string) => {
      const toast = document.createElement("div");
      const toastMessage = document.createElement("span");
      toast.className = `alert alert-${type} cursor-pointer`;
      toastMessage.textContent = message;
      toast.appendChild(toastMessage);

      toast.addEventListener("click", () => {
        toast.remove();
      });

      wgaInternal.els.toastContainer?.appendChild(toast);

      setTimeout(() => {
        toast.remove();
      }, 5000);
    },
    init() {
      console.info("WGA Internal Functions Initialized");
      // Run all internal functions
      wgaInternal.setup.htmx();
      wgaInternal.setup.elements();

      // Run all event listeners
      wgaInternal.eventListeners.forEach((listener) => listener());
    },
    artistSearchModal() {
      const artistSearchModal = document.getElementById("artist_lookup");

      if (!artistSearchModal) {
        return;
      }

      // get the contents of #artistList and parse it as json
      const artistList = document.getElementById("artistList");
      if (!artistList) {
        return;
      }

      const artists = JSON.parse(artistList.innerHTML);

      // create a table within the modal from the json which has label and url keys
      const tableContainer = document.createElement("div");
      tableContainer.className = "overflow-x-auto";
      const table = document.createElement("table");
      table.className = "table";

      tableContainer.appendChild(table);

      artists.forEach((artist: { label: string; url: string }) => {
        const tr = document.createElement("tr");
        const td = document.createElement("td");
        const a = document.createElement("a");

        a.href = artist.url;
        a.textContent = artist.label;

        td.appendChild(a);
        tr.appendChild(td);
        table.appendChild(tr);
      });

      // replace the contents of the dialog with the table
      artistSearchModal.innerHTML = "";
      artistSearchModal.appendChild(tableContainer);
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

(function () {
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
        wgaInternal.els.dialog?.showModal();
      }, 500);
    },
    /**
     * Closes the dialog and resets its content.
     */
    close() {
      // Close the dialog
      wgaInternal.els.dialog?.close();
      // Reset the dialog content
      setTimeout(() => {
        if (wgaInternal.els.dialog) {
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
      let newWin = window.open(
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
