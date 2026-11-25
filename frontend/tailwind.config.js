/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
  theme: {
    extend: {
      colors: {
        canvas: "#0D0E11",
        surface: "#14161B",
        elevated: "#1B1E25",
        edge: "#232730",
        primary: "#F3F4F6",
        muted: "#9CA3AF",
        accent: "#E5E7EB",
        status: {
          ready: "#10B981",
          warn: "#F59E0B",
          cloud: "#6B7280",
          danger: "#EF4444",
        },
      },
      fontFamily: {
        sans: [
          '"Inter"',
          "-apple-system",
          "BlinkMacSystemFont",
          '"Segoe UI"',
          "Roboto",
          "sans-serif",
        ],
        heading: [
          '"Plus Jakarta Sans"',
          '"Inter"',
          "-apple-system",
          "sans-serif",
        ],
        mono: [
          '"JetBrains Mono"',
          '"Cascadia Code"',
          "Consolas",
          "monospace",
        ],
      },
    },
  },
  plugins: [],
};
