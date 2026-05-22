<script setup>
import { DnDProvider } from '@vue-dnd-kit/core'
import { computed, ref, watch } from 'vue'
import SessionInventoryDraggableItem from './SessionInventoryDraggableItem.vue'
import SessionInventoryDropZone from './SessionInventoryDropZone.vue'

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

const placements = ref({})

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

watch(
  () => [props.inventoryItems, props.equipment, safeInventoryWidth.value, safeInventoryHeight.value],
  () => syncPlacements(),
  { immediate: true, deep: true },
)

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

function handleDrop({ event, zone }) {
  const draggedEntry = event.draggedItems?.[0]?.data
  if (!draggedEntry?.id || !allEntriesById.value[draggedEntry.id]) return

  if (zone.kind === 'inventory') {
    placeIntoInventory(draggedEntry.id, zone.x, zone.y)
    return
  }

  if (zone.kind === 'equipment') {
    placeIntoEquipment(draggedEntry.id, zone.slot)
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
  if (!entry?.item?.is_equippable) return

  const isRingSlot = slot === 'ring_1' || slot === 'ring_2'
  const isRingItem = entry.item.equip_slot === 'ring_1' || entry.item.equip_slot === 'ring_2'
  if (!((isRingSlot && isRingItem) || entry.item.equip_slot === slot)) return

  const slotOwner = Object.values(placements.value).find((placement) => placement.kind === 'equipment' && placement.slot === slot)
  if (slotOwner && slotOwner.entryId !== entryId) return

  placements.value = {
    ...placements.value,
    [entryId]: {
      entryId,
      kind: 'equipment',
      slot,
    },
  }
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

function slotDropZone(slot) {
  return {
    kind: 'equipment',
    slot,
  }
}

function equipmentEntry(slot) {
  return equipmentBySlot.value[slot] ?? null
}
</script>

<template>
  <DnDProvider>
    <article class="session-inventory-panel">
      <div class="session-inventory-panel__body">
        <section class="session-inventory-equipment">
          <div class="session-inventory-equipment__columns">
            <div class="session-equipment-column session-equipment-column--weapon-gloves">
              <SessionInventoryDropZone :zone="slotDropZone('main_hand')" @drop="handleDrop">
                <template #default="slotState">
                  <div
                    class="session-equipment-slot session-equipment-slot--weapon-main"
                    :class="{
                      'session-equipment-slot--active': slotState.isDragOver && slotState.isAllowed,
                      'session-equipment-slot--blocked': slotState.isDragOver && !slotState.isAllowed,
                    }"
                  >
                    <SessionInventoryDraggableItem v-if="equipmentEntry('main_hand')" :entry="equipmentEntry('main_hand')" variant="equipment" />
                    <div v-else class="session-equipment-slot__placeholder">
                      <span class="session-equipment-slot__label">Main Hand</span>
                    </div>
                  </div>
                </template>
              </SessionInventoryDropZone>

              <SessionInventoryDropZone :zone="slotDropZone('gloves')" @drop="handleDrop">
                <template #default="slotState">
                  <div
                    class="session-equipment-slot session-equipment-slot--gloves"
                    :class="{
                      'session-equipment-slot--active': slotState.isDragOver && slotState.isAllowed,
                      'session-equipment-slot--blocked': slotState.isDragOver && !slotState.isAllowed,
                    }"
                  >
                    <SessionInventoryDraggableItem v-if="equipmentEntry('gloves')" :entry="equipmentEntry('gloves')" variant="equipment" />
                    <div v-else class="session-equipment-slot__placeholder">
                      <span class="session-equipment-slot__label">Gloves</span>
                    </div>
                  </div>
                </template>
              </SessionInventoryDropZone>
            </div>

            <div class="session-equipment-column session-equipment-column--trinket-left">
              <SessionInventoryDropZone :zone="slotDropZone('ring_1')" @drop="handleDrop">
                <template #default="slotState">
                  <div
                    class="session-equipment-slot session-equipment-slot--ring-left"
                    :class="{
                      'session-equipment-slot--active': slotState.isDragOver && slotState.isAllowed,
                      'session-equipment-slot--blocked': slotState.isDragOver && !slotState.isAllowed,
                    }"
                  >
                    <SessionInventoryDraggableItem v-if="equipmentEntry('ring_1')" :entry="equipmentEntry('ring_1')" variant="equipment" />
                    <div v-else class="session-equipment-slot__placeholder">
                      <span class="session-equipment-slot__label">Ring</span>
                    </div>
                  </div>
                </template>
              </SessionInventoryDropZone>
            </div>

            <div class="session-equipment-column session-equipment-column--helmet-armour-belt">
              <SessionInventoryDropZone :zone="slotDropZone('head')" @drop="handleDrop">
                <template #default="slotState">
                  <div
                    class="session-equipment-slot session-equipment-slot--head"
                    :class="{
                      'session-equipment-slot--active': slotState.isDragOver && slotState.isAllowed,
                      'session-equipment-slot--blocked': slotState.isDragOver && !slotState.isAllowed,
                    }"
                  >
                    <SessionInventoryDraggableItem v-if="equipmentEntry('head')" :entry="equipmentEntry('head')" variant="equipment" />
                    <div v-else class="session-equipment-slot__placeholder">
                      <span class="session-equipment-slot__label">Head</span>
                    </div>
                  </div>
                </template>
              </SessionInventoryDropZone>

              <SessionInventoryDropZone :zone="slotDropZone('chest')" @drop="handleDrop">
                <template #default="slotState">
                  <div
                    class="session-equipment-slot session-equipment-slot--chest"
                    :class="{
                      'session-equipment-slot--active': slotState.isDragOver && slotState.isAllowed,
                      'session-equipment-slot--blocked': slotState.isDragOver && !slotState.isAllowed,
                    }"
                  >
                    <SessionInventoryDraggableItem v-if="equipmentEntry('chest')" :entry="equipmentEntry('chest')" variant="equipment" />
                    <div v-else class="session-equipment-slot__placeholder">
                      <span class="session-equipment-slot__label">Chest</span>
                    </div>
                  </div>
                </template>
              </SessionInventoryDropZone>

              <SessionInventoryDropZone :zone="slotDropZone('belt')" @drop="handleDrop">
                <template #default="slotState">
                  <div
                    class="session-equipment-slot session-equipment-slot--belt"
                    :class="{
                      'session-equipment-slot--active': slotState.isDragOver && slotState.isAllowed,
                      'session-equipment-slot--blocked': slotState.isDragOver && !slotState.isAllowed,
                    }"
                  >
                    <SessionInventoryDraggableItem v-if="equipmentEntry('belt')" :entry="equipmentEntry('belt')" variant="equipment" />
                    <div v-else class="session-equipment-slot__placeholder">
                      <span class="session-equipment-slot__label">Belt</span>
                    </div>
                  </div>
                </template>
              </SessionInventoryDropZone>
            </div>

            <div class="session-equipment-column session-equipment-column--charms-right">
              <SessionInventoryDropZone :zone="slotDropZone('amulet')" @drop="handleDrop">
                <template #default="slotState">
                  <div
                    class="session-equipment-slot session-equipment-slot--amulet"
                    :class="{
                      'session-equipment-slot--active': slotState.isDragOver && slotState.isAllowed,
                      'session-equipment-slot--blocked': slotState.isDragOver && !slotState.isAllowed,
                    }"
                  >
                    <SessionInventoryDraggableItem v-if="equipmentEntry('amulet')" :entry="equipmentEntry('amulet')" variant="equipment" />
                    <div v-else class="session-equipment-slot__placeholder">
                      <span class="session-equipment-slot__label">Amulet</span>
                    </div>
                  </div>
                </template>
              </SessionInventoryDropZone>

              <SessionInventoryDropZone :zone="slotDropZone('ring_2')" @drop="handleDrop">
                <template #default="slotState">
                  <div
                    class="session-equipment-slot session-equipment-slot--ring-right"
                    :class="{
                      'session-equipment-slot--active': slotState.isDragOver && slotState.isAllowed,
                      'session-equipment-slot--blocked': slotState.isDragOver && !slotState.isAllowed,
                    }"
                  >
                    <SessionInventoryDraggableItem v-if="equipmentEntry('ring_2')" :entry="equipmentEntry('ring_2')" variant="equipment" />
                    <div v-else class="session-equipment-slot__placeholder">
                      <span class="session-equipment-slot__label">Ring</span>
                    </div>
                  </div>
                </template>
              </SessionInventoryDropZone>
            </div>

            <div class="session-equipment-column session-equipment-column--offhand-boots">
              <SessionInventoryDropZone :zone="slotDropZone('off_hand')" @drop="handleDrop">
                <template #default="slotState">
                  <div
                    class="session-equipment-slot session-equipment-slot--offhand-main"
                    :class="{
                      'session-equipment-slot--active': slotState.isDragOver && slotState.isAllowed,
                      'session-equipment-slot--blocked': slotState.isDragOver && !slotState.isAllowed,
                    }"
                  >
                    <SessionInventoryDraggableItem v-if="equipmentEntry('off_hand')" :entry="equipmentEntry('off_hand')" variant="equipment" />
                    <div v-else class="session-equipment-slot__placeholder">
                      <span class="session-equipment-slot__label">Off Hand</span>
                    </div>
                  </div>
                </template>
              </SessionInventoryDropZone>

              <SessionInventoryDropZone :zone="slotDropZone('boots')" @drop="handleDrop">
                <template #default="slotState">
                  <div
                    class="session-equipment-slot session-equipment-slot--boots"
                    :class="{
                      'session-equipment-slot--active': slotState.isDragOver && slotState.isAllowed,
                      'session-equipment-slot--blocked': slotState.isDragOver && !slotState.isAllowed,
                    }"
                  >
                    <SessionInventoryDraggableItem v-if="equipmentEntry('boots')" :entry="equipmentEntry('boots')" variant="equipment" />
                    <div v-else class="session-equipment-slot__placeholder">
                      <span class="session-equipment-slot__label">Boots</span>
                    </div>
                  </div>
                </template>
              </SessionInventoryDropZone>
            </div>
          </div>
        </section>

        <section class="session-inventory-grid-shell">
          <div
            class="session-inventory-grid"
            :style="{
              '--inventory-columns': safeInventoryWidth,
              '--inventory-rows': safeInventoryHeight,
            }"
          >
            <SessionInventoryDropZone
              v-for="cell in gridCells"
              :key="cell.id"
              :zone="cell"
              @drop="handleDrop"
            >
              <template #default="slotState">
                <div
                  class="session-inventory-grid__cell"
                  :class="{
                    'session-inventory-grid__cell--active': slotState.isDragOver && slotState.isAllowed,
                    'session-inventory-grid__cell--blocked': slotState.isDragOver && !slotState.isAllowed,
                  }"
                ></div>
              </template>
            </SessionInventoryDropZone>

            <div
              v-for="inventoryEntry in inventoryEntries"
              :key="inventoryEntry.entry.id"
              class="session-inventory-grid__item"
              :style="inventoryItemStyle(inventoryEntry.entry, inventoryEntry.placement)"
            >
              <SessionInventoryDraggableItem :entry="inventoryEntry.entry" variant="grid" />
            </div>
          </div>
        </section>

      </div>
    </article>
  </DnDProvider>
</template>

<style scoped>
.session-inventory-panel {
  --inventory-cell-size: clamp(1.9rem, 5.9vw, 4.95rem);
  --inventory-gap: calc(var(--inventory-cell-size) * 0.28);
  --inventory-weapon-column-width: calc(var(--inventory-cell-size) * 2.6);
  --inventory-trinket-column-width: calc(var(--inventory-cell-size) * 0.98);
  --inventory-center-column-width: calc(var(--inventory-cell-size) * 2.08);
  --inventory-weapon-height: calc(var(--inventory-cell-size) * 3.76);
  --inventory-glove-height: calc(var(--inventory-cell-size) * 1.88);
  --inventory-head-size: calc(var(--inventory-cell-size) * 1.07);
  --inventory-chest-height: calc(var(--inventory-cell-size) * 2.42);
  --inventory-belt-width: calc(var(--inventory-cell-size) * 1.51);
  --inventory-belt-height: calc(var(--inventory-cell-size) * 1.03);
  --inventory-ring-size: calc(var(--inventory-cell-size) * 0.91);
  --inventory-amulet-size: var(--inventory-cell-size);
  position: relative;
  width: fit-content;
  max-width: 100%;
  margin-left: auto;
  margin-right: auto;
  padding: clamp(0.55rem, 1.4vw, 0.95rem);
  border: 1px solid rgba(126, 200, 227, 0.28);
  border-radius: clamp(0.22rem, 0.5vw, 0.34rem);
  background:
    radial-gradient(circle at top, rgba(233, 69, 96, 0.12), transparent 26%),
    linear-gradient(180deg, rgba(16, 24, 44, 0.98), rgba(9, 15, 29, 1)),
    linear-gradient(135deg, rgba(126, 200, 227, 0.05), transparent 42%);
  box-shadow:
    inset 0 0 0 1px rgba(126, 200, 227, 0.06),
    0 18px 34px rgba(0, 0, 0, 0.28);
}

.session-inventory-panel::before {
  content: '';
  position: absolute;
  inset: clamp(0.3rem, 0.8vw, 0.55rem);
  border: 1px solid rgba(126, 200, 227, 0.14);
  border-radius: clamp(0.16rem, 0.4vw, 0.28rem);
  pointer-events: none;
}

.session-inventory-panel__body {
  position: relative;
  z-index: 1;
}

.session-inventory-panel__body {
  display: flex;
  flex-direction: column;
  gap: clamp(0.45rem, 1vw, 0.8rem);
  padding: clamp(0.6rem, 1.5vw, 1rem);
}

.session-inventory-equipment {
  flex: 0 0 auto;
}

.session-inventory-equipment__columns {
  display: flex;
  width: fit-content;
  max-width: 100%;
  margin-left: auto;
  margin-right: auto;
  align-items: stretch;
  justify-content: center;
  gap: var(--inventory-gap);
}

.session-equipment-column {
  display: flex;
  flex-direction: column;
  height: auto;
  gap: var(--inventory-gap);
}

.session-equipment-column--weapon-gloves,
.session-equipment-column--offhand-boots {
  width: var(--inventory-weapon-column-width);
}

.session-equipment-column--trinket-left,
.session-equipment-column--charms-right {
  width: var(--inventory-trinket-column-width);
  align-items: center;
}

.session-equipment-column--trinket-left {
  justify-content: flex-end;
}

.session-equipment-column--helmet-armour-belt {
  width: var(--inventory-center-column-width);
}

.session-equipment-column--charms-right {
  justify-content: flex-end;
  gap: calc(var(--inventory-gap) * 1.4);
}

.session-equipment-column--offhand-boots {
  align-items: stretch;
}

.session-equipment-slot {
  min-height: 100%;
  overflow: hidden;
  border: 1px solid rgba(126, 200, 227, 0.34);
  border-radius: clamp(0.12rem, 0.3vw, 0.24rem);
  background: linear-gradient(180deg, rgba(30, 49, 96, 0.96), rgba(15, 25, 49, 0.98));
  box-shadow: inset 0 0 0 1px rgba(126, 200, 227, 0.06);
}

.session-equipment-slot--weapon-main,
.session-equipment-slot--offhand-main {
  width: 100%;
  height: var(--inventory-weapon-height);
}

.session-equipment-slot--gloves,
.session-equipment-slot--boots {
  width: 100%;
  height: var(--inventory-glove-height);
}

.session-equipment-slot--head {
  width: 100%;
  height: var(--inventory-head-size);
}

.session-equipment-slot--chest {
  width: 100%;
  height: var(--inventory-chest-height);
}

.session-equipment-slot--belt {
  width: 100%;
  height: var(--inventory-belt-height);
}

.session-equipment-slot--ring-left,
.session-equipment-slot--ring-right,
.session-equipment-slot--amulet {
  aspect-ratio: 1 / 1;
  min-height: var(--inventory-ring-size);
  max-width: var(--inventory-ring-size);
  border-radius: 999px;
}

.session-equipment-slot--amulet {
  min-height: var(--inventory-amulet-size);
  max-width: var(--inventory-amulet-size);
}

.session-equipment-slot--active {
  border-color: rgba(233, 69, 96, 0.88);
  box-shadow: inset 0 0 0 1px rgba(233, 69, 96, 0.18), 0 0 0 1px rgba(233, 69, 96, 0.12);
}

.session-equipment-slot--blocked {
  border-color: rgba(233, 69, 96, 0.92);
  background: linear-gradient(180deg, rgba(88, 27, 42, 0.88), rgba(45, 12, 23, 0.94));
}

.session-equipment-slot__placeholder {
  display: flex;
  height: 100%;
  align-items: center;
  justify-content: center;
  padding: clamp(0.25rem, 0.7vw, 0.45rem);
  text-align: center;
}

.session-equipment-slot__label {
  font-size: clamp(0.48rem, 1vw, 0.82rem);
  font-weight: 700;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: rgba(126, 200, 227, 0.62);
}

.session-inventory-grid-shell {
  width: fit-content;
  max-width: 100%;
  margin-left: auto;
  margin-right: auto;
}

.session-inventory-grid {
  display: grid;
  width: fit-content;
  grid-template-columns: repeat(var(--inventory-columns), minmax(0, var(--inventory-cell-size)));
  grid-template-rows: repeat(var(--inventory-rows), minmax(0, var(--inventory-cell-size)));
  gap: 0;
  padding: clamp(0.14rem, 0.35vw, 0.28rem);
  border: 1px solid rgba(126, 200, 227, 0.32);
  border-radius: clamp(0.1rem, 0.25vw, 0.18rem);
  background: linear-gradient(180deg, rgba(12, 25, 49, 0.98), rgba(8, 16, 31, 1));
  box-shadow: inset 0 0 0 1px rgba(126, 200, 227, 0.04);
}

.session-inventory-grid__cell,
.session-inventory-grid__item {
  min-width: 0;
  min-height: 0;
}

.session-inventory-grid__cell {
  height: 100%;
  border: 1px solid rgba(126, 200, 227, 0.18);
  border-radius: 0;
  background: linear-gradient(180deg, rgba(34, 55, 102, 0.58), rgba(15, 28, 57, 0.76));
}

.session-inventory-grid__cell--active {
  border-color: rgba(233, 69, 96, 0.82);
  background: linear-gradient(180deg, rgba(100, 34, 57, 0.62), rgba(66, 18, 37, 0.82));
}

.session-inventory-grid__cell--blocked {
  border-color: rgba(233, 69, 96, 0.82);
  background: linear-gradient(180deg, rgba(86, 25, 34, 0.64), rgba(49, 10, 18, 0.84));
}

.session-inventory-grid__item {
  position: relative;
  z-index: 1;
}

@media (max-width: 1023px) {
  .session-inventory-panel {
    --inventory-cell-size: clamp(1.9rem, 6.2vw, 3.85rem);
    width: 100%;
    padding: 0.7rem;
  }

  .session-inventory-panel__body {
    gap: 0.6rem;
    padding: 0.78rem;
  }
}

@media (max-width: 767px) {
  .session-inventory-panel {
    --inventory-cell-size: clamp(1.72rem, 8vw, 2.6rem);
    width: 100%;
    padding: 0.48rem;
  }

  .session-inventory-panel__body {
    gap: 0.4rem;
    padding: 0.55rem;
  }

  .session-inventory-grid-shell {
    width: 100%;
  }

  .session-inventory-grid {
    max-width: 100%;
  }
}

@media (max-width: 479px) {
  .session-inventory-panel {
    --inventory-cell-size: clamp(1.58rem, 8.8vw, 2.15rem);
  }

  .session-inventory-panel__body {
    padding: 0.42rem;
  }
}
</style>
