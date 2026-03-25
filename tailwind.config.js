/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    './web/templates/**/*.html',
    './web/src/js/**/*.jsx',
    './web/src/js/**/*.js',
  ],
  theme: {
    extend: {
      colors: {
        'terminal-bg': '#0a0e17',
        'terminal-green': '#00ff41',
        'terminal-red': '#ff6b6b',
        'terminal-cyan': '#4ecdc4',
        'terminal-gray': '#888888',
        'terminal-dark': '#0d1117',
        'terminal-border': '#1a2332',
        // New refined tokens
        'surface': '#0d1117',
        'surface-hi': '#151d2b',
        'border-hi': '#243044',
        'green-dim': '#00cc33',
      },
      fontFamily: {
        mono: ['"JetBrains Mono"', '"Fira Code"', 'monospace'],
      },
    },
  },
  plugins: [],
}
