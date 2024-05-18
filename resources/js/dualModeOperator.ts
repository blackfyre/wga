const DEFAULTLEFTURL = "/artists";
const DEFAULTRIGHTURL = "/artists";

export const dualModeOperator = async (): Promise<void> => {
  if (!window.location.pathname.includes("/dualmode")) {
    return;
  }
  try {
    await createDualModePage();
  } catch (error) {
    window.wga.toast(`Failed to create dual mode page: ${error}`, "danger");
  }
};

const createDualModePage = async (): Promise<void> => {
  const page = document.createElement("div");
  page.innerHTML = dualModePage();
  const dualModeSection = document.getElementById("dualModeSection");
  if (!dualModeSection) {
    return;
  }
  dualModeSection.appendChild(page);
  await fetchDefaultPages();

  const leftLinksCheckbox = getInputElementById("rewriteLeftLinksCheckbox");
  const rightLinksCheckbox = getInputElementById("rewriteRightLinksCheckbox");

  if (leftLinksCheckbox && rightLinksCheckbox) {
    updateLinkBehavior(leftLinksCheckbox, rightLinksCheckbox)();
  }
};

const fetchDefaultPages = async (): Promise<void> => {
  const [leftPage, rightPage] = ["leftPageContent", "rightPageContent"].map(
    (id) => document.getElementById(id),
  );

  if (!leftPage || !rightPage || leftPage.innerHTML || rightPage.innerHTML) {
    return;
  }

  const pages = [
    { page: leftPage, start: DEFAULTLEFTURL },
    { page: rightPage, start: DEFAULTRIGHTURL },
  ];

  try {
    await Promise.all(
      pages.map(({ page, start }) => fetchAndUpdatePage(start, page)),
    );
  } catch (error) {
    handleError("Failed to fetch default pages", error);
  }
};

const handleFetchError =
  <T extends (...args: unknown[]) => Promise<void>>(fn: Function) =>
  async (...args: Parameters<T>) => {
    try {
      await fn(...args);
    } catch (error) {
      handleError("Failed to fetch and update page", error);
    }
  };

const fetchAndUpdatePage = handleFetchError(
  async (url: string, page: HTMLElement) => {
    const response = await fetch(url);
    const html = await response.text();
    const parser = new DOMParser();
    const doc = parser.parseFromString(html, "text/html");
    const element = doc.getElementById("mc-area");
    if (element) {
      page.innerHTML = element.innerHTML;
    }
  },
);

function updateLinkBehavior(
  leftLinksCheckbox: HTMLInputElement,
  rightLinksCheckbox: HTMLInputElement,
): () => void {
  const leftPage = document.getElementById("leftPageContent");
  const rightPage = document.getElementById("rightPageContent");

  return function () {
    if (!leftPage || !rightPage) {
      return;
    }
    attachLinkBehavior(leftPage, leftLinksCheckbox, [leftPage, rightPage]);
    attachLinkBehavior(rightPage, rightLinksCheckbox, [rightPage, leftPage]);
  };
}

const createLinkClickHandler =
  (checkbox: HTMLInputElement, pageToUpdate: HTMLElement[]) =>
  async (e: Event): Promise<void> => {
    const link = (e.target as HTMLElement).closest("a");
    const url = link?.href;
    if (!url) return;
    e.preventDefault();

    const page = checkbox.checked ? pageToUpdate[1] : pageToUpdate[0];
    await fetchAndUpdatePage(url, page);
  };

function attachLinkBehavior(
  parentElement: HTMLElement,
  checkbox: HTMLInputElement,
  pageToUpdate: HTMLElement[],
): void {
  parentElement.addEventListener(
    "click",
    createLinkClickHandler(checkbox, pageToUpdate),
  );
}

function getInputElementById(id: string): HTMLInputElement | null {
  const el = document.getElementById(id);
  return el instanceof HTMLInputElement ? el : null;
}

function dualModePage(): string {
  return `
    <section class="container mx-auto flex h-screen">
        <div id="leftPageContent" class="w-1/2 h-full overflow-y-auto border-r"></div>
        <div id="rightPageContent" class="w-1/2 h-full overflow-y-auto"></div>
    </section>
    <section class="container mx-auto flex justify-between p-4">
        <label>
            Rewrite links on the left page
            <input type="checkbox" id="rewriteLeftLinksCheckbox"/>
        </label>
        <label>
            Rewrite links on the right page
            <input type="checkbox" id="rewriteRightLinksCheckbox"/>
        </label>
    </section>
  `;
}

const handleError = (message: string, error: unknown): void => {
  console.error(message, error);
  window.wga.toast(`${message}: ${error}`, "danger");
};
