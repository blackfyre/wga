module.exports = {
  plugins: {
    "@tailwindcss/postcss": {},
    "postcss-import": {},
    autoprefixer: {},
    ...(process.env.NODE_ENV === "production" ? { cssnano: {} } : {}),
  },
};
