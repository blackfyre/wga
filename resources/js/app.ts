import htmx from "htmx.org";
import Viewer from "viewerjs";
import Trix from "trix";
import Choices from "choices.js";

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
    augmentSelects: () => void;
    init: () => void;
  };

  setup: {
    htmx: () => void;
    elements: () => void;
  };
};

interface ToastEvent extends Event {
  detail: {
    closeDialog: boolean;
    message: string;
    type: string;
  };
}

const wgaInternal: wgaInternals = {
  els: {
    dialog: null,
    toastContainer: null,
  },
  existingCloners: [],
  dialogDefaultContent: "",
  eventListeners: [
    () => {
      document.body.addEventListener("notification:toast", function (evt) {
        const event = evt as ToastEvent;
        if (event.detail.closeDialog) {
          wgaInternal.els.dialog?.close();
        }

        wgaInternal.func.toast(event.detail.message, event.detail.type);
      });
    },
    () => {
      document.body.addEventListener("htmx:load", function (evt) {
        wgaInternal.func.viewer();
        wgaInternal.func.cloner();
        wgaInternal.func.augmentSelects();
      });

      document.body.addEventListener("htmx:swapError", function (evt) {
        console.error(evt);

        // TODO: show error toast: An error occurred while processing your request.
        wgaInternal.func.toast(
          "An error occurred while processing your request.",
          "danger",
        );
      });

      document.body.addEventListener("htmx:targetError", function (evt) {
        console.error(evt);
        // TODO: show error toast: An error occurred while processing your request.
        wgaInternal.func.toast(
          "An error occurred while processing your request.",
          "danger",
        );
      });

      document.body.addEventListener("htmx:timeout", function (evt) {
        console.error(evt);
        // TODO: show error toast: Request timed out.
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
        }; // Change Trix.config if you need
      });
    },
    () => {
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
      // find all the elements with data-cloner-target attribute
      const cloners = document.querySelectorAll("[data-cloner-target]");

      // remove not seen items from wga.existingCloners
      wgaInternal.existingCloners = wgaInternal.existingCloners.filter((el) =>
        document.body.contains(el),
      );

      cloners.forEach((c) => {
        const cloner = c as HTMLElement;
        // get the target element
        const target = document.querySelector(
          cloner.dataset.clonerTarget as string,
        );

        if (!target) {
          return;
        }

        // get the target's innerHTML as the template
        const template = target.innerHTML;

        // if target not in wga.existingCloners

        if (!wgaInternal.existingCloners.includes(target as HTMLElement)) {
          // add target to wga.existingCloners
          wgaInternal.existingCloners.push(target as HTMLElement);

          cloner.addEventListener("click", () => {
            // append the template to the target
            target?.insertAdjacentHTML("beforeend", template);

            // find all the elements with data-cloner-remove-me attribute
            const removeMe = target.querySelectorAll("[data-cloner-remove-me]");

            // loop through each removeMe

            removeMe.forEach((el) => {
              const removeMe = () => {
                // find the closest .field element
                const field = el.closest("label.input");

                // remove the field
                field?.remove();
              };

              el.removeEventListener("click", removeMe);

              // add click event listener
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
      // run all internal functions
      wgaInternal.setup.htmx();
      wgaInternal.setup.elements();

      // run all event listeners
      wgaInternal.eventListeners.forEach((listener) => listener());
    },
    augmentSelects() {
      // find all the elements with data-choices attribute
      const choices = document.querySelectorAll("[data-choices]");
      choices.forEach((c) => {
        const selector = c as HTMLSelectElement;
        const listId = selector.dataset.choices;

        if (!listId) {
          console.error("data-choices attribute is required");
          return;
        }

        // create a new Choices instance
        const i = new Choices(selector);

        const list = JSON.parse(
          document.getElementById(listId)?.textContent || "",
        );

        console.log(list);

        i.setChoices(list, "url", "label", true);
      });
    },
  },

  setup: {
    htmx: () => {
      htmx.config.globalViewTransitions = true;
      htmx.config.selfRequestsOnly = true;
      htmx.config.allowScriptTags = false;
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
      // open the dialog
      setTimeout(() => {
        wgaInternal.els.dialog?.showModal();
      }, 500);
    },
    /**
     * Closes the dialog and resets its content.
     */
    close() {
      // close the dialog
      wgaInternal.els.dialog?.close();
      // reset the dialog content
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
