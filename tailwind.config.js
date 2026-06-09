/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        'panel': '#0a1628',
        'panel-light': '#111d35',
        'panel-border': '#1a2d4d',
        'tech-blue': '#00d4ff',
        'tech-green': '#00ff88',
        'tech-yellow': '#ffdd00',
        'tech-orange': '#ff8800',
        'tech-red': '#ff4444',
        'tech-purple': '#b44dff',
      },
      fontFamily: {
        'mono': ['JetBrains Mono', 'Consolas', 'monospace'],
        'sans': ['Inter', 'system-ui', 'sans-serif'],
      },
      animation: {
        'pulse-ring': 'pulse-ring 2s cubic-bezier(0.215, 0.61, 0.355, 1) infinite',
        'blink': 'blink 1s step-end infinite',
      },
      keyframes: {
        'pulse-ring': {
          '0%': { transform: 'scale(0.5)', opacity: '1' },
          '100%': { transform: 'scale(2)', opacity: '0' },
        },
        'blink': {
          '0%, 100%': { opacity: '1' },
          '50%': { opacity: '0' },
        },
      },
    },
  },
  plugins: [],
}
