<script setup>
import { API_URL } from '@/api'
import { Eye } from '@lucide/vue'
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
})

const FILL_MS = 750
const RING_CIRCUMFERENCE = 94.25

const characterAttributes = inject('inventoryCharacterAttributes', null)
const attributesEnabled = inject('inventoryAttributesEnabled', null)

const rootRef = ref(null)
const hovering = ref(false)
const ringActive = ref(false)
const showInfo = ref(false)
const infoPos = ref({ left: 0, top: 0 })
let revealTimer = null

const RARITY_TEXT = {
  common: 'text-[#e2e8f0]',
  uncommon: 'text-[#fde68a]',
  rare: 'text-[#93c5fd]',
  epic: 'text-[#e9d5ff]',
  masterwork: 'text-[#fdba74]',
  legendary: 'text-[#86efac]',
  unique: 'text-[#fca5a5]',
}

const item = computed(() => props.entry?.item ?? {})
const itemImageUrl = computed(() => item.value?.image_id ? `${API_URL}/api/uploads/${item.value.image_id}` : '')
const itemInitial = computed(() => (item.value?.name || '?').charAt(0).toUpperCase())
const sizeLabel = computed(() => `${item.value?.grid_width || 1}x${item.value?.grid_height || 1}`)
const descriptionPreview = computed(() => {
  const text = String(item.value?.description || '').trim()
  if (!text) return ''
  return text.length > 140 ? `${text.slice(0, 137).trimEnd()}...` : text
})
const requirementChecks = computed(() => {
  if (!(attributesEnabled?.value ?? true)) return []
  return (item.value?.required_attributes ?? []).map((requirement) => {
    const name = String(requirement?.attribute_name || '')
    const current = Number(characterAttributes?.value?.[name] ?? 0)
    const required = Number(requirement?.min_value ?? 0)
    return { name, label: formatAttr(name), required, current, met: current >= required }
  })
})
const modifierList = computed(() => ((attributesEnabled?.value ?? true) ? (item.value?.attribute_modifiers ?? []) : []).map((modifier) => ({
  label: formatAttr(modifier?.attribute_name),
  value: Number(modifier?.modifier_value ?? 0),
  percent: Boolean(modifier?.is_percentage),
})))
const meetsRequirements = computed(() => requirementChecks.value.every((requirement) => requirement.met))
const rarityLabel = computed(() => formatAttr(item.value?.rarity || 'common'))
const rarityTextClass = computed(() => RARITY_TEXT[item.value?.rarity] || RARITY_TEXT.common)
const rarityBorderClass = computed(() => {
  const variants = {
    common: 'border-[rgba(226,232,240,0.95)] shadow-[inset_0_0_10px_rgba(226,232,240,0.25),0_6px_14px_rgba(0,0,0,0.3)]',
    uncommon: 'border-[rgba(250,204,21,1)] shadow-[inset_0_0_10px_rgba(250,204,21,0.3),0_6px_14px_rgba(0,0,0,0.3)]',
    rare: 'border-[rgba(96,165,250,1)] shadow-[inset_0_0_10px_rgba(96,165,250,0.3),0_6px_14px_rgba(0,0,0,0.3)]',
    epic: 'border-[rgba(192,132,252,1)] shadow-[inset_0_0_10px_rgba(192,132,252,0.3),0_6px_14px_rgba(0,0,0,0.3)]',
    masterwork: 'border-[rgba(251,146,60,1)] shadow-[inset_0_0_10px_rgba(251,146,60,0.3),0_6px_14px_rgba(0,0,0,0.3)]',
    legendary: 'border-[rgba(74,222,128,1)] shadow-[inset_0_0_10px_rgba(74,222,128,0.3),0_6px_14px_rgba(0,0,0,0.3)]',
    unique: 'border-[rgba(247,118,118,1)] shadow-[inset_0_0_10px_rgba(247,118,118,0.3),0_6px_14px_rgba(0,0,0,0.3)]',
  }

  return variants[item.value?.rarity] || variants.common
})

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
      class="relative h-full w-full overflow-hidden rounded-[0.12rem] border-[3px] bg-[linear-gradient(180deg,rgba(27,49,93,0.98),rgba(13,24,47,1)),linear-gradient(135deg,rgba(106,147,211,0.2),rgba(7,12,24,0))]"
      :class="rarityBorderClass"
    >
      <div class="flex h-full w-full min-h-0 items-center justify-center bg-[radial-gradient(circle_at_top,rgba(255,247,226,0.2),transparent_46%),linear-gradient(180deg,rgba(222,211,182,0.9),rgba(170,151,104,0.78))]">
        <img
          v-if="itemImageUrl"
          :src="itemImageUrl"
          :alt="item.name"
          draggable="false"
          class="h-full w-full object-cover"
        />
        <span v-else class="font-[Cinzel] text-[clamp(1rem,1.5vw,1.45rem)] font-bold text-[rgba(23,36,66,0.92)]">{{ itemInitial }}</span>
      </div>

      <span class="absolute bottom-[0.22rem] left-[0.22rem] z-[1] inline-flex h-[1rem] min-w-[1rem] items-center justify-center rounded-full border border-[rgba(205,217,242,0.12)] bg-[rgba(7,14,28,0.72)] px-[0.28rem] text-[0.5rem] font-bold uppercase tracking-[0.08em] text-[rgba(225,237,255,0.84)]">{{ sizeLabel }}</span>
      <span v-if="entry.quantity > 1" class="absolute bottom-[0.22rem] right-[0.22rem] z-[1] inline-flex h-[1rem] min-w-[1rem] items-center justify-center rounded-full border border-[rgba(205,217,242,0.12)] bg-[rgba(7,14,28,0.72)] px-[0.28rem] text-[0.5rem] font-bold uppercase tracking-[0.08em] text-[rgba(225,237,255,0.84)]">x{{ entry.quantity }}</span>
      <span v-if="entry.enchantment" class="absolute right-[0.22rem] top-[0.22rem] z-[1] inline-flex h-[1rem] min-w-[1rem] items-center justify-center rounded-full border border-[rgba(205,217,242,0.12)] bg-[rgba(7,14,28,0.72)] px-[0.28rem] text-[0.5rem] font-bold uppercase tracking-[0.08em] text-[rgba(225,237,255,0.84)]">+{{ entry.enchantment }}</span>

      <!-- Hover inspect indicator: eye + filling progress ring -->
      <div
        v-show="hovering"
        class="pointer-events-none absolute bottom-[0.22rem] right-[0.22rem] z-[2] h-[2.1rem] w-[2.1rem]"
      >
        <svg viewBox="0 0 36 36" class="absolute inset-0 h-full w-full -rotate-90">
          <circle cx="18" cy="18" r="15" fill="rgba(7,12,24,0.85)" stroke="rgba(255,255,255,0.28)" stroke-width="4" />
          <circle
            cx="18"
            cy="18"
            r="15"
            fill="none"
            stroke="rgba(126,200,227,1)"
            stroke-width="4"
            stroke-linecap="round"
            :stroke-dasharray="RING_CIRCUMFERENCE"
            :stroke-dashoffset="ringActive ? 0 : RING_CIRCUMFERENCE"
            :style="{ transition: `stroke-dashoffset ${FILL_MS}ms linear` }"
          />
        </svg>
        <span class="absolute inset-0 flex items-center justify-center">
          <Eye class="h-[58%] w-[58%] text-[#e8f4fb]" :stroke-width="2.5" />
        </span>
      </div>
    </div>

    <Teleport to="body">
      <div
        v-if="showInfo"
        class="fixed z-[12000] w-[16rem] overflow-hidden rounded-[1rem] border border-[rgba(126,200,227,0.2)] bg-[linear-gradient(180deg,rgba(9,18,34,0.98),rgba(5,10,22,0.98))] p-4 shadow-[0_24px_60px_rgba(0,0,0,0.5)]"
        :style="{ left: `${infoPos.left}px`, top: `${infoPos.top}px` }"
      >
        <div class="flex items-start justify-between gap-2">
          <h4 class="break-words text-[14px] font-semibold leading-snug text-[#f6f7fb] [overflow-wrap:anywhere]">{{ item.name || 'Unnamed item' }}</h4>
          <span class="shrink-0 text-[10px] font-bold uppercase tracking-[0.16em]" :class="rarityTextClass">{{ rarityLabel }}</span>
        </div>
        <p v-if="descriptionPreview" class="mt-1.5 break-words whitespace-pre-line text-[12px] leading-relaxed text-[#d8dce7]/62">{{ descriptionPreview }}</p>

        <div v-if="requirementChecks.length" class="mt-3">
          <p class="text-[10px] uppercase tracking-[0.18em] text-[#7ec8e3]/55">Requirements</p>
          <ul class="mt-1.5 space-y-1">
            <li
              v-for="requirement in requirementChecks"
              :key="`req-${requirement.name}`"
              class="flex items-center justify-between gap-2 text-[12px]"
              :class="requirement.met ? 'text-[#86efac]' : 'text-[#fca5a5]'"
            >
              <span>{{ requirement.label }} {{ requirement.required }}</span>
              <span class="text-[11px] opacity-80">you: {{ requirement.current }}</span>
            </li>
          </ul>
          <p v-if="!meetsRequirements" class="mt-1.5 text-[11px] font-semibold text-[#fca5a5]">Requirements not met</p>
        </div>

        <div v-if="modifierList.length" class="mt-3">
          <p class="text-[10px] uppercase tracking-[0.18em] text-[#7ec8e3]/55">Grants</p>
          <ul class="mt-1.5 space-y-1">
            <li
              v-for="(modifier, index) in modifierList"
              :key="`mod-${index}`"
              class="flex items-center justify-between gap-2 text-[12px] text-[#d8dce7]/82"
            >
              <span>{{ modifier.label }}</span>
              <span class="font-semibold" :class="modifier.value >= 0 ? 'text-[#8fd7ef]' : 'text-[#fca5a5]'">
                {{ modifier.value >= 0 ? '+' : '' }}{{ modifier.value }}{{ modifier.percent ? '%' : '' }}
              </span>
            </li>
          </ul>
        </div>

        <p v-if="!requirementChecks.length && !modifierList.length && !descriptionPreview" class="mt-2 text-[12px] text-[#d8dce7]/50">No additional details.</p>
      </div>
    </Teleport>
  </div>
</template>
