<script setup>
import { ref, computed, nextTick, onMounted, onBeforeUnmount } from 'vue'

const props = defineProps({
  modelValue: { type: [String, Number], default: '' },
  // [{ value, label }]
  options: { type: Array, default: () => [] },
  disabled: { type: Boolean, default: false },
  placeholder: { type: String, default: 'Search…' },
  clearable: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue'])

const rootRef = ref(null)
const inputRef = ref(null)
const dropdownRef = ref(null)
const open = ref(false)
const query = ref('')
const highlightedIndex = ref(0)
const dropdownStyle = ref({})

const selectedOption = computed(() => props.options.find((option) => option.value === props.modelValue) ?? null)
const selectedLabel = computed(() => selectedOption.value?.label ?? '')

const filteredOptions = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return props.options
  return props.options.filter((option) =>
    String(option.label).toLowerCase().includes(q) || String(option.value).toLowerCase().includes(q),
  )
})

// The dropdown is teleported to <body> with fixed positioning so it never gets
// clipped by an ancestor's overflow (e.g. the game header or a modal).
function updatePosition() {
  const el = rootRef.value
  if (!el) return
  const rect = el.getBoundingClientRect()
  dropdownStyle.value = {
    position: 'fixed',
    left: `${rect.left}px`,
    top: `${rect.bottom + 4}px`,
    width: `${rect.width}px`,
  }
}

function openDropdown() {
  if (props.disabled) return
  open.value = true
  query.value = ''
  const current = filteredOptions.value.findIndex((option) => option.value === props.modelValue)
  highlightedIndex.value = current >= 0 ? current : 0
  nextTick(() => {
    updatePosition()
    inputRef.value?.focus()
  })
}

function closeDropdown() {
  open.value = false
  query.value = ''
}

function selectOption(option) {
  if (!option) return
  emit('update:modelValue', option.value)
  closeDropdown()
  // Drop focus so the field no longer looks active after a pick.
  inputRef.value?.blur()
}

function clear() {
  emit('update:modelValue', '')
  closeDropdown()
  inputRef.value?.blur()
}

function onInput(event) {
  query.value = event.target.value
  open.value = true
  highlightedIndex.value = 0
  nextTick(updatePosition)
}

function moveHighlight(delta) {
  if (!open.value) {
    openDropdown()
    return
  }
  const count = filteredOptions.value.length
  if (count === 0) return
  highlightedIndex.value = (highlightedIndex.value + delta + count) % count
}

function selectHighlighted() {
  if (!open.value) {
    openDropdown()
    return
  }
  selectOption(filteredOptions.value[highlightedIndex.value])
}

function onDocumentMouseDown(event) {
  const target = event.target
  if (rootRef.value?.contains(target) || dropdownRef.value?.contains(target)) return
  closeDropdown()
}

function onReposition() {
  if (open.value) updatePosition()
}

onMounted(() => {
  document.addEventListener('mousedown', onDocumentMouseDown)
  window.addEventListener('resize', onReposition)
  window.addEventListener('scroll', onReposition, true)
})
onBeforeUnmount(() => {
  document.removeEventListener('mousedown', onDocumentMouseDown)
  window.removeEventListener('resize', onReposition)
  window.removeEventListener('scroll', onReposition, true)
})
</script>

<template>
  <div ref="rootRef" class="relative">
    <input
      ref="inputRef"
      type="text"
      :value="open ? query : selectedLabel"
      :disabled="disabled"
      :placeholder="selectedLabel || placeholder"
      autocomplete="off"
      @focus="openDropdown"
      @input="onInput"
      @keydown.down.prevent="moveHighlight(1)"
      @keydown.up.prevent="moveHighlight(-1)"
      @keydown.enter.prevent="selectHighlighted"
      @keydown.esc.prevent="closeDropdown"
      class="session-input w-full rounded-xl border border-[rgba(126,200,227,0.12)] bg-[rgba(126,200,227,0.08)] pl-3 py-2.5 text-[14px] text-[#f6f7fb] outline-none transition-colors focus:border-[rgba(233,69,96,0.4)] disabled:cursor-not-allowed disabled:opacity-65"
      :class="clearable && modelValue ? 'pr-9' : 'pr-3'"
    />

    <button
      v-if="clearable && modelValue && !disabled"
      type="button"
      aria-label="Clear selection"
      @mousedown.prevent="clear"
      class="absolute right-2 top-1/2 flex h-6 w-6 -translate-y-1/2 cursor-pointer items-center justify-center rounded-md text-[16px] leading-none text-[#7ec8e3]/55 transition-colors hover:bg-[rgba(233,69,96,0.16)] hover:text-[#fca5a5]"
    >
      ×
    </button>

    <Teleport to="body">
      <div
        v-if="open"
        ref="dropdownRef"
        :style="dropdownStyle"
        class="z-[13000] max-h-56 overflow-y-auto rounded-xl border border-[rgba(126,200,227,0.2)] bg-[#0b1730] shadow-[0_18px_48px_rgba(0,0,0,0.55)]"
      >
        <button
          v-for="(option, index) in filteredOptions"
          :key="option.value"
          type="button"
          @mousedown.prevent="selectOption(option)"
          @mouseenter="highlightedIndex = index"
          class="block w-full cursor-pointer px-3 py-2 text-left text-[14px] transition-colors"
          :class="[
            index === highlightedIndex ? 'bg-[rgba(126,200,227,0.14)]' : 'bg-transparent',
            option.value === modelValue ? 'text-[#e94560] font-semibold' : 'text-[#e8e8f0]',
          ]"
        >
          {{ option.label }}
        </button>
        <div v-if="!filteredOptions.length" class="px-3 py-2 text-[13px] text-[#7ec8e3]/45">No matches</div>
      </div>
    </Teleport>
  </div>
</template>
