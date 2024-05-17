export const dualModeOperator = () => {
  if (!window.location.pathname.includes("/dualmode")) {
    return;
  }
  try {
    createDualModePage();
  } catch (error) {
    console.error(error);
  }
};

const createDualModePage = async () => {
  const page = document.createElement("div");
  page.innerHTML = dualModePage();
  const dualModeSection = document.getElementById("dualModeSection");
  if (!dualModeSection) {
    return;
  }
  dualModeSection.appendChild(page);
  console.log("Before fetchDefaultPages");
  await fetchDefaultPages();
  console.log("After fetchDefaultPages");

  const leftLinks = document.querySelectorAll("#leftPageContent a");
  const rightLinks = document.querySelectorAll("#rightPageContent a");

  const rewriteLeftLinksCheckbox = document.getElementById(
    "rewriteLeftLinksCheckbox",
  );
  const rewriteRightLinksCheckbox = document.getElementById(
    "rewriteRightLinksCheckbox",
  );

  if (
    !rewriteLeftLinksCheckbox ||
    !rewriteRightLinksCheckbox ||
    leftLinks.length === 0 ||
    rightLinks.length === 0
  ) {
    return;
  }

  const elements = {
    leftLinks,
    rightLinks,
    rewriteLeftLinksCheckbox,
    rewriteRightLinksCheckbox,
  };

  const listener: EventListenerOrEventListenerObject =
    updateLinkBehavior(elements);
  rewriteLeftLinksCheckbox.addEventListener("change", listener);
  rewriteRightLinksCheckbox.addEventListener("change", listener);
};

function updateLinkBehavior(elements) {
  let leftPage = document.getElementById("leftPageContent");
  let rightPage = document.getElementById("rightPageContent");
  if (!leftPage || !rightPage) {
    return;
  }
  elements.leftLinks.forEach((link: HTMLElement) => {
    link.addEventListener("click", function (e) {
      e.preventDefault();
      const url = link.getAttribute("href");
      if (!url) return;
      if (elements.rewriteLeftLinksCheckbox.checked) {
        fetch(url)
          .then((response) => response.text())
          .then((html) => {
            const parser = new DOMParser();
            const doc = parser.parseFromString(html, "text/html");
            const element = doc.getElementById("mc-area");
            console.log("🚀 ~ .then ~ element:", element);
            if (!element) {
              return;
            }
            rightPage.innerHTML = element.innerHTML;
          });
      } else {
        fetch(url)
          .then((response) => response.text())
          .then((html) => {
            const parser = new DOMParser();
            const doc = parser.parseFromString(html, "text/html");
            const element = doc.getElementById("mc-area");
            if (!element) {
              return;
            }
            leftPage.innerHTML = element.innerHTML;
          });
      }
    });
  });

  elements.rightLinks.forEach((link: HTMLElement) => {
    link.addEventListener("click", function (e) {
      e.preventDefault();
      const url = link.getAttribute("href");
      if (!url) return;
      if (elements.rewriteRightLinksCheckbox.checked) {
        fetch(url)
          .then((response) => response.text())
          .then((html) => {
            const parser = new DOMParser();
            const doc = parser.parseFromString(html, "text/html");
            const element = doc.getElementById("mc-area");
            if (!element) {
              return;
            }
            leftPage.innerHTML = element.innerHTML;
          });
      } else {
        fetch(url)
          .then((response) => response.text())
          .then((html) => {
            const parser = new DOMParser();
            const doc = parser.parseFromString(html, "text/html");
            const element = doc.getElementById("mc-area");
            if (!element) {
              return;
            }
            rightPage.innerHTML = element.innerHTML;
          });
      }
    });
  });
}

function fetchDefaultPages() {
  let leftPage = document.getElementById("leftPageContent");
  let rightPage = document.getElementById("rightPageContent");
  if (!leftPage || !rightPage) {
    return;
  }
  if (leftPage?.innerHTML && rightPage?.innerHTML) {
    return;
  }

  return Promise.all([
    fetch("/artists")
      .then((response) => response.text())
      .then((html) => {
        const parser = new DOMParser();
        const doc = parser.parseFromString(html, "text/html");
        const element = doc.getElementById("mc-area");
        if (!element) {
          return;
        }
        leftPage.innerHTML = element.innerHTML;
      }),
    fetch("/guestbook")
      .then((response) => response.text())
      .then((html) => {
        const parser = new DOMParser();
        const doc = parser.parseFromString(html, "text/html");
        const element = doc.getElementById("mc-area");
        if (!element) {
          return;
        }
        rightPage.innerHTML = element.innerHTML;
      }),
  ]);
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
