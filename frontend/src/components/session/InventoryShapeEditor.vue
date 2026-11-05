<script setup>
import { computed, ref } from 'vue'
import { countOpen, fullMask } from './inventoryMask'

// Paint a backpack shape: press on a cell to open or close it, keep the pointer
// down and drag to paint more cells the same way.
const props = defineProps({
  width: { type: Number, required: true },
  height: { type: Number, required: true },
  // Always a full-length mask here; the parent decides whether "all open" means a plain grid.
  modelValue: { type: String, required: true },
  // "x,y" keys of cells holding items; they stay open so nothing ends up on a closed cell.
  lockedCells: { type: Object, default: () => new Set() },
  disabled: { type: Boolean, default: false },
})

const emit = defineEmits(['update:modelValue'])

const cells = computed(() => {
  const list = []
  for (let y = 0; y < props.height; y += 1) {
    for (let x = 0; x < props.width; x += 1) {
      const index = y * props.width + x
      list.push({ index, x, y, open: props.modelValue[index] === '1', locked: props.lockedCells.has(`${x},${y}`) })
    }
  }
  return list
})

const openCount = computed(() => countOpen(props.modelValue))
const total = computed(() => props.width * props.height)

let paintValue = null
let draft = ''
const painting = ref(false)

function setCell(index) {
  if (index == null || draft[index] === paintValue) return
  const x = index % props.width
  const y = Math.floor(index / props.width)
  if (paintValue === '0' && props.lockedCells.has(`${x},${y}`)) return
  draft = draft.slice(0, index) + paintValue + draft.slice(index + 1)
  emit('update:modelValue', draft)
}

function cellIndexAt(clientX, clientY) {
  const target = document.elementFromPoint(clientX, clientY)?.closest?.('[data-shape-cell]')
  return target ? Number(target.dataset.shapeCell) : null
}

function onPointerDown(event) {
  if (props.disabled || event.button > 0) return
  const index = cellIndexAt(event.clientX, event.clientY)
  if (index == null) return
  event.preventDefault()
  event.currentTarget.setPointerCapture?.(event.pointerId)
  draft = props.modelValue
  paintValue = draft[index] === '1' ? '0' : '1'
  painting.value = true
  setCell(index)
}

function onPointerMove(event) {
  if (!painting.value) return
  setCell(cellIndexAt(event.clientX, event.clientY))
}

function stopPainting() {
  painting.value = false
  paintValue = null
}

function fillAll() {
  emit('update:modelValue', fullMask(props.width, props.height))
}

function clearAll() {
  emit('update:modelValue', cells.value.map(cell => (cell.locked ? '1' : '0')).join(''))
}

function invert() {
  emit('update:modelValue', cells.value.map(cell => (cell.locked || !cell.open ? '1' : '0')).join(''))
}
</script>

<template>
  <div class="shape-editor" :class="{ 'is-disabled': disabled }">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <p class="text-[13px] text-dim">
        <span class="num text-[15px] font-semibold text-ink">{{ openCount }}</span> of {{ total }} cells open
      </p>
      <div class="flex gap-1.5">
        <button type="button" class="shape-editor__tool session-own" :disabled="disabled" @click="fillAll">Open all</button>
        <button type="button" class="shape-editor__tool session-own" :disabled="disabled" @click="invert">Invert</button>
        <button type="button" class="shape-editor__tool session-own" :disabled="disabled" @click="clearAll">Clear</button>
      </div>
    </div>

    <div
      class="shape-editor__grid mt-3"
      :style="{ '--cols': width, '--rows': height }"
      role="grid"
      aria-label="Backpack shape. Press and drag to open or close cells."
      @pointerdown="onPointerDown"
      @pointermove="onPointerMove"
      @pointerup="stopPainting"
      @pointercancel="stopPainting"
      @lostpointercapture="stopPainting"
    >
      <div
        v-for="cell in cells"
        :key="cell.index"
        :data-shape-cell="cell.index"
        class="shape-editor__cell"
        :class="{ 'is-open': cell.open, 'is-locked': cell.locked }"
        :title="cell.locked ? 'An item sits here' : ''"
      ></div>
    </div>

    <p v-if="openCount === 0" class="mt-2 text-[12px] font-semibold text-danger">Open at least one cell.</p>
    <p v-else-if="lockedCells.size" class="mt-2 text-[12px] text-faint">Dotted cells hold items and stay open.</p>
  </div>
</template>

<style scoped>
.shape-editor__grid {
  display: grid;
  grid-template-columns: repeat(var(--cols), minmax(0, 1fr));
  gap: 3px;
  width: min(100%, calc(var(--cols) * 2.6rem));
  padding: 6px;
  border-radius: 0.6rem;
  background: var(--color-n0);
  touch-action: none;
  user-select: none;
  cursor: crosshair;
}

.is-disabled .shape-editor__grid {
  cursor: not-allowed;
  opacity: 0.6;
}

.shape-editor__cell {
  aspect-ratio: 1;
  border-radius: 3px;
  box-shadow: inset 0 0 0 1px rgb(74 90 150 / 0.28);
  transition: background-color 90ms ease, box-shadow 90ms ease;
}

.shape-editor__cell.is-open {
  background: color-mix(in oklab, var(--color-primary) 34%, var(--color-n3));
  box-shadow: inset 0 0 0 1px color-mix(in oklab, var(--color-primary-ink) 45%, transparent);
}

.shape-editor__cell.is-locked {
  background-image: radial-gradient(circle, var(--color-primary-ink) 0 18%, transparent 22%);
}

@media (hover: hover) {
  .shape-editor__grid:not(:active) .shape-editor__cell:hover {
    box-shadow: inset 0 0 0 1px var(--color-sky);
  }
}

.shape-editor__tool {
  border-radius: 0.5rem;
  background: var(--color-n4);
  padding: 0.3rem 0.65rem;
  font-size: 12px;
  font-weight: 600;
  color: var(--color-ink);
  cursor: pointer;
  transition: background-color 150ms ease;
}

.shape-editor__tool:hover:not(:disabled) {
  background: var(--color-n5);
}

.shape-editor__tool:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}
</style>
