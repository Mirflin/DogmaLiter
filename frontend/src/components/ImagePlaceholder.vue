<script setup>
import { computed } from 'vue'
import { Dices, ScrollText } from '@lucide/vue'
import { iconForItem } from '@/assets/icons/itemIcons'

// Stands in for a missing cover, news image or item picture. Fills its parent, so the parent sets the size.
const props = defineProps({
  kind: { type: String, default: 'game' },
  // Anything stable (an id) to pick the tint, so neighbouring games don't all look the same.
  seed: { type: String, default: '' },
  item: { type: Object, default: null },
})

const TINTS = ['var(--color-sky)', 'var(--color-primary-ink)', 'var(--color-orchid)', 'var(--color-r-epic)', 'var(--color-r-legendary)', 'var(--color-warning)']

const tint = computed(() => {
  if (props.kind === 'item') return `var(--color-r-${props.item?.rarity || 'common'})`
  if (props.kind === 'news') return 'var(--color-sky)'
  let hash = 0
  for (const char of props.seed) hash = (hash * 31 + char.charCodeAt(0)) >>> 0
  return TINTS[hash % TINTS.length]
})

const itemPath = computed(() => (props.kind === 'item' ? iconForItem(props.item) : ''))
</script>

<template>
  <div class="placeholder" :class="`placeholder--${kind}`" :style="{ '--tint': tint }" aria-hidden="true">
    <svg v-if="kind === 'item'" class="placeholder__icon" viewBox="0 0 512 512"><path :d="itemPath" /></svg>
    <ScrollText v-else-if="kind === 'news'" class="placeholder__icon" :stroke-width="1.4" />
    <Dices v-else class="placeholder__icon" :stroke-width="1.4" />
  </div>
</template>

<style scoped>
.placeholder {
  container-type: size;
  display: grid;
  place-items: center;
  width: 100%;
  height: 100%;
  color: color-mix(in oklab, var(--tint) 70%, white);
  background:
    radial-gradient(circle at 30% 20%, color-mix(in oklab, var(--tint) 22%, transparent), transparent 65%),
    color-mix(in oklab, var(--tint) 8%, var(--color-n2));
}

.placeholder--item {
  background: color-mix(in oklab, var(--tint) 20%, var(--color-n2));
}

.placeholder__icon {
  width: clamp(1.25rem, 38cqmin, 5.5rem);
  height: clamp(1.25rem, 38cqmin, 5.5rem);
  opacity: 0.6;
}

.placeholder--item .placeholder__icon {
  width: clamp(1.25rem, 62cqmin, 9rem);
  height: clamp(1.25rem, 62cqmin, 9rem);
  fill: currentColor;
  opacity: 0.75;
}
</style>
