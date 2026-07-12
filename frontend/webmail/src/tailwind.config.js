// tailwind.config.js
/** @type {import('tailwindcss').Config} */
export default {
    darkMode: 'class',
    content: [
      "./index.html",
      "./src/**/*.{vue,js,ts,jsx,tsx}",
    ],
    theme: {
      extend: {
        colors: {
          primary: 'rgb(0, 120, 212)',
          bg: 'rgb(249, 249, 249)',
          sidebar: 'rgb(244, 244, 244)',
          border: 'rgb(234, 234, 234)',
          'text-primary': 'rgb(33, 33, 33)',
          'text-secondary': 'rgb(102, 102, 102)',
          dark: {
            bg: 'rgb(32, 32, 32)',
            surface: 'rgb(41, 41, 41)',
            border: 'rgb(60, 60, 60)',
            text: 'rgb(243, 243, 243)',
          }
        },
        fontFamily: {
          sans: ['Segoe UI', 'system-ui', 'sans-serif'],
        }
      }
    },
    plugins: [],
  }