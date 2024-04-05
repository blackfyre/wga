const SUCCESS = "is-success";
const WARNING = "is-warning";
const ERROR = "is-error";
const DEFAULT = "is-default";
const icons = {
  [SUCCESS]: `<svg class="w-6 h-6 text-green-400" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" aria-hidden="true">
                <path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
              </svg>`,
  [WARNING]: `<svg class="w-6 h-6" fill="#eaea00" version="1.1" id="Capa_1" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="64px" height="64px" viewBox="0 0 123.996 123.996" xml:space="preserve" stroke="#eaea00"><g id="SVGRepo_bgCarrier" stroke-width="0"></g><g id="SVGRepo_tracerCarrier" stroke-linecap="round" stroke-linejoin="round"></g><g id="SVGRepo_iconCarrier"><g><path d="M9.821,118.048h104.4c7.3,0,12-7.7,8.7-14.2l-52.2-92.5c-3.601-7.199-13.9-7.199-17.5,0l-52.2,92.5 C-2.179,110.348,2.521,118.048,9.821,118.048z M70.222,96.548c0,4.8-3.5,8.5-8.5,8.5s-8.5-3.7-8.5-8.5v-0.2c0-4.8,3.5-8.5,8.5-8.5 s8.5,3.7,8.5,8.5V96.548z M57.121,34.048h9.801c2.699,0,4.3,2.3,4,5.2l-4.301,37.6c-0.3,2.7-2.1,4.4-4.6,4.4s-4.3-1.7-4.6-4.4 l-4.301-37.6C52.821,36.348,54.422,34.048,57.121,34.048z"></path> </g> </g></svg>`,
  [ERROR]: `<svg class="w-6 h-6" fill="#ff0000" height="64px" width="64px" version="1.1" id="Layer_1" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 493.636 493.636" xml:space="preserve" stroke="#ff0000"><g id="SVGRepo_bgCarrier" stroke-width="0"></g><g id="SVGRepo_tracerCarrier" stroke-linecap="round" stroke-linejoin="round"></g><g id="SVGRepo_iconCarrier"><g><g><path d="M421.428,72.476C374.868,25.84,312.86,0.104,246.724,0.044C110.792,0.044,0.112,110.624,0,246.548 c-0.068,65.912,25.544,127.944,72.1,174.584c46.564,46.644,108.492,72.46,174.4,72.46h0.58v-0.048 c134.956,0,246.428-110.608,246.556-246.532C493.7,181.12,468,119.124,421.428,72.476z M257.516,377.292 c-2.852,2.856-6.844,4.5-10.904,4.5c-4.052,0-8.044-1.66-10.932-4.516c-2.856-2.864-4.496-6.852-4.492-10.916 c0.004-4.072,1.876-8.044,4.732-10.884c2.884-2.86,7.218-4.511,11.047-4.542c3.992,0.038,7.811,1.689,10.677,4.562 c2.872,2.848,4.46,6.816,4.456,10.884C262.096,370.46,260.404,374.432,257.516,377.292z M262.112,304.692 c-0.008,8.508-6.928,15.404-15.448,15.404c-8.5-0.008-15.42-6.916-15.416-15.432L231.528,135 c0.004-8.484,3.975-15.387,15.488-15.414c4.093,0.021,7.895,1.613,10.78,4.522c2.912,2.916,4.476,6.788,4.472,10.912 L262.112,304.692z"></path> </g> </g> </g></svg>`,
  [DEFAULT]: `<svg class="w-6 h-6" xmlns="http://www.w3.org/2000/svg" id="Layer_1" data-name="Layer 1" viewBox="0 0 24 24" width="512" height="512"><path d="m17.994,2.287C16.086.582,13.517-.227,10.956.059,6.904.517,3.59,3.781,3.075,7.821c-.374,2.933.644,5.761,2.793,7.762,1.375,1.278,2.132,2.9,2.132,4.566v3.851h8v-3.685c0-1.817.704-3.476,1.932-4.552,1.95-1.71,3.068-4.175,3.068-6.764,0-2.56-1.096-5.007-3.006-6.713Zm-3.994,19.713h-4s-.001-1.95-.002-2h4.008c-.004.105-.006,2-.006,2Zm2.613-7.74c-1.106.969-1.897,2.271-2.303,3.74h-4.622c-.42-1.431-1.258-2.765-2.458-3.881-1.671-1.557-2.463-3.76-2.171-6.045.399-3.137,2.974-5.672,6.121-6.027,2.028-.234,3.976.387,5.481,1.731,1.485,1.327,2.338,3.23,2.338,5.222,0,2.013-.87,3.93-2.387,5.26Zm-1.66-6.794c.224,1.271-.381,2.542-1.506,3.163-.188.103-.447.563-.447.876v.495h-2v-.495c0-1.033.637-2.163,1.481-2.628.289-.16.595-.535.502-1.065-.069-.393-.402-.726-.793-.794-.31-.058-.603.021-.832.216-.228.19-.358.47-.358.767h-2c0-.889.391-1.727,1.072-2.299.681-.572,1.578-.814,2.464-.653,1.21.211,2.205,1.206,2.417,2.418Zm-3.953,6.534h2v2h-2v-2Z"/></svg>`,
};

export const toast = (message, type) => {
  try {
    createToast(message, type);
    closeToast();
  } catch (error) {
    console.error(error);
  }
};

const handleToastClose = () => closeToast(0);
const decorateToast =
  (func) =>
  (...args) => {
    const toast = document.getElementById("toast");
    toast && func(toast, ...args);
  };
const switchClass = (el, removeClass, addClass) => {
  el.classList?.remove(removeClass);
  el.classList?.add(addClass);
};
const createToast = (message, type) => {
  if (!message) console.warn("message parameter is missing for toast");
  if (!type) console.warn("type parameter is missing for toast");

  const toast = document.createElement("div");
  toast.id = "toast";
  toast.setAttribute("aria-live", "assertive");
  toast.classList.add(
    "toast",
    "fixed",
    "inset-0",
    "z-50",
    "flex",
    "items-end",
    "px-4",
    "py-6",
    "pointer-events-none",
    "entering",
    "sm:items-start",
    "sm:p-6",
  );
  decorateToast(switchClass)("entering", "entered");
  toast.innerHTML = toastComponent(message, type);
  toast
    .querySelector("#closeToast")
    .addEventListener("click", handleToastClose);
  document.body.appendChild(toast);
};
const closeToast = (time = 3000) => {
  const toast = document.getElementById("toast");
  decorateToast(switchClass)("entered", "leaving");
  setTimeout(() => {
    decorateToast(switchClass)("leaving", "left");
    setTimeout(() => toast.remove(), 200);
  }, time);
};
const getIcon = (type, iconList = icons) =>
  iconList[type] ? iconList[type] : iconList[DEFAULT];

function toastComponent(message, type) {
  const icon = getIcon(type);
  return `
  <div class="flex flex-col items-center w-full space-y-4 sm:items-end">
    <div class="w-full max-w-sm overflow-hidden bg-white rounded-lg shadow-lg pointer-events-auto ring-1 ring-black ring-opacity-5">
      <div class="p-4">
        <div class="flex items-start">
          <div class="flex-shrink-0">
            ${icon}
          </div>
          <div class="ml-3 w-0 flex-1 pt-0.5">
            <p class="text-sm font-medium text-gray-500">${message}</p>
          </div>
          <div class="flex flex-shrink-0 ml-4">
            <button id="closeToast" type="button" class="inline-flex text-gray-400 bg-white rounded-md hover:text-gray-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2">
              <span class="sr-only">Close</span>
              <svg class="w-5 h-5" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                <path d="M6.28 5.22a.75.75 0 00-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 101.06 1.06L10 11.06l3.72 3.72a.75.75 0 101.06-1.06L11.06 10l3.72-3.72a.75.75 0 00-1.06-1.06L10 8.94 6.28 5.22z"></path>
              </svg>
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
  `;
}
