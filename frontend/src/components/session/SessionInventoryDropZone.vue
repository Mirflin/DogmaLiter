<script setup>
import { makeDroppable } from '@vue-dnd-kit/core'
import { computed, ref } from 'vue'

const props = defineProps({
  zone: {
    type: Object,
    required: true,
  },
  groups: {
    type: Array,
    default: () => ['inventory-placement'],
  },
})

const emit = defineEmits(['drop'])

const zoneRef = ref(null)

const { isAllowed, isDragOver } = makeDroppable(zoneRef, {
  groups: computed(() => props.groups),
  data: () => props.zone,
  events: {
    onDrop: (event) => emit('drop', { event, zone: props.zone }),
  },
})
</script>

<template>
  <div ref="zoneRef">
    <slot :is-allowed="isAllowed" :is-drag-over="isDragOver" />
  </div>
</template>