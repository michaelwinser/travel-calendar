import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

export default {
  preprocess: vitePreprocess(),
  onwarn(warning, handler) {
    // Suppress "captures initial value" warnings — our modal components
    // intentionally capture initial prop values as form state.
    if (warning.code === 'state_referenced_locally') return;
    handler(warning);
  },
};
