import { reactive } from 'vue'

// Tracks image URLs that failed to load (404, deleted upload, offline dev
// environment with no uploads) so templates can swap to a glyph/initials
// placeholder instead of the browser's native broken-image + alt text.
export function useImageFallback() {
  const failed = reactive(new Set())

  function markFailed(key) {
    if (key) failed.add(key)
  }

  function isFailed(key) {
    return Boolean(key) && failed.has(key)
  }

  function reset(key) {
    failed.delete(key)
  }

  return { markFailed, isFailed, reset }
}
