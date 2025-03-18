/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./resources/{css,js}/*.{html,js,ts}",
    "./assets/templ/**/*.templ",
    "utils/**/*.go",
  ],
  theme: {
    extend: {},
    fontFamily: {
      sans: ["Lexend", "Arial", "sans-serif"],
      serif: ["Merriweather", "Georgia", "serif"],
      mono: ["JetBrains Mono", "monospace"],
    },
    container: {
      center: true,
      padding: "1rem",
    },
  },
  plugins: [require("@tailwindcss/typography"), require("daisyui")],
  daisyui: {
    themes: [
      {
        light: {
          ...require("daisyui/src/theming/themes")["light"],
          primary: "#013365",
          secondary: "#489393",
          base100: "#f0f6ff",
        },
      },
      {
        dark: {
          ...require("daisyui/src/theming/themes")["dark"],
          primary: "#013365",
          secondary: "#489393",
          "--rounded-box": "0.25rem",
          "--rounded-btn": ".125rem",
          "--rounded-badge": ".125rem",
          "--tab-radius": "0.25rem",
          "--animation-btn": "0",
          "--animation-input": "0",
          "--btn-focus-scale": "1",
        },
      },
      {
        eldritch: {
          primary: "#37f499",
          secondary: "#04d1f9",
          accent: "#a48cf2",
          neutral: "#323449",
          "base-100": "#212337",
          info: "#04d1f9",
          success: "#37f499",
          warning: "#f7c67f",
          error: "#f16c75",
        },
      },
    ], // false: only light + dark | true: all themes | array: specific themes like this ["light", "dark", "cupcake"]
    darkTheme: "dark", // name of one of the included themes for dark mode
    base: true, // applies background color and foreground color for root element by default
    styled: true, // include daisyUI colors and design decisions for all components
    utils: true, // adds responsive and modifier utility classes
    prefix: "", // prefix for daisyUI classnames (components, modifiers and responsive class names. Not colors)
    logs: true, // Shows info about daisyUI version and used config in the console when building your CSS
    themeRoot: ":root", // The element that receives theme color CSS variables
  },
  safelist: [
    {
      pattern: /alert-.+/,
    },
  ],
};
