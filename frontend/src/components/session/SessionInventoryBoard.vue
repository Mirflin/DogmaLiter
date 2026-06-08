<script setup>
import interact from 'interactjs'
import { computed, onBeforeUnmount, onMounted, provide, ref, watch } from 'vue'
import { API_URL } from '@/api'
import { notify } from '@/notify'
import { Send, Trash2, X } from '@lucide/vue'
import SessionInventoryDraggableItem from './SessionInventoryDraggableItem.vue'
import weaponImg from '@/assets/weapon.png'
import offHandImg from '@/assets/lamp.png'
import headImg from '@/assets/helmet.png'
import chestImg from '@/assets/armour.png'
import amuletImg from '@/assets/charm.png'
import ringImg from '@/assets/ring1.png'
import ring2Img from '@/assets/ring2.png'
import beltImg from '@/assets/belt.png'
import glovesImg from '@/assets/gloves.png'
import bootsImg from '@/assets/boots.png'

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
  characterAttributes: {
    type: Object,
    default: () => ({}),
  },
  attributesEnabled: {
    type: Boolean,
    default: true,
  },
  characterId: {
    type: String,
    default: '',
  },
  canEdit: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits(['persist', 'update-item', 'delete-item', 'share', 'split'])

provide('inventoryCharacterAttributes', computed(() => props.characterAttributes))
provide('inventoryAttributesEnabled', computed(() => props.attributesEnabled))

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
const mounted = ref(false)

let interactable = null
let grabPxX = 0
let grabPxY = 0
let persistTimer = null

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
  requestAnimationFrame(() => {
    mounted.value = true
  })

  if (!boardRef.value) return

  interactable = interact('.session-inventory-item', { context: boardRef.value })
  interactable.draggable({
    enabled: props.canEdit,
    inertia: false,
    autoScroll: true,
    listeners: {
      start: onDragStart,
      move: onDragMove,
      end: onDragEnd,
    },
  })
})

watch(() => props.canEdit, (value) => {
  if (interactable) interactable.draggable({ enabled: value })
})

onBeforeUnmount(() => {
  flushPersist()
  if (interactable) {
    interactable.unset()
    interactable = null
  }
})

function setDragVisual(element, active, geometry) {
  if (active) {
    element.style.position = 'fixed'
    element.style.width = `${geometry.width}px`
    element.style.height = `${geometry.height}px`
    element.style.left = `${geometry.left}px`
    element.style.top = `${geometry.top}px`
    element.style.margin = '0'
    element.style.zIndex = '9999'
    element.style.pointerEvents = 'none'
    element.style.transition = 'transform 130ms ease, filter 130ms ease'
    element.style.transform = 'scale(1.06) rotate(1.4deg)'
    element.style.filter = 'drop-shadow(0 18px 28px rgba(0, 0, 0, 0.6))'
    element.style.cursor = 'grabbing'
    return
  }

  for (const prop of ['position', 'width', 'height', 'left', 'top', 'margin', 'zIndex', 'pointerEvents', 'transition', 'transform', 'filter', 'cursor']) {
    element.style[prop] = ''
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

  // Pixel offset from the cursor to the dragged item's top-left, so the grabbed
  // cell sits centered under the pointer once the item snaps to its real grid size.
  grabPxX = grabOffset.value.dx * cellWidth + cellWidth / 2
  grabPxY = grabOffset.value.dy * cellHeight + cellHeight / 2

  setDragVisual(element, true, {
    width: width * cellWidth,
    height: height * cellHeight,
    left: event.clientX - grabPxX,
    top: event.clientY - grabPxY,
  })
  updateHoverFromPointer(event.clientX, event.clientY)
}

function onDragMove(event) {
  const element = event.target
  element.style.left = `${event.clientX - grabPxX}px`
  element.style.top = `${event.clientY - grabPxY}px`
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
  schedulePersist()
}

function buildLayout() {
  const inventory = []
  const equipment = []

  for (const placement of Object.values(placements.value)) {
    if (placement.kind === 'inventory') {
      inventory.push({ id: placement.entryId, grid_x: placement.x, grid_y: placement.y, is_rotated: false })
    } else if (placement.kind === 'equipment') {
      equipment.push({ slot: placement.slot, inventory_item_id: placement.entryId })
    }
  }

  return { inventory, equipment }
}

function schedulePersist() {
  if (!props.canEdit || !props.characterId) return
  clearTimeout(persistTimer)
  persistTimer = setTimeout(() => {
    persistTimer = null
    emit('persist', buildLayout())
  }, 400)
}

function flushPersist() {
  if (!persistTimer) return
  clearTimeout(persistTimer)
  persistTimer = null
  emit('persist', buildLayout())
}

const detailEntryId = ref('')
const detailDurability = ref(0)
const detailMaxDurability = ref(100)
const detailConfirmDelete = ref(false)
const detailEntry = computed(() => allEntriesById.value[detailEntryId.value] ?? null)
const detailImageUrl = computed(() => detailEntry.value?.item?.image_id ? `${API_URL}/api/uploads/${detailEntry.value.item.image_id}` : '')
const detailRequirements = computed(() => {
  if (!props.attributesEnabled) return []
  return (detailEntry.value?.item?.required_attributes ?? []).map((requirement) => {
    const name = String(requirement?.attribute_name || '')
    const current = Number(props.characterAttributes?.[name] ?? 0)
    const required = Number(requirement?.min_value ?? 0)
    return { label: formatLabel(name), required, current, met: current >= required }
  })
})
const detailModifiers = computed(() => (props.attributesEnabled ? (detailEntry.value?.item?.attribute_modifiers ?? []) : []).map((modifier) => ({
  label: formatLabel(modifier?.attribute_name),
  value: Number(modifier?.modifier_value) || 0,
  percent: Boolean(modifier?.is_percentage),
})))
const detailMeetsRequirements = computed(() => detailRequirements.value.every((requirement) => requirement.met))

function formatLabel(value) {
  return String(value || '').split('_').filter(Boolean).map((part) => part.charAt(0).toUpperCase() + part.slice(1)).join(' ')
}

function rarityTextClass(rarity) {
  const variants = {
    common: 'text-[#e2e8f0]',
    uncommon: 'text-[#fde68a]',
    rare: 'text-[#93c5fd]',
    epic: 'text-[#e9d5ff]',
    masterwork: 'text-[#fdba74]',
    legendary: 'text-[#86efac]',
    unique: 'text-[#fca5a5]',
  }
  return variants[rarity] || variants.common
}

function onBoardDblClick(event) {
  const element = event.target.closest?.('.session-inventory-item')
  if (!element) return
  const entry = allEntriesById.value[element.dataset.entryId]
  if (entry) openItemDetail(entry)
}

function openItemDetail(entry) {
  detailEntryId.value = entry.id
  detailMaxDurability.value = Number(entry.max_durability ?? entry.durability ?? 100) || 100
  detailDurability.value = Number(entry.durability ?? detailMaxDurability.value) || 0
  detailConfirmDelete.value = false
}

function closeItemDetail() {
  detailEntryId.value = ''
  detailConfirmDelete.value = false
}

function saveItemDetail() {
  const entry = detailEntry.value
  if (!entry) return
  const max = clampInt(Number(detailMaxDurability.value) || 1, 1, 1000000)
  const current = clampInt(Number(detailDurability.value) || 0, 0, max)
  emit('update-item', { id: entry.id, durability: current, max_durability: max })
  closeItemDetail()
}

function deleteItemDetail() {
  if (!detailConfirmDelete.value) {
    detailConfirmDelete.value = true
    return
  }
  const entry = detailEntry.value
  if (entry) emit('delete-item', entry.id)
  closeItemDetail()
}

function shareItemDetail() {
  const entry = detailEntry.value
  if (entry) emit('share', entry)
  closeItemDetail()
}

function splitItemDetail() {
  const entry = detailEntry.value
  if (!entry || (entry.quantity || 1) <= 1) return
  emit('split', entry.id)
  closeItemDetail()
}

function meetsItemRequirements(entry) {
  if (!props.attributesEnabled) return true
  const requirements = entry?.item?.required_attributes ?? []
  return requirements.every((requirement) => {
    const name = String(requirement?.attribute_name || '')
    const current = Number(props.characterAttributes?.[name] ?? 0)
    return current >= Number(requirement?.min_value ?? 0)
  })
}

function placeIntoEquipment(entryId, slot) {
  const entry = allEntriesById.value[entryId]
  if (!canEquip(entry, slot)) return

  if (!meetsItemRequirements(entry)) {
    notify.warning({
      title: 'Requirements not met',
      message: `Your character does not meet the requirements to equip ${entry.item?.name || 'this item'}.`,
    })
    return
  }

  placements.value = {
    ...placements.value,
    [entryId]: {
      entryId,
      kind: 'equipment',
      slot,
    },
  }
  schedulePersist()
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
    @dblclick="onBoardDblClick"
    :class="mounted ? 'opacity-100' : 'opacity-0'"
    class="relative mx-auto w-full max-w-full transition-opacity duration-500 ease-out rounded-[clamp(0.22rem,0.5vw,0.34rem)] border border-[rgba(126,200,227,0.28)] bg-[radial-gradient(circle_at_top,rgba(233,69,96,0.12),transparent_26%),linear-gradient(180deg,rgba(16,24,44,0.98),rgba(9,15,29,1)),linear-gradient(135deg,rgba(126,200,227,0.05),transparent_42%)] p-[0.48rem] shadow-[inset_0_0_0_1px_rgba(126,200,227,0.06),0_18px_34px_rgba(0,0,0,0.28)] before:pointer-events-none before:absolute before:inset-[clamp(0.3rem,0.8vw,0.55rem)] before:rounded-[clamp(0.16rem,0.4vw,0.28rem)] before:border before:border-[rgba(126,200,227,0.14)] before:content-[''] md:p-[0.7rem] lg:w-fit lg:p-[clamp(0.55rem,1.4vw,0.95rem)] [--inv-cell:clamp(1.58rem,8.8vw,2.15rem)] min-[480px]:[--inv-cell:clamp(1.72rem,8vw,2.6rem)] md:[--inv-cell:clamp(1.9rem,6.2vw,3.85rem)] lg:[--inv-cell:clamp(1.9rem,5.9vw,4.95rem)] [--inv-gap:calc(var(--inv-cell)*0.28)] [--inv-weapon-col:calc(var(--inv-cell)*2.6)] [--inv-trinket-col:calc(var(--inv-cell)*0.98)] [--inv-center-col:calc(var(--inv-cell)*2.08)] [--inv-weapon-h:calc(var(--inv-cell)*3.76)] [--inv-glove-h:calc(var(--inv-cell)*1.88)] [--inv-head:calc(var(--inv-cell)*2)] [--inv-chest-h:calc(var(--inv-cell)*2.42)] [--inv-belt-h:calc(var(--inv-cell)*1.03)] [--inv-ring:calc(var(--inv-cell)*0.91)] [--inv-amulet:var(--inv-cell)]"
  >
    <div class="relative z-[1] flex flex-col gap-[0.4rem] p-[0.42rem] min-[480px]:p-[0.55rem] md:gap-[0.6rem] md:p-[0.78rem] lg:gap-[clamp(0.45rem,1vw,0.8rem)] lg:p-[clamp(0.6rem,1.5vw,1rem)]">
      <section class="shrink-0">
        <div class="mx-auto flex w-fit max-w-full items-stretch justify-center gap-[var(--inv-gap)]">
          <div class="flex h-auto w-[var(--inv-weapon-col)] flex-col gap-[var(--inv-gap)]">
            <div
              data-slot="main_hand"
              class="h-[var(--inv-weapon-h)] w-full relative rounded-[clamp(0.12rem,0.3vw,0.24rem)] border transition-[border-color,box-shadow,background-color] duration-150"
              :class="['overflow-hidden', slotStateClasses('main_hand')]"
            >
              <SessionInventoryDraggableItem v-if="equipmentEntry('main_hand')" :entry="equipmentEntry('main_hand')" variant="equipment" class="relative z-[1]" />
              <div :style="{
                backgroundImage: `url(${weaponImg})`,
                backgroundSize: '85%',
                backgroundPosition: 'center',
                backgroundRepeat: 'no-repeat'
              }" class="pointer-events-none absolute inset-0 z-0 bg-center bg-no-repeat">
              </div>
            </div>

            <div
              data-slot="gloves"
              class="h-[var(--inv-glove-h)] w-full relative rounded-[clamp(0.12rem,0.3vw,0.24rem)] border transition-[border-color,box-shadow,background-color] duration-150"
              :class="['overflow-hidden', slotStateClasses('gloves')]"
            >
              <SessionInventoryDraggableItem v-if="equipmentEntry('gloves')" :entry="equipmentEntry('gloves')" variant="equipment" class="relative z-[1]" />
              <div :style="{
                backgroundImage: `url(${glovesImg})`,
                backgroundSize: '75%',
                backgroundPosition: 'center',
                backgroundRepeat: 'no-repeat'
              }" class="pointer-events-none absolute inset-0 z-0 bg-center bg-no-repeat">
                
              </div>
            </div>
          </div>

          <div class="flex h-auto w-[var(--inv-trinket-col)] flex-col items-center justify-end gap-[var(--inv-gap)]">
            <div
              data-slot="ring_1"
              class="aspect-square min-h-[var(--inv-ring)] max-w-[var(--inv-ring)] relative rounded-full border transition-[border-color,box-shadow,background-color] duration-150"
              :class="['overflow-hidden', slotStateClasses('ring_1')]"
            >
              <SessionInventoryDraggableItem v-if="equipmentEntry('ring_1')" :entry="equipmentEntry('ring_1')" variant="equipment" class="relative z-[1]" />
              <div :style="{
                backgroundImage: `url(${ringImg})`,
                backgroundSize: '75%',
                backgroundPosition: 'center',
                backgroundRepeat: 'no-repeat'
              }" class="pointer-events-none absolute inset-0 z-0 bg-center bg-no-repeat">
                
              </div>
            </div>
          </div>

          <div class="flex h-auto w-[var(--inv-center-col)] flex-col gap-[var(--inv-gap)]">
            <div
              data-slot="head"
              class="h-[var(--inv-head)] w-full relative rounded-[clamp(0.12rem,0.3vw,0.24rem)] border transition-[border-color,box-shadow,background-color] duration-150"
              :class="['overflow-hidden', slotStateClasses('head')]"
            >
              <SessionInventoryDraggableItem v-if="equipmentEntry('head')" :entry="equipmentEntry('head')" variant="equipment" class="relative z-[1]" />
              <div :style="{
                backgroundImage: `url(${headImg})`,
                backgroundSize: '85%',
                backgroundPosition: 'center',
                backgroundRepeat: 'no-repeat'
              }" class="pointer-events-none absolute inset-0 z-0 bg-center bg-no-repeat">
              </div>
            </div>

            <div
              data-slot="chest"
              class="h-[var(--inv-chest-h)] w-full relative rounded-[clamp(0.12rem,0.3vw,0.24rem)] border transition-[border-color,box-shadow,background-color] duration-150"
              :class="['overflow-hidden', slotStateClasses('chest')]"
            >
              <SessionInventoryDraggableItem v-if="equipmentEntry('chest')" :entry="equipmentEntry('chest')" variant="equipment" class="relative z-[1]" />
              <div :style="{
                backgroundImage: `url(${chestImg})`,
                backgroundSize: '85%',
                backgroundPosition: 'center',
                backgroundRepeat: 'no-repeat'
              }" class="pointer-events-none absolute inset-0 z-0 bg-center bg-no-repeat">
                
              </div>
            </div>

            <div
              data-slot="belt"
              class="h-[var(--inv-belt-h)] w-full relative rounded-[clamp(0.12rem,0.3vw,0.24rem)] border transition-[border-color,box-shadow,background-color] duration-150"
              :class="['overflow-hidden', slotStateClasses('belt')]"
            >
              <SessionInventoryDraggableItem v-if="equipmentEntry('belt')" :entry="equipmentEntry('belt')" variant="equipment" class="relative z-[1]" />
              <div :style="{
                backgroundImage: `url(${beltImg})`,
                backgroundSize: '85%',
                backgroundPosition: 'center',
                backgroundRepeat: 'no-repeat'
              }" class="pointer-events-none absolute inset-0 z-0 bg-center bg-no-repeat">
                
              </div>
            </div>
          </div>

          <div class="flex h-auto w-[var(--inv-trinket-col)] flex-col items-center justify-end gap-[calc(var(--inv-gap)*1.4)]">
            <div
              data-slot="amulet"
              class="aspect-square min-h-[var(--inv-amulet)] max-w-[var(--inv-amulet)] relative rounded-full border transition-[border-color,box-shadow,background-color] duration-150"
              :class="['overflow-hidden', slotStateClasses('amulet')]"
            >
              <SessionInventoryDraggableItem v-if="equipmentEntry('amulet')" :entry="equipmentEntry('amulet')" variant="equipment" class="relative z-[1]" />
              <div :style="{
                backgroundImage: `url(${amuletImg})`,
                backgroundSize: '85%',
                backgroundPosition: 'center',
                backgroundRepeat: 'no-repeat'
              }" class="pointer-events-none absolute inset-0 z-0 bg-center bg-no-repeat">
                
              </div>
            </div>

            <div
              data-slot="ring_2"
              class="aspect-square min-h-[var(--inv-ring)] max-w-[var(--inv-ring)] relative rounded-full border transition-[border-color,box-shadow,background-color] duration-150"
              :class="['overflow-hidden', slotStateClasses('ring_2')]"
            >
              <SessionInventoryDraggableItem v-if="equipmentEntry('ring_2')" :entry="equipmentEntry('ring_2')" variant="equipment" class="relative z-[1]" />
              <div :style="{
                backgroundImage: `url(${ring2Img})`,
                backgroundSize: '75%',
                backgroundPosition: 'center',
                backgroundRepeat: 'no-repeat'
              }" class="pointer-events-none absolute inset-0 z-0 bg-center bg-no-repeat">
                
              </div>
            </div>
          </div>

          <div class="flex h-auto w-[var(--inv-weapon-col)] flex-col items-stretch gap-[var(--inv-gap)]">
            <div
              data-slot="off_hand"
              class="h-[var(--inv-weapon-h)] w-full relative rounded-[clamp(0.12rem,0.3vw,0.24rem)] border transition-[border-color,box-shadow,background-color] duration-150"
              :class="['overflow-hidden', slotStateClasses('off_hand')]"
            >
              <SessionInventoryDraggableItem v-if="equipmentEntry('off_hand')" :entry="equipmentEntry('off_hand')" variant="equipment" class="relative z-[1]" />
              <div :style="{
                backgroundImage: `url(${offHandImg})`,
                backgroundSize: '85%',
                backgroundPosition: 'center',
                backgroundRepeat: 'no-repeat'
              }" class="pointer-events-none absolute inset-0 z-0 bg-center bg-no-repeat">
                
              </div>
            </div>

            <div
              data-slot="boots"
              class="h-[var(--inv-glove-h)] w-full relative rounded-[clamp(0.12rem,0.3vw,0.24rem)] border transition-[border-color,box-shadow,background-color] duration-150"
              :class="['overflow-hidden', slotStateClasses('boots')]"
            >
              <SessionInventoryDraggableItem v-if="equipmentEntry('boots')" :entry="equipmentEntry('boots')" variant="equipment" class="relative z-[1]" />
              <div :style="{
                backgroundImage: `url(${bootsImg})`,
                backgroundSize: '70%',
                backgroundPosition: 'center',
                backgroundRepeat: 'no-repeat'
              }" class="pointer-events-none absolute inset-0 z-0 bg-center bg-no-repeat">
                
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
            class="session-inventory-grid__cell h-full min-w-0 min-h-0 border transition-[border-color,box-shadow,background-color] duration-150"
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

  <Teleport to="body">
    <div v-if="detailEntry" class="fixed inset-0 z-[12700] flex items-center justify-center p-4">
      <div class="absolute inset-0 bg-[rgba(5,8,12,0.82)] backdrop-blur-md" @click="closeItemDetail"></div>

      <div class="relative flex max-h-full w-full max-w-[40rem] flex-col overflow-hidden rounded-[1.6rem] border border-[rgba(126,200,227,0.18)] bg-[linear-gradient(180deg,rgba(9,18,34,0.98),rgba(5,10,22,0.98))] shadow-[0_40px_120px_rgba(0,0,0,0.55)]">
        <button
          type="button"
          @click="closeItemDetail"
          class="absolute right-4 top-4 z-10 flex h-9 w-9 cursor-pointer items-center justify-center rounded-xl border border-[rgba(126,200,227,0.14)] bg-[rgba(126,200,227,0.08)] text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(233,69,96,0.32)]"
          aria-label="Close item details"
        >
          <X class="h-5 w-5" :stroke-width="2" />
        </button>

        <div class="min-h-0 flex-1 overflow-y-auto p-5 sm:p-6">
          <div class="flex flex-col gap-4 sm:flex-row sm:items-start">
            <div class="h-28 w-28 shrink-0 overflow-hidden rounded-[1.2rem] border-2 border-[rgba(126,200,227,0.2)] bg-[rgba(7,17,31,0.66)]">
              <img v-if="detailImageUrl" :src="detailImageUrl" :alt="detailEntry.item?.name" class="h-full w-full object-cover" />
              <div v-else class="flex h-full w-full items-center justify-center font-[Cinzel] text-[32px] font-bold text-[#7ec8e3]/40">
                {{ (detailEntry.item?.name || '?').charAt(0).toUpperCase() }}
              </div>
            </div>

            <div class="min-w-0 flex-1 pr-8">
              <h3 class="break-words text-[20px] font-bold leading-tight text-[#f6f7fb] [overflow-wrap:anywhere]">{{ detailEntry.item?.name || 'Unnamed item' }}</h3>
              <div class="mt-1.5 flex flex-wrap items-center gap-2 text-[11px] uppercase tracking-[0.16em]">
                <span class="font-bold" :class="rarityTextClass(detailEntry.item?.rarity)">{{ formatLabel(detailEntry.item?.rarity || 'common') }}</span>
                <span class="text-[#7ec8e3]/45">{{ formatLabel(detailEntry.item?.category || 'other') }}</span>
                <span class="text-[#7ec8e3]/45">{{ detailEntry.item?.grid_width || 1 }}x{{ detailEntry.item?.grid_height || 1 }}</span>
                <span v-if="detailEntry.item?.equip_slot" class="text-[#7ec8e3]/45">{{ formatLabel(detailEntry.item.equip_slot) }}</span>
              </div>
              <div class="mt-2 flex flex-wrap gap-3 text-[12px] text-[#d8dce7]/70">
                <span>Quantity: <span class="font-semibold text-[#f6f7fb]">{{ detailEntry.quantity || 1 }}</span></span>
                <span v-if="detailEntry.enchantment">Enchantment: <span class="font-semibold text-[#8fd7ef]">+{{ detailEntry.enchantment }}</span></span>
              </div>
              <div class="mt-3">
                <p class="text-[10px] uppercase tracking-[0.18em] text-[#7ec8e3]/55">Description</p>
                <p class="mt-1 break-words whitespace-pre-line text-[13px] leading-relaxed text-[#d8dce7]/72 [overflow-wrap:anywhere]">{{ detailEntry.item?.description?.trim() || 'No description provided.' }}</p>
              </div>
            </div>
          </div>

          <div v-if="detailRequirements.length" class="mt-5">
            <p class="text-[10px] uppercase tracking-[0.18em] text-[#7ec8e3]/55">Requirements</p>
            <ul class="mt-2 grid gap-1.5 sm:grid-cols-2">
              <li
                v-for="(requirement, index) in detailRequirements"
                :key="`req-${index}`"
                class="flex items-center justify-between gap-2 rounded-lg border px-3 py-1.5 text-[12px]"
                :class="requirement.met ? 'border-[rgba(74,222,128,0.2)] bg-[rgba(21,128,61,0.12)] text-[#86efac]' : 'border-[rgba(248,113,113,0.2)] bg-[rgba(127,29,29,0.14)] text-[#fca5a5]'"
              >
                <span>{{ requirement.label }} {{ requirement.required }}</span>
                <span class="opacity-80">you: {{ requirement.current }}</span>
              </li>
            </ul>
            <p v-if="!detailMeetsRequirements" class="mt-1.5 text-[11px] font-semibold text-[#fca5a5]">Requirements not met</p>
          </div>

          <div v-if="detailModifiers.length" class="mt-5">
            <p class="text-[10px] uppercase tracking-[0.18em] text-[#7ec8e3]/55">Grants</p>
            <ul class="mt-2 grid gap-1.5 sm:grid-cols-2">
              <li v-for="(modifier, index) in detailModifiers" :key="`mod-${index}`" class="flex items-center justify-between gap-2 rounded-lg border border-[rgba(126,200,227,0.12)] bg-[rgba(126,200,227,0.06)] px-3 py-1.5 text-[12px] text-[#d8dce7]/82">
                <span>{{ modifier.label }}</span>
                <span class="font-semibold" :class="modifier.value >= 0 ? 'text-[#8fd7ef]' : 'text-[#fca5a5]'">{{ modifier.value >= 0 ? '+' : '' }}{{ modifier.value }}{{ modifier.percent ? '%' : '' }}</span>
              </li>
            </ul>
          </div>

          <div class="mt-5">
            <p class="text-[10px] uppercase tracking-[0.18em] text-[#7ec8e3]/55">Durability</p>
            <div class="mt-2 h-2 w-full overflow-hidden rounded-full bg-[rgba(7,17,31,0.8)]">
              <div class="h-full rounded-full bg-blue-200" :style="{ width: `${Math.max(0, Math.min(100, detailMaxDurability ? (detailDurability / detailMaxDurability) * 100 : 0))}%` }"></div>
            </div>
            <div v-if="canEdit" class="mt-3 grid gap-3 sm:grid-cols-2">
              <label class="block">
                <span class="text-[11px] uppercase tracking-[0.16em] text-[#7ec8e3]/45">Current</span>
                <input v-model.number="detailDurability" type="number" min="0" :max="detailMaxDurability" class="session-input mt-1.5 w-full rounded-[0.9rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.72)] px-3 py-2.5 text-[14px] text-[#f6f7fb] outline-none" />
              </label>
              <label class="block">
                <span class="text-[11px] uppercase tracking-[0.16em] text-[#7ec8e3]/45">Max</span>
                <input disabled v-model.number="detailMaxDurability" type="number" min="1" class="session-input mt-1.5 w-full rounded-[0.9rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.72)] px-3 py-2.5 text-[14px] text-[#f6f7fb] outline-none" />
              </label>
            </div>
            <p v-else class="mt-2 text-[13px] text-[#d8dce7]/70">{{ detailDurability }} / {{ detailMaxDurability }}</p>
          </div>
        </div>

        <div class="flex flex-wrap items-center justify-between gap-3 border-t border-[rgba(126,200,227,0.1)] px-5 py-4 sm:px-6">
          <div class="flex flex-wrap gap-3">
            <button
              type="button"
              @click="shareItemDetail"
              class="inline-flex cursor-pointer items-center gap-2 rounded-xl border border-[rgba(126,200,227,0.2)] bg-[rgba(126,200,227,0.08)] px-4 py-2.5 text-[13px] font-semibold text-[#8fd7ef] transition-all duration-200 hover:border-[rgba(126,200,227,0.4)]"
            >
              <Send class="h-4 w-4" :stroke-width="2" />
              Share to chat
            </button>
            <button
              v-if="canEdit && (detailEntry.quantity || 1) > 1"
              type="button"
              @click="splitItemDetail"
              class="inline-flex cursor-pointer items-center gap-2 rounded-xl border border-[rgba(126,200,227,0.2)] bg-[rgba(126,200,227,0.08)] px-4 py-2.5 text-[13px] font-semibold text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(126,200,227,0.4)]"
            >
              Unstack
            </button>
            <button
              v-if="canEdit"
              type="button"
              @click="deleteItemDetail"
              class="inline-flex cursor-pointer items-center gap-2 rounded-xl border border-[rgba(248,113,113,0.28)] bg-[rgba(248,113,113,0.12)] px-4 py-2.5 text-[13px] font-semibold text-[#fecaca] transition-all duration-200 hover:border-[rgba(248,113,113,0.45)]"
            >
              <Trash2 class="h-4 w-4" :stroke-width="2" />
              {{ detailConfirmDelete ? 'Confirm delete' : 'Delete item' }}
            </button>
          </div>
          <div v-if="canEdit" class="flex gap-3">
            <button type="button" @click="closeItemDetail" class="cursor-pointer rounded-xl border border-[rgba(126,200,227,0.16)] bg-[rgba(126,200,227,0.08)] px-4 py-2.5 text-[13px] font-semibold text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(126,200,227,0.3)]">
              Cancel
            </button>
            <button type="button" @click="saveItemDetail" class="inline-flex cursor-pointer items-center gap-2 rounded-xl border border-[rgba(233,69,96,0.28)] bg-[linear-gradient(135deg,rgba(233,69,96,0.92),rgba(194,49,82,0.92))] px-4 py-2.5 text-[13px] font-semibold text-white transition-all duration-200 hover:-translate-y-0.5">
              Save changes
            </button>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>
