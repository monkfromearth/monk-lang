import { defineConfig } from 'astro/config';
import solidJs from '@astrojs/solid-js';
import tailwind from '@astrojs/tailwind';

/** @type {import('shiki').ThemeRegistration} */
const monkTheme = {
  name: 'monk-light',
  type: 'light',
  colors: {
    'editor.background': '#FAFAF9',
    'editor.foreground': '#1A1A1A',
  },
  tokenColors: [
    { scope: ['keyword', 'storage.type', 'storage.modifier'], settings: { foreground: '#E8590C' } },
    { scope: ['entity.name.type', 'support.type'], settings: { foreground: '#0D9488' } },
    { scope: ['entity.name.function', 'support.function'], settings: { foreground: '#2563EB' } },
    { scope: ['string', 'string.quoted'], settings: { foreground: '#16A34A' } },
    { scope: ['constant.numeric'], settings: { foreground: '#2563EB' } },
    { scope: ['comment', 'punctuation.definition.comment'], settings: { foreground: '#9C9590' } },
    { scope: ['keyword.operator'], settings: { foreground: '#D97706' } },
    { scope: ['support.function.builtin', 'entity.name.tag'], settings: { foreground: '#E8590C' } },
    { scope: ['variable', 'meta.definition.variable'], settings: { foreground: '#1A1A1A' } },
    { scope: ['punctuation'], settings: { foreground: '#1A1A1A' } },
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
