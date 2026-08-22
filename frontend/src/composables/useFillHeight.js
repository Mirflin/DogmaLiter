import { onBeforeUnmount, onMounted, ref, watch } from 'vue'

// Measures how far a section sits from the top of the viewport and turns
// that into an explicit height so it fills exactly what's left on screen,
// instead of guessing a fixed "100vh - Nrem" offset that breaks whenever a
// sibling panel (page header, character context bar, ...) changes height.
export function useFillHeight(elRef, { bottom = 24, minWidth = 1280, minHeight = 480 } = {}) {
  const heightPx = ref('')

  function recalc() {
    const el = elRef.value
    if (!el || window.innerWidth < minWidth) {
      heightPx.value = ''
      return
    }
    const top = el.getBoundingClientRect().top
    heightPx.value = `${Math.max(minHeight, window.innerHeight - top - bottom)}px`
  }

  let resizeObserver
  onMounted(() => {
    recalc()
    window.addEventListener('resize', recalc)
    resizeObserver = new ResizeObserver(recalc)
    resizeObserver.observe(document.body)
  })
  onBeforeUnmount(() => {
    window.removeEventListener('resize', recalc)
    resizeObserver?.disconnect()
  })
  watch(elRef, recalc)

  return { heightPx, recalc }
}
