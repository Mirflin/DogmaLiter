<script setup>
import { API_URL } from '@/api'
import { Eye } from '@lucide/vue'
import { useImageFallback } from '@/composables/useImageFallback'
import { iconForItem } from '@/assets/icons/itemIcons'
import { computed, inject, onBeforeUnmount, ref } from 'vue'

const props = defineProps({
  entry: {
    type: Object,
    required: true,
  },
  variant: {
    type: String,
    default: 'grid',
  },
  round: {
    type: Boolean,
    default: false,
  },
})

const FILL_MS = 750
const RING_CIRCUMFERENCE = 94.25

const characterAttributes = inject('inventoryCharacterAttributes', null)
const attributesEnabled = inject('inventoryAttributesEnabled', null)
const disabledStandardAttrs = inject('inventoryDisabledStandardAttrs', null)

function isAttrDisabled(name) {
  const list = disabledStandardAttrs?.value ?? []
  return list.includes(String(name || '').trim().toLowerCase())
}

const rootRef = ref(null)
const hovering = ref(false)
const ringActive = ref(false)
const showInfo = ref(false)
const infoPos = ref({ left: 0, top: 0 })
let revealTimer = null

const RARITY_TEXT = {
  common: 'text-ink',
  uncommon: 'text-warning',
  rare: 'text-sky',
  epic: 'text-r-epic',
  masterwork: 'text-r-masterwork',
  legendary: 'text-success',
  unique: 'text-danger',
}

const { markFailed: markImageFailed, isFailed: isImageFailed } = useImageFallback()

const item = computed(() => props.entry?.item ?? {})
const itemImageUrl = computed(() => {
  if (!item.value?.image_id) return ''
  const url = `${API_URL}/api/uploads/${item.value.image_id}`
  return isImageFailed(url) ? '' : url
})
const descriptionPreview = computed(() => {
  const text = String(item.value?.description || '').trim()
  if (!text) return ''
  return text.length > 140 ? `${text.slice(0, 137).trimEnd()}...` : text
})
const requirementChecks = computed(() => {
  return (item.value?.required_attributes ?? [])
    .filter((requirement) => !isAttrDisabled(requirement?.attribute_name))
    .map((requirement) => {
      const name = String(requirement?.attribute_name || '')
      const current = Number(characterAttributes?.value?.[name] ?? 0)
      const required = Number(requirement?.min_value ?? 0)
      return { name, label: formatAttr(name), required, current, met: current >= required }
    })
})
const modifierList = computed(() => (item.value?.attribute_modifiers ?? [])
  .filter((modifier) => !isAttrDisabled(modifier?.attribute_name))
  .map((modifier) => ({
    label: formatAttr(modifier?.attribute_name),
    value: Number(modifier?.modifier_value ?? 0),
    percent: Boolean(modifier?.is_percentage),
  })))
const itemTags = computed(() => item.value?.tags ?? [])
const meetsRequirements = computed(() => requirementChecks.value.every((requirement) => requirement.met))
const rarityLabel = computed(() => formatAttr(item.value?.rarity || 'common'))
const rarityTextClass = computed(() => RARITY_TEXT[item.value?.rarity] || RARITY_TEXT.common)
// Resin colourway per rarity: the tile is tinted glass of its tier, as on the landing demo.
const resinColor = computed(() => `var(--color-r-${item.value?.rarity || 'common'})`)
const fallbackIcon = computed(() => iconForItem(item.value))

function formatAttr(value) {
  return String(value || '')
    .split('_')
    .filter(Boolean)
    .map(part => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ')
}

function onEnter() {
  hovering.value = true
  ringActive.value = false
  clearTimeout(revealTimer)
  revealTimer = setTimeout(revealInfo, FILL_MS)
  // Force a frame with the ring empty, then flip so the stroke transition animates.
  requestAnimationFrame(() => requestAnimationFrame(() => {
    if (hovering.value) ringActive.value = true
  }))
}

function onLeave() {
  resetHover()
}

function resetHover() {
  hovering.value = false
  ringActive.value = false
  clearTimeout(revealTimer)
  showInfo.value = false
}

function revealInfo() {
  const element = rootRef.value
  if (!element) return

  const rect = element.getBoundingClientRect()
  const width = 256
  const gap = 10
  const onLeftSide = rect.left > window.innerWidth / 2
  let left = onLeftSide ? rect.left - width - gap : rect.right + gap
  left = Math.max(8, Math.min(left, window.innerWidth - width - 8))
  const top = Math.max(8, Math.min(rect.top, window.innerHeight - 8 - 280))

  infoPos.value = { left, top }
  showInfo.value = true
}

onBeforeUnmount(() => {
  clearTimeout(revealTimer)
})
</script>

<template>
  <div
    ref="rootRef"
    class="session-inventory-item group relative h-full w-full cursor-grab touch-none select-none outline-none transition-transform duration-150 ease-out hover:scale-[1.05]"
    :data-entry-id="entry.id"
    tabindex="0"
    @mouseenter="onEnter"
    @mouseleave="onLeave"
    @pointerdown="resetHover"
  >
    <div
      class="inv-tile"
      :class="{ 'is-round': round, 'is-locked': !meetsRequirements }"
      :style="{ '--resin': resinColor }"
    >
      <img
        v-if="itemImageUrl"
        :src="itemImageUrl"
        :alt="item.name"
        draggable="false"
        class="inv-tile__image"
        @error="markImageFailed(itemImageUrl)"
      />
      <svg v-else class="inv-tile__icon" viewBox="0 0 512 512" aria-hidden="true"><path :d="fallbackIcon" /></svg>

      <span v-if="variant === 'grid' && !round" class="inv-tile__name">{{ item.name }}</span>
      <span v-if="entry.quantity > 1" class="inv-tile__chip inv-tile__chip--qty">×{{ entry.quantity }}</span>
      <span v-if="entry.enchantment" class="inv-tile__chip inv-tile__chip--ench">+{{ entry.enchantment }}</span>
      <span v-if="entry.max_durability" class="inv-tile__dur" aria-hidden="true"><i :style="{ width: `${Math.max(0, Math.min(100, (entry.durability / entry.max_durability) * 100))}%` }" /></span>

      <!-- Hover inspect indicator: eye + filling progress ring -->
      <div
        v-show="hovering"
        class="pointer-events-none absolute bottom-[0.22rem] right-[0.22rem] z-[2] h-[2.1rem] w-[2.1rem]"
      >
        <svg viewBox="0 0 36 36" class="absolute inset-0 h-full w-full -rotate-90">
          <circle cx="18" cy="18" r="15" fill="rgba(6,10,28,0.85)" stroke="rgba(255,255,255,0.22)" stroke-width="4" />
          <circle
            cx="18"
            cy="18"
            r="15"
            fill="none"
            stroke="var(--color-primary-ink)"
            stroke-width="4"
            stroke-linecap="round"
            :stroke-dasharray="RING_CIRCUMFERENCE"
            :stroke-dashoffset="ringActive ? 0 : RING_CIRCUMFERENCE"
            :style="{ transition: `stroke-dashoffset ${FILL_MS}ms linear` }"
          />
        </svg>
        <span class="absolute inset-0 flex items-center justify-center">
          <Eye class="h-[58%] w-[58%] text-ink" :stroke-width="2.5" />
        </span>
      </div>
    </div>

    <Teleport to="body">
      <div
        v-if="showInfo"
        class="fixed z-[12990] w-[16rem] overflow-hidden rounded-[1rem] border border-line bg-[linear-gradient(180deg,rgba(10,16,41,0.98),rgba(6,10,28,0.98))] p-4 shadow-[0_24px_60px_rgba(0,0,0,0.5)]"
        :style="{ left: `${infoPos.left}px`, top: `${infoPos.top}px` }"
      >
        <div class="flex items-start justify-between gap-2">
          <h4 class="break-words text-[14px] font-semibold leading-snug text-ink [overflow-wrap:anywhere]">{{ item.name || 'Unnamed item' }}</h4>
          <span class="shrink-0 text-[10px] font-bold uppercase tracking-[0.16em]" :class="rarityTextClass">{{ rarityLabel }}</span>
        </div>
        <p v-if="descriptionPreview" class="mt-1.5 break-words whitespace-pre-line text-[12px] leading-relaxed text-ink-soft/62">{{ descriptionPreview }}</p>

        <div v-if="requirementChecks.length" class="mt-3">
          <p class="text-[13px] font-semibold text-dim">Requirements</p>
          <ul class="mt-1.5 space-y-1">
            <li
              v-for="requirement in requirementChecks"
              :key="`req-${requirement.name}`"
              class="flex items-center justify-between gap-2 text-[12px]"
              :class="requirement.met ? 'text-success' : 'text-danger'"
            >
              <span>{{ requirement.label }} {{ requirement.required }}</span>
              <span class="text-[11px] opacity-80">you: {{ requirement.current }}</span>
            </li>
          </ul>
          <p v-if="!meetsRequirements" class="mt-1.5 text-[11px] font-semibold text-danger">Requirements not met</p>
        </div>

        <div v-if="modifierList.length" class="mt-3">
          <p class="text-[13px] font-semibold text-dim">Grants</p>
          <ul class="mt-1.5 space-y-1">
            <li
              v-for="(modifier, index) in modifierList"
              :key="`mod-${index}`"
              class="flex items-center justify-between gap-2 text-[12px] text-ink-soft/82"
            >
              <span>{{ modifier.label }}</span>
              <span class="font-semibold" :class="modifier.value >= 0 ? 'text-sky' : 'text-danger'">
                {{ modifier.value >= 0 ? '+' : '' }}{{ modifier.value }}{{ modifier.percent ? '%' : '' }}
              </span>
            </li>
          </ul>
        </div>

        <div v-if="itemTags.length" class="mt-3 flex flex-wrap gap-1.5">
          <span v-for="tag in itemTags" :key="tag" class="rounded-full border border-primary/18 bg-primary/8 px-2 py-0.5 text-[11px] text-primary-soft">{{ tag }}</span>
        </div>

        <p v-if="!requirementChecks.length && !modifierList.length && !itemTags.length && !descriptionPreview" class="mt-2 text-[12px] text-ink-soft/50">No additional details.</p>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.inv-tile {
  position: relative;
  height: 100%;
  width: 100%;
  overflow: hidden;
  border-radius: 0.5rem;
  color: color-mix(in oklab, var(--resin) 35%, white);
  background: color-mix(in oklab, var(--resin) 24%, var(--color-n2));
  box-shadow: inset 0 0 0 1.5px color-mix(in oklab, var(--resin) 70%, transparent), inset 0 1px 0 rgb(255 255 255 / 0.08);
  transition: background-color 0.2s ease, box-shadow 0.2s ease;
}

.inv-tile.is-round {
  border-radius: 999px;
}

.session-inventory-item:hover .inv-tile,
.session-inventory-item:focus-visible .inv-tile {
  background: color-mix(in oklab, var(--resin) 34%, var(--color-n2));
  box-shadow: inset 0 0 0 2px var(--resin), inset 0 1px 0 rgb(255 255 255 / 0.12);
}

/* Gear the character can't wear yet stays greyed out. */
.inv-tile.is-locked {
  filter: saturate(0.25) brightness(0.75);
}

.inv-tile__image {
  height: 100%;
  width: 100%;
  object-fit: cover;
}

.inv-tile__icon {
  position: absolute;
  left: 50%;
  top: 50%;
  width: min(72%, 4.5rem);
  aspect-ratio: 1;
  transform: translate(-50%, -50%);
  fill: currentColor;
  opacity: 0.55;
}

.inv-tile__name {
  position: absolute;
  left: 0.4rem;
  right: 0.4rem;
  top: 0.3rem;
  overflow: hidden;
  font-size: clamp(0.6rem, 0.9vw, 0.78rem);
  font-weight: 600;
  line-height: 1.15;
  text-overflow: ellipsis;
  white-space: nowrap;
  text-shadow: 0 1px 2px rgb(0 0 0 / 0.5);
}

.inv-tile__chip {
  position: absolute;
  z-index: 1;
  padding: 0 0.3rem;
  border-radius: 999px;
  font-family: var(--font-num);
  font-size: 0.72rem;
  font-weight: 600;
  line-height: 1.1rem;
  color: var(--color-ink);
  background: rgb(6 10 28 / 0.72);
}

.inv-tile__chip--qty {
  right: 0.25rem;
  bottom: 0.35rem;
}

.inv-tile__chip--ench {
  left: 0.25rem;
  bottom: 0.35rem;
  color: var(--color-orchid);
}

.inv-tile__dur {
  position: absolute;
  left: 0.3rem;
  right: 0.3rem;
  bottom: 0.2rem;
  height: 3px;
  border-radius: 2px;
  background: rgb(255 255 255 / 0.14);
}

.inv-tile__dur i {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: currentColor;
}
</style>
