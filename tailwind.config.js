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
  plugins: [require("@tailwindcss/typography")],
  safelist: [
    {
      pattern: /alert-.+/,
    },
  ],
};
