import { defineConfig } from 'astro/config';
import solidJs from '@astrojs/solid-js';
import tailwind from '@astrojs/tailwind';

/** @type {import('shiki').ThemeRegistration} */
const monkTheme = {
  name: 'monk-dark',
  type: 'dark',
  colors: {
    'editor.background': '#1C1917',
    'editor.foreground': '#E7E5E4',
  },
  tokenColors: [
    { scope: ['keyword', 'storage.type', 'storage.modifier'], settings: { foreground: '#FB923C', fontStyle: 'bold' } },
    { scope: ['entity.name.type', 'support.type'], settings: { foreground: '#5EEAD4' } },
    { scope: ['entity.name.function', 'support.function'], settings: { foreground: '#93C5FD' } },
    { scope: ['string', 'string.quoted'], settings: { foreground: '#86EFAC' } },
    { scope: ['constant.numeric'], settings: { foreground: '#93C5FD' } },
    { scope: ['comment', 'punctuation.definition.comment'], settings: { foreground: '#78716C', fontStyle: 'italic' } },
    { scope: ['keyword.operator', 'keyword.operator.assignment', 'keyword.operator.arithmetic'], settings: { foreground: '#FCD34D' } },
    { scope: ['support.function.builtin', 'entity.name.tag'], settings: { foreground: '#FB923C' } },
    { scope: ['variable', 'variable.other', 'meta.definition.variable', 'variable.other.readwrite'], settings: { foreground: '#E7E5E4' } },
    { scope: ['punctuation', 'meta.brace'], settings: { foreground: '#A8A29E' } },
  ],
};

export default defineConfig({
  integrations: [solidJs(), tailwind()],
  output: 'static',
  markdown: {
    shikiConfig: {
      theme: monkTheme,
    },
  },
});
