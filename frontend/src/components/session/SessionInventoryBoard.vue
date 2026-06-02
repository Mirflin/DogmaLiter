<script setup>
import interact from 'interactjs'
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import SessionInventoryDraggableItem from './SessionInventoryDraggableItem.vue'

const props = defineProps({
  characterName: {
    type: String,
    default: '',
  },
  inventoryItems: {
    type: Array,
    default: () => [],
  },
  equipment: {
    type: Array,
    default: () => [],
  },
  currencyCards: {
    type: Array,
    default: () => [],
  },
  inventoryWidth: {
    type: Number,
    default: 10,
  },
  inventoryHeight: {
    type: Number,
    default: 6,
  },
})

const SLOT_DEFAULT = 'border-[rgba(126,200,227,0.34)] bg-[linear-gradient(180deg,rgba(30,49,96,0.96),rgba(15,25,49,0.98))] shadow-[inset_0_0_0_1px_rgba(126,200,227,0.06)]'
const SLOT_ACTIVE = 'border-[rgba(74,222,128,0.85)] bg-[linear-gradient(180deg,rgba(21,128,61,0.5),rgba(13,70,38,0.72))] shadow-[inset_0_0_0_1px_rgba(74,222,128,0.2),0_0_0_1px_rgba(74,222,128,0.12)]'
const SLOT_BLOCKED = 'border-[rgba(248,113,113,0.85)] bg-[linear-gradient(180deg,rgba(88,27,42,0.88),rgba(45,12,23,0.94))] shadow-[inset_0_0_0_1px_rgba(248,113,113,0.2)]'
const CELL_DEFAULT = 'border-[rgba(126,200,227,0.18)] bg-[linear-gradient(180deg,rgba(34,55,102,0.58),rgba(15,28,57,0.76))]'
const CELL_ACTIVE = 'border-[rgba(74,222,128,0.85)] bg-[linear-gradient(180deg,rgba(21,128,61,0.55),rgba(13,70,38,0.72))] shadow-[inset_0_0_0_1px_rgba(74,222,128,0.22)]'
const CELL_BLOCKED = 'border-[rgba(248,113,113,0.85)] bg-[linear-gradient(180deg,rgba(127,29,29,0.6),rgba(60,12,18,0.8))] shadow-[inset_0_0_0_1px_rgba(248,113,113,0.22)]'

const boardRef = ref(null)
const gridRef = ref(null)
const placements = ref({})

const dragging = ref(false)
const dragEntryId = ref('')
const grabOffset = ref({ dx: 0, dy: 0 })
const hoverCell = ref(null)
const hoverSlot = ref(null)

let interactable = null
let dragTx = 0
let dragTy = 0

const safeInventoryWidth = computed(() => Math.max(1, Number(props.inventoryWidth) || 10))
const safeInventoryHeight = computed(() => Math.max(1, Number(props.inventoryHeight) || 6))

const allEntriesById = computed(() => {
  const entries = {}

  for (const entry of props.inventoryItems) {
    entries[entry.id] = entry
  }

  for (const slot of props.equipment) {
    const entry = slot.inventory_item
    if (entry?.id) {
      entries[entry.id] = entry
    }
  }

  return entries
})

const gridCells = computed(() => {
  const cells = []

  for (let y = 0; y < safeInventoryHeight.value; y += 1) {
    for (let x = 0; x < safeInventoryWidth.value; x += 1) {
      cells.push({ id: `cell-${x}-${y}`, x, y, kind: 'inventory' })
    }
  }

  return cells
})

const equipmentBySlot = computed(() => {
  const slots = {}

  for (const placement of Object.values(placements.value)) {
    if (placement.kind !== 'equipment') continue
    slots[placement.slot] = allEntriesById.value[placement.entryId]
  }

  return slots
})

const inventoryEntries = computed(() => Object.values(placements.value)
  .filter(placement => placement.kind === 'inventory' && allEntriesById.value[placement.entryId])
  .map((placement) => ({
    entry: allEntriesById.value[placement.entryId],
    placement,
  })))

const dragFootprint = computed(() => {
  if (!dragging.value || !hoverCell.value) return null

  const entry = allEntriesById.value[dragEntryId.value]
  if (!entry) return null

  const { width, height } = getEntrySize(entry)
  const x = hoverCell.value.x - grabOffset.value.dx
  const y = hoverCell.value.y - grabOffset.value.dy

  return {
    x,
    y,
    width,
    height,
    valid: canPlaceInInventory(dragEntryId.value, x, y),
  }
})

watch(
  () => [props.inventoryItems, props.equipment, safeInventoryWidth.value, safeInventoryHeight.value],
  () => syncPlacements(),
  { immediate: true, deep: true },
)

onMounted(() => {
  if (!boardRef.value) return

  interactable = interact('.session-inventory-item', { context: boardRef.value })
  interactable.draggable({
    inertia: false,
    autoScroll: true,
    listeners: {
      start: onDragStart,
      move: onDragMove,
      end: onDragEnd,
    },
  })
})

onBeforeUnmount(() => {
  if (interactable) {
    interactable.unset()
    interactable = null
  }
})

function setDragVisual(element, active) {
  element.style.transform = active ? element.style.transform : ''
  element.style.zIndex = active ? '1000' : ''
  element.style.pointerEvents = active ? 'none' : ''
  element.style.opacity = active ? '0.95' : ''
  element.style.cursor = active ? 'grabbing' : ''
  element.style.filter = active ? 'drop-shadow(0 12px 20px rgba(0, 0, 0, 0.55))' : ''
  if (element.parentElement) {
    element.parentElement.style.zIndex = active ? '1000' : ''
  }
}

function onDragStart(event) {
  const element = event.target
  const entryId = element.dataset.entryId
  const entry = allEntriesById.value[entryId]
  if (!entry) return

  const firstCell = gridRef.value?.querySelector('.session-inventory-grid__cell')
  let cellWidth = 0
  let cellHeight = 0
  if (firstCell) {
    const cellRect = firstCell.getBoundingClientRect()
    cellWidth = cellRect.width
    cellHeight = cellRect.height
  }

  const itemRect = element.getBoundingClientRect()
  const { width, height } = getEntrySize(entry)
  const grabDX = cellWidth > 0 ? Math.floor((event.clientX - itemRect.left) / cellWidth) : 0
  const grabDY = cellHeight > 0 ? Math.floor((event.clientY - itemRect.top) / cellHeight) : 0

  grabOffset.value = {
    dx: clampInt(grabDX, 0, width - 1),
    dy: clampInt(grabDY, 0, height - 1),
  }
  dragEntryId.value = entryId
  dragging.value = true
  dragTx = 0
  dragTy = 0

  setDragVisual(element, true)
  updateHoverFromPointer(event.clientX, event.clientY)
}

function onDragMove(event) {
  dragTx += event.dx
  dragTy += event.dy
  event.target.style.transform = `translate(${dragTx}px, ${dragTy}px)`
  updateHoverFromPointer(event.clientX, event.clientY)
}

function onDragEnd(event) {
  const element = event.target
  const entryId = dragEntryId.value
  const entry = allEntriesById.value[entryId]
  const slot = hoverSlot.value
  const cell = hoverCell.value
  const offset = grabOffset.value

  setDragVisual(element, false)

  dragging.value = false
  dragEntryId.value = ''
  hoverCell.value = null
  hoverSlot.value = null

  if (!entry) return

  if (slot) {
    placeIntoEquipment(entryId, slot)
    return
  }

  if (cell) {
    placeIntoInventory(entryId, cell.x - offset.dx, cell.y - offset.dy)
  }
}

function updateHoverFromPointer(clientX, clientY) {
  const element = document.elementFromPoint(clientX, clientY)
  if (!element) {
    hoverCell.value = null
    hoverSlot.value = null
    return
  }

  const slotElement = element.closest('[data-slot]')
  if (slotElement) {
    hoverSlot.value = slotElement.dataset.slot
    hoverCell.value = null
    return
  }

  const gridElement = element.closest('.session-inventory-grid')
  if (gridElement) {
    const firstCell = gridElement.querySelector('.session-inventory-grid__cell')
    if (firstCell) {
      const cellRect = firstCell.getBoundingClientRect()
      if (cellRect.width > 0 && cellRect.height > 0) {
        hoverCell.value = {
          x: Math.floor((clientX - cellRect.left) / cellRect.width),
          y: Math.floor((clientY - cellRect.top) / cellRect.height),
        }
        hoverSlot.value = null
        return
      }
    }
  }

  hoverCell.value = null
  hoverSlot.value = null
}

function syncPlacements() {
  const next = {}

  for (const entry of props.inventoryItems) {
    next[entry.id] = {
      entryId: entry.id,
      kind: 'inventory',
      x: normalizeGridCoordinate(entry.grid_x),
      y: normalizeGridCoordinate(entry.grid_y),
    }
  }

  for (const slot of props.equipment) {
    const entryId = slot.inventory_item?.id || slot.inventory_item_id
    if (!entryId) continue

    next[entryId] = {
      entryId,
      kind: 'equipment',
      slot: slot.slot,
    }
  }

  placements.value = next
}

function normalizeGridCoordinate(value) {
  const numericValue = Number(value)
  if (!Number.isFinite(numericValue)) return 0
  return Math.max(0, Math.trunc(numericValue))
}

function getEntrySize(entry) {
  const item = entry?.item ?? {}
  const width = Math.max(1, Number(item.grid_width) || 1)
  const height = Math.max(1, Number(item.grid_height) || 1)

  if (entry?.is_rotated) {
    return { width: height, height: width }
  }

  return { width, height }
}

function inventoryItemStyle(entry, placement) {
  const { width, height } = getEntrySize(entry)

  return {
    gridColumn: `${placement.x + 1} / span ${width}`,
    gridRow: `${placement.y + 1} / span ${height}`,
  }
}

function placeIntoInventory(entryId, x, y) {
  const entry = allEntriesById.value[entryId]
  if (!entry) return
  if (!canPlaceInInventory(entryId, x, y)) return

  placements.value = {
    ...placements.value,
    [entryId]: {
      entryId,
      kind: 'inventory',
      x,
      y,
    },
  }
}

function placeIntoEquipment(entryId, slot) {
  const entry = allEntriesById.value[entryId]
  if (!canEquip(entry, slot)) return

  placements.value = {
    ...placements.value,
    [entryId]: {
      entryId,
      kind: 'equipment',
      slot,
    },
  }
}

function canEquip(entry, slot) {
  if (!entry?.item?.equip_slot) return false

  const isRingSlot = slot === 'ring_1' || slot === 'ring_2'
  const isRingItem = entry.item.equip_slot === 'ring'
  if (!((isRingSlot && isRingItem) || entry.item.equip_slot === slot)) return false

  const slotOwner = Object.values(placements.value).find((placement) => placement.kind === 'equipment' && placement.slot === slot)
  if (slotOwner && slotOwner.entryId !== entry.id) return false

  return true
}

function canPlaceInInventory(entryId, x, y) {
  const entry = allEntriesById.value[entryId]
  if (!entry) return false

  const { width, height } = getEntrySize(entry)
  if (x < 0 || y < 0) return false
  if (x + width > safeInventoryWidth.value) return false
  if (y + height > safeInventoryHeight.value) return false

  for (const placement of Object.values(placements.value)) {
    if (placement.entryId === entryId || placement.kind !== 'inventory') continue

    const otherEntry = allEntriesById.value[placement.entryId]
    if (!otherEntry) continue

    const otherSize = getEntrySize(otherEntry)
    const overlaps = x < placement.x + otherSize.width
      && x + width > placement.x
      && y < placement.y + otherSize.height
      && y + height > placement.y

    if (overlaps) return false
  }

  return true
}

function cellHighlight(cell) {
  const footprint = dragFootprint.value
  if (!footprint) return ''

  const inside = cell.x >= footprint.x
    && cell.x < footprint.x + footprint.width
    && cell.y >= footprint.y
    && cell.y < footprint.y + footprint.height

  if (!inside) return ''
  return footprint.valid ? 'active' : 'blocked'
}

function slotHighlight(slot) {
  if (!dragging.value || hoverSlot.value !== slot) return ''
  const entry = allEntriesById.value[dragEntryId.value]
  return canEquip(entry, slot) ? 'active' : 'blocked'
}

function cellStateClasses(cell) {
  const state = cellHighlight(cell)
  if (state === 'active') return CELL_ACTIVE
  if (state === 'blocked') return CELL_BLOCKED
  return CELL_DEFAULT
}

function slotStateClasses(slot) {
  const state = slotHighlight(slot)
  if (state === 'active') return SLOT_ACTIVE
  if (state === 'blocked') return SLOT_BLOCKED
  return SLOT_DEFAULT
}

function clampInt(value, min, max) {
  return Math.min(max, Math.max(min, value))
}

function equipmentEntry(slot) {
  return equipmentBySlot.value[slot] ?? null
}
</script>

<template>
  <article
    ref="boardRef"
    class="relative mx-auto w-full max-w-full rounded-[clamp(0.22rem,0.5vw,0.34rem)] border border-[rgba(126,200,227,0.28)] bg-[radial-gradient(circle_at_top,rgba(233,69,96,0.12),transparent_26%),linear-gradient(180deg,rgba(16,24,44,0.98),rgba(9,15,29,1)),linear-gradient(135deg,rgba(126,200,227,0.05),transparent_42%)] p-[0.48rem] shadow-[inset_0_0_0_1px_rgba(126,200,227,0.06),0_18px_34px_rgba(0,0,0,0.28)] before:pointer-events-none before:absolute before:inset-[clamp(0.3rem,0.8vw,0.55rem)] before:rounded-[clamp(0.16rem,0.4vw,0.28rem)] before:border before:border-[rgba(126,200,227,0.14)] before:content-[''] md:p-[0.7rem] lg:w-fit lg:p-[clamp(0.55rem,1.4vw,0.95rem)] [--inv-cell:clamp(1.58rem,8.8vw,2.15rem)] min-[480px]:[--inv-cell:clamp(1.72rem,8vw,2.6rem)] md:[--inv-cell:clamp(1.9rem,6.2vw,3.85rem)] lg:[--inv-cell:clamp(1.9rem,5.9vw,4.95rem)] [--inv-gap:calc(var(--inv-cell)*0.28)] [--inv-weapon-col:calc(var(--inv-cell)*2.6)] [--inv-trinket-col:calc(var(--inv-cell)*0.98)] [--inv-center-col:calc(var(--inv-cell)*2.08)] [--inv-weapon-h:calc(var(--inv-cell)*3.76)] [--inv-glove-h:calc(var(--inv-cell)*1.88)] [--inv-head:calc(var(--inv-cell)*1.07)] [--inv-chest-h:calc(var(--inv-cell)*2.42)] [--inv-belt-h:calc(var(--inv-cell)*1.03)] [--inv-ring:calc(var(--inv-cell)*0.91)] [--inv-amulet:var(--inv-cell)]"
  >
    <div class="relative z-[1] flex flex-col gap-[0.4rem] p-[0.42rem] min-[480px]:p-[0.55rem] md:gap-[0.6rem] md:p-[0.78rem] lg:gap-[clamp(0.45rem,1vw,0.8rem)] lg:p-[clamp(0.6rem,1.5vw,1rem)]">
      <section class="shrink-0">
        <div class="mx-auto flex w-fit max-w-full items-stretch justify-center gap-[var(--inv-gap)]">
          <div class="flex h-auto w-[var(--inv-weapon-col)] flex-col gap-[var(--inv-gap)]">
            <div
              data-slot="main_hand"
              class="h-[var(--inv-weapon-h)] w-full rounded-[clamp(0.12rem,0.3vw,0.24rem)] border"
              :class="[dragging ? 'overflow-visible' : 'overflow-hidden', slotStateClasses('main_hand')]"
            >
              <SessionInventoryDraggableItem v-if="equipmentEntry('main_hand')" :entry="equipmentEntry('main_hand')" variant="equipment" />
              <div v-else class="flex h-full items-center justify-center p-[clamp(0.25rem,0.7vw,0.45rem)] text-center">
                <span class="text-[clamp(0.48rem,1vw,0.82rem)] font-bold uppercase tracking-[0.14em] text-[rgba(126,200,227,0.62)]">Main Hand</span>
              </div>
            </div>

            <div
              data-slot="gloves"
              class="h-[var(--inv-glove-h)] w-full rounded-[clamp(0.12rem,0.3vw,0.24rem)] border"
              :class="[dragging ? 'overflow-visible' : 'overflow-hidden', slotStateClasses('gloves')]"
            >
              <SessionInventoryDraggableItem v-if="equipmentEntry('gloves')" :entry="equipmentEntry('gloves')" variant="equipment" />
              <div v-else class="flex h-full items-center justify-center p-[clamp(0.25rem,0.7vw,0.45rem)] text-center">
                <span class="text-[clamp(0.48rem,1vw,0.82rem)] font-bold uppercase tracking-[0.14em] text-[rgba(126,200,227,0.62)]">Gloves</span>
              </div>
            </div>
          </div>

          <div class="flex h-auto w-[var(--inv-trinket-col)] flex-col items-center justify-end gap-[var(--inv-gap)]">
            <div
              data-slot="ring_1"
              class="aspect-square min-h-[var(--inv-ring)] max-w-[var(--inv-ring)] rounded-full border"
              :class="[dragging ? 'overflow-visible' : 'overflow-hidden', slotStateClasses('ring_1')]"
            >
              <SessionInventoryDraggableItem v-if="equipmentEntry('ring_1')" :entry="equipmentEntry('ring_1')" variant="equipment" />
              <div v-else class="flex h-full items-center justify-center p-[clamp(0.25rem,0.7vw,0.45rem)] text-center">
                <span class="text-[clamp(0.48rem,1vw,0.82rem)] font-bold uppercase tracking-[0.14em] text-[rgba(126,200,227,0.62)]">Ring</span>
              </div>
            </div>
          </div>

          <div class="flex h-auto w-[var(--inv-center-col)] flex-col gap-[var(--inv-gap)]">
            <div
              data-slot="head"
              class="h-[var(--inv-head)] w-full rounded-[clamp(0.12rem,0.3vw,0.24rem)] border"
              :class="[dragging ? 'overflow-visible' : 'overflow-hidden', slotStateClasses('head')]"
            >
              <SessionInventoryDraggableItem v-if="equipmentEntry('head')" :entry="equipmentEntry('head')" variant="equipment" />
              <div v-else class="flex h-full items-center justify-center p-[clamp(0.25rem,0.7vw,0.45rem)] text-center">
                <span class="text-[clamp(0.48rem,1vw,0.82rem)] font-bold uppercase tracking-[0.14em] text-[rgba(126,200,227,0.62)]">Head</span>
              </div>
            </div>

            <div
              data-slot="chest"
              class="h-[var(--inv-chest-h)] w-full rounded-[clamp(0.12rem,0.3vw,0.24rem)] border"
              :class="[dragging ? 'overflow-visible' : 'overflow-hidden', slotStateClasses('chest')]"
            >
              <SessionInventoryDraggableItem v-if="equipmentEntry('chest')" :entry="equipmentEntry('chest')" variant="equipment" />
              <div v-else class="flex h-full items-center justify-center p-[clamp(0.25rem,0.7vw,0.45rem)] text-center">
                <span class="text-[clamp(0.48rem,1vw,0.82rem)] font-bold uppercase tracking-[0.14em] text-[rgba(126,200,227,0.62)]">Chest</span>
              </div>
            </div>

            <div
              data-slot="belt"
              class="h-[var(--inv-belt-h)] w-full rounded-[clamp(0.12rem,0.3vw,0.24rem)] border"
              :class="[dragging ? 'overflow-visible' : 'overflow-hidden', slotStateClasses('belt')]"
            >
              <SessionInventoryDraggableItem v-if="equipmentEntry('belt')" :entry="equipmentEntry('belt')" variant="equipment" />
              <div v-else class="flex h-full items-center justify-center p-[clamp(0.25rem,0.7vw,0.45rem)] text-center">
                <span class="text-[clamp(0.48rem,1vw,0.82rem)] font-bold uppercase tracking-[0.14em] text-[rgba(126,200,227,0.62)]">Belt</span>
              </div>
            </div>
          </div>

          <div class="flex h-auto w-[var(--inv-trinket-col)] flex-col items-center justify-end gap-[calc(var(--inv-gap)*1.4)]">
            <div
              data-slot="amulet"
              class="aspect-square min-h-[var(--inv-amulet)] max-w-[var(--inv-amulet)] rounded-full border"
              :class="[dragging ? 'overflow-visible' : 'overflow-hidden', slotStateClasses('amulet')]"
            >
              <SessionInventoryDraggableItem v-if="equipmentEntry('amulet')" :entry="equipmentEntry('amulet')" variant="equipment" />
              <div v-else class="flex h-full items-center justify-center p-[clamp(0.25rem,0.7vw,0.45rem)] text-center">
                <span class="text-[clamp(0.48rem,1vw,0.82rem)] font-bold uppercase tracking-[0.14em] text-[rgba(126,200,227,0.62)]">Amulet</span>
              </div>
            </div>

            <div
              data-slot="ring_2"
              class="aspect-square min-h-[var(--inv-ring)] max-w-[var(--inv-ring)] rounded-full border"
              :class="[dragging ? 'overflow-visible' : 'overflow-hidden', slotStateClasses('ring_2')]"
            >
              <SessionInventoryDraggableItem v-if="equipmentEntry('ring_2')" :entry="equipmentEntry('ring_2')" variant="equipment" />
              <div v-else class="flex h-full items-center justify-center p-[clamp(0.25rem,0.7vw,0.45rem)] text-center">
                <span class="text-[clamp(0.48rem,1vw,0.82rem)] font-bold uppercase tracking-[0.14em] text-[rgba(126,200,227,0.62)]">Ring</span>
              </div>
            </div>
          </div>

          <div class="flex h-auto w-[var(--inv-weapon-col)] flex-col items-stretch gap-[var(--inv-gap)]">
            <div
              data-slot="off_hand"
              class="h-[var(--inv-weapon-h)] w-full rounded-[clamp(0.12rem,0.3vw,0.24rem)] border"
              :class="[dragging ? 'overflow-visible' : 'overflow-hidden', slotStateClasses('off_hand')]"
            >
              <SessionInventoryDraggableItem v-if="equipmentEntry('off_hand')" :entry="equipmentEntry('off_hand')" variant="equipment" />
              <div v-else class="flex h-full items-center justify-center p-[clamp(0.25rem,0.7vw,0.45rem)] text-center">
                <span class="text-[clamp(0.48rem,1vw,0.82rem)] font-bold uppercase tracking-[0.14em] text-[rgba(126,200,227,0.62)]">Off Hand</span>
              </div>
            </div>

            <div
              data-slot="boots"
              class="h-[var(--inv-glove-h)] w-full rounded-[clamp(0.12rem,0.3vw,0.24rem)] border"
              :class="[dragging ? 'overflow-visible' : 'overflow-hidden', slotStateClasses('boots')]"
            >
              <SessionInventoryDraggableItem v-if="equipmentEntry('boots')" :entry="equipmentEntry('boots')" variant="equipment" />
              <div v-else class="flex h-full items-center justify-center p-[clamp(0.25rem,0.7vw,0.45rem)] text-center">
                <span class="text-[clamp(0.48rem,1vw,0.82rem)] font-bold uppercase tracking-[0.14em] text-[rgba(126,200,227,0.62)]">Boots</span>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section class="mx-auto w-fit max-w-full max-md:w-full">
        <div
          ref="gridRef"
          class="session-inventory-grid grid w-fit gap-0 rounded-[clamp(0.1rem,0.25vw,0.18rem)] border border-[rgba(126,200,227,0.32)] bg-[linear-gradient(180deg,rgba(12,25,49,0.98),rgba(8,16,31,1))] p-[clamp(0.14rem,0.35vw,0.28rem)] shadow-[inset_0_0_0_1px_rgba(126,200,227,0.04)] grid-cols-[repeat(var(--inv-cols),minmax(0,var(--inv-cell)))] grid-rows-[repeat(var(--inv-rows),minmax(0,var(--inv-cell)))] max-md:max-w-full"
          :style="{
            '--inv-cols': safeInventoryWidth,
            '--inv-rows': safeInventoryHeight,
          }"
        >
          <div
            v-for="cell in gridCells"
            :key="cell.id"
            class="session-inventory-grid__cell h-full min-w-0 min-h-0 border"
            :class="cellStateClasses(cell)"
            :style="{ gridColumn: `${cell.x + 1}`, gridRow: `${cell.y + 1}` }"
          ></div>

          <div
            v-for="inventoryEntry in inventoryEntries"
            :key="inventoryEntry.entry.id"
            class="relative z-[1] min-w-0 min-h-0"
            :style="inventoryItemStyle(inventoryEntry.entry, inventoryEntry.placement)"
          >
            <SessionInventoryDraggableItem :entry="inventoryEntry.entry" variant="grid" />
          </div>
        </div>
      </section>

    </div>
  </article>
</template>
