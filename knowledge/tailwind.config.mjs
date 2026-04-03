/** @type {import('tailwindcss').Config} */
export default {
  content: ['./src/**/*.{astro,html,js,jsx,md,mdx,svelte,ts,tsx,vue}'],
  theme: {
    extend: {
      colors: {
        monk: {
          bg: '#FAF8F5',
          card: '#FFFFFF',
          border: '#E8E4DF',
          text: '#1A1A1A',
          muted: '#6B6560',
          accent: '#E8590C',
          accentLight: '#FFF4ED',
          green: '#16A34A',
          greenLight: '#F0FDF4',
          blue: '#2563EB',
          blueLight: '#EFF6FF',
          teal: '#0D9488',
          tealLight: '#F0FDFA',
          amber: '#D97706',
          amberLight: '#FFFBEB',
          locked: '#D1CDC8',
        },
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'Fira Code', 'monospace'],
      },
    },
  },
  plugins: [],
};
