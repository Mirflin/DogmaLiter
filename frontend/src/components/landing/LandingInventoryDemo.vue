<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import interact from 'interactjs'
import { ITEM_ICONS } from './itemIcons'

// A small working copy of the session inventory: same library (interact.js) and the same feel —
// the grabbed item lifts and follows the pointer, cells under it light up, gear goes into slots,
// and the character's stats follow what is equipped.

const COLS = 8
const ROWS = 5

const RARITY = {
  common: { label: 'Common', color: 'var(--color-r-common)' },
  uncommon: { label: 'Uncommon', color: 'var(--color-r-uncommon)' },
  rare: { label: 'Rare', color: 'var(--color-r-rare)' },
  epic: { label: 'Epic', color: 'var(--color-r-epic)' },
  masterwork: { label: 'Masterwork', color: 'var(--color-r-masterwork)' },
  legendary: { label: 'Legendary', color: 'var(--color-r-legendary)' },
  unique: { label: 'Unique', color: 'var(--color-r-unique)' },
}

const SLOTS = [
  { id: 'main_hand', label: 'Main hand' },
  { id: 'off_hand', label: 'Off hand' },
  { id: 'chest', label: 'Chest' },
  { id: 'ring', label: 'Ring' },
  { id: 'amulet', label: 'Amulet' },
]
const SLOT_LABEL = Object.fromEntries(SLOTS.map((slot) => [slot.id, slot.label]))

const BASE = { STR: 12, DEX: 18, CON: 14, INT: 10, WIS: 15, CHA: 9 }

const items = ref([
  { id: 'bow', name: 'Longbow of the Gull', short: 'Longbow', rarity: 'rare', w: 1, h: 3, durability: 80, mods: { DEX: 1 }, req: { DEX: 13 }, slot: 'main_hand', at: { x: 0, y: 0 } },
  { id: 'dagger', name: 'Moon Dagger', short: 'Dagger', rarity: 'rare', w: 1, h: 2, durability: 92, mods: { DEX: 1 }, slot: 'main_hand', at: { x: 1, y: 0 } },
  { id: 'potion', name: 'Potion of Healing', short: 'PH', rarity: 'common', w: 1, h: 1, qty: 3, at: { x: 2, y: 0 } },
  { id: 'arrows', name: 'Bundle of Arrows', short: 'Arrows', rarity: 'common', w: 2, h: 1, qty: 46, at: { x: 3, y: 0 } },
  { id: 'ring', name: 'Ember Ring', short: 'ER', rarity: 'epic', w: 1, h: 1, mods: { CHA: 2 }, enchant: 2, slot: 'ring', at: { x: 2, y: 1 } },
  { id: 'amulet', name: 'Dragonbone Amulet', short: 'DA', rarity: 'legendary', w: 1, h: 1, mods: { WIS: 2, CON: 1 }, req: { WIS: 14 }, slot: 'amulet', at: { x: 3, y: 1 } },
  { id: 'shield', name: "Warden's Greatshield", short: 'Greatshield', rarity: 'unique', w: 2, h: 2, mods: { CON: 2 }, req: { STR: 16 }, slot: 'off_hand', at: { x: 6, y: 0 } },
  { id: 'rope', name: 'Rope, 50 ft', short: 'Rope', rarity: 'common', w: 2, h: 1, at: { x: 0, y: 3 } },
  { id: 'cloak', name: 'Cloak of the Tide', short: 'Cloak', rarity: 'masterwork', w: 2, h: 2, mods: { WIS: 1 }, at: { x: 6, y: 3 } },
  { id: 'armor', name: 'Studded Leather', short: 'Leather', rarity: 'uncommon', w: 2, h: 3, durability: 55, mods: { CON: 1 }, slot: 'chest', at: { slot: 'chest' } },
])

const byId = computed(() => Object.fromEntries(items.value.map((item) => [item.id, item])))
const gridItems = computed(() => items.value.filter((item) => item.at.slot === undefined))
const equipped = computed(() => Object.fromEntries(items.value.filter((item) => item.at.slot).map((item) => [item.at.slot, item])))
const used = computed(() => gridItems.value.reduce((sum, item) => sum + item.w * item.h, 0))

const stats = computed(() => Object.entries(BASE).map(([key, base]) => {
  const bonus = Object.values(equipped.value).reduce((sum, item) => sum + (item.mods?.[key] || 0), 0)
  return { key, total: base + bonus, bonus }
}))

function unmet(item) {
  return Object.entries(item.req || {}).filter(([key, min]) => BASE[key] < min)
}

const selectedId = ref('bow')
const selected = computed(() => byId.value[selectedId.value])
const message = ref('')

// ---- Placement rules (the same checks the session board makes) ----

function fits(item, x, y) {
  if (x < 0 || y < 0 || x + item.w > COLS || y + item.h > ROWS) return false
  return gridItems.value.every((other) => other.id === item.id
    || x + item.w <= other.at.x || other.at.x + other.w <= x
    || y + item.h <= other.at.y || other.at.y + other.h <= y)
}

function canEquip(item, slot) {
  if (item.slot !== slot) return false
  const owner = equipped.value[slot]
  return !owner || owner.id === item.id
}

function firstFreeCell(item) {
  for (let y = 0; y <= ROWS - item.h; y += 1) {
    for (let x = 0; x <= COLS - item.w; x += 1) {
      if (fits(item, x, y)) return { x, y }
    }
  }
  return null
}

function equip(item, slot) {
  if (!canEquip(item, slot)) {
    message.value = item.slot ? `${SLOT_LABEL[slot]} is taken, or ${item.name} doesn’t go there.` : `${item.name} can’t be equipped.`
    return
  }
  const missing = unmet(item)
  if (missing.length) {
    message.value = `Kaela needs ${missing.map(([key, min]) => `${key} ${min}`).join(', ')} to equip ${item.name}.`
    return
  }
  item.at = { slot }
  message.value = ''
}

function unequip(item) {
  const cell = firstFreeCell(item)
  if (!cell) {
    message.value = 'No room in the backpack for that.'
    return
  }
  item.at = cell
  message.value = ''
}

// ---- Dragging with interact.js ----

const root = ref(null)
const gridEl = ref(null)
const dragging = ref(null)
const hoverCell = ref(null)
const hoverSlot = ref(null)
let interactable = null
let grabPx = { x: 0, y: 0 }
// The section panel's backdrop-filter makes it the containing block for position: fixed,
// so fixed coordinates are measured from its corner rather than the viewport's.
let origin = { x: 0, y: 0 }

const footprint = computed(() => {
  const d = dragging.value
  if (!d || !hoverCell.value) return null
  const item = byId.value[d.id]
  const x = hoverCell.value.x - d.grab.x
  const y = hoverCell.value.y - d.grab.y
  return { x, y, w: item.w, h: item.h, valid: fits(item, x, y), resin: RARITY[item.rarity].color }
})

function cellState(x, y) {
  const f = footprint.value
  if (!f || x < f.x || x >= f.x + f.w || y < f.y || y >= f.y + f.h) return ''
  return f.valid ? 'is-active' : 'is-blocked'
}

function slotState(slot) {
  const d = dragging.value
  if (!d) return ''
  const item = byId.value[d.id]
  if (item.slot !== slot) return ''
  const ok = canEquip(item, slot) && !unmet(item).length
  if (hoverSlot.value === slot) return ok ? 'is-active' : 'is-blocked'
  return ok ? 'is-candidate' : ''
}

function cellSize() {
  const rect = gridEl.value?.querySelector('.demo__cell')?.getBoundingClientRect()
  return { w: rect?.width || 0, h: rect?.height || 0 }
}

function setDragVisual(element, active, geometry) {
  if (active) {
    Object.assign(element.style, { position: 'fixed', left: '0px', top: '0px', margin: '0', transform: 'none', transition: 'none' })
    const corner = element.getBoundingClientRect()
    origin = { x: corner.left, y: corner.top }
    Object.assign(element.style, {
      position: 'fixed',
      width: `${geometry.width}px`,
      height: `${geometry.height}px`,
      left: `${geometry.left - origin.x}px`,
      top: `${geometry.top - origin.y}px`,
      margin: '0',
      zIndex: '9999',
      pointerEvents: 'none',
      transition: 'transform 130ms ease, filter 130ms ease',
      transform: 'scale(1.06) rotate(1.4deg)',
      filter: 'drop-shadow(0 18px 28px rgba(0, 0, 0, 0.6))',
      cursor: 'grabbing',
    })
    return
  }
  for (const prop of ['position', 'width', 'height', 'left', 'top', 'margin', 'zIndex', 'pointerEvents', 'transition', 'transform', 'filter', 'cursor']) {
    element.style[prop] = ''
  }
}

function onDragStart(event) {
  const element = event.target
  const item = byId.value[element.dataset.itemId]
  if (!item) return
  selectedId.value = item.id
  message.value = ''

  const cell = cellSize()
  const rect = element.getBoundingClientRect()
  // Which cell of the item was grabbed; worn gear is picked up by its middle.
  const grab = item.at.slot
    ? { x: Math.floor((item.w - 1) / 2), y: Math.floor((item.h - 1) / 2) }
    : {
        x: Math.min(item.w - 1, Math.max(0, Math.floor((event.clientX - rect.left) / (cell.w || 1)))),
        y: Math.min(item.h - 1, Math.max(0, Math.floor((event.clientY - rect.top) / (cell.h || 1)))),
      }
  dragging.value = { id: item.id, grab }
  grabPx = { x: grab.x * cell.w + cell.w / 2, y: grab.y * cell.h + cell.h / 2 }

  setDragVisual(element, true, {
    width: item.w * cell.w,
    height: item.h * cell.h,
    left: event.clientX - grabPx.x,
    top: event.clientY - grabPx.y,
  })
  updateHover(event.clientX, event.clientY)
}

function onDragMove(event) {
  event.target.style.left = `${event.clientX - grabPx.x - origin.x}px`
  event.target.style.top = `${event.clientY - grabPx.y - origin.y}px`
  updateHover(event.clientX, event.clientY)
}

function onDragEnd(event) {
  const d = dragging.value
  setDragVisual(event.target, false)
  const slot = hoverSlot.value
  const cell = hoverCell.value
  dragging.value = null
  hoverCell.value = null
  hoverSlot.value = null
  if (!d) return

  const item = byId.value[d.id]
  if (slot) {
    equip(item, slot)
  } else if (cell) {
    const x = cell.x - d.grab.x
    const y = cell.y - d.grab.y
    if (fits(item, x, y)) item.at = { x, y }
  }
}

function updateHover(clientX, clientY) {
  const element = document.elementFromPoint(clientX, clientY)
  const slotElement = element?.closest('[data-slot]')
  if (slotElement && root.value.contains(slotElement)) {
    hoverSlot.value = slotElement.dataset.slot
    hoverCell.value = null
    return
  }
  hoverSlot.value = null
  const firstCell = element?.closest('.demo__grid') ? gridEl.value?.querySelector('.demo__cell') : null
  if (firstCell) {
    const rect = firstCell.getBoundingClientRect()
    const gap = parseFloat(getComputedStyle(gridEl.value).columnGap) || 0
    hoverCell.value = {
      x: Math.floor((clientX - rect.left) / (rect.width + gap)),
      y: Math.floor((clientY - rect.top) / (rect.height + gap)),
    }
    return
  }
  hoverCell.value = null
}

// ---- Keyboard: arrows move, Enter equips or takes off ----

const STEPS = { ArrowLeft: [-1, 0], ArrowRight: [1, 0], ArrowUp: [0, -1], ArrowDown: [0, 1] }

function onKeydown(event, item) {
  if (event.key === 'Enter') {
    event.preventDefault()
    if (item.at.slot) unequip(item)
    else if (item.slot) equip(item, item.slot)
    return
  }
  const step = STEPS[event.key]
  if (!step || item.at.slot) return
  event.preventDefault()
  const x = item.at.x + step[0]
  const y = item.at.y + step[1]
  if (fits(item, x, y)) item.at = { x, y }
}

// ---- Entrance: items drop into the tray when the board first scrolls into view ----

const armed = ref(false)
const dropped = ref(false)
let observer = null

onMounted(() => {
  interactable = interact('.demo__item', { context: root.value }).draggable({
    inertia: false,
    autoScroll: true,
    listeners: { start: onDragStart, move: onDragMove, end: onDragEnd },
  })

  if (window.matchMedia('(prefers-reduced-motion: reduce)').matches || !('IntersectionObserver' in window)) return
  armed.value = true
  observer = new IntersectionObserver((entries) => {
    if (entries.some((entry) => entry.isIntersecting)) {
      dropped.value = true
      observer.disconnect()
    }
  }, { threshold: 0.35 })
  observer.observe(root.value)
})

onBeforeUnmount(() => {
  observer?.disconnect()
  interactable?.unset()
})

function tileStyle(item, index) {
  return {
    gridColumn: `${item.at.x + 1} / span ${item.w}`,
    gridRow: `${item.at.y + 1} / span ${item.h}`,
    '--resin': RARITY[item.rarity].color,
    '--delay': `${index * 85}ms`,
  }
}

function describe(item) {
  const where = item.at.slot ? `worn in ${SLOT_LABEL[item.at.slot]}` : 'in the backpack'
  const lock = unmet(item).length ? ', requirements not met' : ''
  return `${item.name}, ${RARITY[item.rarity].label}, ${where}${lock}. Arrow keys move it${item.slot ? ', Enter equips or takes it off' : ''}.`
}
</script>

<template>
  <div ref="root" class="demo" :class="{ 'demo--armed': armed, 'demo--dropped': dropped }">
    <div class="demo__board">
      <div class="demo__meta">
        <span class="text-ink font-semibold">Kaela Thornwood</span>
        <span class="demo__tag" title="A simplified demo; the in-game inventory looks different">Demo</span>
        <ul class="demo__stats" aria-label="Attributes">
          <li v-for="stat in stats" :key="stat.key" :class="{ 'has-bonus': stat.bonus }">
            <span>{{ stat.key }}</span>
            <strong class="num">{{ stat.total }}</strong>
            <em v-if="stat.bonus" class="num">+{{ stat.bonus }}</em>
          </li>
        </ul>
      </div>

      <div class="demo__body">
        <div class="demo__slots" aria-label="Equipment">
          <div
            v-for="slot in SLOTS"
            :key="slot.id"
            class="demo__slot"
            :class="slotState(slot.id)"
            :data-slot="slot.id"
          >
            <span class="demo__slot-label">{{ slot.label }}</span>
            <button
              v-if="equipped[slot.id]"
              type="button"
              class="demo__item demo__item--worn"
              :class="{ 'is-selected': equipped[slot.id].id === selectedId }"
              :data-item-id="equipped[slot.id].id"
              :style="{ '--resin': RARITY[equipped[slot.id].rarity].color }"
              :aria-label="describe(equipped[slot.id])"
              @pointerenter="!dragging && (selectedId = equipped[slot.id].id)"
              @focus="selectedId = equipped[slot.id].id"
              @keydown="onKeydown($event, equipped[slot.id])"
            >
              <svg class="demo__icon" viewBox="0 0 512 512" aria-hidden="true"><path :d="ITEM_ICONS[equipped[slot.id].id]" /></svg>
            </button>
          </div>
        </div>

        <div>
          <div ref="gridEl" class="demo__grid" :style="{ '--cols': COLS, '--rows': ROWS }" role="list" aria-label="Backpack">
            <span
              v-for="n in COLS * ROWS"
              :key="`c${n}`"
              class="demo__cell"
              :class="cellState((n - 1) % COLS, Math.floor((n - 1) / COLS))"
              :style="{ gridColumn: ((n - 1) % COLS) + 1, gridRow: Math.floor((n - 1) / COLS) + 1, '--resin': footprint?.resin }"
              aria-hidden="true"
            />
            <button
              v-for="(item, index) in gridItems"
              :key="item.id"
              type="button"
              role="listitem"
              class="demo__item"
              :class="{
                'is-selected': item.id === selectedId,
                'is-locked': unmet(item).length,
                'is-small': item.w * item.h === 1,
                'is-narrow': item.w === 1,
              }"
              :data-item-id="item.id"
              :style="tileStyle(item, index)"
              :aria-label="describe(item)"
              @pointerenter="!dragging && (selectedId = item.id)"
              @focus="selectedId = item.id"
              @keydown="onKeydown($event, item)"
            >
              <svg class="demo__icon" viewBox="0 0 512 512" aria-hidden="true"><path :d="ITEM_ICONS[item.id]" /></svg>
              <span v-if="item.w * item.h > 1" class="demo__name">{{ item.short || item.name }}</span>
              <span v-if="unmet(item).length" class="demo__lock">Needs {{ unmet(item).map(([k, v]) => `${k} ${v}`).join(', ') }}</span>
              <span v-if="item.qty" class="demo__qty num">×{{ item.qty }}</span>
              <span v-if="item.durability" class="demo__dur" aria-hidden="true"><i :style="{ width: `${item.durability}%` }" /></span>
            </button>
          </div>
          <p class="demo__hint" aria-live="polite">
            <span v-if="message" class="text-danger">{{ message }}</span>
            <template v-else>Drag items around the backpack or onto Kaela’s gear. <span class="num text-dim">{{ used }} / {{ COLS * ROWS }}</span> cells used.</template>
          </p>
        </div>
      </div>
    </div>

    <aside class="demo__inspector" aria-live="polite">
      <p class="text-[13px] font-semibold" :style="{ color: RARITY[selected.rarity].color }">{{ RARITY[selected.rarity].label }}</p>
      <h3 class="mt-1 text-[22px] leading-tight font-semibold text-ink">{{ selected.name }}</h3>
      <dl class="demo__facts">
        <div><dt>Size</dt><dd class="num">{{ selected.w }} × {{ selected.h }}</dd></div>
        <div v-if="selected.slot"><dt>Slot</dt><dd>{{ SLOT_LABEL[selected.slot] }}<template v-if="selected.at.slot"> · worn</template></dd></div>
        <div v-if="selected.qty"><dt>Stack</dt><dd class="num">{{ selected.qty }}</dd></div>
        <div v-if="selected.durability"><dt>Durability</dt><dd class="num">{{ selected.durability }} / 100</dd></div>
        <div v-if="selected.enchant"><dt>Enchantment</dt><dd class="num">+{{ selected.enchant }}</dd></div>
        <div v-if="selected.mods"><dt>Modifiers</dt><dd class="text-orchid">{{ Object.entries(selected.mods).map(([k, v]) => `+${v} ${k}`).join(', ') }}</dd></div>
        <div v-if="selected.req">
          <dt>Requires</dt>
          <dd :class="unmet(selected).length ? 'text-danger' : ''">
            {{ Object.entries(selected.req).map(([k, v]) => `${k} ${v}`).join(', ') }}<template v-if="unmet(selected).length"> · Kaela has {{ unmet(selected).map(([k]) => BASE[k]).join(', ') }}</template>
          </dd>
        </div>
      </dl>
    </aside>
  </div>
</template>

<style scoped>
.demo {
  --cell: min(66px, calc((100vw - 5rem - 7 * 4px) / 8));
  display: grid;
  grid-template-columns: max-content minmax(14rem, 1fr);
  gap: 1.25rem;
  align-items: stretch;
}

.demo__board {
  border-radius: 1.25rem;
  background: var(--color-n0);
  box-shadow: var(--shadow-well);
  padding: 1rem;
  min-width: 0;
}

.demo__meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.5rem 1rem;
  padding: 0.1rem 0.25rem 0.9rem;
  font-size: 14px;
}

.demo__tag {
  padding: 0.1rem 0.5rem;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 600;
  color: var(--color-dim);
  background: var(--color-n3);
}

.demo__stats {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem 0.9rem;
  margin: 0 0 0 auto;
  padding: 0;
  list-style: none;
}

.demo__stats li {
  display: inline-flex;
  align-items: baseline;
  gap: 0.3rem;
  color: var(--color-faint);
  font-size: 12px;
  font-weight: 600;
}

.demo__stats strong {
  font-size: 17px;
  color: var(--color-ink);
  transition: color 0.3s ease;
}

.demo__stats em {
  font-style: normal;
  font-size: 13px;
  color: var(--color-orchid);
}

.demo__stats li.has-bonus strong {
  color: color-mix(in oklab, var(--color-orchid) 45%, white);
}

.demo__body {
  display: flex;
  gap: 0.85rem;
}

/* Equipment column */
.demo__slots {
  display: grid;
  grid-template-rows: repeat(5, 1fr);
  gap: 4px;
  width: calc(var(--cell) * 1.75);
}

.demo__slot {
  position: relative;
  display: flex;
  flex-direction: column;
  min-height: 0;
  padding: 4px;
  border-radius: 9px;
  background: var(--color-n2);
  transition: background-color 0.15s ease, box-shadow 0.15s ease;
}

.demo__slot-label {
  position: absolute;
  inset: auto 0 6px;
  text-align: center;
  font-size: 11px;
  font-weight: 600;
  color: var(--color-faint);
  pointer-events: none;
}

.demo__slot.is-candidate {
  box-shadow: inset 0 0 0 1px rgb(164 174 255 / 0.45);
}

.demo__slot.is-active {
  background: rgb(98 214 157 / 0.14);
  box-shadow: inset 0 0 0 1.5px var(--color-success);
}

.demo__slot.is-blocked {
  background: rgb(242 122 133 / 0.14);
  box-shadow: inset 0 0 0 1.5px var(--color-danger);
}

/* Grid */
.demo__grid {
  --gap: 4px;
  display: grid;
  grid-template-columns: repeat(var(--cols), var(--cell));
  grid-template-rows: repeat(var(--rows), var(--cell));
  gap: var(--gap);
  width: max-content;
}

.demo__cell {
  border-radius: 7px;
  background: var(--color-n2);
  transition: background-color 0.12s ease, box-shadow 0.12s ease;
}

.demo__cell.is-active {
  background: color-mix(in oklab, var(--resin) 22%, var(--color-n2));
  box-shadow: inset 0 0 0 1px color-mix(in oklab, var(--resin) 70%, transparent);
}

.demo__cell.is-blocked {
  background: rgb(242 122 133 / 0.18);
  box-shadow: inset 0 0 0 1px rgb(242 122 133 / 0.7);
}

/* Items */
.demo__item {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 2px;
  padding: 7px 8px;
  border: 0;
  border-radius: 9px;
  font: inherit;
  text-align: left;
  color: color-mix(in oklab, var(--resin) 35%, white);
  background: color-mix(in oklab, var(--resin) 24%, var(--color-n2));
  box-shadow: inset 0 0 0 1px color-mix(in oklab, var(--resin) 62%, transparent), inset 0 1px 0 rgb(255 255 255 / 0.08);
  cursor: grab;
  overflow: hidden;
  touch-action: none;
  user-select: none;
  transition: background-color 0.2s ease, box-shadow 0.2s ease;
}

.demo__item--worn {
  flex: 1;
  justify-content: center;
  align-items: center;
  text-align: center;
  padding-bottom: 1.3rem;
}

.demo__item:hover,
.demo__item.is-selected {
  background: color-mix(in oklab, var(--resin) 34%, var(--color-n2));
  box-shadow: inset 0 0 0 1.5px var(--resin), inset 0 1px 0 rgb(255 255 255 / 0.12);
}

.demo__item:focus-visible {
  outline: 2px solid var(--color-primary-ink);
  outline-offset: 2px;
}

.demo__item.is-locked {
  filter: saturate(0.2) brightness(0.72);
}

/* The item's picture, tinted by its resin; text sits above it. */
.demo__icon {
  position: absolute;
  left: 50%;
  top: 50%;
  width: min(74%, calc(var(--cell) * 1.25));
  aspect-ratio: 1;
  transform: translate(-50%, -50%);
  fill: currentColor;
  opacity: 0.42;
  pointer-events: none;
}

.is-small .demo__icon {
  width: 72%;
  opacity: 0.9;
}

.demo__item--worn .demo__icon {
  top: 42%;
  width: min(62%, calc(var(--cell) * 0.85));
  opacity: 0.85;
}

.demo__name,
.demo__lock,
.demo__qty,
.demo__dur {
  position: relative;
}

.demo__name {
  font-size: 13px;
  font-weight: 600;
  line-height: 1.2;
}

.is-small .demo__qty {
  margin: auto 0 0 auto;
  font-size: 14px;
  opacity: 1;
  text-shadow: 0 1px 2px rgb(0 0 0 / 0.6);
}

.demo__lock {
  font-size: 12px;
  font-weight: 600;
  color: var(--color-ink-soft);
}

.demo__qty {
  margin-top: auto;
  font-size: 13px;
  opacity: 0.85;
}

.demo__dur {
  margin-top: auto;
  width: 100%;
  height: 3px;
  border-radius: 2px;
  background: rgb(255 255 255 / 0.14);
}

.demo__dur i {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: currentColor;
}

.demo__hint {
  margin: 0.8rem 0.25rem 0;
  max-width: calc(var(--cell) * 8 + 28px);
  min-height: 2.9em;
  font-size: 13px;
  line-height: 1.45;
  color: var(--color-faint);
}

/* Hidden only until the board scrolls into view; afterwards the animation leaves no fill behind. */
.demo--armed:not(.demo--dropped) .demo__item {
  opacity: 0;
  transform: translateY(-22px) rotate(-7deg) scale(1.05);
}

.demo--armed.demo--dropped .demo__item {
  animation: settle 0.62s cubic-bezier(0.16, 1, 0.3, 1) var(--delay, 0ms) backwards;
}

@keyframes settle {
  0% { opacity: 0; transform: translateY(-22px) rotate(-7deg) scale(1.05); }
  45% { opacity: 1; }
  78% { transform: translateY(0) rotate(1.2deg) scale(1); }
  100% { opacity: 1; transform: none; }
}

/* Inspector */
.demo__inspector {
  padding: 0.5rem 0.25rem;
}

.demo__facts {
  margin: 1rem 0 0;
  display: grid;
  gap: 0.55rem;
  font-size: 14px;
}

.demo__facts div {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
}

.demo__facts dt {
  color: var(--color-faint);
}

.demo__facts dd {
  margin: 0;
  color: var(--color-ink);
  text-align: right;
}

.demo__facts .num {
  font-size: 16px;
}

@media (max-width: 1100px) {
  .demo {
    grid-template-columns: minmax(0, 1fr);
  }

  .demo__board {
    width: max-content;
    max-width: 100%;
  }
}

@media (max-width: 720px) {
  .demo {
    /* Viewport minus the section gutters, the panel padding and the board padding. */
    --cell: calc((100vw - 5.6rem - 7 * 3px) / 8);
  }

  .demo__board {
    padding: 0.75rem;
  }

  .demo__body {
    flex-direction: column;
  }

  .demo__slots {
    grid-template-columns: repeat(5, 1fr);
    grid-template-rows: none;
    width: 100%;
    height: calc(var(--cell) * 1.6);
  }

  .demo__slot-label {
    font-size: 10px;
  }

  .demo__grid {
    --gap: 3px;
  }

  .demo__item {
    padding: 4px 5px;
    border-radius: 7px;
  }

  .demo__name {
    font-size: 10.5px;
  }

  .is-small .demo__qty {
    font-size: 11px;
  }

  /* One cell wide is too narrow for a name on a phone; the picture carries it. */
  .is-narrow .demo__name {
    display: none;
  }

  .is-narrow .demo__icon {
    width: 80%;
    opacity: 0.8;
  }

  .demo__stats {
    margin-left: 0;
  }
}
</style>
