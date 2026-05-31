<script setup>
import { API_URL } from '@/api'
import { makeDraggable } from '@vue-dnd-kit/core'
import { computed, ref } from 'vue'

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

const itemRef = ref(null)

const item = computed(() => props.entry?.item ?? {})
const itemImageUrl = computed(() => item.value?.image_id ? `${API_URL}/api/uploads/${item.value.image_id}` : '')
const itemInitial = computed(() => (item.value?.name || '?').charAt(0).toUpperCase())
const sizeLabel = computed(() => `${item.value?.grid_width || 1}x${item.value?.grid_height || 1}`)
const rarityClass = computed(() => {
  const variants = {
    common: 'session-inventory-item--common',
    uncommon: 'session-inventory-item--uncommon',
    rare: 'session-inventory-item--rare',
    epic: 'session-inventory-item--epic',
    masterwork: 'session-inventory-item--masterwork',
    legendary: 'session-inventory-item--legendary',
    unique: 'session-inventory-item--unique',
  }

  return variants[item.value?.rarity] || variants.common
})

const { isDragging } = makeDraggable(itemRef, {
  id: computed(() => props.entry?.id),
  groups: ['inventory-placement'],
  data: () => props.entry,
  activation: { distance: 6 },
})
</script>

<template>
  <div
    ref="itemRef"
    class="session-inventory-item"
    :class="[
      `session-inventory-item--${variant}`,
      rarityClass,
      { 'session-inventory-item--dragging': isDragging },
    ]"
    tabindex="0"
    :title="item.name || 'Unnamed item'"
  >
    <div class="session-inventory-item__frame">
      <div class="session-inventory-item__art">
        <img v-if="itemImageUrl" :src="itemImageUrl" :alt="item.name" class="session-inventory-item__image" />
        <span v-else class="session-inventory-item__glyph">{{ itemInitial }}</span>
      </div>

      <span class="session-inventory-item__badge session-inventory-item__badge--size">{{ sizeLabel }}</span>
      <span v-if="entry.quantity > 1" class="session-inventory-item__badge session-inventory-item__badge--quantity">x{{ entry.quantity }}</span>
      <span v-if="entry.enchantment" class="session-inventory-item__badge session-inventory-item__badge--enchantment">+{{ entry.enchantment }}</span>
    </div>
  </div>
</template>

<style scoped>
.session-inventory-item {
  width: 100%;
  height: 100%;
  cursor: grab;
  user-select: none;
  touch-action: none;
  outline: none;
}

.session-inventory-item--dragging {
  opacity: 0.5;
}

.session-inventory-item__frame {
  position: relative;
  width: 100%;
  height: 100%;
  overflow: hidden;
  border-radius: 0.12rem;
  border: 1px solid rgba(94, 122, 174, 0.8);
  background:
    linear-gradient(180deg, rgba(27, 49, 93, 0.98), rgba(13, 24, 47, 1)),
    linear-gradient(135deg, rgba(106, 147, 211, 0.2), rgba(7, 12, 24, 0));
  box-shadow:
    inset 0 0 0 1px rgba(171, 204, 255, 0.08),
    0 8px 16px rgba(0, 0, 0, 0.18);
}

.session-inventory-item__frame::after {
  content: '';
  position: absolute;
  inset: 0.16rem;
  border: 1px solid rgba(151, 188, 245, 0.1);
  border-radius: 0.08rem;
  pointer-events: none;
}

.session-inventory-item__art {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  min-height: 0;
  border-radius: 0.05rem;
  background:
    radial-gradient(circle at top, rgba(255, 247, 226, 0.2), transparent 46%),
    linear-gradient(180deg, rgba(222, 211, 182, 0.9), rgba(170, 151, 104, 0.78));
}

.session-inventory-item__image {
  width: 100%;
  height: 100%;
  object-fit: contain;
  padding: 0.3rem;
}

.session-inventory-item__glyph {
  font-family: Cinzel, serif;
  font-size: clamp(1rem, 1.5vw, 1.45rem);
  font-weight: 700;
  color: rgba(23, 36, 66, 0.92);
}

.session-inventory-item__badge {
  position: absolute;
  z-index: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 1rem;
  height: 1rem;
  border-radius: 999px;
  border: 1px solid rgba(205, 217, 242, 0.12);
  background: rgba(7, 14, 28, 0.72);
  padding: 0 0.28rem;
  font-size: 0.5rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: rgba(225, 237, 255, 0.84);
}

.session-inventory-item__badge--size {
  left: 0.22rem;
  bottom: 0.22rem;
}

.session-inventory-item__badge--quantity {
  right: 0.22rem;
  bottom: 0.22rem;
}

.session-inventory-item__badge--enchantment {
  top: 0.22rem;
  right: 0.22rem;
}

.session-inventory-item--equipment .session-inventory-item__image {
  padding: 0.36rem;
}

.session-inventory-item--common .session-inventory-item__frame {
  border-color: rgba(255, 255, 255, 0.72);
}

.session-inventory-item--uncommon .session-inventory-item__frame {
  border-color: rgba(250, 204, 21, 0.78);
}

.session-inventory-item--rare .session-inventory-item__frame {
  border-color: rgba(96, 165, 250, 0.8);
}

.session-inventory-item--epic .session-inventory-item__frame {
  border-color: rgba(192, 132, 252, 0.8);
}

.session-inventory-item--masterwork .session-inventory-item__frame {
  border-color: rgba(251, 146, 60, 0.82);
}

.session-inventory-item--legendary .session-inventory-item__frame {
  border-color: rgba(74, 222, 128, 0.8);
}

.session-inventory-item--unique .session-inventory-item__frame {
  border-color: rgba(247, 118, 118, 0.82);
}
</style>