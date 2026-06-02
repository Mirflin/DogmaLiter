<script setup>
import { API_URL } from '@/api'
import { computed } from 'vue'

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

const item = computed(() => props.entry?.item ?? {})
const itemImageUrl = computed(() => item.value?.image_id ? `${API_URL}/api/uploads/${item.value.image_id}` : '')
const itemInitial = computed(() => (item.value?.name || '?').charAt(0).toUpperCase())
const sizeLabel = computed(() => `${item.value?.grid_width || 1}x${item.value?.grid_height || 1}`)
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
</script>

<template>
  <div
    class="session-inventory-item h-full w-full cursor-grab touch-none select-none outline-none transition-transform duration-150 ease-out hover:scale-[1.05]"
    :data-entry-id="entry.id"
    tabindex="0"
    :title="item.name || 'Unnamed item'"
  >
    <div
      class="relative h-full w-full overflow-hidden rounded-[0.12rem] border-2 bg-[linear-gradient(180deg,rgba(27,49,93,0.98),rgba(13,24,47,1)),linear-gradient(135deg,rgba(106,147,211,0.2),rgba(7,12,24,0))]"
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
    </div>
  </div>
</template>
