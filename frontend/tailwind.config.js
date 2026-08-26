/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,jsx}'],
  theme: {
    extend: {
      fontFamily: {
        display: ['"Poppins"', 'ui-sans-serif', 'system-ui', 'sans-serif'],
        sans: ['"Inter"', 'ui-sans-serif', 'system-ui', 'sans-serif'],
      },
      colors: {
        ink: {
          900: '#131A2C', // headings, primary buttons
          800: '#1B2338',
        },
        slate: {
          25: '#F8FAFC',
        },
        accent: {
          DEFAULT: '#5B6B95', // step badge / muted brand blue
        },
      },
      boxShadow: {
        field: '0 1px 2px 0 rgb(0 0 0 / 0.03)',
      },
      borderRadius: {
        xl2: '0.875rem',
      },
    },
  },
  plugins: [],
}
