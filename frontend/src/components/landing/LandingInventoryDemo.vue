<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'

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

const items = [
  { id: 'bow', name: 'Longbow of the Gull', short: 'Longbow', rarity: 'rare', x: 0, y: 0, w: 1, h: 3, durability: 80, mods: ['+1 DEX'], req: 'DEX 13', slot: 'Main hand' },
  { id: 'armor', name: 'Studded Leather', rarity: 'uncommon', x: 1, y: 0, w: 2, h: 3, durability: 55, mods: ['+1 CON'], slot: 'Chest' },
  { id: 'potion', name: 'Potion of Healing', short: 'PH', rarity: 'common', x: 3, y: 0, w: 1, h: 1, qty: 3 },
  { id: 'arrows', name: 'Bundle of Arrows', short: 'Arrows', rarity: 'common', x: 4, y: 0, w: 2, h: 1, qty: 46 },
  { id: 'ring', name: 'Ember Ring', short: 'ER', rarity: 'epic', x: 3, y: 1, w: 1, h: 1, mods: ['+2 CHA'], enchant: 2, slot: 'Ring' },
  { id: 'amulet', name: 'Dragonbone Amulet', short: 'DA', rarity: 'legendary', x: 4, y: 1, w: 1, h: 1, mods: ['+2 WIS', '+1 CON'], req: 'WIS 14', slot: 'Amulet' },
  { id: 'shield', name: "Warden's Greatshield", short: 'Greatshield', rarity: 'unique', x: 6, y: 0, w: 2, h: 2, mods: ['+2 CON'], req: 'STR 16', locked: true, slot: 'Off hand' },
  { id: 'rope', name: 'Rope, 50 ft', short: 'Rope', rarity: 'common', x: 0, y: 3, w: 2, h: 1 },
  { id: 'dagger', name: 'Moon Dagger', short: 'Dagger', rarity: 'rare', x: 2, y: 3, w: 1, h: 2, durability: 92, mods: ['+1 DEX'], slot: 'Main hand' },
  { id: 'cloak', name: 'Cloak of the Tide', short: 'Cloak', rarity: 'masterwork', x: 6, y: 3, w: 2, h: 2, mods: ['+1 WIS'] },
]

const selectedId = ref('bow')
const selected = computed(() => items.find((item) => item.id === selectedId.value))
const used = items.reduce((sum, item) => sum + item.w * item.h, 0)

const root = ref(null)
const armed = ref(false)
const dropped = ref(false)
let observer = null

onMounted(() => {
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

onBeforeUnmount(() => observer?.disconnect())

function tileStyle(item, index) {
  return {
    gridColumn: `${item.x + 1} / span ${item.w}`,
    gridRow: `${item.y + 1} / span ${item.h}`,
    '--resin': RARITY[item.rarity].color,
    '--delay': `${index * 85}ms`,
  }
}
</script>

<template>
  <div ref="root" class="demo" :class="{ 'demo--armed': armed, 'demo--dropped': dropped }">
    <div class="demo__board">
      <div class="demo__meta">
        <span class="text-ink font-semibold">Kaela Thornwood</span>
        <span class="text-faint">Backpack</span>
        <span class="ml-auto num text-[15px] text-dim"><span class="text-ink">{{ used }}</span> / {{ COLS * ROWS }} cells</span>
      </div>

      <div class="demo__grid" :style="{ '--cols': COLS, '--rows': ROWS }" role="list" aria-label="Example inventory">
        <span
          v-for="n in COLS * ROWS"
          :key="`c${n}`"
          class="demo__cell"
          :style="{ gridColumn: ((n - 1) % COLS) + 1, gridRow: Math.floor((n - 1) / COLS) + 1 }"
          aria-hidden="true"
        />
        <button
          v-for="(item, index) in items"
          :key="item.id"
          type="button"
          role="listitem"
          class="demo__item"
          :class="{ 'is-selected': item.id === selectedId, 'is-locked': item.locked, 'is-small': item.w * item.h === 1 }"
          :style="tileStyle(item, index)"
          :aria-label="`${item.name}, ${RARITY[item.rarity].label}${item.locked ? ', requirements not met' : ''}`"
          @pointerenter="selectedId = item.id"
          @focus="selectedId = item.id"
          @click="selectedId = item.id"
        >
          <span class="demo__name">{{ item.short || item.name }}</span>
          <span v-if="item.locked" class="demo__lock">Needs {{ item.req }}</span>
          <span v-if="item.qty" class="demo__qty num">×{{ item.qty }}</span>
          <span v-if="item.durability" class="demo__dur" aria-hidden="true"><i :style="{ width: `${item.durability}%` }" /></span>
        </button>
      </div>
    </div>

    <aside class="demo__inspector" aria-live="polite">
      <p class="text-[13px] font-semibold" :style="{ color: RARITY[selected.rarity].color }">{{ RARITY[selected.rarity].label }}</p>
      <h3 class="mt-1 text-[22px] leading-tight font-semibold text-ink">{{ selected.name }}</h3>
      <dl class="demo__facts">
        <div><dt>Size</dt><dd class="num">{{ selected.w }} × {{ selected.h }}</dd></div>
        <div v-if="selected.slot"><dt>Slot</dt><dd>{{ selected.slot }}</dd></div>
        <div v-if="selected.qty"><dt>Stack</dt><dd class="num">{{ selected.qty }}</dd></div>
        <div v-if="selected.durability"><dt>Durability</dt><dd class="num">{{ selected.durability }} / 100</dd></div>
        <div v-if="selected.enchant"><dt>Enchantment</dt><dd class="num">+{{ selected.enchant }}</dd></div>
        <div v-if="selected.mods"><dt>Modifiers</dt><dd class="text-orchid">{{ selected.mods.join(', ') }}</dd></div>
        <div v-if="selected.req"><dt>Requires</dt><dd :class="selected.locked ? 'text-danger' : ''">{{ selected.req }}<template v-if="selected.locked"> · Kaela has 12</template></dd></div>
      </dl>
    </aside>
  </div>
</template>

<style scoped>
.demo {
  display: grid;
  grid-template-columns: max-content minmax(15rem, 1fr);
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
  align-items: baseline;
  gap: 0.6rem;
  padding: 0.1rem 0.25rem 0.85rem;
  font-size: 14px;
}

.demo__grid {
  --gap: 4px;
  --cell: min(84px, calc((100vw - 4.5rem - (var(--cols) - 1) * var(--gap)) / var(--cols)));
  display: grid;
  grid-template-columns: repeat(var(--cols), var(--cell));
  grid-template-rows: repeat(var(--rows), var(--cell));
  gap: var(--gap);
  width: max-content;
  max-width: 100%;
}

.demo__cell {
  border-radius: 7px;
  background: var(--color-n2);
}

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
  cursor: pointer;
  overflow: hidden;
  transition: background-color 0.2s ease, box-shadow 0.2s ease, transform 0.2s cubic-bezier(0.16, 1, 0.3, 1);
}

.demo__item:hover,
.demo__item.is-selected {
  background: color-mix(in oklab, var(--resin) 34%, var(--color-n2));
  box-shadow: inset 0 0 0 1.5px var(--resin), inset 0 1px 0 rgb(255 255 255 / 0.12);
}

.demo__item:active {
  transform: scale(0.97);
}

.demo__item:focus-visible {
  outline: 2px solid var(--color-primary-ink);
  outline-offset: 2px;
}

.demo__item.is-locked {
  filter: saturate(0.2) brightness(0.72);
}

.demo__lock {
  font-size: 12px;
  font-weight: 600;
  color: var(--color-ink-soft);
}

.demo__name {
  font-size: 13px;
  font-weight: 600;
  line-height: 1.2;
}

.is-small .demo__name {
  font-family: var(--font-num);
  font-size: 17px;
  letter-spacing: 0.02em;
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

/* Items drop into the tray and settle, one after another. */
.demo--armed .demo__item {
  opacity: 0;
  transform: translateY(-22px) rotate(-7deg) scale(1.05);
}

.demo--armed.demo--dropped .demo__item {
  animation: settle 0.62s cubic-bezier(0.16, 1, 0.3, 1) var(--delay) both;
}

@keyframes settle {
  0% { opacity: 0; transform: translateY(-22px) rotate(-7deg) scale(1.05); }
  45% { opacity: 1; }
  78% { transform: translateY(0) rotate(1.2deg) scale(1); }
  100% { opacity: 1; transform: none; }
}

.demo--dropped .demo__item.is-locked {
  animation-name: settle-locked;
}

@keyframes settle-locked {
  0% { opacity: 0; transform: translateY(-22px) rotate(-7deg) scale(1.05); }
  78% { transform: translateY(0) rotate(1.2deg) scale(1); }
  100% { opacity: 1; transform: none; }
}

.demo__inspector {
  border-radius: 1.25rem;
  background: var(--color-n2);
  padding: 1.25rem 1.25rem 1.1rem;
  min-height: 100%;
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

@media (max-width: 900px) {
  .demo {
    grid-template-columns: minmax(0, 1fr);
  }

  .demo__board {
    width: max-content;
    max-width: 100%;
  }
}

@media (max-width: 520px) {
  .demo__board {
    padding: 0.75rem;
  }

  .demo__grid {
    --gap: 3px;
    --cell: calc((100vw - 3.5rem - (var(--cols) - 1) * var(--gap)) / var(--cols));
  }

  .demo__item {
    padding: 4px 5px;
    border-radius: 7px;
  }

  .demo__name {
    font-size: 10.5px;
  }

  .is-small .demo__name {
    font-size: 12px;
  }
}
</style>
