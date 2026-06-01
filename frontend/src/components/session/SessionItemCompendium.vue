<script setup>
import {
  FilePlus2,
  Package,
  Plus,
  Search,
  X,
} from '@lucide/vue'
import { API_URL } from '@/api'
import { getErrorMessage, notify } from '@/notify'
import { useAuthStore } from '@/stores/auth'
import { computed, ref, watch } from 'vue'

const emit = defineEmits(['created'])

const props = defineProps({
  items: {
    type: Array,
    default: () => [],
  },
  characters: {
    type: Array,
    default: () => [],
  },
  gameId: {
    type: String,
    default: '',
  },
  availableTags: {
    type: Array,
    default: () => [],
  },
})

const auth = useAuthStore()

const RARITY_OPTIONS = ['common', 'uncommon', 'rare', 'epic', 'masterwork', 'legendary', 'unique']
const CATEGORY_OPTIONS = ['loot', 'consumable', 'equipment', 'other']
const EQUIP_SLOT_OPTIONS = ['head', 'chest', 'gloves', 'belt', 'boots', 'main_hand', 'off_hand', 'ring', 'amulet']
const BASE_ATTRIBUTE_OPTIONS = [
  { value: 'strength', label: 'Strength' },
  { value: 'dexterity', label: 'Dexterity' },
  { value: 'constitution', label: 'Constitution' },
  { value: 'intelligence', label: 'Intelligence' },
  { value: 'wisdom', label: 'Wisdom' },
  { value: 'charisma', label: 'Charisma' },
]
const SORT_OPTIONS = [
  { value: 'recent', label: 'Recently Updated' },
  { value: 'name-asc', label: 'Name A-Z' },
  { value: 'name-desc', label: 'Name Z-A' },
  { value: 'rarity', label: 'Quality' },
  { value: 'size', label: 'Grid Size' },
]
const RARITY_RANK = Object.fromEntries(RARITY_OPTIONS.map((value, index) => [value, index]))
const dateFormatter = new Intl.DateTimeFormat('en-US', {
  dateStyle: 'medium',
  timeStyle: 'short',
})

const searchQuery = ref('')
const rarityFilter = ref('all')
const categoryFilter = ref('all')
const slotFilter = ref('all')
const tagFilter = ref('all')
const sortMode = ref('recent')
const selectedItemId = ref('')
const hoveredItemId = ref('')
const showCreateItemModal = ref(false)
const createItemSubmitting = ref(false)
const itemCreateError = ref('')
const newTagDraft = ref('')
const itemDraft = ref(createEmptyItemDraft())

const searchNeedle = computed(() => searchQuery.value.trim().toLowerCase())
const allItems = computed(() => (props.items ?? []).map(item => normalizeItem(item)))
const itemById = computed(() => Object.fromEntries(allItems.value.map(item => [item.id, item])))
const attributeOptions = computed(() => {
  const seen = new Set(BASE_ATTRIBUTE_OPTIONS.map(option => option.value))
  const customOptions = []

  for (const character of props.characters ?? []) {
    for (const attribute of character?.custom_attributes ?? []) {
      const attributeName = normalizeAttributeName(attribute?.name)
      if (!attributeName || seen.has(attributeName)) {
        continue
      }

      seen.add(attributeName)
      customOptions.push({
        value: attributeName,
        label: formatAttributeLabel(attribute.name || attributeName),
      })
    }
  }

  customOptions.sort((left, right) => left.label.localeCompare(right.label))
  return [...BASE_ATTRIBUTE_OPTIONS, ...customOptions]
})
const availableTagNames = computed(() => {
  const result = new Map()

  for (const tag of props.availableTags ?? []) {
    const normalized = normalizeTagName(typeof tag === 'string' ? tag : tag?.name)
    if (!normalized) {
      continue
    }

    const lookupKey = normalized.toLowerCase()
    if (!result.has(lookupKey)) {
      result.set(lookupKey, normalized)
    }
  }

  for (const item of allItems.value) {
    for (const tag of item.tags) {
      const lookupKey = tag.toLowerCase()
      if (!result.has(lookupKey)) {
        result.set(lookupKey, tag)
      }
    }
  }

  return [...result.values()].sort((left, right) => left.localeCompare(right))
})
const filteredItems = computed(() => allItems.value
  .filter(item => itemMatchesFilters(item))
  .sort(compareItems))
const activeFilterBadges = computed(() => {
  const badges = []

  if (searchNeedle.value) {
    badges.push(`Query: ${searchQuery.value.trim()}`)
  }
  if (rarityFilter.value !== 'all') {
    badges.push(`Quality: ${formatLabel(rarityFilter.value)}`)
  }
  if (categoryFilter.value !== 'all') {
    badges.push(`Category: ${formatLabel(categoryFilter.value)}`)
  }
  if (slotFilter.value !== 'all') {
    badges.push(`Slot: ${formatLabel(slotFilter.value)}`)
  }
  if (tagFilter.value !== 'all') {
    badges.push(`Tag: ${tagFilter.value}`)
  }

  return badges
})
const hasActiveFilters = computed(() => activeFilterBadges.value.length > 0)
const compendiumStats = computed(() => ({
  totalItems: allItems.value.length,
  visibleItems: filteredItems.value.length,
  equipmentItems: allItems.value.filter(item => item.category === 'equipment').length,
  totalTags: availableTagNames.value.length,
}))
const inspectorItem = computed(() => itemById.value[hoveredItemId.value]
  ?? itemById.value[selectedItemId.value]
  ?? filteredItems.value[0]
  ?? allItems.value[0]
  ?? null)
const selectedTagSuggestions = computed(() => availableTagNames.value.filter(tag => !itemDraft.value.tags
  .some(selectedTag => selectedTag.toLowerCase() === tag.toLowerCase())))
const draftPreviewItem = computed(() => normalizeItem({
  id: 'draft-preview',
  name: itemDraft.value.name || 'Untitled Item',
  description: itemDraft.value.description,
  rarity: itemDraft.value.rarity,
  category: itemDraft.value.category,
  grid_width: normalizePositiveInteger(itemDraft.value.gridWidth, 1),
  grid_height: normalizePositiveInteger(itemDraft.value.gridHeight, 1),
  equip_slot: itemDraft.value.category === 'equipment' ? itemDraft.value.equipSlot : null,
  tags: itemDraft.value.tags,
  required_attributes: itemDraft.value.requiredAttributes,
  attribute_modifiers: itemDraft.value.attributeModifiers,
  updated_at: new Date().toISOString(),
}))

watch(
  () => filteredItems.value.map(item => item.id),
  (visibleIds) => {
    if (visibleIds.includes(selectedItemId.value)) {
      return
    }

    if (visibleIds.length > 0) {
      selectedItemId.value = visibleIds[0]
      return
    }

    selectedItemId.value = allItems.value[0]?.id ?? ''
  },
  { immediate: true },
)

function normalizeItem(item) {
  const rarity = RARITY_OPTIONS.includes(String(item?.rarity || '').toLowerCase())
    ? String(item.rarity).toLowerCase()
    : 'common'
  const category = CATEGORY_OPTIONS.includes(String(item?.category || '').toLowerCase())
    ? String(item.category).toLowerCase()
    : 'other'
  const equipSlot = normalizeEquipSlot(item?.equip_slot)
  const tags = uniqueLabels(Array.isArray(item?.tags) ? item.tags : [])
  const types = Array.isArray(item?.types)
    ? uniqueLabels(item.types)
    : []
  const requiredAttributes = Array.isArray(item?.required_attributes)
    ? item.required_attributes
      .map(entry => ({
        attribute_name: normalizeAttributeName(entry?.attribute_name),
        min_value: normalizeInteger(entry?.min_value, 0),
      }))
      .filter(entry => entry.attribute_name)
    : []
  const attributeModifiers = Array.isArray(item?.attribute_modifiers)
    ? item.attribute_modifiers
      .map(entry => ({
        attribute_name: normalizeAttributeName(entry?.attribute_name),
        modifier_value: normalizeSignedInteger(entry?.modifier_value, 0),
        is_percentage: Boolean(entry?.is_percentage),
      }))
      .filter(entry => entry.attribute_name)
    : []

  return {
    id: String(item?.id || `item-${Math.random().toString(36).slice(2)}`),
    name: String(item?.name || 'Untitled Item').trim() || 'Untitled Item',
    description: String(item?.description || '').trim(),
    image_id: item?.image_id ? String(item.image_id) : '',
    rarity,
    category,
    equip_slot: category === 'equipment' ? equipSlot : '',
    grid_width: normalizePositiveInteger(item?.grid_width, 1),
    grid_height: normalizePositiveInteger(item?.grid_height, 1),
    tags,
    types,
    required_attributes: requiredAttributes,
    attribute_modifiers: attributeModifiers,
    created_at: item?.created_at || '',
    updated_at: item?.updated_at || item?.created_at || '',
    search_text: [
      item?.name,
      item?.description,
      rarity,
      category,
      equipSlot,
      ...tags,
      ...types,
      ...requiredAttributes.map(entry => `${entry.attribute_name} ${entry.min_value}`),
      ...attributeModifiers.map(entry => `${entry.attribute_name} ${entry.modifier_value}${entry.is_percentage ? '%' : ''}`),
    ]
      .filter(Boolean)
      .join(' ')
      .toLowerCase(),
  }
}

function itemMatchesFilters(item) {
  if (searchNeedle.value && !item.search_text.includes(searchNeedle.value)) {
    return false
  }
  if (rarityFilter.value !== 'all' && item.rarity !== rarityFilter.value) {
    return false
  }
  if (categoryFilter.value !== 'all' && item.category !== categoryFilter.value) {
    return false
  }
  if (slotFilter.value !== 'all' && item.equip_slot !== slotFilter.value) {
    return false
  }
  if (tagFilter.value !== 'all' && !item.tags.some(tag => tag.toLowerCase() === tagFilter.value.toLowerCase())) {
    return false
  }

  return true
}

function compareItems(left, right) {
  if (sortMode.value === 'name-asc') {
    return left.name.localeCompare(right.name)
  }
  if (sortMode.value === 'name-desc') {
    return right.name.localeCompare(left.name)
  }
  if (sortMode.value === 'rarity') {
    const rarityDelta = (RARITY_RANK[right.rarity] ?? 0) - (RARITY_RANK[left.rarity] ?? 0)
    if (rarityDelta !== 0) {
      return rarityDelta
    }
    return left.name.localeCompare(right.name)
  }
  if (sortMode.value === 'size') {
    const sizeDelta = (right.grid_width * right.grid_height) - (left.grid_width * left.grid_height)
    if (sizeDelta !== 0) {
      return sizeDelta
    }
    return left.name.localeCompare(right.name)
  }

  return compareDates(right.updated_at, left.updated_at) || left.name.localeCompare(right.name)
}

function compareDates(left, right) {
  const leftTime = Date.parse(left || '')
  const rightTime = Date.parse(right || '')

  if (!Number.isFinite(leftTime) && !Number.isFinite(rightTime)) {
    return 0
  }
  if (!Number.isFinite(leftTime)) {
    return -1
  }
  if (!Number.isFinite(rightTime)) {
    return 1
  }

  return leftTime - rightTime
}

function clearFilters() {
  searchQuery.value = ''
  rarityFilter.value = 'all'
  categoryFilter.value = 'all'
  slotFilter.value = 'all'
  tagFilter.value = 'all'
  sortMode.value = 'recent'
}

function selectItem(itemId) {
  selectedItemId.value = itemId
}

function openCreateItemModal() {
  itemDraft.value = createEmptyItemDraft()
  newTagDraft.value = ''
  itemCreateError.value = ''
  showCreateItemModal.value = true
}

function closeCreateItemModal(force = false) {
  if (createItemSubmitting.value && !force) {
    return
  }

  showCreateItemModal.value = false
  itemCreateError.value = ''
  newTagDraft.value = ''
}

function addRequirementRow() {
  itemDraft.value.requiredAttributes.push(createEmptyRequirementRow(defaultAttributeName()))
}

function removeRequirementRow(index) {
  itemDraft.value.requiredAttributes.splice(index, 1)
}

function addModifierRow() {
  itemDraft.value.attributeModifiers.push(createEmptyModifierRow(defaultAttributeName()))
}

function removeModifierRow(index) {
  itemDraft.value.attributeModifiers.splice(index, 1)
}

function toggleDraftTag(tag) {
  const lookupKey = tag.toLowerCase()
  if (itemDraft.value.tags.some(entry => entry.toLowerCase() === lookupKey)) {
    itemDraft.value.tags = itemDraft.value.tags.filter(entry => entry.toLowerCase() !== lookupKey)
    return
  }

  if (itemDraft.value.tags.length >= 20) {
    itemCreateError.value = 'Each item can have up to 20 tags.'
    return
  }

  itemDraft.value.tags = [...itemDraft.value.tags, tag].sort((left, right) => left.localeCompare(right))
}

function removeDraftTag(tag) {
  itemDraft.value.tags = itemDraft.value.tags.filter(entry => entry.toLowerCase() !== tag.toLowerCase())
}

function addDraftTagFromInput() {
  const normalized = normalizeTagName(newTagDraft.value)
  if (!normalized) {
    newTagDraft.value = ''
    return
  }
  if (normalized.length > 60) {
    itemCreateError.value = 'Tag names must be 60 characters or fewer.'
    return
  }
  if (itemDraft.value.tags.length >= 20) {
    itemCreateError.value = 'Each item can have up to 20 tags.'
    return
  }

  itemCreateError.value = ''
  if (!itemDraft.value.tags.some(entry => entry.toLowerCase() === normalized.toLowerCase())) {
    itemDraft.value.tags = [...itemDraft.value.tags, normalized].sort((left, right) => left.localeCompare(right))
  }
  newTagDraft.value = ''
}

async function createItem() {
  if (createItemSubmitting.value) {
    return
  }

  const payload = {
    name: itemDraft.value.name.trim(),
    description: itemDraft.value.description.trim(),
    rarity: itemDraft.value.rarity,
    category: itemDraft.value.category,
    tags: itemDraft.value.tags,
    grid_width: normalizePositiveInteger(itemDraft.value.gridWidth, 1),
    grid_height: normalizePositiveInteger(itemDraft.value.gridHeight, 1),
    equip_slot: itemDraft.value.category === 'equipment' ? normalizeEquipSlot(itemDraft.value.equipSlot) : undefined,
    required_attributes: itemDraft.value.requiredAttributes
      .map(entry => ({
        attribute_name: normalizeAttributeName(entry.attribute_name),
        min_value: normalizeInteger(entry.min_value, 0),
      }))
      .filter(entry => entry.attribute_name),
    attribute_modifiers: itemDraft.value.attributeModifiers
      .map(entry => ({
        attribute_name: normalizeAttributeName(entry.attribute_name),
        modifier_value: normalizeSignedInteger(entry.modifier_value, 0),
        is_percentage: Boolean(entry.is_percentage),
      }))
      .filter(entry => entry.attribute_name),
  }

  if (!props.gameId) {
    itemCreateError.value = 'Game context is missing.'
    return
  }
  if (!payload.name) {
    itemCreateError.value = 'Item name cannot be empty.'
    return
  }
  if (payload.category === 'equipment' && !payload.equip_slot) {
    itemCreateError.value = 'Equipment items need an equip slot.'
    return
  }

  createItemSubmitting.value = true
  itemCreateError.value = ''

  try {
    await auth.createGameItem(props.gameId, payload)
    notify.success({
      title: 'Item created',
      message: `${payload.name} was added to the campaign compendium.`,
    })
    closeCreateItemModal(true)
    emit('created')
  } catch (error) {
    itemCreateError.value = getErrorMessage(error, 'Failed to create item')
  } finally {
    createItemSubmitting.value = false
  }
}

function createEmptyItemDraft() {
  return {
    name: '',
    description: '',
    rarity: 'common',
    category: 'other',
    gridWidth: 1,
    gridHeight: 1,
    equipSlot: 'main_hand',
    tags: [],
    requiredAttributes: [],
    attributeModifiers: [],
  }
}

function createEmptyRequirementRow(attributeName = BASE_ATTRIBUTE_OPTIONS[0].value) {
  return {
    attribute_name: attributeName,
    min_value: 0,
  }
}

function createEmptyModifierRow(attributeName = BASE_ATTRIBUTE_OPTIONS[0].value) {
  return {
    attribute_name: attributeName,
    modifier_value: 0,
    is_percentage: false,
  }
}

function defaultAttributeName() {
  return attributeOptions.value[0]?.value || BASE_ATTRIBUTE_OPTIONS[0].value
}

function normalizeTagName(value) {
  return String(value || '')
    .trim()
    .split(/\s+/)
    .filter(Boolean)
    .join(' ')
}

function uniqueLabels(values) {
  const uniqueValues = new Map()

  for (const value of values ?? []) {
    const normalized = normalizeTagName(value)
    if (!normalized) {
      continue
    }

    const lookupKey = normalized.toLowerCase()
    if (!uniqueValues.has(lookupKey)) {
      uniqueValues.set(lookupKey, normalized)
    }
  }

  return [...uniqueValues.values()].sort((left, right) => left.localeCompare(right))
}

function normalizeAttributeName(value) {
  return String(value || '')
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9\s_-]+/g, '')
    .replace(/[\s-]+/g, '_')
}

function normalizeEquipSlot(value) {
  const normalized = String(value || '').trim().toLowerCase()
  return EQUIP_SLOT_OPTIONS.includes(normalized) ? normalized : ''
}

function normalizeInteger(value, fallback = 0) {
  const parsed = Number.parseInt(value, 10)
  return Number.isNaN(parsed) ? fallback : parsed
}

function normalizeSignedInteger(value, fallback = 0) {
  return normalizeInteger(value, fallback)
}

function normalizePositiveInteger(value, fallback = 1) {
  return Math.max(1, normalizeInteger(value, fallback))
}

function formatLabel(value) {
  return String(value || '')
    .split('_')
    .filter(Boolean)
    .map(part => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ')
}

function formatAttributeLabel(value) {
  return formatLabel(value)
}

function formatDateTime(value) {
  if (!value) {
    return 'Unknown update'
  }

  const parsed = Date.parse(value)
  if (!Number.isFinite(parsed)) {
    return 'Unknown update'
  }

  return dateFormatter.format(new Date(parsed))
}

function itemFrameClass(rarity) {
  const variants = {
    common: 'border-[rgba(255,255,255,0.24)] bg-[linear-gradient(180deg,rgba(16,32,52,0.92),rgba(8,16,30,0.92))] text-[#f6f7fb]',
    uncommon: 'border-[rgba(250,204,21,0.36)] bg-[linear-gradient(180deg,rgba(94,67,0,0.92),rgba(44,31,2,0.92))] text-[#fde68a]',
    rare: 'border-[rgba(96,165,250,0.36)] bg-[linear-gradient(180deg,rgba(12,38,79,0.94),rgba(8,19,44,0.92))] text-[#bfdbfe]',
    epic: 'border-[rgba(192,132,252,0.34)] bg-[linear-gradient(180deg,rgba(59,20,88,0.94),rgba(28,12,44,0.92))] text-[#e9d5ff]',
    masterwork: 'border-[rgba(251,146,60,0.34)] bg-[linear-gradient(180deg,rgba(102,47,10,0.94),rgba(43,21,8,0.92))] text-[#fdba74]',
    legendary: 'border-[rgba(74,222,128,0.34)] bg-[linear-gradient(180deg,rgba(10,78,39,0.94),rgba(8,40,22,0.92))] text-[#bbf7d0]',
    unique: 'border-[rgba(248,113,113,0.38)] bg-[linear-gradient(180deg,rgba(111,24,24,0.94),rgba(55,14,18,0.92))] text-[#fecaca]',
  }

  return variants[rarity] ?? variants.common
}

function rarityBadgeClass(rarity) {
  const variants = {
    common: 'border-[rgba(255,255,255,0.24)] bg-[rgba(255,255,255,0.08)] text-[#f8fafc]',
    uncommon: 'border-[rgba(250,204,21,0.35)] bg-[rgba(202,138,4,0.16)] text-[#fde68a]',
    rare: 'border-[rgba(96,165,250,0.35)] bg-[rgba(37,99,235,0.16)] text-[#93c5fd]',
    epic: 'border-[rgba(192,132,252,0.35)] bg-[rgba(126,34,206,0.18)] text-[#e9d5ff]',
    masterwork: 'border-[rgba(251,146,60,0.35)] bg-[rgba(194,65,12,0.18)] text-[#fdba74]',
    legendary: 'border-[rgba(74,222,128,0.35)] bg-[rgba(21,128,61,0.18)] text-[#86efac]',
    unique: 'border-[rgba(248,113,113,0.38)] bg-[rgba(153,27,27,0.18)] text-[#fecaca]',
  }

  return variants[rarity] ?? variants.common
}

function itemImageUrl(item) {
  if (!item?.image_id) {
    return ''
  }

  return `${API_URL}/api/uploads/${item.image_id}`
}

function itemGlyph(item) {
  return item.name
    .split(' ')
    .filter(Boolean)
    .slice(0, 2)
    .map(part => part.charAt(0).toUpperCase())
    .join('') || 'IT'
}

function itemSizeLabel(item) {
  return `${item.grid_width}x${item.grid_height}`
}
</script>

<template>
  <section class="space-y-6">
    <article class="overflow-hidden rounded-[1.9rem] border border-[rgba(126,200,227,0.14)] bg-[linear-gradient(180deg,rgba(9,18,34,0.98),rgba(5,10,22,0.98))] p-5 shadow-[0_32px_90px_rgba(0,0,0,0.35)] sm:p-6">
      <div class="flex flex-col gap-5 xl:flex-row xl:items-end xl:justify-between">
        <div>
          <p class="text-[11px] uppercase tracking-[0.24em] text-[#7ec8e3]/58">GM Compendium</p>
          <div class="mt-3 flex flex-wrap items-center gap-3">
            <h2 class="font-[Cinzel] text-[28px] font-bold text-[#f6f7fb] sm:text-[34px]">Item Compendium</h2>
          </div>
        </div>

        <div class="flex flex-wrap gap-3">
          <button
            type="button"
            @click="openCreateItemModal"
            class="inline-flex cursor-pointer items-center gap-2 rounded-xl border border-[rgba(233,69,96,0.28)] bg-[linear-gradient(135deg,rgba(233,69,96,0.92),rgba(194,49,82,0.92))] px-4 py-2.5 text-[13px] font-semibold text-white transition-all duration-200 hover:-translate-y-0.5 hover:shadow-[0_12px_30px_rgba(233,69,96,0.24)]"
          >
            <FilePlus2 class="h-4 w-4" :stroke-width="2" />
            Create Item
          </button>
        </div>
      </div>
    </article>

    <article class="rounded-[1.75rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(8,16,30,0.78)] p-5 sm:p-6">
      <div class="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Filters</p>
          <p class="mt-2 text-[14px] leading-relaxed text-[#d8dce7]/58">Search across item names, descriptions, requirements, modifiers, and custom tags.</p>
        </div>
        <button
          v-if="hasActiveFilters"
          type="button"
          @click="clearFilters"
          class="inline-flex cursor-pointer items-center gap-2 rounded-xl border border-[rgba(126,200,227,0.16)] bg-[rgba(126,200,227,0.08)] px-4 py-2.5 text-[13px] font-semibold text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(126,200,227,0.3)]"
        >
          Clear Filters
        </button>
      </div>

      <div class="mt-5 grid gap-4 xl:grid-cols-[minmax(0,1.5fr)_repeat(5,minmax(0,1fr))]">
        <label class="block xl:col-span-1">
          <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Search</span>
          <div class="mt-2 flex items-center gap-3 rounded-[1.2rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.72)] px-4 py-3">
            <Search class="h-4 w-4 text-[#7ec8e3]/45" :stroke-width="2" />
            <input
              v-model="searchQuery"
              type="text"
              placeholder="Search the compendium"
              class="session-input w-full border-0 bg-transparent p-0 text-[14px] text-[#f6f7fb] outline-none placeholder:text-[#7ec8e3]/30"
            />
          </div>
        </label>

        <label class="block">
          <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Quality</span>
          <select v-model="rarityFilter" class="session-input mt-2 w-full rounded-[1.2rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.72)] px-4 py-3 text-[14px] text-[#f6f7fb] outline-none">
            <option value="all">All qualities</option>
            <option v-for="rarity in RARITY_OPTIONS" :key="rarity" :value="rarity">{{ formatLabel(rarity) }}</option>
          </select>
        </label>

        <label class="block">
          <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Category</span>
          <select v-model="categoryFilter" class="session-input mt-2 w-full rounded-[1.2rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.72)] px-4 py-3 text-[14px] text-[#f6f7fb] outline-none">
            <option value="all">All categories</option>
            <option v-for="category in CATEGORY_OPTIONS" :key="category" :value="category">{{ formatLabel(category) }}</option>
          </select>
        </label>

        <label class="block">
          <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Slot</span>
          <select v-model="slotFilter" class="session-input mt-2 w-full rounded-[1.2rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.72)] px-4 py-3 text-[14px] text-[#f6f7fb] outline-none">
            <option value="all">All slots</option>
            <option v-for="slot in EQUIP_SLOT_OPTIONS" :key="slot" :value="slot">{{ formatLabel(slot) }}</option>
          </select>
        </label>

        <label class="block">
          <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Tag</span>
          <select v-model="tagFilter" class="session-input mt-2 w-full rounded-[1.2rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.72)] px-4 py-3 text-[14px] text-[#f6f7fb] outline-none">
            <option value="all">All tags</option>
            <option v-for="tag in availableTagNames" :key="tag" :value="tag">{{ tag }}</option>
          </select>
        </label>

        <label class="block">
          <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Sort</span>
          <select v-model="sortMode" class="session-input mt-2 w-full rounded-[1.2rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.72)] px-4 py-3 text-[14px] text-[#f6f7fb] outline-none">
            <option v-for="option in SORT_OPTIONS" :key="option.value" :value="option.value">{{ option.label }}</option>
          </select>
        </label>
      </div>

      <div v-if="activeFilterBadges.length" class="mt-4 flex flex-wrap gap-2">
        <span
          v-for="badge in activeFilterBadges"
          :key="badge"
          class="rounded-full border border-[rgba(126,200,227,0.14)] bg-[rgba(126,200,227,0.08)] px-3 py-1.5 text-[12px] text-[#d8dce7]/72"
        >
          {{ badge }}
        </span>
      </div>
    </article>

    <div class="grid gap-6 xl:grid-cols-[minmax(0,1.4fr)_380px]">
      <article class="overflow-hidden rounded-[1.75rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(8,16,30,0.78)]">
        <div class="flex flex-wrap items-center justify-between gap-3 border-b border-[rgba(126,200,227,0.1)] px-5 py-4 sm:px-6">
          <div>
            <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Compendium Entries</p>
          </div>
        </div>

        <div v-if="filteredItems.length" class="grid gap-4 p-5 sm:grid-cols-2 2xl:grid-cols-3 sm:p-6">
          <button
            v-for="item in filteredItems"
            :key="item.id"
            type="button"
            @click="selectItem(item.id)"
            @mouseenter="hoveredItemId = item.id"
            @mouseleave="hoveredItemId = ''"
            @focus="selectedItemId = item.id"
            class="group cursor-pointer rounded-[1.45rem] border bg-[rgba(7,17,31,0.72)] p-4 text-left transition-all duration-200 hover:-translate-y-0.5 hover:border-[rgba(126,200,227,0.24)]"
            :class="selectedItemId === item.id ? 'border-[rgba(233,69,96,0.32)] shadow-[0_12px_36px_rgba(233,69,96,0.12)]' : 'border-[rgba(126,200,227,0.12)]'"
          >
            <div class="flex items-start gap-4">
              <div class="relative flex h-20 w-20 shrink-0 items-center justify-center overflow-hidden rounded-[1.3rem] border text-[24px] font-bold uppercase"
                :class="itemFrameClass(item.rarity)">
                <img v-if="itemImageUrl(item)" :src="itemImageUrl(item)" :alt="item.name" class="h-full w-full object-cover" />
                <span v-else>{{ itemGlyph(item) }}</span>
                <span class="absolute right-2 top-2 rounded-full border border-[rgba(255,255,255,0.16)] bg-[rgba(7,17,31,0.82)] px-2 py-0.5 text-[10px] font-semibold text-[#f6f7fb]">
                  {{ itemSizeLabel(item) }}
                </span>
              </div>

              <div class="min-w-0 flex-1">
                <div class="flex flex-wrap items-start gap-2">
                  <h3 class="min-w-0 flex-1 break-words text-[16px] font-semibold text-[#f6f7fb]">{{ item.name }}</h3>
                  <span class="rounded-full border px-2.5 py-1 text-[10px] font-semibold uppercase tracking-[0.16em]" :class="rarityBadgeClass(item.rarity)">
                    {{ formatLabel(item.rarity) }}
                  </span>
                </div>

                <p class="mt-2 line-clamp-3 text-[13px] leading-relaxed text-[#d8dce7]/58">
                  {{ item.description || 'No description yet.' }}
                </p>

                <div class="mt-3 flex flex-wrap gap-2">
                  <span class="rounded-full border border-[rgba(126,200,227,0.14)] bg-[rgba(126,200,227,0.08)] px-2.5 py-1 text-[11px] uppercase tracking-[0.12em] text-[#8fd7ef]">
                    {{ formatLabel(item.category) }}
                  </span>
                  <span v-if="item.equip_slot" class="rounded-full border border-[rgba(126,200,227,0.14)] bg-[rgba(255,255,255,0.05)] px-2.5 py-1 text-[11px] uppercase tracking-[0.12em] text-[#d8dce7]/68">
                    {{ formatLabel(item.equip_slot) }}
                  </span>
                  <span v-for="tag in item.tags.slice(0, 3)" :key="`${item.id}-${tag}`" class="rounded-full border border-[rgba(233,69,96,0.16)] bg-[rgba(233,69,96,0.08)] px-2.5 py-1 text-[11px] text-[#ffe0e7]">
                    {{ tag }}
                  </span>
                  <span v-if="item.tags.length > 3" class="rounded-full border border-[rgba(126,200,227,0.14)] bg-[rgba(126,200,227,0.05)] px-2.5 py-1 text-[11px] text-[#d8dce7]/65">
                    +{{ item.tags.length - 3 }} more
                  </span>
                </div>
              </div>
            </div>

            <div class="mt-4 flex flex-wrap items-center gap-3 border-t border-[rgba(126,200,227,0.08)] pt-3 text-[12px] text-[#d8dce7]/56">
              <span>{{ item.required_attributes.length }} requirements</span>
              <span>{{ item.attribute_modifiers.length }} modifiers</span>
              <span>Updated {{ formatDateTime(item.updated_at) }}</span>
            </div>
          </button>
        </div>

        <div v-else class="flex min-h-[320px] flex-col items-center justify-center gap-4 px-6 py-12 text-center">
          <div class="flex h-16 w-16 items-center justify-center rounded-full border border-[rgba(126,200,227,0.14)] bg-[rgba(126,200,227,0.08)] text-[#8fd7ef]">
            <Package class="h-7 w-7" :stroke-width="2" />
          </div>
          <div>
            <h3 class="font-[Cinzel] text-[24px] font-bold text-[#f6f7fb]">{{ allItems.length ? 'No matches found' : 'No items yet' }}</h3>
            <p class="mt-3 max-w-[34rem] text-[14px] leading-relaxed text-[#d8dce7]/58">
              {{ allItems.length
                ? 'Adjust the active filters or search query to widen the compendium results.'
                : 'Create the first compendium entry and start tagging equipment, loot, and consumables for this game.' }}
            </p>
          </div>
        </div>
      </article>

      <aside class="space-y-6">
        <article class="rounded-[1.75rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(8,16,30,0.78)] p-5 sm:p-6">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div>
              <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Inspector</p>
              <p class="mt-2 text-[14px] leading-relaxed text-[#d8dce7]/58">Pinned preview for the currently selected compendium entry.</p>
            </div>
          </div>

          <div v-if="inspectorItem" class="mt-5 space-y-5">
            <div class="overflow-hidden rounded-[1.5rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.66)]">
              <div class="flex h-64 items-center justify-center" :class="itemFrameClass(inspectorItem.rarity)">
                <img v-if="itemImageUrl(inspectorItem)" :src="itemImageUrl(inspectorItem)" :alt="inspectorItem.name" class="h-full w-full object-cover" />
                <span v-else class="font-[Cinzel] text-[40px] font-bold">{{ itemGlyph(inspectorItem) }}</span>
              </div>
            </div>

            <div>
              <div class="flex flex-wrap items-center gap-2">
                <h3 class="font-[Cinzel] text-[28px] font-bold text-[#f6f7fb]">{{ inspectorItem.name }}</h3>
                <span class="rounded-full border px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.16em]" :class="rarityBadgeClass(inspectorItem.rarity)">
                  {{ formatLabel(inspectorItem.rarity) }}
                </span>
              </div>
              <p class="mt-3 text-[14px] leading-relaxed text-[#d8dce7]/62">{{ inspectorItem.description || 'No description yet.' }}</p>
            </div>

            <div class="grid gap-3 sm:grid-cols-2">
              <div class="rounded-[1.2rem] border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] px-4 py-3">
                <p class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Category</p>
                <p class="mt-2 text-[15px] font-semibold text-[#f6f7fb]">{{ formatLabel(inspectorItem.category) }}</p>
              </div>
              <div class="rounded-[1.2rem] border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] px-4 py-3">
                <p class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Grid Size</p>
                <p class="mt-2 text-[15px] font-semibold text-[#f6f7fb]">{{ itemSizeLabel(inspectorItem) }}</p>
              </div>
              <div class="rounded-[1.2rem] border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] px-4 py-3">
                <p class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Equip Slot</p>
                <p class="mt-2 text-[15px] font-semibold text-[#f6f7fb]">{{ inspectorItem.equip_slot ? formatLabel(inspectorItem.equip_slot) : 'Not equippable' }}</p>
              </div>
              <div class="rounded-[1.2rem] border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] px-4 py-3">
                <p class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Updated</p>
                <p class="mt-2 text-[15px] font-semibold text-[#f6f7fb]">{{ formatDateTime(inspectorItem.updated_at) }}</p>
              </div>
            </div>

            <div>
              <p class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Custom Tags</p>
              <div v-if="inspectorItem.tags.length" class="mt-3 flex flex-wrap gap-2">
                <span v-for="tag in inspectorItem.tags" :key="`${inspectorItem.id}-${tag}`" class="rounded-full border border-[rgba(233,69,96,0.18)] bg-[rgba(233,69,96,0.08)] px-3 py-1.5 text-[12px] text-[#ffe0e7]">
                  {{ tag }}
                </span>
              </div>
              <p v-else class="mt-3 text-[14px] text-[#d8dce7]/58">No custom tags assigned to this item.</p>
            </div>

            <div class="grid gap-4">
              <div class="rounded-[1.4rem] border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] p-4">
                <div class="flex items-center justify-between gap-3">
                  <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Requirements</span>
                  <span class="text-[12px] text-[#d8dce7]/45">{{ inspectorItem.required_attributes.length }}</span>
                </div>
                <div v-if="inspectorItem.required_attributes.length" class="mt-4 space-y-2">
                  <div v-for="requirement in inspectorItem.required_attributes" :key="`${inspectorItem.id}-${requirement.attribute_name}-${requirement.min_value}`" class="rounded-[1rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.62)] px-3 py-2.5 text-[13px] text-[#f6f7fb]">
                    {{ formatAttributeLabel(requirement.attribute_name) }} {{ requirement.min_value }}+
                  </div>
                </div>
                <p v-else class="mt-4 text-[14px] text-[#d8dce7]/58">No requirements.</p>
              </div>

              <div class="rounded-[1.4rem] border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] p-4">
                <div class="flex items-center justify-between gap-3">
                  <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Modifiers</span>
                  <span class="text-[12px] text-[#d8dce7]/45">{{ inspectorItem.attribute_modifiers.length }}</span>
                </div>
                <div v-if="inspectorItem.attribute_modifiers.length" class="mt-4 space-y-2">
                  <div v-for="modifier in inspectorItem.attribute_modifiers" :key="`${inspectorItem.id}-${modifier.attribute_name}-${modifier.modifier_value}-${modifier.is_percentage}`" class="rounded-[1rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.62)] px-3 py-2.5 text-[13px] text-[#f6f7fb]">
                    {{ formatAttributeLabel(modifier.attribute_name) }} {{ modifier.modifier_value > 0 ? '+' : '' }}{{ modifier.modifier_value }}{{ modifier.is_percentage ? '%' : '' }}
                  </div>
                </div>
                <p v-else class="mt-4 text-[14px] text-[#d8dce7]/58">No modifiers.</p>
              </div>
            </div>
          </div>

          <div v-else class="mt-6 text-[14px] leading-relaxed text-[#d8dce7]/58">Select an item from the compendium to inspect it here.</div>
        </article>
      </aside>
    </div>

    <Teleport to="body">
      <div v-if="showCreateItemModal" class="fixed inset-0 z-[12500] p-3 sm:p-4">
        <div class="absolute inset-0 bg-[rgba(5,8,12,0.82)] backdrop-blur-md" @click="closeCreateItemModal"></div>

        <div class="relative flex h-full w-full flex-col overflow-hidden rounded-[2rem] border border-[rgba(126,200,227,0.16)] bg-[linear-gradient(180deg,rgba(9,18,34,0.98),rgba(5,10,22,0.98))] shadow-[0_40px_120px_rgba(0,0,0,0.52)]">
          <button
            type="button"
            @click="closeCreateItemModal"
            :disabled="createItemSubmitting"
            class="absolute right-5 top-5 z-10 flex h-10 w-10 cursor-pointer items-center justify-center rounded-xl border border-[rgba(126,200,227,0.14)] bg-[rgba(126,200,227,0.08)] text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(233,69,96,0.32)] disabled:cursor-not-allowed disabled:opacity-60"
            aria-label="Close item creation modal"
          >
            <X class="h-5 w-5" :stroke-width="2" />
          </button>

          <div class="border-b border-[rgba(126,200,227,0.1)] px-5 py-5 pr-16 sm:px-6">
            <p class="text-[11px] uppercase tracking-[0.24em] text-[#7ec8e3]/58">GM Item Forge</p>
            <div class="mt-3 flex flex-wrap items-center gap-3">
              <h2 class="font-[Cinzel] text-[28px] font-bold text-[#f6f7fb] sm:text-[34px]">Create Compendium Entry</h2>
            </div>
          </div>

          <p v-if="itemCreateError" class="mx-5 mt-5 rounded-[1.3rem] border border-[rgba(248,113,113,0.24)] bg-[rgba(127,29,29,0.18)] px-4 py-3 text-[13px] text-[#fecaca] sm:mx-6">
            {{ itemCreateError }}
          </p>

          <div class="grid min-h-0 flex-1 gap-6 overflow-hidden px-5 pb-5 pt-5 sm:px-6 lg:grid-cols-[minmax(0,1.45fr)_380px]">
            <section class="min-h-0 space-y-5 overflow-y-auto pr-1">
              <article class="rounded-[1.6rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(8,16,30,0.78)] p-5">
                <div class="grid gap-5 lg:grid-cols-[minmax(0,1fr)_minmax(320px,0.9fr)]">
                  <label class="block">
                    <span class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Name</span>
                    <input
                      v-model="itemDraft.name"
                      type="text"
                      maxlength="120"
                      :disabled="createItemSubmitting"
                      class="session-input mt-2 w-full rounded-[1.25rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.72)] px-4 py-3 text-[15px] text-[#f6f7fb] outline-none placeholder:text-[#7ec8e3]/30"
                    />
                  </label>

                  <div class="grid gap-4 sm:grid-cols-3">
                    <label class="block">
                      <span class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Quality</span>
                      <select v-model="itemDraft.rarity" class="session-input mt-2 w-full rounded-[1.25rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.72)] px-4 py-3 text-[14px] text-[#f6f7fb] outline-none">
                        <option v-for="rarity in RARITY_OPTIONS" :key="rarity" :value="rarity">{{ formatLabel(rarity) }}</option>
                      </select>
                    </label>

                    <label class="block">
                      <span class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Category</span>
                      <select v-model="itemDraft.category" class="session-input mt-2 w-full rounded-[1.25rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.72)] px-4 py-3 text-[14px] text-[#f6f7fb] outline-none">
                        <option v-for="category in CATEGORY_OPTIONS" :key="category" :value="category">{{ formatLabel(category) }}</option>
                      </select>
                    </label>

                    <label class="block">
                      <span class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Equip Slot</span>
                      <select v-model="itemDraft.equipSlot" :disabled="itemDraft.category !== 'equipment'" class="session-input mt-2 w-full rounded-[1.25rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.72)] px-4 py-3 text-[14px] text-[#f6f7fb] outline-none disabled:cursor-not-allowed disabled:opacity-45">
                        <option v-for="slot in EQUIP_SLOT_OPTIONS" :key="slot" :value="slot">{{ formatLabel(slot) }}</option>
                      </select>
                    </label>
                  </div>
                </div>

                <label class="mt-5 block">
                  <span class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Description</span>
                  <textarea
                    v-model="itemDraft.description"
                    rows="6"
                    :disabled="createItemSubmitting"
                    class="session-input mt-2 w-full resize-y rounded-[1.25rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.72)] px-4 py-3 text-[15px] text-[#f6f7fb] outline-none placeholder:text-[#7ec8e3]/30"
                  ></textarea>
                </label>
              </article>

              <article class="rounded-[1.6rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(8,16,30,0.78)] p-5">
                <div class="flex flex-wrap items-center justify-between gap-3">
                  <div>
                    <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Footprint</p>
                    <p class="mt-2 text-[14px] leading-relaxed text-[#d8dce7]/58">Give the item enough breathing room inside inventory grids and equipment layouts.</p>
                  </div>
                </div>

                <div class="mt-5 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
                  <label class="rounded-[1.2rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.62)] px-4 py-3">
                    <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Grid Width</span>
                    <input v-model.number="itemDraft.gridWidth" type="number" min="1" max="12" class="session-input mt-3 w-full border-0 bg-transparent p-0 text-[24px] font-semibold text-[#f6f7fb] outline-none" />
                  </label>
                  <label class="rounded-[1.2rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.62)] px-4 py-3">
                    <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Grid Height</span>
                    <input v-model.number="itemDraft.gridHeight" type="number" min="1" max="12" class="session-input mt-3 w-full border-0 bg-transparent p-0 text-[24px] font-semibold text-[#f6f7fb] outline-none" />
                  </label>
                  <div class="rounded-[1.2rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.62)] px-4 py-3 sm:col-span-2">
                    <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Preview Summary</span>
                    <p class="mt-3 text-[18px] font-semibold text-[#f6f7fb]">{{ itemSizeLabel(draftPreviewItem) }}</p>
                  </div>
                </div>
              </article>

              <article class="rounded-[1.6rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(8,16,30,0.78)] p-5">
                <div class="flex flex-wrap items-center justify-between gap-3">
                  <div>
                    <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Tags</p>
                    <p class="mt-2 text-[14px] leading-relaxed text-[#d8dce7]/58">Reuse existing game tags or mint a new one here. The backend keeps tags unique per game.</p>
                  </div>
                </div>

                <div class="mt-5 grid gap-4 xl:grid-cols-[minmax(0,1fr)_auto]">
                  <label class="block">
                    <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Create New Tag</span>
                    <input
                      v-model="newTagDraft"
                      type="text"
                      maxlength="60"
                      :disabled="createItemSubmitting"
                      placeholder="Example: Boss Loot"
                      class="session-input mt-2 w-full rounded-[1.25rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.72)] px-4 py-3 text-[14px] text-[#f6f7fb] outline-none placeholder:text-[#7ec8e3]/30"
                      @keydown.enter.prevent="addDraftTagFromInput"
                    />
                  </label>

                  <button
                    type="button"
                    @click="addDraftTagFromInput"
                    :disabled="createItemSubmitting"
                    class="inline-flex cursor-pointer items-center justify-center gap-2 self-end rounded-xl border border-[rgba(233,69,96,0.22)] bg-[rgba(233,69,96,0.12)] px-4 py-3 text-[13px] font-semibold text-[#ffe0e7] transition-all duration-200 hover:border-[rgba(233,69,96,0.38)] disabled:cursor-not-allowed disabled:opacity-60"
                  >
                    <Plus class="h-4 w-4" :stroke-width="2" />
                    Create Tag
                  </button>
                </div>

                <div class="mt-5">
                  <p class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Selected Tags</p>
                  <div v-if="itemDraft.tags.length" class="mt-3 flex flex-wrap gap-2">
                    <button
                      v-for="tag in itemDraft.tags"
                      :key="`selected-${tag}`"
                      type="button"
                      @click="removeDraftTag(tag)"
                      class="inline-flex cursor-pointer items-center gap-2 rounded-full border border-[rgba(233,69,96,0.24)] bg-[rgba(233,69,96,0.12)] px-3 py-1.5 text-[12px] text-[#ffe0e7]"
                    >
                      {{ tag }}
                      <X class="h-3.5 w-3.5" :stroke-width="2" />
                    </button>
                  </div>
                  <p v-else class="mt-3 text-[14px] text-[#d8dce7]/58">No tags assigned yet.</p>
                </div>

                <div v-if="selectedTagSuggestions.length" class="mt-5">
                  <p class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Existing Game Tags</p>
                  <div class="mt-3 flex flex-wrap gap-2">
                    <button
                      v-for="tag in selectedTagSuggestions"
                      :key="`suggestion-${tag}`"
                      type="button"
                      @click="toggleDraftTag(tag)"
                      class="cursor-pointer rounded-full border border-[rgba(126,200,227,0.14)] bg-[rgba(126,200,227,0.08)] px-3 py-1.5 text-[12px] text-[#d8dce7]/72 transition-all duration-200 hover:border-[rgba(126,200,227,0.24)]"
                    >
                      {{ tag }}
                    </button>
                  </div>
                </div>
              </article>

              <article class="rounded-[1.6rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(8,16,30,0.78)] p-5">
                <div class="flex flex-wrap items-center justify-between gap-3">
                  <div>
                    <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Requirements</p>
                    <p class="mt-2 text-[14px] leading-relaxed text-[#d8dce7]/58">Add explicit stat gates row by row so nothing is hidden behind dense text parsing.</p>
                  </div>
                  <button
                    type="button"
                    @click="addRequirementRow"
                    :disabled="createItemSubmitting"
                    class="inline-flex cursor-pointer items-center gap-2 rounded-xl border border-[rgba(233,69,96,0.22)] bg-[rgba(233,69,96,0.12)] px-4 py-2.5 text-[13px] font-semibold text-[#ffe0e7] transition-all duration-200 hover:border-[rgba(233,69,96,0.38)] disabled:cursor-not-allowed disabled:opacity-60"
                  >
                    <Plus class="h-4 w-4" :stroke-width="2" />
                    Add Requirement
                  </button>
                </div>

                <div v-if="itemDraft.requiredAttributes.length" class="mt-5 space-y-3">
                  <article
                    v-for="(requirement, index) in itemDraft.requiredAttributes"
                    :key="`requirement-${index}`"
                    class="rounded-[1.2rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.62)] p-4"
                  >
                    <div class="grid gap-3 lg:grid-cols-[minmax(0,1fr)_180px_auto]">
                      <label class="block">
                        <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Attribute</span>
                        <select v-model="requirement.attribute_name" class="session-input mt-2 w-full rounded-xl border border-[rgba(126,200,227,0.12)] bg-[rgba(126,200,227,0.08)] px-3 py-2.5 text-[14px] text-[#f6f7fb] outline-none">
                          <option v-for="option in attributeOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
                        </select>
                      </label>
                      <label class="block">
                        <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Minimum</span>
                        <input v-model.number="requirement.min_value" type="number" class="session-input mt-2 w-full rounded-xl border border-[rgba(126,200,227,0.12)] bg-[rgba(126,200,227,0.08)] px-3 py-2.5 text-[14px] text-[#f6f7fb] outline-none" />
                      </label>
                      <div class="flex items-end justify-start lg:justify-end">
                        <button type="button" @click="removeRequirementRow(index)" :disabled="createItemSubmitting" class="inline-flex cursor-pointer items-center rounded-xl border border-[rgba(248,113,113,0.2)] bg-[rgba(248,113,113,0.12)] px-3 py-2.5 text-[12px] font-semibold text-[#fecaca] transition-all duration-200 hover:border-[rgba(248,113,113,0.35)] disabled:cursor-not-allowed disabled:opacity-60">Remove</button>
                      </div>
                    </div>
                  </article>
                </div>

                <p v-else class="mt-5 text-[14px] text-[#d8dce7]/58">No requirements yet. Add rows as needed.</p>
              </article>

              <article class="rounded-[1.6rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(8,16,30,0.78)] p-5">
                <div class="flex flex-wrap items-center justify-between gap-3">
                  <div>
                    <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Modifiers</p>
                    <p class="mt-2 text-[14px] leading-relaxed text-[#d8dce7]/58">Build bonuses and penalties row by row, with a clear toggle for flat versus percent values.</p>
                  </div>
                  <button
                    type="button"
                    @click="addModifierRow"
                    :disabled="createItemSubmitting"
                    class="inline-flex cursor-pointer items-center gap-2 rounded-xl border border-[rgba(233,69,96,0.22)] bg-[rgba(233,69,96,0.12)] px-4 py-2.5 text-[13px] font-semibold text-[#ffe0e7] transition-all duration-200 hover:border-[rgba(233,69,96,0.38)] disabled:cursor-not-allowed disabled:opacity-60"
                  >
                    <Plus class="h-4 w-4" :stroke-width="2" />
                    Add Modifier
                  </button>
                </div>

                <div v-if="itemDraft.attributeModifiers.length" class="mt-5 space-y-3">
                  <article
                    v-for="(modifier, index) in itemDraft.attributeModifiers"
                    :key="`modifier-${index}`"
                    class="rounded-[1.2rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.62)] p-4"
                  >
                    <div class="grid gap-3 xl:grid-cols-[minmax(0,1fr)_160px_180px_auto]">
                      <label class="block">
                        <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Attribute</span>
                        <select v-model="modifier.attribute_name" class="session-input mt-2 w-full rounded-xl border border-[rgba(126,200,227,0.12)] bg-[rgba(126,200,227,0.08)] px-3 py-2.5 text-[14px] text-[#f6f7fb] outline-none">
                          <option v-for="option in attributeOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
                        </select>
                      </label>
                      <label class="block">
                        <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Value</span>
                        <input v-model.number="modifier.modifier_value" type="number" class="session-input mt-2 w-full rounded-xl border border-[rgba(126,200,227,0.12)] bg-[rgba(126,200,227,0.08)] px-3 py-2.5 text-[14px] text-[#f6f7fb] outline-none" />
                      </label>
                      <label class="block">
                        <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Mode</span>
                        <select v-model="modifier.is_percentage" class="session-input mt-2 w-full rounded-xl border border-[rgba(126,200,227,0.12)] bg-[rgba(126,200,227,0.08)] px-3 py-2.5 text-[14px] text-[#f6f7fb] outline-none">
                          <option :value="false">Flat</option>
                          <option :value="true">Percent</option>
                        </select>
                      </label>
                      <div class="flex items-end justify-start xl:justify-end">
                        <button type="button" @click="removeModifierRow(index)" :disabled="createItemSubmitting" class="inline-flex cursor-pointer items-center rounded-xl border border-[rgba(248,113,113,0.2)] bg-[rgba(248,113,113,0.12)] px-3 py-2.5 text-[12px] font-semibold text-[#fecaca] transition-all duration-200 hover:border-[rgba(248,113,113,0.35)] disabled:cursor-not-allowed disabled:opacity-60">Remove</button>
                      </div>
                    </div>
                  </article>
                </div>

                <p v-else class="mt-5 text-[14px] text-[#d8dce7]/58">No modifiers yet. Add rows here for bonuses and penalties.</p>
              </article>
            </section>

            <aside class="min-h-0 overflow-y-auto">
              <article class="rounded-[1.6rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(8,16,30,0.78)] p-5">
                <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Live Preview</p>
                <div class="mt-5 overflow-hidden rounded-[1.5rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.66)]">
                  <div class="flex h-64 items-center justify-center" :class="itemFrameClass(draftPreviewItem.rarity)">
                    <span class="font-[Cinzel] text-[40px] font-bold">{{ itemGlyph(draftPreviewItem) }}</span>
                  </div>
                </div>

                <div class="mt-5">
                  <div class="flex flex-wrap items-center gap-2">
                    <h3 class="font-[Cinzel] text-[28px] font-bold text-[#f6f7fb]">{{ draftPreviewItem.name }}</h3>
                    <span class="rounded-full border px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.16em]" :class="rarityBadgeClass(draftPreviewItem.rarity)">
                      {{ formatLabel(draftPreviewItem.rarity) }}
                    </span>
                  </div>
                  <p class="mt-3 text-[14px] leading-relaxed text-[#d8dce7]/62">{{ draftPreviewItem.description || 'The description preview appears here as you type.' }}</p>
                </div>

                <div class="mt-5 grid gap-3 sm:grid-cols-2">
                  <div class="rounded-[1.2rem] border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] px-4 py-3">
                    <p class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Category</p>
                    <p class="mt-2 text-[15px] font-semibold text-[#f6f7fb]">{{ formatLabel(draftPreviewItem.category) }}</p>
                  </div>
                  <div class="rounded-[1.2rem] border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] px-4 py-3">
                    <p class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Grid Size</p>
                    <p class="mt-2 text-[15px] font-semibold text-[#f6f7fb]">{{ itemSizeLabel(draftPreviewItem) }}</p>
                  </div>
                  <div class="rounded-[1.2rem] border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] px-4 py-3 sm:col-span-2">
                    <p class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Equip Slot</p>
                    <p class="mt-2 text-[15px] font-semibold text-[#f6f7fb]">{{ draftPreviewItem.equip_slot ? formatLabel(draftPreviewItem.equip_slot) : 'Not equippable' }}</p>
                  </div>
                </div>

                <div class="mt-5">
                  <p class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Custom Tags</p>
                  <div v-if="draftPreviewItem.tags.length" class="mt-3 flex flex-wrap gap-2">
                    <span v-for="tag in draftPreviewItem.tags" :key="`preview-tag-${tag}`" class="rounded-full border border-[rgba(233,69,96,0.18)] bg-[rgba(233,69,96,0.08)] px-3 py-1.5 text-[12px] text-[#ffe0e7]">
                      {{ tag }}
                    </span>
                  </div>
                  <p v-else class="mt-3 text-[14px] text-[#d8dce7]/58">No tags selected yet.</p>
                </div>

                <div class="mt-5 grid gap-4">
                  <div class="rounded-[1.4rem] border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] p-4">
                    <div class="flex items-center justify-between gap-3">
                      <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Requirements</span>
                      <span class="text-[12px] text-[#d8dce7]/45">{{ draftPreviewItem.required_attributes.length }}</span>
                    </div>
                    <div v-if="draftPreviewItem.required_attributes.length" class="mt-4 space-y-2">
                      <div v-for="requirement in draftPreviewItem.required_attributes" :key="`preview-requirement-${requirement.attribute_name}-${requirement.min_value}`" class="rounded-[1rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.62)] px-3 py-2.5 text-[13px] text-[#f6f7fb]">
                        {{ formatAttributeLabel(requirement.attribute_name) }} {{ requirement.min_value }}+
                      </div>
                    </div>
                    <p v-else class="mt-4 text-[14px] text-[#d8dce7]/58">No requirements.</p>
                  </div>

                  <div class="rounded-[1.4rem] border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] p-4">
                    <div class="flex items-center justify-between gap-3">
                      <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Modifiers</span>
                      <span class="text-[12px] text-[#d8dce7]/45">{{ draftPreviewItem.attribute_modifiers.length }}</span>
                    </div>
                    <div v-if="draftPreviewItem.attribute_modifiers.length" class="mt-4 space-y-2">
                      <div v-for="modifier in draftPreviewItem.attribute_modifiers" :key="`preview-modifier-${modifier.attribute_name}-${modifier.modifier_value}-${modifier.is_percentage}`" class="rounded-[1rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.62)] px-3 py-2.5 text-[13px] text-[#f6f7fb]">
                        {{ formatAttributeLabel(modifier.attribute_name) }} {{ modifier.modifier_value > 0 ? '+' : '' }}{{ modifier.modifier_value }}{{ modifier.is_percentage ? '%' : '' }}
                      </div>
                    </div>
                    <p v-else class="mt-4 text-[14px] text-[#d8dce7]/58">No modifiers.</p>
                  </div>
                </div>
              </article>
            </aside>
          </div>

          <div class="flex flex-col gap-4 border-t border-[rgba(126,200,227,0.1)] px-5 py-4 sm:flex-row sm:items-center sm:justify-between sm:px-6">
            <div class="flex flex-wrap gap-3">
              <button
                type="button"
                @click="closeCreateItemModal"
                :disabled="createItemSubmitting"
                class="cursor-pointer rounded-xl border border-[rgba(126,200,227,0.16)] bg-[rgba(126,200,227,0.08)] px-4 py-2.5 text-[13px] font-semibold text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(126,200,227,0.3)] disabled:cursor-not-allowed disabled:opacity-60"
              >
                Cancel
              </button>
              <button
                type="button"
                @click="createItem"
                :disabled="createItemSubmitting"
                class="cursor-pointer rounded-xl border border-[rgba(233,69,96,0.28)] bg-[linear-gradient(135deg,rgba(233,69,96,0.9),rgba(194,49,82,0.9))] px-4 py-2.5 text-[13px] font-semibold text-white transition-all duration-200 hover:-translate-y-0.5 hover:shadow-[0_12px_30px_rgba(233,69,96,0.24)] disabled:cursor-not-allowed disabled:opacity-60"
              >
                {{ createItemSubmitting ? 'Creating...' : 'Create Item' }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </Teleport>
  </section>
</template>