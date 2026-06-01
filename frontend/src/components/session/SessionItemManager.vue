<script setup>
import {
  ChevronDown,
  ChevronRight,
  FilePlus2,
  FolderPlus,
  Package,
  Plus,
  Search,
  Trash2,
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
  { value: 'name-asc', label: 'Name A-Z' },
  { value: 'name-desc', label: 'Name Z-A' },
  { value: 'rarity', label: 'Quality' },
  { value: 'recent', label: 'Recently Updated' },
  { value: 'size', label: 'Grid Size' },
]

const ui = {
  panel: 'relative overflow-hidden rounded-3xl border border-sky-300/15 bg-slate-950/75',
  card: 'relative overflow-hidden rounded-2xl border border-sky-300/15 bg-slate-950/75',
  statCard: 'relative overflow-hidden rounded-2xl border border-sky-300/15 bg-slate-950/75 px-5 py-4',
  block: 'relative overflow-hidden rounded-2xl border border-sky-300/15 bg-slate-950/75 p-5',
  nestedBlock: 'relative overflow-hidden rounded-2xl border border-sky-300/15 bg-slate-950/60 p-4',
  pill: 'inline-flex min-h-7 items-center justify-center rounded-full border border-sky-300/20 bg-sky-300/10 px-3 py-0.5 text-xs text-slate-300',
  eyebrow: 'm-0 text-xs uppercase tracking-widest text-sky-300/60',
  title: 'mt-2 font-[Cinzel] text-3xl font-bold text-slate-50 lg:text-4xl',
  description: 'mt-3 text-sm leading-7 text-slate-300/70 break-words',
  ghostButton: 'inline-flex min-h-11 items-center justify-center gap-2 rounded-2xl border border-sky-300/20 bg-sky-300/10 px-4 py-3 text-sm font-semibold text-slate-50 transition duration-200 hover:-translate-y-px disabled:cursor-not-allowed disabled:opacity-60',
  accentButton: 'inline-flex min-h-11 items-center justify-center gap-2 rounded-2xl border border-rose-400/30 bg-rose-400/15 px-4 py-3 text-sm font-semibold text-rose-100 transition duration-200 hover:-translate-y-px disabled:cursor-not-allowed disabled:opacity-60',
  searchField: 'flex min-h-12 w-full items-center gap-3 rounded-2xl border border-sky-300/15 bg-slate-950/90 px-4',
  fieldLabel: 'grid gap-2',
  fieldLabelText: 'text-xs uppercase tracking-widest text-sky-300/60',
  input: 'min-h-12 w-full rounded-2xl border border-sky-300/15 bg-slate-950/90 px-4 py-3 text-slate-50 outline-none',
  textarea: 'min-h-28 w-full resize-y rounded-2xl border border-sky-300/15 bg-slate-950/90 px-4 py-3 text-slate-50 outline-none',
  inlineButton: 'inline-flex min-h-10 items-center justify-center gap-2 rounded-xl border border-sky-300/15 bg-sky-300/10 px-4 py-2.5 text-slate-50 transition duration-200 hover:border-rose-400/30 hover:bg-rose-400/10',
  iconButton: 'inline-flex h-12 w-12 self-end items-center justify-center rounded-xl border border-sky-300/15 bg-sky-300/10 text-slate-50 transition duration-200 hover:border-rose-400/30 hover:bg-rose-400/10',
  dropHighlight: 'border-rose-400/35 bg-rose-400/10',
  activeHighlight: 'border-rose-400/35 bg-rose-400/15',
  stackItem: 'rounded-xl border border-sky-300/15 bg-sky-300/5 px-4 py-3 text-sm text-slate-50 break-words',
  itemCard: 'grid cursor-pointer gap-3 rounded-3xl border border-sky-300/15 bg-slate-950/75 p-4 text-left transition duration-200 hover:-translate-y-px',
  treeRowButton: 'flex min-h-11 w-full items-center justify-between gap-3 rounded-2xl border border-sky-300/15 bg-slate-950/65 px-4 py-3 text-left',
  chipButton: 'cursor-pointer rounded-full border border-sky-300/15 bg-sky-300/10 px-4 py-2.5 text-slate-300 transition duration-200 hover:border-sky-300/25 hover:bg-sky-300/15',
  panelHeader: 'flex items-start justify-between gap-4 px-5 pb-3 pt-5',
  sectionLabelRow: 'mb-4 flex items-start justify-between gap-4',
  copy: 'text-sm leading-7 text-slate-300/70 break-words',
  titleText: 'min-w-0 text-sm font-bold leading-6 text-slate-50 break-words',
  badge: 'inline-flex min-h-7 items-center justify-center rounded-full border px-3 py-0.5 text-xs font-bold uppercase tracking-wide',
  modalBackdrop: 'absolute inset-0 bg-black/70 backdrop-blur-md',
  modalCard: 'relative z-10 flex flex-col overflow-hidden rounded-3xl border border-sky-300/15 bg-gradient-to-b from-slate-900 to-slate-950 shadow-2xl shadow-black/50',
  closeButton: 'absolute right-5 top-5 inline-flex h-9 w-9 items-center justify-center rounded-full text-sky-300/60',
  modalHeader: 'px-4 py-6 pr-16 md:px-6',
  modalBody: 'grid gap-5 px-4 pb-6 md:px-6',
  modalFooter: 'flex justify-end gap-3 border-t border-sky-300/10 px-4 py-6 md:px-6',
  artFrame: 'relative flex size-24 shrink-0 items-center justify-center overflow-hidden rounded-2xl border-2 font-[Cinzel] text-3xl font-bold text-slate-50',
  sizeBadge: 'absolute right-2 top-2 min-h-6 rounded-full border border-white/15 bg-slate-950/80 px-2 py-0.5 text-xs font-bold text-slate-50',
  rowCard: 'grid items-end gap-3 rounded-2xl border border-sky-300/15 bg-sky-300/5 p-4',
}

const artFrameClasses = {
  common: 'border-white/70',
  uncommon: 'border-yellow-400/80',
  rare: 'border-blue-400/80',
  epic: 'border-purple-400/80',
  masterwork: 'border-orange-400/85',
  legendary: 'border-green-400/80',
  unique: 'border-red-400/85',
}

const rarityBadgeClasses = {
  common: 'border-white/25 bg-white/10 text-slate-50',
  uncommon: 'border-yellow-400/30 bg-yellow-700/20 text-yellow-200',
  rare: 'border-blue-400/30 bg-blue-700/20 text-blue-300',
  epic: 'border-purple-400/30 bg-purple-700/20 text-purple-200',
  masterwork: 'border-orange-400/30 bg-orange-700/20 text-orange-300',
  legendary: 'border-green-400/30 bg-green-700/20 text-green-300',
  unique: 'border-red-400/30 bg-red-900/20 text-red-300',
}

const currentFolderId = ref(null)
const searchQuery = ref('')
const rarityFilter = ref('all')
const categoryFilter = ref('all')
const slotFilter = ref('all')
const sortMode = ref('name-asc')
const folders = ref([])
const expandedFolderIds = ref([])
const itemLocations = ref({})
const draggingEntry = ref(null)
const dragTargetKey = ref('')
const showCreateFolderModal = ref(false)
const showCreateItemModal = ref(false)
const folderNameDraft = ref('')
const folderParentDraft = ref('')
const selectedItemId = ref('')
const hoveredItemId = ref('')
const saveStateReady = ref(false)
const createItemSubmitting = ref(false)
const itemCreateError = ref('')
const itemDraft = ref(createEmptyItemDraft())

const storageKey = computed(() => props.gameId ? `dogmaliter:item-manager:${props.gameId}` : '')
const searchNeedle = computed(() => searchQuery.value.trim().toLowerCase())
const folderMap = computed(() => Object.fromEntries(folders.value.map(folder => [folder.id, folder])))
const allItems = computed(() => (props.items ?? []).map(item => normalizeItem(item)))
const itemById = computed(() => Object.fromEntries(allItems.value.map(item => [item.id, item])))
const currentFolder = computed(() => folderMap.value[currentFolderId.value] ?? null)
const attributeOptions = computed(() => {
  const seen = new Set(BASE_ATTRIBUTE_OPTIONS.map(option => option.value))
  const customOptions = []

  for (const character of props.characters ?? []) {
    for (const attribute of character?.custom_attributes ?? []) {
      const attributeName = normalizeAttributeName(attribute?.name)
      if (!attributeName || seen.has(attributeName)) continue

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
const activeFilterCount = computed(() => {
  let count = 0
  if (searchNeedle.value) count += 1
  if (rarityFilter.value !== 'all') count += 1
  if (categoryFilter.value !== 'all') count += 1
  if (slotFilter.value !== 'all') count += 1
  return count
})
const hasActiveFilters = computed(() => activeFilterCount.value > 0)
const breadcrumbs = computed(() => {
  const trail = [{ id: null, name: 'Campaign Archive' }]
  if (!currentFolderId.value) return trail

  const chain = []
  let cursorId = currentFolderId.value
  while (cursorId && folderMap.value[cursorId]) {
    const folder = folderMap.value[cursorId]
    chain.unshift({ id: folder.id, name: folder.name })
    cursorId = normalizeFolderId(folder.parentId)
  }

  return [...trail, ...chain]
})
const folderOptions = computed(() => [{ value: '', label: 'Campaign Archive' }, ...folders.value
  .map(folder => ({ value: folder.id, label: folderPathLabel(folder.id) }))
  .sort((left, right) => left.label.localeCompare(right.label))])
const matchingFolderIds = computed(() => {
  const ids = new Set()
  if (!hasActiveFilters.value) return ids

  for (const item of allItems.value) {
    if (!itemMatchesFilters(item)) continue

    let cursorId = normalizeFolderId(itemLocations.value[item.id])
    while (cursorId) {
      ids.add(cursorId)
      cursorId = normalizeFolderId(folderMap.value[cursorId]?.parentId)
    }
  }

  return ids
})
const treeRows = computed(() => flattenTreeRows())
const visibleFolders = computed(() => {
  const pool = hasActiveFilters.value
    ? folders.value.filter(folder => matchingFolderIds.value.has(folder.id) || folderMatchesSearch(folder))
    : childFoldersOf(currentFolderId.value)

  return [...pool].sort((left, right) => left.name.localeCompare(right.name))
})
const visibleItems = computed(() => {
  const pool = hasActiveFilters.value
    ? allItems.value.filter(item => itemMatchesFilters(item))
    : allItems.value.filter(item => normalizeFolderId(itemLocations.value[item.id]) === normalizeFolderId(currentFolderId.value))

  return [...pool].sort(compareItems)
})
const inspectorItem = computed(() => itemById.value[hoveredItemId.value] ?? itemById.value[selectedItemId.value] ?? visibleItems.value[0] ?? allItems.value[0] ?? null)
const inspectorModeLabel = computed(() => {
  if (hoveredItemId.value && itemById.value[hoveredItemId.value]) return 'Hover Preview'
  if (selectedItemId.value && itemById.value[selectedItemId.value]) return 'Pinned Preview'
  if (inspectorItem.value) return 'Auto Preview'
  return 'Inspector'
})
const currentFolderDropKey = computed(() => currentFolderId.value ? `folder:${currentFolderId.value}` : 'folder:root')
const filterBadges = computed(() => {
  const badges = []
  if (searchNeedle.value) badges.push(`Query: ${searchQuery.value.trim()}`)
  if (rarityFilter.value !== 'all') badges.push(`Quality: ${formatLabel(rarityFilter.value)}`)
  if (categoryFilter.value !== 'all') badges.push(`Category: ${formatLabel(categoryFilter.value)}`)
  if (slotFilter.value !== 'all') badges.push(`Slot: ${formatLabel(slotFilter.value)}`)
  return badges
})
const explorerStats = computed(() => ({
  totalItems: allItems.value.length,
  visibleItems: visibleItems.value.length,
  totalFolders: folders.value.length,
  activeFilters: activeFilterCount.value,
}))
const draftPreviewItem = computed(() => normalizeItem({
  id: 'draft-preview',
  name: itemDraft.value.name.trim() || 'Untitled Item',
  description: itemDraft.value.description.trim(),
  rarity: itemDraft.value.rarity,
  category: itemDraft.value.category,
  grid_width: Math.max(1, Number(itemDraft.value.gridWidth) || 1),
  grid_height: Math.max(1, Number(itemDraft.value.gridHeight) || 1),
  equip_slot: itemDraft.value.equipSlot || null,
  required_attributes: parseRequirements(itemDraft.value.requirements),
  attribute_modifiers: parseModifiers(itemDraft.value.modifiers),
  created_at: new Date().toISOString(),
  updated_at: new Date().toISOString(),
}))

watch(() => props.gameId, () => {
  loadState()
}, { immediate: true })

watch([folders, expandedFolderIds, itemLocations, currentFolderId], () => {
  saveState()
}, { deep: true })

watch(allItems, () => {
  if (selectedItemId.value && !itemById.value[selectedItemId.value]) {
    selectedItemId.value = ''
  }

  if (hoveredItemId.value && !itemById.value[hoveredItemId.value]) {
    hoveredItemId.value = ''
  }
}, { deep: true })

function createEmptyItemDraft() {
  return {
    folderId: currentFolderId.value ?? '',
    name: '',
    description: '',
    rarity: 'common',
    category: 'other',
    gridWidth: 1,
    gridHeight: 1,
    equipSlot: '',
    requirements: [createEmptyRequirementRow()],
    modifiers: [createEmptyModifierRow()],
  }
}

function createEmptyRequirementRow() {
  return {
    id: createId('requirement'),
    attribute_name: '',
    min_value: 0,
  }
}

function createEmptyModifierRow() {
  return {
    id: createId('modifier'),
    attribute_name: '',
    sign: '+',
    magnitude: 0,
    is_percentage: false,
  }
}

function addRequirementRow() {
  itemDraft.value.requirements = [...itemDraft.value.requirements, createEmptyRequirementRow()]
}

function removeRequirementRow(rowId) {
  itemDraft.value.requirements = itemDraft.value.requirements.filter(row => row.id !== rowId)
}

function addModifierRow() {
  itemDraft.value.modifiers = [...itemDraft.value.modifiers, createEmptyModifierRow()]
}

function removeModifierRow(rowId) {
  itemDraft.value.modifiers = itemDraft.value.modifiers.filter(row => row.id !== rowId)
}

function normalizeFolderId(folderId) {
  return folderId || null
}

function createId(prefix) {
  if (window.crypto?.randomUUID) {
    return `${prefix}-${window.crypto.randomUUID()}`
  }

  return `${prefix}-${Date.now()}-${Math.random().toString(16).slice(2)}`
}

function extractTypeName(type) {
  if (typeof type === 'string') return type.trim()
  if (type && typeof type === 'object') {
    return String(type.type_name || type.name || '').trim()
  }
  return ''
}

function normalizeAttributeName(value) {
  return String(value || '')
    .trim()
    .toLowerCase()
    .replace(/\s+/g, '_')
    .replace(/^_+|_+$/g, '')
}

function formatAttributeLabel(value) {
  return formatLabel(String(value || '').trim())
}

function normalizeRarityValue(value) {
  const normalized = String(value || '').trim().toLowerCase()
  if (normalized === 'artifact') return 'unique'
  return RARITY_OPTIONS.includes(normalized) ? normalized : 'common'
}

function normalizeCategoryValue(value, types = [], equipSlotValue = null) {
  const normalized = String(value || '').trim().toLowerCase()
  const equipSlot = normalizeEquipSlotValue(equipSlotValue)

  if (CATEGORY_OPTIONS.includes(normalized)) {
    if (equipSlot && (normalized === 'other' || normalized === 'loot')) return 'equipment'
    return normalized
  }

  const fallback = types
    .map(type => String(type || '').trim().toLowerCase())
    .find(type => CATEGORY_OPTIONS.includes(type))

  if (fallback) {
    if (equipSlot && (fallback === 'other' || fallback === 'loot')) return 'equipment'
    return fallback
  }

  return equipSlot ? 'equipment' : 'other'
}

function normalizeEquipSlotValue(value) {
  const normalized = String(value || '').trim().toLowerCase()
  if (!normalized) return null
  if (normalized === 'ring_1' || normalized === 'ring_2') return 'ring'
  return EQUIP_SLOT_OPTIONS.includes(normalized) ? normalized : null
}

function normalizeRequiredAttributes(entries) {
  return (Array.isArray(entries) ? entries : [])
    .map(entry => ({
      attribute_name: normalizeAttributeName(entry?.attribute_name),
      min_value: Number(entry?.min_value) || 0,
    }))
    .filter(entry => entry.attribute_name)
}

function normalizeAttributeModifiers(entries) {
  return (Array.isArray(entries) ? entries : [])
    .map(entry => ({
      attribute_name: normalizeAttributeName(entry?.attribute_name),
      modifier_value: Number(entry?.modifier_value) || 0,
      is_percentage: Boolean(entry?.is_percentage),
    }))
    .filter(entry => entry.attribute_name)
}

function normalizeItem(item) {
  const rawTypes = Array.isArray(item.types) ? item.types.map(extractTypeName).filter(Boolean) : []
  const equipSlot = normalizeEquipSlotValue(item.equip_slot)
  const category = normalizeCategoryValue(item.category, rawTypes, equipSlot)
  const providedTags = Array.isArray(item.tags)
    ? item.tags.map(tag => String(tag || '').trim()).filter(Boolean)
    : []
  const tags = Array.from(new Set([
    category,
    ...providedTags,
    ...rawTypes.filter(type => !CATEGORY_OPTIONS.includes(type.toLowerCase())),
  ]))

  return {
    ...item,
    name: item.name || 'Unnamed item',
    description: item.description || '',
    rarity: normalizeRarityValue(item.rarity),
    category,
    equip_slot: equipSlot,
    is_equippable: Boolean(equipSlot),
    grid_width: Math.max(1, Number(item.grid_width) || 1),
    grid_height: Math.max(1, Number(item.grid_height) || 1),
    tags,
    required_attributes: normalizeRequiredAttributes(item.required_attributes),
    attribute_modifiers: normalizeAttributeModifiers(item.attribute_modifiers),
  }
}

function itemArtFrameClass(rarity) {
  return artFrameClasses[normalizeRarityValue(rarity)] ?? artFrameClasses.common
}

function rarityBadgeClass(rarity) {
  return rarityBadgeClasses[normalizeRarityValue(rarity)] ?? rarityBadgeClasses.common
}

function childFoldersOf(folderId) {
  const parentId = normalizeFolderId(folderId)
  return folders.value.filter(folder => normalizeFolderId(folder.parentId) === parentId)
}

function isFolderExpanded(folderId) {
  return expandedFolderIds.value.includes(folderId)
}

function toggleFolderExpanded(folderId) {
  const next = new Set(expandedFolderIds.value)
  if (next.has(folderId)) {
    next.delete(folderId)
  } else {
    next.add(folderId)
  }
  expandedFolderIds.value = [...next]
}

function ensureFolderExpanded(folderId) {
  let cursorId = normalizeFolderId(folderId)
  if (!cursorId) return

  const next = new Set(expandedFolderIds.value)
  while (cursorId) {
    next.add(cursorId)
    cursorId = normalizeFolderId(folderMap.value[cursorId]?.parentId)
  }
  expandedFolderIds.value = [...next]
}

function flattenTreeRows(parentId = null, depth = 0) {
  const rows = []

  for (const folder of childFoldersOf(parentId).sort((left, right) => left.name.localeCompare(right.name))) {
    const hasChildren = childFoldersOf(folder.id).length > 0
    rows.push({ ...folder, depth, hasChildren })

    if (hasChildren && isFolderExpanded(folder.id)) {
      rows.push(...flattenTreeRows(folder.id, depth + 1))
    }
  }

  return rows
}

function buildFolderPath(folderId) {
  const chain = []
  let cursorId = normalizeFolderId(folderId)

  while (cursorId && folderMap.value[cursorId]) {
    const folder = folderMap.value[cursorId]
    chain.unshift(folder)
    cursorId = normalizeFolderId(folder.parentId)
  }

  return chain
}

function folderPathLabel(folderId) {
  const path = buildFolderPath(folderId)
  if (!path.length) return 'Campaign Archive'
  return `Campaign Archive / ${path.map(folder => folder.name).join(' / ')}`
}

function folderMatchesSearch(folder) {
  if (!searchNeedle.value) return false
  return folderPathLabel(folder.id).toLowerCase().includes(searchNeedle.value)
}

function folderContainsLocation(folderId, locationId) {
  let cursorId = normalizeFolderId(locationId)
  while (cursorId) {
    if (cursorId === folderId) return true
    cursorId = normalizeFolderId(folderMap.value[cursorId]?.parentId)
  }

  return false
}

function folderDirectItemCount(folderId) {
  return allItems.value.filter(item => normalizeFolderId(itemLocations.value[item.id]) === normalizeFolderId(folderId)).length
}

function folderVisibleItemCount(folderId) {
  if (!hasActiveFilters.value) {
    return folderDirectItemCount(folderId)
  }

  return allItems.value.filter(item => itemMatchesFilters(item) && folderContainsLocation(folderId, itemLocations.value[item.id])).length
}

function folderDescription(folder) {
  const childCount = childFoldersOf(folder.id).length
  const itemCount = folderVisibleItemCount(folder.id)
  return `${childCount} folders · ${itemCount} items`
}

function formatLabel(value) {
  if (!value) return 'None'
  return String(value)
    .replaceAll('_', ' ')
    .split(' ')
    .filter(Boolean)
    .map(part => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ')
}

function formatDate(value) {
  if (!value) return 'Just now'
  const timestamp = Date.parse(value)
  if (Number.isNaN(timestamp)) return 'Just now'
  return new Intl.DateTimeFormat('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  }).format(new Date(timestamp))
}

function itemImageUrl(item) {
  if (!item?.image_id) return ''
  return `${API_URL}/api/uploads/${item.image_id}`
}

function itemInitial(item) {
  return (item?.name || '?').charAt(0).toUpperCase()
}

function shortText(value, maxLength = 120) {
  const text = String(value || '').trim()
  if (!text) return 'No description provided yet.'
  if (text.length <= maxLength) return text
  return `${text.slice(0, maxLength - 3)}...`
}

function rarityRank(rarity) {
  return {
    common: 0,
    uncommon: 1,
    rare: 2,
    epic: 3,
    masterwork: 4,
    legendary: 5,
    unique: 6,
  }[rarity] ?? 0
}

function compareItems(left, right) {
  if (sortMode.value === 'name-desc') {
    return right.name.localeCompare(left.name)
  }

  if (sortMode.value === 'rarity') {
    const delta = rarityRank(right.rarity) - rarityRank(left.rarity)
    if (delta !== 0) return delta
    return left.name.localeCompare(right.name)
  }

  if (sortMode.value === 'recent') {
    const leftTime = Date.parse(left.updated_at || left.created_at || 0)
    const rightTime = Date.parse(right.updated_at || right.created_at || 0)
    if (rightTime !== leftTime) return rightTime - leftTime
    return left.name.localeCompare(right.name)
  }

  if (sortMode.value === 'size') {
    const leftSize = left.grid_width * left.grid_height
    const rightSize = right.grid_width * right.grid_height
    if (rightSize !== leftSize) return rightSize - leftSize
    return left.name.localeCompare(right.name)
  }

  return left.name.localeCompare(right.name)
}

function itemMatchesFilters(item) {
  if (searchNeedle.value) {
    const haystack = [
      item.name,
      item.description,
      item.rarity,
      item.category,
      item.equip_slot,
      folderPathLabel(itemLocations.value[item.id]),
      ...item.tags,
      ...item.required_attributes.map(entry => `${entry.attribute_name} ${entry.min_value}`),
      ...item.attribute_modifiers.map(entry => `${entry.attribute_name} ${entry.modifier_value}${entry.is_percentage ? '%' : ''}`),
    ]
      .filter(Boolean)
      .join(' ')
      .toLowerCase()

    if (!haystack.includes(searchNeedle.value)) return false
  }

  if (rarityFilter.value !== 'all' && item.rarity !== rarityFilter.value) return false
  if (categoryFilter.value !== 'all' && item.category !== categoryFilter.value) return false
  if (slotFilter.value !== 'all' && item.equip_slot !== slotFilter.value) return false

  return true
}

function loadState() {
  saveStateReady.value = false
  currentFolderId.value = null
  searchQuery.value = ''
  rarityFilter.value = 'all'
  categoryFilter.value = 'all'
  slotFilter.value = 'all'
  sortMode.value = 'name-asc'
  folders.value = []
  expandedFolderIds.value = []
  itemLocations.value = {}

  if (!storageKey.value) {
    saveStateReady.value = true
    return
  }

  try {
    const raw = window.localStorage.getItem(storageKey.value)
    if (raw) {
      const parsed = JSON.parse(raw)
      folders.value = Array.isArray(parsed.folders) ? parsed.folders : []
      expandedFolderIds.value = Array.isArray(parsed.expandedFolderIds) ? parsed.expandedFolderIds : []
      itemLocations.value = parsed.itemLocations && typeof parsed.itemLocations === 'object' ? parsed.itemLocations : {}
      currentFolderId.value = normalizeFolderId(parsed.currentFolderId)
    }
  } catch {
    folders.value = []
    expandedFolderIds.value = []
    itemLocations.value = {}
    currentFolderId.value = null
  }

  if (currentFolderId.value && !folderMap.value[currentFolderId.value]) {
    currentFolderId.value = null
  }

  ensureFolderExpanded(currentFolderId.value)
  saveStateReady.value = true
}

function saveState() {
  if (!saveStateReady.value || !storageKey.value) return

  window.localStorage.setItem(storageKey.value, JSON.stringify({
    folders: folders.value,
    expandedFolderIds: expandedFolderIds.value,
    itemLocations: itemLocations.value,
    currentFolderId: currentFolderId.value,
  }))
}

function selectFolder(folderId) {
  currentFolderId.value = normalizeFolderId(folderId)
  ensureFolderExpanded(currentFolderId.value)
}

function openCreateFolderModal(parentId = currentFolderId.value) {
  folderNameDraft.value = ''
  folderParentDraft.value = parentId ?? ''
  showCreateFolderModal.value = true
}

function createFolder() {
  const name = folderNameDraft.value.trim()
  if (!name) return

  const parentId = normalizeFolderId(folderParentDraft.value)
  const folder = {
    id: createId('folder'),
    name,
    parentId,
    created_at: new Date().toISOString(),
  }

  folders.value = [...folders.value, folder]
  if (parentId) ensureFolderExpanded(parentId)
  expandedFolderIds.value = [...new Set([...expandedFolderIds.value, folder.id])]
  currentFolderId.value = folder.id
  showCreateFolderModal.value = false
}

function openCreateItemModal() {
  itemDraft.value = createEmptyItemDraft()
  itemCreateError.value = ''
  showCreateItemModal.value = true
}

function parseRequirements(entries) {
  return normalizeRequiredAttributes(entries).map(entry => ({
    attribute_name: entry.attribute_name,
    min_value: Number(entry.min_value) || 0,
  }))
}

function parseModifiers(entries) {
  return (Array.isArray(entries) ? entries : [])
    .map(entry => ({
      attribute_name: normalizeAttributeName(entry?.attribute_name),
      modifier_value: (entry?.sign === '-' ? -1 : 1) * Math.abs(Number(entry?.magnitude) || 0),
      is_percentage: Boolean(entry?.is_percentage),
    }))
    .filter(entry => entry.attribute_name)
}

async function createItem() {
  const name = itemDraft.value.name.trim()
  if (!name || createItemSubmitting.value) return

  const folderId = normalizeFolderId(itemDraft.value.folderId)
  createItemSubmitting.value = true
  itemCreateError.value = ''

  try {
    const response = await auth.createGameItem(props.gameId, {
      name,
      description: itemDraft.value.description.trim(),
      rarity: itemDraft.value.rarity,
      category: itemDraft.value.category,
      grid_width: Math.max(1, Number(itemDraft.value.gridWidth) || 1),
      grid_height: Math.max(1, Number(itemDraft.value.gridHeight) || 1),
      equip_slot: itemDraft.value.equipSlot || null,
      required_attributes: parseRequirements(itemDraft.value.requirements),
      attribute_modifiers: parseModifiers(itemDraft.value.modifiers),
    })

    const createdItem = normalizeItem(response?.item ?? {})
    if (createdItem.id) {
      itemLocations.value = {
        ...itemLocations.value,
        [createdItem.id]: folderId,
      }
      selectedItemId.value = createdItem.id
      hoveredItemId.value = createdItem.id
    }

    currentFolderId.value = folderId
    ensureFolderExpanded(folderId)
    showCreateItemModal.value = false
    notify.success({ title: 'Item created', message: `${createdItem.name || name} was saved to the campaign archive.` })
    emit('created', { itemId: createdItem.id, folderId })
  } catch (error) {
    itemCreateError.value = getErrorMessage(error, 'Failed to create item')
    notify.error(error, 'Failed to create item')
  } finally {
    createItemSubmitting.value = false
  }
}

function openItemDetails(item) {
  selectedItemId.value = item.id
  hoveredItemId.value = item.id
}

function previewItem(itemId) {
  hoveredItemId.value = itemId || ''
}

function startItemDrag(itemId) {
  draggingEntry.value = { kind: 'item', id: itemId }
}

function startFolderDrag(folderId) {
  draggingEntry.value = { kind: 'folder', id: folderId }
}

function stopDrag() {
  draggingEntry.value = null
  dragTargetKey.value = ''
}

function isFolderAncestor(targetFolderId, sourceFolderId) {
  let cursorId = normalizeFolderId(targetFolderId)
  while (cursorId) {
    if (cursorId === sourceFolderId) return true
    cursorId = normalizeFolderId(folderMap.value[cursorId]?.parentId)
  }
  return false
}

function canDropIntoFolder(targetFolderId) {
  if (!draggingEntry.value) return false

  if (draggingEntry.value.kind === 'item') {
    return normalizeFolderId(itemLocations.value[draggingEntry.value.id]) !== normalizeFolderId(targetFolderId)
  }

  if (draggingEntry.value.id === targetFolderId) return false
  if (isFolderAncestor(targetFolderId, draggingEntry.value.id)) return false

  const sourceFolder = folderMap.value[draggingEntry.value.id]
  return normalizeFolderId(sourceFolder?.parentId) !== normalizeFolderId(targetFolderId)
}

function handleFolderDragOver(folderId) {
  if (!canDropIntoFolder(folderId)) return
  dragTargetKey.value = folderId ? `folder:${folderId}` : 'folder:root'
}

function dropIntoFolder(targetFolderId) {
  if (!canDropIntoFolder(targetFolderId)) return

  if (draggingEntry.value.kind === 'item') {
    itemLocations.value = {
      ...itemLocations.value,
      [draggingEntry.value.id]: normalizeFolderId(targetFolderId),
    }
  } else {
    folders.value = folders.value.map(folder => {
      if (folder.id !== draggingEntry.value.id) return folder
      return {
        ...folder,
        parentId: normalizeFolderId(targetFolderId),
      }
    })
  }

  dragTargetKey.value = ''
  draggingEntry.value = null
}

function dragTargetActive(targetKey) {
  return dragTargetKey.value === targetKey
}

function clearFilters() {
  searchQuery.value = ''
  rarityFilter.value = 'all'
  categoryFilter.value = 'all'
  slotFilter.value = 'all'
}
</script>

<template>
  <section class="grid gap-5">
    <section
      :class="[ui.panel, 'p-5']"
      style="background: radial-gradient(circle at top right, rgba(233, 69, 96, 0.14), transparent 26%), radial-gradient(circle at top left, rgba(126, 200, 227, 0.12), transparent 34%), linear-gradient(180deg, rgba(10, 18, 32, 0.96), rgba(8, 12, 24, 0.98)); box-shadow: 0 28px 80px rgba(0, 0, 0, 0.28);"
    >
      <div class="flex flex-col gap-6 xl:flex-row xl:items-end xl:justify-between">
        <div class="min-w-0">
          <p :class="ui.eyebrow">Item Explorer</p>
          <h2 :class="ui.title">Campaign Archive</h2>
          <p :class="ui.description">Server-backed GM library with explorer folders, search, drag-and-drop placement, and a large create modal with live preview.</p>
        </div>

        <div class="flex flex-wrap gap-3 xl:justify-end">
          <button :class="ui.ghostButton" @click="openCreateFolderModal()">
            <FolderPlus :size="16" />
            <span>New Folder</span>
          </button>
          <button :class="ui.accentButton" @click="openCreateItemModal()">
            <FilePlus2 :size="16" />
            <span>Create Item</span>
          </button>
        </div>
      </div>

      <div class="mt-5 grid gap-3 md:grid-cols-2 2xl:grid-cols-4">
        <article :class="ui.statCard">
          <span :class="ui.fieldLabelText">Total Items</span>
          <strong>{{ explorerStats.totalItems }}</strong>
        </article>
        <article :class="ui.statCard">
          <span :class="ui.fieldLabelText">Visible</span>
          <strong>{{ explorerStats.visibleItems }}</strong>
        </article>
        <article :class="ui.statCard">
          <span :class="ui.fieldLabelText">Folders</span>
          <strong>{{ explorerStats.totalFolders }}</strong>
        </article>
        <article :class="ui.statCard">
          <span :class="ui.fieldLabelText">Active Filters</span>
          <strong>{{ explorerStats.activeFilters }}</strong>
        </article>
      </div>
    </section>

    <div class="grid gap-5 xl:grid-cols-[18rem_minmax(0,1fr)] 2xl:grid-cols-[18rem_minmax(0,1fr)_24rem]">
      <aside :class="[ui.panel, 'pb-5']">
        <div :class="ui.panelHeader">
          <div>
            <p :class="ui.eyebrow">Directory Tree</p>
            <h3 :class="ui.title">Folders</h3>
          </div>
          <span :class="ui.pill">{{ folders.length }}</span>
        </div>

        <button
          :class="[
            ui.card,
            'mx-5 mt-5 flex items-center justify-between p-5 text-left',
            !currentFolderId ? ui.activeHighlight : '',
            dragTargetActive('folder:root') ? ui.dropHighlight : '',
          ]"
          @click="selectFolder(null)"
          @dragover.prevent="handleFolderDragOver(null)"
          @dragenter.prevent="handleFolderDragOver(null)"
          @drop.prevent="dropIntoFolder(null)"
        >
          <div>
            <span class="block text-sm font-bold text-slate-50">Campaign Archive</span>
            <span class="mt-1 block text-xs text-sky-300/60">{{ allItems.length }} items</span>
          </div>
          <Package :size="16" class="text-slate-300/80" />
        </button>

        <div v-if="treeRows.length" class="mt-4 grid max-h-[38rem] gap-1 overflow-auto px-2">
          <div
            v-for="folder in treeRows"
            :key="folder.id"
            class="grid items-center gap-1 [grid-template-columns:auto_minmax(0,1fr)]"
            :style="{ paddingLeft: `calc(${folder.depth} * 0.95rem)` }"
          >
            <button
              class="inline-flex h-7 w-7 items-center justify-center rounded-xl text-sky-300/60"
              :class="{ 'cursor-default': !folder.hasChildren, 'cursor-pointer': folder.hasChildren }"
              @click.stop="folder.hasChildren ? toggleFolderExpanded(folder.id) : null"
            >
              <ChevronDown v-if="folder.hasChildren && isFolderExpanded(folder.id)" :size="14" />
              <ChevronRight v-else-if="folder.hasChildren" :size="14" />
            </button>

            <button
              :class="[
                ui.treeRowButton,
                currentFolderId === folder.id ? ui.activeHighlight : '',
                dragTargetActive(`folder:${folder.id}`) ? ui.dropHighlight : '',
              ]"
              draggable="true"
              @click="selectFolder(folder.id)"
              @dragstart="startFolderDrag(folder.id)"
              @dragover.prevent="handleFolderDragOver(folder.id)"
              @dragenter.prevent="handleFolderDragOver(folder.id)"
              @drop.prevent="dropIntoFolder(folder.id)"
              @dragend="stopDrag"
            >
              <span :class="ui.titleText" :title="folder.name">{{ folder.name }}</span>
              <span class="mt-1 block text-xs text-sky-300/60">{{ folderVisibleItemCount(folder.id) }}</span>
            </button>
          </div>
        </div>

        <p v-else class="px-5 pb-5 pt-4 text-sm leading-7 text-slate-300/70">No folders yet. Create your first archive directory.</p>

        <div
          :class="[
            ui.card,
            'mx-5 mt-5 p-5 text-slate-300/80 leading-7',
            dragTargetActive(currentFolderDropKey) ? ui.dropHighlight : '',
          ]"
          @dragover.prevent="handleFolderDragOver(currentFolderId)"
          @dragenter.prevent="handleFolderDragOver(currentFolderId)"
          @drop.prevent="dropIntoFolder(currentFolderId)"
        >
          Drop items or folders into {{ currentFolder?.name || 'Campaign Archive' }}.
        </div>
      </aside>

      <div class="grid gap-5">
        <section :class="[ui.panel, 'pb-5']">
          <div class="mt-5 flex flex-col gap-4 px-5 xl:flex-row xl:items-start xl:justify-between">
            <label :class="ui.searchField">
              <Search :size="16" class="text-sky-300/60" />
              <input v-model="searchQuery" class="border-0 bg-transparent text-slate-50 outline-none" type="text" placeholder="Search by name, category, quality, slot or stats" />
              <button v-if="searchQuery" class="inline-flex items-center justify-center text-sky-300/60" @click="searchQuery = ''">
                <X :size="14" />
              </button>
            </label>

            <label :class="[ui.fieldLabel, 'min-w-0 xl:min-w-52']">
              <span :class="ui.fieldLabelText">Sort</span>
              <select v-model="sortMode" :class="ui.input">
                <option v-for="option in SORT_OPTIONS" :key="option.value" :value="option.value">{{ option.label }}</option>
              </select>
            </label>
          </div>

          <div class="grid gap-4 px-5 py-5 lg:grid-cols-3">
            <label :class="ui.fieldLabel">
              <span :class="ui.fieldLabelText">Quality</span>
              <select v-model="rarityFilter" :class="ui.input">
                <option value="all">All qualities</option>
                <option v-for="rarity in RARITY_OPTIONS" :key="rarity" :value="rarity">{{ formatLabel(rarity) }}</option>
              </select>
            </label>

            <label :class="ui.fieldLabel">
              <span :class="ui.fieldLabelText">Category</span>
              <select v-model="categoryFilter" :class="ui.input">
                <option value="all">All categories</option>
                <option v-for="category in CATEGORY_OPTIONS" :key="category" :value="category">{{ formatLabel(category) }}</option>
              </select>
            </label>

            <label :class="ui.fieldLabel">
              <span :class="ui.fieldLabelText">Equip Slot</span>
              <select v-model="slotFilter" :class="ui.input">
                <option value="all">All slots</option>
                <option v-for="slot in EQUIP_SLOT_OPTIONS" :key="slot" :value="slot">{{ formatLabel(slot) }}</option>
              </select>
            </label>
          </div>

          <div class="flex flex-col gap-4 px-5 xl:flex-row xl:items-start xl:justify-between">
            <div class="flex flex-wrap gap-3">
              <button
                v-for="crumb in breadcrumbs"
                :key="crumb.id ?? 'root'"
                :class="ui.chipButton"
                @click="selectFolder(crumb.id)"
              >
                {{ crumb.name }}
              </button>
            </div>

            <button
              v-if="activeFilterCount"
              :class="ui.chipButton"
              @click="clearFilters"
            >
              Reset {{ activeFilterCount }} filter{{ activeFilterCount === 1 ? '' : 's' }}
            </button>
          </div>
        </section>

        <section :class="ui.panel">
          <div :class="ui.panelHeader">
            <div>
              <p :class="ui.eyebrow">Folders</p>
              <h3 :class="ui.title">Directory Surface</h3>
              <p :class="ui.description">{{ hasActiveFilters ? 'Folders connected to the current search and filters.' : `Folders inside ${currentFolder?.name || 'Campaign Archive'}.` }}</p>
            </div>
            <span :class="ui.pill">{{ visibleFolders.length }}</span>
          </div>

          <div v-if="visibleFolders.length" class="grid gap-4 px-5 pb-5 pt-1 2xl:grid-cols-3">
            <article
              v-for="folder in visibleFolders"
              :key="folder.id"
              :class="[ui.card, 'p-4', dragTargetActive(`folder:${folder.id}`) ? ui.dropHighlight : '']"
              draggable="true"
              @dragstart="startFolderDrag(folder.id)"
              @dragover.prevent="handleFolderDragOver(folder.id)"
              @dragenter.prevent="handleFolderDragOver(folder.id)"
              @drop.prevent="dropIntoFolder(folder.id)"
              @dragend="stopDrag"
            >
              <button class="flex w-full items-start justify-between gap-4 text-left" @click="selectFolder(folder.id)">
                <div>
                  <p :class="ui.titleText" :title="folder.name">{{ folder.name }}</p>
                  <p class="mt-1 text-sm leading-7 text-slate-300/70 break-words">{{ folderDescription(folder) }}</p>
                </div>
                <ChevronRight :size="16" class="shrink-0 text-slate-300/80" />
              </button>
            </article>
          </div>

          <p v-else class="px-5 pb-5 pt-4 text-sm leading-7 text-slate-300/70">{{ hasActiveFilters ? 'No folders match the current filters.' : 'No subfolders here yet.' }}</p>
        </section>

        <section
          :class="[ui.panel, dragTargetActive(currentFolderDropKey) ? ui.dropHighlight : '']"
          @dragover.prevent="handleFolderDragOver(currentFolderId)"
          @dragenter.prevent="handleFolderDragOver(currentFolderId)"
          @drop.prevent="dropIntoFolder(currentFolderId)"
        >
          <div :class="ui.panelHeader">
            <div>
              <p :class="ui.eyebrow">Items</p>
              <h3 :class="ui.title">Archive Listing</h3>
              <p :class="ui.description">{{ hasActiveFilters ? `${visibleItems.length} matches across the campaign archive.` : `${visibleItems.length} items in ${currentFolder?.name || 'Campaign Archive'}.` }}</p>
            </div>
            <div v-if="filterBadges.length" class="flex flex-wrap gap-3">
              <span v-for="badge in filterBadges" :key="badge" :class="ui.pill">{{ badge }}</span>
            </div>
          </div>

          <div v-if="visibleItems.length" class="grid max-h-[45rem] grid-cols-2 gap-4 overflow-auto px-5 pb-5 pt-1 md:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5">
            <button
              v-for="item in visibleItems"
              :key="item.id"
              :class="[
                ui.itemCard,
                selectedItemId === item.id ? 'border-rose-400/35 ring-1 ring-inset ring-rose-400/20' : '',
              ]"
              draggable="true"
              @dragstart="startItemDrag(item.id)"
              @dragend="stopDrag"
              @mouseenter="previewItem(item.id)"
              @mouseleave="previewItem('')"
              @focus="previewItem(item.id)"
              @blur="previewItem('')"
              @click="openItemDetails(item)"
            >
              <div
                class="relative flex aspect-square items-center justify-center overflow-hidden rounded-2xl border-2"
                :class="itemArtFrameClass(item.rarity)"
                style="background: linear-gradient(180deg, rgba(27, 49, 93, 0.98), rgba(13, 24, 47, 1));"
              >
                <img v-if="itemImageUrl(item)" :src="itemImageUrl(item)" :alt="item.name" />
                <span v-else class="font-[Cinzel] text-3xl font-bold text-slate-50">{{ itemInitial(item) }}</span>
                <span :class="ui.sizeBadge">{{ item.grid_width }}x{{ item.grid_height }}</span>
              </div>

              <div class="grid min-w-0 gap-1">
                <strong :class="ui.titleText" :title="item.name">{{ item.name }}</strong>
                <span class="text-xs text-slate-300/60 break-words">{{ formatLabel(item.category) }}<template v-if="item.equip_slot"> · {{ formatLabel(item.equip_slot) }}</template></span>
              </div>
            </button>
          </div>

          <p v-else class="px-5 pb-5 pt-4 text-sm leading-7 text-slate-300/70">{{ hasActiveFilters ? 'No items match the current filters.' : 'This directory has no items yet.' }}</p>
        </section>
      </div>

      <aside :class="[ui.panel, 'pb-4 xl:col-span-2 2xl:col-span-1']">
        <div :class="ui.panelHeader">
          <div>
            <p :class="ui.eyebrow">{{ inspectorModeLabel }}</p>
            <h3 :class="ui.title">Detail Inspector</h3>
            <p :class="ui.description">Hover to preview, click to pin.</p>
          </div>
          <span v-if="inspectorItem" :class="ui.pill">{{ formatLabel(inspectorItem.rarity) }}</span>
        </div>

        <div v-if="inspectorItem" class="grid gap-5 px-5 pt-1">
          <div :class="[ui.card, 'grid gap-5 p-5 md:grid-cols-[auto_minmax(0,1fr)]']">
            <div
              :class="[ui.artFrame, itemArtFrameClass(inspectorItem.rarity)]"
              style="background: linear-gradient(180deg, rgba(27, 49, 93, 0.98), rgba(13, 24, 47, 1));"
            >
              <img v-if="itemImageUrl(inspectorItem)" :src="itemImageUrl(inspectorItem)" :alt="inspectorItem.name" />
              <span v-else>{{ itemInitial(inspectorItem) }}</span>
            </div>

            <div class="grid min-w-0 gap-4">
              <div class="flex flex-wrap items-start justify-between gap-4">
                <h4 class="m-0 basis-56 grow break-words text-slate-50" :title="inspectorItem.name">{{ inspectorItem.name }}</h4>
                <span :class="ui.pill">{{ formatDate(inspectorItem.updated_at || inspectorItem.created_at) }}</span>
              </div>
              <p :class="ui.copy">{{ shortText(inspectorItem.description, 220) }}</p>

              <div class="flex flex-wrap gap-3">
                <span :class="[ui.badge, rarityBadgeClass(inspectorItem.rarity)]">{{ formatLabel(inspectorItem.rarity) }}</span>
                <span :class="ui.pill">{{ formatLabel(inspectorItem.category) }}</span>
                <span :class="ui.pill">{{ inspectorItem.grid_width }}x{{ inspectorItem.grid_height }}</span>
                <span :class="ui.pill">{{ inspectorItem.equip_slot ? formatLabel(inspectorItem.equip_slot) : 'No slot' }}</span>
              </div>
            </div>
          </div>

          <div class="grid gap-4 xl:grid-cols-2">
            <article :class="ui.statCard">
              <span :class="ui.fieldLabelText">Location</span>
              <strong>{{ folderPathLabel(itemLocations[inspectorItem.id]) }}</strong>
            </article>
            <article :class="ui.statCard">
              <span :class="ui.fieldLabelText">Requirements</span>
              <strong>{{ inspectorItem.required_attributes.length }}</strong>
            </article>
            <article :class="ui.statCard">
              <span :class="ui.fieldLabelText">Modifiers</span>
              <strong>{{ inspectorItem.attribute_modifiers.length }}</strong>
            </article>
            <article :class="ui.statCard">
              <span :class="ui.fieldLabelText">Tags</span>
              <strong>{{ inspectorItem.tags.length }}</strong>
            </article>
          </div>

          <section :class="ui.block">
            <div :class="ui.sectionLabelRow">
              <span :class="ui.fieldLabelText">Category & Tags</span>
            </div>
            <div class="flex flex-wrap gap-3">
              <span :class="ui.pill">{{ formatLabel(inspectorItem.category) }}</span>
              <span v-if="inspectorItem.equip_slot" :class="ui.pill">{{ formatLabel(inspectorItem.equip_slot) }}</span>
              <span v-for="tag in inspectorItem.tags.filter(tag => tag.toLowerCase() !== inspectorItem.category.toLowerCase())" :key="`${inspectorItem.id}-${tag}`" :class="ui.pill">{{ tag }}</span>
            </div>
          </section>

          <div class="grid gap-4 xl:grid-cols-2">
            <section :class="ui.block">
              <div :class="ui.sectionLabelRow">
                <span :class="ui.fieldLabelText">Requirements</span>
              </div>
              <div v-if="inspectorItem.required_attributes.length" class="grid gap-2">
                <div v-for="requirement in inspectorItem.required_attributes" :key="`${inspectorItem.id}-${requirement.attribute_name}`" :class="ui.stackItem">
                  {{ formatAttributeLabel(requirement.attribute_name) }} >= {{ requirement.min_value }}
                </div>
              </div>
              <p v-else class="text-sm leading-7 text-slate-300/70">No requirements.</p>
            </section>

            <section :class="ui.block">
              <div :class="ui.sectionLabelRow">
                <span :class="ui.fieldLabelText">Modifiers</span>
              </div>
              <div v-if="inspectorItem.attribute_modifiers.length" class="grid gap-2">
                <div v-for="modifier in inspectorItem.attribute_modifiers" :key="`${inspectorItem.id}-${modifier.attribute_name}`" :class="ui.stackItem">
                  {{ formatAttributeLabel(modifier.attribute_name) }} {{ modifier.modifier_value > 0 ? '+' : '' }}{{ modifier.modifier_value }}{{ modifier.is_percentage ? '%' : '' }}
                </div>
              </div>
              <p v-else class="text-sm leading-7 text-slate-300/70">No modifiers.</p>
            </section>
          </div>
        </div>

        <div v-else class="grid min-h-72 place-items-center p-4 text-center">
          <Package :size="18" class="text-slate-300/80" />
          <p class="mt-3 text-sm leading-7 text-slate-300/70">Hover or select an item to inspect it here.</p>
        </div>
      </aside>
    </div>

    <Teleport to="body">
      <div v-if="showCreateFolderModal" class="fixed inset-0 z-50 flex items-center justify-center p-3 md:p-5" @click.self="showCreateFolderModal = false">
        <div :class="ui.modalBackdrop"></div>
        <div :class="[ui.modalCard, 'w-full max-w-xl']">
          <button :class="ui.closeButton" @click="showCreateFolderModal = false">
            <X :size="18" />
          </button>

          <div :class="ui.modalHeader">
            <p :class="ui.eyebrow">New Folder</p>
            <h3 :class="ui.title">Create Archive Directory</h3>
            <p :class="ui.description">Structure the item library before dropping gear into it.</p>
          </div>

          <div :class="ui.modalBody">
            <label :class="ui.fieldLabel">
              <span :class="ui.fieldLabelText">Folder Name</span>
              <input v-model="folderNameDraft" :class="ui.input" type="text" placeholder="Boss Drops, Temple Loot, Shops" />
            </label>

            <label :class="ui.fieldLabel">
              <span :class="ui.fieldLabelText">Parent Folder</span>
              <select v-model="folderParentDraft" :class="ui.input">
                <option v-for="option in folderOptions" :key="option.value || 'root'" :value="option.value">{{ option.label }}</option>
              </select>
            </label>
          </div>

          <div :class="ui.modalFooter">
            <button :class="ui.ghostButton" @click="showCreateFolderModal = false">Cancel</button>
            <button :class="ui.accentButton" :disabled="!folderNameDraft.trim()" @click="createFolder">Create Folder</button>
          </div>
        </div>
      </div>
    </Teleport>

    <Teleport to="body">
      <div v-if="showCreateItemModal" class="fixed inset-0 z-50 flex items-center justify-center p-3 md:p-5" @click.self="showCreateItemModal = false">
        <div :class="ui.modalBackdrop"></div>
        <div :class="[ui.modalCard, 'max-h-screen w-full max-w-7xl']">
          <button :class="ui.closeButton" @click="showCreateItemModal = false">
            <X :size="18" />
          </button>

          <div :class="ui.modalHeader">
            <p :class="ui.eyebrow">Create Item</p>
            <h3 :class="ui.title">New Explorer Entry</h3>
            <p :class="ui.description">Form fields are on the left, live preview is on the right. This save now goes straight to the backend.</p>
          </div>

          <div class="grid min-h-0 gap-5 px-4 pb-6 md:px-6 xl:grid-cols-[minmax(0,1.35fr)_23rem]">
            <div class="grid min-h-0 gap-5 overflow-auto md:pr-1">
              <section :class="ui.block">
                <div :class="ui.sectionLabelRow">
                  <span :class="ui.fieldLabelText">Core Identity</span>
                </div>

                <label :class="ui.fieldLabel">
                  <span :class="ui.fieldLabelText">Name</span>
                  <input v-model="itemDraft.name" :class="ui.input" type="text" placeholder="Astral Lantern" />
                </label>

                <label :class="[ui.fieldLabel, 'mt-2']">
                  <span :class="ui.fieldLabelText">Description</span>
                  <textarea v-model="itemDraft.description" :class="ui.textarea" rows="5" placeholder="Describe what the item is, what it does, and how it feels in the world."></textarea>
                </label>

                <div class="grid gap-4 xl:grid-cols-2">
                  <label :class="ui.fieldLabel">
                    <span :class="ui.fieldLabelText">Folder</span>
                    <select v-model="itemDraft.folderId" :class="ui.input">
                      <option v-for="option in folderOptions" :key="option.value || 'root'" :value="option.value">{{ option.label }}</option>
                    </select>
                  </label>

                  <label :class="ui.fieldLabel">
                    <span :class="ui.fieldLabelText">Quality</span>
                    <select v-model="itemDraft.rarity" :class="ui.input">
                      <option v-for="rarity in RARITY_OPTIONS" :key="rarity" :value="rarity">{{ formatLabel(rarity) }}</option>
                    </select>
                  </label>
                </div>

                <div class="grid gap-4 xl:grid-cols-2">
                  <label :class="ui.fieldLabel">
                    <span :class="ui.fieldLabelText">Category</span>
                    <select v-model="itemDraft.category" :class="ui.input">
                      <option v-for="category in CATEGORY_OPTIONS" :key="category" :value="category">{{ formatLabel(category) }}</option>
                    </select>
                  </label>

                  <label :class="ui.fieldLabel">
                    <span :class="ui.fieldLabelText">Equip Slot</span>
                    <select v-model="itemDraft.equipSlot" :class="ui.input">
                      <option value="">No slot</option>
                      <option v-for="slot in EQUIP_SLOT_OPTIONS" :key="slot" :value="slot">{{ formatLabel(slot) }}</option>
                    </select>
                  </label>
                </div>
              </section>

              <section :class="ui.block">
                <div :class="ui.sectionLabelRow">
                  <span :class="ui.fieldLabelText">Footprint</span>
                </div>

                <div class="grid gap-4 xl:grid-cols-2">
                  <label :class="ui.fieldLabel">
                    <span :class="ui.fieldLabelText">Grid Width</span>
                    <input v-model.number="itemDraft.gridWidth" :class="ui.input" type="number" min="1" />
                  </label>

                  <label :class="ui.fieldLabel">
                    <span :class="ui.fieldLabelText">Grid Height</span>
                    <input v-model.number="itemDraft.gridHeight" :class="ui.input" type="number" min="1" />
                  </label>
                </div>
              </section>

              <section :class="ui.block">
                <div :class="ui.sectionLabelRow">
                  <div>
                    <span :class="ui.fieldLabelText">Stats</span>
                    <p class="mt-3 text-sm leading-7 text-slate-300/70 break-words">Use explicit stat rows instead of text parsing. Base attributes are always available, and custom attributes are pulled from current characters.</p>
                  </div>
                </div>

                <div class="grid gap-4 xl:grid-cols-2">
                  <section :class="ui.nestedBlock">
                    <div :class="ui.sectionLabelRow">
                      <div>
                        <span :class="ui.fieldLabelText">Requirements</span>
                        <p class="mt-3 text-sm leading-7 text-slate-300/70 break-words">Minimum values a character must meet.</p>
                      </div>
                      <button :class="ui.inlineButton" @click="addRequirementRow">
                        <Plus :size="14" />
                        <span>Add Row</span>
                      </button>
                    </div>

                    <div class="grid gap-3">
                      <div v-for="requirement in itemDraft.requirements" :key="requirement.id" :class="[ui.rowCard, 'md:grid-cols-[minmax(0,1fr)_9rem_auto]']">
                        <label :class="[ui.fieldLabel, 'min-w-0']">
                          <span :class="ui.fieldLabelText">Stat</span>
                          <select v-model="requirement.attribute_name" :class="ui.input">
                            <option value="">Select stat</option>
                            <option v-for="option in attributeOptions" :key="`requirement-${option.value}`" :value="option.value">{{ option.label }}</option>
                          </select>
                        </label>

                        <label :class="ui.fieldLabel">
                          <span :class="ui.fieldLabelText">Min Value</span>
                          <input v-model.number="requirement.min_value" :class="ui.input" type="number" />
                        </label>

                        <button :class="ui.iconButton" @click="removeRequirementRow(requirement.id)">
                          <Trash2 :size="16" />
                        </button>
                      </div>
                    </div>
                  </section>

                  <section :class="ui.nestedBlock">
                    <div :class="ui.sectionLabelRow">
                      <div>
                        <span :class="ui.fieldLabelText">Modifiers</span>
                        <p class="mt-3 text-sm leading-7 text-slate-300/70 break-words">Choose a stat, sign, amount and whether it is flat or percentage.</p>
                      </div>
                      <button :class="ui.inlineButton" @click="addModifierRow">
                        <Plus :size="14" />
                        <span>Add Row</span>
                      </button>
                    </div>

                    <div class="grid gap-3">
                      <div v-for="modifier in itemDraft.modifiers" :key="modifier.id" :class="[ui.rowCard, 'md:grid-cols-[minmax(0,1fr)_5.5rem_7rem_8rem_auto]']">
                        <label :class="[ui.fieldLabel, 'min-w-0']">
                          <span :class="ui.fieldLabelText">Stat</span>
                          <select v-model="modifier.attribute_name" :class="ui.input">
                            <option value="">Select stat</option>
                            <option v-for="option in attributeOptions" :key="`modifier-${option.value}`" :value="option.value">{{ option.label }}</option>
                          </select>
                        </label>

                        <label :class="ui.fieldLabel">
                          <span :class="ui.fieldLabelText">Sign</span>
                          <select v-model="modifier.sign" :class="ui.input">
                            <option value="+">+</option>
                            <option value="-">-</option>
                          </select>
                        </label>

                        <label :class="ui.fieldLabel">
                          <span :class="ui.fieldLabelText">Amount</span>
                          <input v-model.number="modifier.magnitude" :class="ui.input" type="number" min="0" />
                        </label>

                        <label :class="ui.fieldLabel">
                          <span :class="ui.fieldLabelText">Unit</span>
                          <select v-model="modifier.is_percentage" :class="ui.input">
                            <option :value="false">Flat</option>
                            <option :value="true">Percent</option>
                          </select>
                        </label>

                        <button :class="ui.iconButton" @click="removeModifierRow(modifier.id)">
                          <Trash2 :size="16" />
                        </button>
                      </div>
                    </div>
                  </section>
                </div>
              </section>
            </div>

            <aside class="grid content-start gap-4">
              <div :class="[ui.card, 'grid gap-5 p-5 md:grid-cols-[auto_minmax(0,1fr)]']">
                <div
                  :class="[ui.artFrame, itemArtFrameClass(draftPreviewItem.rarity)]"
                  style="background: linear-gradient(180deg, rgba(27, 49, 93, 0.98), rgba(13, 24, 47, 1));"
                >
                  <span>{{ itemInitial(draftPreviewItem) }}</span>
                </div>

                <div class="grid min-w-0 gap-4">
                  <div class="flex flex-wrap items-start justify-between gap-4">
                    <h4 class="m-0 basis-56 grow break-words text-slate-50" :title="draftPreviewItem.name">{{ draftPreviewItem.name }}</h4>
                    <span :class="ui.pill">Preview</span>
                  </div>
                  <p :class="ui.copy">{{ shortText(draftPreviewItem.description, 240) }}</p>

                  <div class="flex flex-wrap gap-3">
                    <span :class="[ui.badge, rarityBadgeClass(draftPreviewItem.rarity)]">{{ formatLabel(draftPreviewItem.rarity) }}</span>
                    <span :class="ui.pill">{{ formatLabel(draftPreviewItem.category) }}</span>
                    <span :class="ui.pill">{{ draftPreviewItem.grid_width }}x{{ draftPreviewItem.grid_height }}</span>
                    <span :class="ui.pill">{{ draftPreviewItem.equip_slot ? formatLabel(draftPreviewItem.equip_slot) : 'No slot' }}</span>
                  </div>
                </div>
              </div>

              <article :class="ui.statCard">
                <span :class="ui.fieldLabelText">Destination</span>
                <strong>{{ folderPathLabel(itemDraft.folderId) }}</strong>
              </article>

              <div class="grid gap-4 xl:grid-cols-2">
                <section :class="ui.block">
                  <div :class="ui.sectionLabelRow">
                    <span :class="ui.fieldLabelText">Requirements</span>
                  </div>
                  <div v-if="draftPreviewItem.required_attributes.length" class="grid gap-2">
                    <div v-for="requirement in draftPreviewItem.required_attributes" :key="`${requirement.attribute_name}-${requirement.min_value}`" :class="ui.stackItem">
                      {{ formatAttributeLabel(requirement.attribute_name) }} >= {{ requirement.min_value }}
                    </div>
                  </div>
                  <p v-else class="text-sm leading-7 text-slate-300/70">No requirements.</p>
                </section>

                <section :class="ui.block">
                  <div :class="ui.sectionLabelRow">
                    <span :class="ui.fieldLabelText">Modifiers</span>
                  </div>
                  <div v-if="draftPreviewItem.attribute_modifiers.length" class="grid gap-2">
                    <div v-for="modifier in draftPreviewItem.attribute_modifiers" :key="`${modifier.attribute_name}-${modifier.modifier_value}-${modifier.is_percentage}`" :class="ui.stackItem">
                      {{ formatAttributeLabel(modifier.attribute_name) }} {{ modifier.modifier_value > 0 ? '+' : '' }}{{ modifier.modifier_value }}{{ modifier.is_percentage ? '%' : '' }}
                    </div>
                  </div>
                  <p v-else class="text-sm leading-7 text-slate-300/70">No modifiers.</p>
                </section>
              </div>
            </aside>
          </div>

          <p v-if="itemCreateError" class="mx-4 rounded-2xl border border-red-400/20 bg-red-900/20 px-4 py-4 text-sm text-red-200 md:mx-6">{{ itemCreateError }}</p>

          <div :class="ui.modalFooter">
            <button :class="ui.ghostButton" @click="showCreateItemModal = false">Cancel</button>
            <button :class="ui.accentButton" :disabled="!itemDraft.name.trim() || createItemSubmitting" @click="createItem">
              <FilePlus2 :size="16" />
              <span>{{ createItemSubmitting ? 'Saving...' : 'Save Item' }}</span>
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </section>
</template>