<script setup>
import {
  ChevronDown,
  ChevronRight,
  FilePlus2,
  FolderPlus,
  Package,
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
  gameId: {
    type: String,
    default: '',
  },
})

const auth = useAuthStore()

const RARITY_OPTIONS = ['common', 'uncommon', 'rare', 'epic', 'masterwork', 'legendary', 'unique']
const CATEGORY_OPTIONS = ['loot', 'consumable', 'other']
const EQUIP_SLOT_OPTIONS = ['head', 'chest', 'gloves', 'belt', 'boots', 'main_hand', 'off_hand', 'ring', 'amulet']
const SORT_OPTIONS = [
  { value: 'name-asc', label: 'Name A-Z' },
  { value: 'name-desc', label: 'Name Z-A' },
  { value: 'rarity', label: 'Quality' },
  { value: 'recent', label: 'Recently Updated' },
  { value: 'size', label: 'Grid Size' },
]

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
  required_attributes: parseRequirements(itemDraft.value.requirementsText),
  attribute_modifiers: parseModifiers(itemDraft.value.modifiersText),
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
    requirementsText: '',
    modifiersText: '',
  }
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

function normalizeRarityValue(value) {
  const normalized = String(value || '').trim().toLowerCase()
  if (normalized === 'artifact') return 'unique'
  return RARITY_OPTIONS.includes(normalized) ? normalized : 'common'
}

function normalizeCategoryValue(value, types = []) {
  const normalized = String(value || '').trim().toLowerCase()
  if (CATEGORY_OPTIONS.includes(normalized)) return normalized

  const fallback = types
    .map(type => extractTypeName(type).toLowerCase())
    .find(type => CATEGORY_OPTIONS.includes(type))

  return fallback || 'other'
}

function normalizeEquipSlotValue(value) {
  const normalized = String(value || '').trim().toLowerCase()
  if (!normalized) return null
  if (normalized === 'ring_1' || normalized === 'ring_2') return 'ring'
  return EQUIP_SLOT_OPTIONS.includes(normalized) ? normalized : null
}

function normalizeItem(item) {
  const rawTypes = Array.isArray(item.types) ? item.types.map(extractTypeName).filter(Boolean) : []
  const category = normalizeCategoryValue(item.category, rawTypes)
  const equipSlot = normalizeEquipSlotValue(item.equip_slot)
  const tags = Array.from(new Set([category, ...rawTypes.filter(type => !CATEGORY_OPTIONS.includes(type.toLowerCase()))]))

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
    required_attributes: Array.isArray(item.required_attributes) ? item.required_attributes : [],
    attribute_modifiers: Array.isArray(item.attribute_modifiers) ? item.attribute_modifiers : [],
  }
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

function parseRequirements(text) {
  return String(text || '')
    .split('\n')
    .map(line => line.trim())
    .filter(Boolean)
    .map(line => {
      const [attributeName, minValue] = line.split(':').map(part => part.trim())
      return {
        attribute_name: attributeName.toLowerCase(),
        min_value: Number(minValue) || 0,
      }
    })
    .filter(entry => entry.attribute_name)
}

function parseModifiers(text) {
  return String(text || '')
    .split('\n')
    .map(line => line.trim())
    .filter(Boolean)
    .map(line => {
      const [attributeName, rawValue] = line.split(':').map(part => part.trim())
      const isPercentage = rawValue?.endsWith('%') ?? false
      const parsedValue = Number((rawValue || '').replace('%', '')) || 0
      return {
        attribute_name: attributeName.toLowerCase(),
        modifier_value: parsedValue,
        is_percentage: isPercentage,
      }
    })
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
      required_attributes: parseRequirements(itemDraft.value.requirementsText),
      attribute_modifiers: parseModifiers(itemDraft.value.modifiersText),
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
  <section class="session-item-manager">
    <div class="session-item-manager__hero">
      <div>
        <p class="session-item-manager__eyebrow">Item Explorer</p>
        <h2 class="session-item-manager__title">Campaign Archive</h2>
        <p class="session-item-manager__description">
          Server-backed GM library with explorer folders, search, drag-and-drop placement, and a large create modal with live preview.
        </p>
      </div>

      <div class="session-item-manager__actions">
        <button class="session-item-button session-item-button--ghost" @click="openCreateFolderModal()">
          <FolderPlus :size="16" />
          <span>New Folder</span>
        </button>
        <button class="session-item-button session-item-button--accent" @click="openCreateItemModal()">
          <FilePlus2 :size="16" />
          <span>Create Item</span>
        </button>
      </div>
    </div>

    <div class="session-item-manager__stats">
      <article class="session-item-stat">
        <span>Total Items</span>
        <strong>{{ explorerStats.totalItems }}</strong>
      </article>
      <article class="session-item-stat">
        <span>Visible</span>
        <strong>{{ explorerStats.visibleItems }}</strong>
      </article>
      <article class="session-item-stat">
        <span>Folders</span>
        <strong>{{ explorerStats.totalFolders }}</strong>
      </article>
      <article class="session-item-stat">
        <span>Active Filters</span>
        <strong>{{ explorerStats.activeFilters }}</strong>
      </article>
    </div>

    <div class="session-item-manager__shell">
      <aside class="session-panel session-sidebar">
        <div class="session-panel__head">
          <div>
            <p class="session-panel__eyebrow">Directory Tree</p>
            <h3 class="session-panel__title">Folders</h3>
          </div>
          <span class="session-pill">{{ folders.length }}</span>
        </div>

        <button
          class="session-tree-root"
          :class="{ 'session-tree-root--active': !currentFolderId, 'session-tree-root--drop': dragTargetActive('folder:root') }"
          @click="selectFolder(null)"
          @dragover.prevent="handleFolderDragOver(null)"
          @dragenter.prevent="handleFolderDragOver(null)"
          @drop.prevent="dropIntoFolder(null)"
        >
          <div>
            <span class="session-tree-root__label">Campaign Archive</span>
            <span class="session-tree-root__meta">{{ allItems.length }} items</span>
          </div>
          <Package :size="16" />
        </button>

        <div v-if="treeRows.length" class="session-scroll session-tree-list">
          <div v-for="folder in treeRows" :key="folder.id" class="session-tree-row" :style="{ '--tree-depth': folder.depth }">
            <button class="session-tree-row__toggle" :class="{ 'session-tree-row__toggle--placeholder': !folder.hasChildren }" @click.stop="folder.hasChildren ? toggleFolderExpanded(folder.id) : null">
              <ChevronDown v-if="folder.hasChildren && isFolderExpanded(folder.id)" :size="14" />
              <ChevronRight v-else-if="folder.hasChildren" :size="14" />
            </button>

            <button
              class="session-tree-row__body"
              :class="{ 'session-tree-row__body--active': currentFolderId === folder.id, 'session-tree-row__body--drop': dragTargetActive(`folder:${folder.id}`) }"
              draggable="true"
              @click="selectFolder(folder.id)"
              @dragstart="startFolderDrag(folder.id)"
              @dragover.prevent="handleFolderDragOver(folder.id)"
              @dragenter.prevent="handleFolderDragOver(folder.id)"
              @drop.prevent="dropIntoFolder(folder.id)"
              @dragend="stopDrag"
            >
              <span class="session-tree-row__name">{{ folder.name }}</span>
              <span class="session-tree-row__meta">{{ folderVisibleItemCount(folder.id) }}</span>
            </button>
          </div>
        </div>

        <p v-else class="session-empty-copy">No folders yet. Create your first archive directory.</p>

        <div
          class="session-sidebar__dropzone"
          :class="{ 'session-sidebar__dropzone--active': dragTargetActive(currentFolderDropKey) }"
          @dragover.prevent="handleFolderDragOver(currentFolderId)"
          @dragenter.prevent="handleFolderDragOver(currentFolderId)"
          @drop.prevent="dropIntoFolder(currentFolderId)"
        >
          Drop items or folders into {{ currentFolder?.name || 'Campaign Archive' }}.
        </div>
      </aside>

      <div class="session-workspace">
        <section class="session-panel session-toolbar">
          <div class="session-toolbar__top">
            <label class="session-search-field">
              <Search :size="16" />
              <input v-model="searchQuery" type="text" placeholder="Search by name, category, quality, slot or stats" />
              <button v-if="searchQuery" class="session-search-field__clear" @click="searchQuery = ''">
                <X :size="14" />
              </button>
            </label>

            <label class="session-filter-field session-filter-field--sort">
              <span>Sort</span>
              <select v-model="sortMode">
                <option v-for="option in SORT_OPTIONS" :key="option.value" :value="option.value">{{ option.label }}</option>
              </select>
            </label>
          </div>

          <div class="session-toolbar__filters">
            <label class="session-filter-field">
              <span>Quality</span>
              <select v-model="rarityFilter">
                <option value="all">All qualities</option>
                <option v-for="rarity in RARITY_OPTIONS" :key="rarity" :value="rarity">{{ formatLabel(rarity) }}</option>
              </select>
            </label>

            <label class="session-filter-field">
              <span>Category</span>
              <select v-model="categoryFilter">
                <option value="all">All categories</option>
                <option v-for="category in CATEGORY_OPTIONS" :key="category" :value="category">{{ formatLabel(category) }}</option>
              </select>
            </label>

            <label class="session-filter-field">
              <span>Equip Slot</span>
              <select v-model="slotFilter">
                <option value="all">All slots</option>
                <option v-for="slot in EQUIP_SLOT_OPTIONS" :key="slot" :value="slot">{{ formatLabel(slot) }}</option>
              </select>
            </label>
          </div>

          <div class="session-toolbar__bottom">
            <div class="session-breadcrumbs">
              <button v-for="crumb in breadcrumbs" :key="crumb.id ?? 'root'" @click="selectFolder(crumb.id)">
                {{ crumb.name }}
              </button>
            </div>

            <button v-if="activeFilterCount" class="session-reset-button" @click="clearFilters">
              Reset {{ activeFilterCount }} filter{{ activeFilterCount === 1 ? '' : 's' }}
            </button>
          </div>
        </section>

        <section class="session-panel session-folder-surface">
          <div class="session-panel__head session-panel__head--compact">
            <div>
              <p class="session-panel__eyebrow">Folders</p>
              <h3 class="session-panel__title">Directory Surface</h3>
              <p class="session-panel__description">{{ hasActiveFilters ? 'Folders connected to the current search and filters.' : `Folders inside ${currentFolder?.name || 'Campaign Archive'}.` }}</p>
            </div>
            <span class="session-pill">{{ visibleFolders.length }}</span>
          </div>

          <div v-if="visibleFolders.length" class="session-folder-grid">
            <article
              v-for="folder in visibleFolders"
              :key="folder.id"
              class="session-folder-card"
              :class="{ 'session-folder-card--drop': dragTargetActive(`folder:${folder.id}`) }"
              draggable="true"
              @dragstart="startFolderDrag(folder.id)"
              @dragover.prevent="handleFolderDragOver(folder.id)"
              @dragenter.prevent="handleFolderDragOver(folder.id)"
              @drop.prevent="dropIntoFolder(folder.id)"
              @dragend="stopDrag"
            >
              <button class="session-folder-card__button" @click="selectFolder(folder.id)">
                <div>
                  <p class="session-folder-card__title">{{ folder.name }}</p>
                  <p class="session-folder-card__meta">{{ folderDescription(folder) }}</p>
                </div>
                <ChevronRight :size="16" />
              </button>
            </article>
          </div>

          <p v-else class="session-empty-copy">{{ hasActiveFilters ? 'No folders match the current filters.' : 'No subfolders here yet.' }}</p>
        </section>

        <section
          class="session-panel session-item-surface"
          :class="{ 'session-item-surface--drop': dragTargetActive(currentFolderDropKey) }"
          @dragover.prevent="handleFolderDragOver(currentFolderId)"
          @dragenter.prevent="handleFolderDragOver(currentFolderId)"
          @drop.prevent="dropIntoFolder(currentFolderId)"
        >
          <div class="session-panel__head session-panel__head--compact">
            <div>
              <p class="session-panel__eyebrow">Items</p>
              <h3 class="session-panel__title">Archive Listing</h3>
              <p class="session-panel__description">{{ hasActiveFilters ? `${visibleItems.length} matches across the campaign archive.` : `${visibleItems.length} items in ${currentFolder?.name || 'Campaign Archive'}.` }}</p>
            </div>
            <div v-if="filterBadges.length" class="session-badge-row">
              <span v-for="badge in filterBadges" :key="badge" class="session-pill">{{ badge }}</span>
            </div>
          </div>

          <div v-if="visibleItems.length" class="session-scroll session-item-grid">
            <button
              v-for="item in visibleItems"
              :key="item.id"
              class="session-item-card"
              :class="{ 'session-item-card--selected': selectedItemId === item.id }"
              draggable="true"
              @dragstart="startItemDrag(item.id)"
              @dragend="stopDrag"
              @mouseenter="previewItem(item.id)"
              @mouseleave="previewItem('')"
              @focus="previewItem(item.id)"
              @blur="previewItem('')"
              @click="openItemDetails(item)"
            >
              <div class="session-item-card__art" :class="`session-item-card__art--${item.rarity}`">
                <img v-if="itemImageUrl(item)" :src="itemImageUrl(item)" :alt="item.name" />
                <span v-else class="session-item-card__glyph">{{ itemInitial(item) }}</span>
                <span class="session-item-card__size">{{ item.grid_width }}x{{ item.grid_height }}</span>
              </div>

              <div class="session-item-card__body">
                <strong class="session-item-card__name">{{ item.name }}</strong>
                <span class="session-item-card__meta">{{ formatLabel(item.category) }}<template v-if="item.equip_slot"> · {{ formatLabel(item.equip_slot) }}</template></span>
              </div>
            </button>
          </div>

          <p v-else class="session-empty-copy">{{ hasActiveFilters ? 'No items match the current filters.' : 'This directory has no items yet.' }}</p>
        </section>
      </div>

      <aside class="session-panel session-inspector">
        <div class="session-panel__head session-panel__head--compact">
          <div>
            <p class="session-panel__eyebrow">{{ inspectorModeLabel }}</p>
            <h3 class="session-panel__title">Detail Inspector</h3>
            <p class="session-panel__description">Hover to preview, click to pin.</p>
          </div>
          <span v-if="inspectorItem" class="session-pill">{{ formatLabel(inspectorItem.rarity) }}</span>
        </div>

        <div v-if="inspectorItem" class="session-inspector__content">
          <div class="session-preview-card">
            <div class="session-preview-card__media" :class="`session-preview-card__media--${inspectorItem.rarity}`">
              <img v-if="itemImageUrl(inspectorItem)" :src="itemImageUrl(inspectorItem)" :alt="inspectorItem.name" />
              <span v-else>{{ itemInitial(inspectorItem) }}</span>
            </div>

            <div class="session-preview-card__copy">
              <div class="session-preview-card__row">
                <h4>{{ inspectorItem.name }}</h4>
                <span class="session-preview-card__chip">{{ formatDate(inspectorItem.updated_at || inspectorItem.created_at) }}</span>
              </div>
              <p>{{ shortText(inspectorItem.description, 220) }}</p>

              <div class="session-badge-row">
                <span class="session-rarity-badge" :class="`session-rarity-badge--${inspectorItem.rarity}`">{{ formatLabel(inspectorItem.rarity) }}</span>
                <span class="session-pill">{{ formatLabel(inspectorItem.category) }}</span>
                <span class="session-pill">{{ inspectorItem.grid_width }}x{{ inspectorItem.grid_height }}</span>
                <span class="session-pill">{{ inspectorItem.equip_slot ? formatLabel(inspectorItem.equip_slot) : 'No slot' }}</span>
              </div>
            </div>
          </div>

          <div class="session-inspector__stats">
            <article class="session-mini-card">
              <span>Location</span>
              <strong>{{ folderPathLabel(itemLocations[inspectorItem.id]) }}</strong>
            </article>
            <article class="session-mini-card">
              <span>Requirements</span>
              <strong>{{ inspectorItem.required_attributes.length }}</strong>
            </article>
            <article class="session-mini-card">
              <span>Modifiers</span>
              <strong>{{ inspectorItem.attribute_modifiers.length }}</strong>
            </article>
            <article class="session-mini-card">
              <span>Tags</span>
              <strong>{{ inspectorItem.tags.length }}</strong>
            </article>
          </div>

          <section class="session-block">
            <div class="session-block__head">
              <span>Category & Tags</span>
            </div>
            <div class="session-badge-row">
              <span class="session-pill">{{ formatLabel(inspectorItem.category) }}</span>
              <span v-if="inspectorItem.equip_slot" class="session-pill">{{ formatLabel(inspectorItem.equip_slot) }}</span>
              <span v-for="tag in inspectorItem.tags.filter(tag => tag.toLowerCase() !== inspectorItem.category.toLowerCase())" :key="`${inspectorItem.id}-${tag}`" class="session-pill">{{ tag }}</span>
            </div>
          </section>

          <div class="session-inspector__split">
            <section class="session-block">
              <div class="session-block__head">
                <span>Requirements</span>
              </div>
              <div v-if="inspectorItem.required_attributes.length" class="session-stack">
                <div v-for="requirement in inspectorItem.required_attributes" :key="`${inspectorItem.id}-${requirement.attribute_name}`" class="session-stack__item">
                  {{ requirement.attribute_name }} >= {{ requirement.min_value }}
                </div>
              </div>
              <p v-else class="session-empty-copy session-empty-copy--tight">No requirements.</p>
            </section>

            <section class="session-block">
              <div class="session-block__head">
                <span>Modifiers</span>
              </div>
              <div v-if="inspectorItem.attribute_modifiers.length" class="session-stack">
                <div v-for="modifier in inspectorItem.attribute_modifiers" :key="`${inspectorItem.id}-${modifier.attribute_name}`" class="session-stack__item">
                  {{ modifier.attribute_name }} {{ modifier.modifier_value > 0 ? '+' : '' }}{{ modifier.modifier_value }}{{ modifier.is_percentage ? '%' : '' }}
                </div>
              </div>
              <p v-else class="session-empty-copy session-empty-copy--tight">No modifiers.</p>
            </section>
          </div>
        </div>

        <div v-else class="session-inspector__empty">
          <Package :size="18" />
          <p>Hover or select an item to inspect it here.</p>
        </div>
      </aside>
    </div>

    <Teleport to="body">
      <div v-if="showCreateFolderModal" class="session-modal" @click.self="showCreateFolderModal = false">
        <div class="session-modal__scrim"></div>
        <div class="session-modal__card session-modal__card--narrow">
          <button class="session-modal__close" @click="showCreateFolderModal = false">
            <X :size="18" />
          </button>

          <div class="session-modal__header">
            <p class="session-panel__eyebrow">New Folder</p>
            <h3 class="session-panel__title">Create Archive Directory</h3>
            <p class="session-panel__description">Structure the item library before dropping gear into it.</p>
          </div>

          <div class="session-modal__form session-modal__form--compact">
            <label class="session-form-field">
              <span>Folder Name</span>
              <input v-model="folderNameDraft" type="text" placeholder="Boss Drops, Temple Loot, Shops" />
            </label>

            <label class="session-form-field">
              <span>Parent Folder</span>
              <select v-model="folderParentDraft">
                <option v-for="option in folderOptions" :key="option.value || 'root'" :value="option.value">{{ option.label }}</option>
              </select>
            </label>
          </div>

          <div class="session-modal__footer">
            <button class="session-item-button session-item-button--ghost" @click="showCreateFolderModal = false">Cancel</button>
            <button class="session-item-button session-item-button--accent" :disabled="!folderNameDraft.trim()" @click="createFolder">Create Folder</button>
          </div>
        </div>
      </div>
    </Teleport>

    <Teleport to="body">
      <div v-if="showCreateItemModal" class="session-modal" @click.self="showCreateItemModal = false">
        <div class="session-modal__scrim"></div>
        <div class="session-modal__card session-modal__card--wide">
          <button class="session-modal__close" @click="showCreateItemModal = false">
            <X :size="18" />
          </button>

          <div class="session-modal__header">
            <p class="session-panel__eyebrow">Create Item</p>
            <h3 class="session-panel__title">New Explorer Entry</h3>
            <p class="session-panel__description">Form fields are on the left, live preview is on the right. This save now goes straight to the backend.</p>
          </div>

          <div class="session-modal__layout">
            <div class="session-scroll session-modal__form">
              <section class="session-block">
                <div class="session-block__head">
                  <span>Core Identity</span>
                </div>

                <label class="session-form-field">
                  <span>Name</span>
                  <input v-model="itemDraft.name" type="text" placeholder="Astral Lantern" />
                </label>

                <label class="session-form-field">
                  <span>Description</span>
                  <textarea v-model="itemDraft.description" rows="5" placeholder="Describe what the item is, what it does, and how it feels in the world."></textarea>
                </label>

                <div class="session-form-grid">
                  <label class="session-form-field">
                    <span>Folder</span>
                    <select v-model="itemDraft.folderId">
                      <option v-for="option in folderOptions" :key="option.value || 'root'" :value="option.value">{{ option.label }}</option>
                    </select>
                  </label>

                  <label class="session-form-field">
                    <span>Quality</span>
                    <select v-model="itemDraft.rarity">
                      <option v-for="rarity in RARITY_OPTIONS" :key="rarity" :value="rarity">{{ formatLabel(rarity) }}</option>
                    </select>
                  </label>
                </div>

                <div class="session-form-grid">
                  <label class="session-form-field">
                    <span>Category</span>
                    <select v-model="itemDraft.category">
                      <option v-for="category in CATEGORY_OPTIONS" :key="category" :value="category">{{ formatLabel(category) }}</option>
                    </select>
                  </label>

                  <label class="session-form-field">
                    <span>Equip Slot</span>
                    <select v-model="itemDraft.equipSlot">
                      <option value="">No slot</option>
                      <option v-for="slot in EQUIP_SLOT_OPTIONS" :key="slot" :value="slot">{{ formatLabel(slot) }}</option>
                    </select>
                  </label>
                </div>
              </section>

              <section class="session-block">
                <div class="session-block__head">
                  <span>Footprint</span>
                </div>

                <div class="session-form-grid">
                  <label class="session-form-field">
                    <span>Grid Width</span>
                    <input v-model.number="itemDraft.gridWidth" type="number" min="1" />
                  </label>

                  <label class="session-form-field">
                    <span>Grid Height</span>
                    <input v-model.number="itemDraft.gridHeight" type="number" min="1" />
                  </label>
                </div>
              </section>

              <section class="session-block">
                <div class="session-block__head">
                  <span>Stats</span>
                </div>

                <div class="session-form-grid">
                  <label class="session-form-field">
                    <span>Requirements</span>
                    <textarea v-model="itemDraft.requirementsText" rows="6" placeholder="strength:12&#10;dexterity:10"></textarea>
                  </label>

                  <label class="session-form-field">
                    <span>Modifiers</span>
                    <textarea v-model="itemDraft.modifiersText" rows="6" placeholder="strength:2&#10;damage:5%"></textarea>
                  </label>
                </div>
              </section>
            </div>

            <aside class="session-modal__preview">
              <div class="session-preview-card session-preview-card--modal">
                <div class="session-preview-card__media" :class="`session-preview-card__media--${draftPreviewItem.rarity}`">
                  <span>{{ itemInitial(draftPreviewItem) }}</span>
                </div>

                <div class="session-preview-card__copy">
                  <div class="session-preview-card__row">
                    <h4>{{ draftPreviewItem.name }}</h4>
                    <span class="session-preview-card__chip">Preview</span>
                  </div>
                  <p>{{ shortText(draftPreviewItem.description, 240) }}</p>

                  <div class="session-badge-row">
                    <span class="session-rarity-badge" :class="`session-rarity-badge--${draftPreviewItem.rarity}`">{{ formatLabel(draftPreviewItem.rarity) }}</span>
                    <span class="session-pill">{{ formatLabel(draftPreviewItem.category) }}</span>
                    <span class="session-pill">{{ draftPreviewItem.grid_width }}x{{ draftPreviewItem.grid_height }}</span>
                    <span class="session-pill">{{ draftPreviewItem.equip_slot ? formatLabel(draftPreviewItem.equip_slot) : 'No slot' }}</span>
                  </div>
                </div>
              </div>

              <article class="session-mini-card session-mini-card--wide">
                <span>Destination</span>
                <strong>{{ folderPathLabel(itemDraft.folderId) }}</strong>
              </article>

              <div class="session-inspector__split">
                <section class="session-block">
                  <div class="session-block__head">
                    <span>Requirements</span>
                  </div>
                  <div v-if="draftPreviewItem.required_attributes.length" class="session-stack">
                    <div v-for="requirement in draftPreviewItem.required_attributes" :key="`${requirement.attribute_name}-${requirement.min_value}`" class="session-stack__item">
                      {{ requirement.attribute_name }} >= {{ requirement.min_value }}
                    </div>
                  </div>
                  <p v-else class="session-empty-copy session-empty-copy--tight">No requirements.</p>
                </section>

                <section class="session-block">
                  <div class="session-block__head">
                    <span>Modifiers</span>
                  </div>
                  <div v-if="draftPreviewItem.attribute_modifiers.length" class="session-stack">
                    <div v-for="modifier in draftPreviewItem.attribute_modifiers" :key="`${modifier.attribute_name}-${modifier.modifier_value}-${modifier.is_percentage}`" class="session-stack__item">
                      {{ modifier.attribute_name }} {{ modifier.modifier_value > 0 ? '+' : '' }}{{ modifier.modifier_value }}{{ modifier.is_percentage ? '%' : '' }}
                    </div>
                  </div>
                  <p v-else class="session-empty-copy session-empty-copy--tight">No modifiers.</p>
                </section>
              </div>
            </aside>
          </div>

          <p v-if="itemCreateError" class="session-create-error">{{ itemCreateError }}</p>

          <div class="session-modal__footer">
            <button class="session-item-button session-item-button--ghost" @click="showCreateItemModal = false">Cancel</button>
            <button class="session-item-button session-item-button--accent" :disabled="!itemDraft.name.trim() || createItemSubmitting" @click="createItem">
              <FilePlus2 :size="16" />
              <span>{{ createItemSubmitting ? 'Saving...' : 'Save Item' }}</span>
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </section>
</template>

<style scoped>
.session-item-manager {
  position: relative;
  overflow: hidden;
  padding: 1.5rem;
  border: 1px solid rgba(126, 200, 227, 0.14);
  border-radius: 2rem;
  background:
    radial-gradient(circle at top right, rgba(233, 69, 96, 0.12), transparent 24%),
    radial-gradient(circle at top left, rgba(126, 200, 227, 0.12), transparent 34%),
    linear-gradient(180deg, rgba(10, 18, 32, 0.96), rgba(8, 12, 24, 0.98));
  box-shadow: 0 28px 80px rgba(0, 0, 0, 0.28);
}

.session-item-manager::before,
.session-panel::before,
.session-modal__card::before {
  content: '';
  position: absolute;
  inset: 0;
  pointer-events: none;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.03), transparent 26%),
    repeating-linear-gradient(135deg, rgba(255, 255, 255, 0.016) 0 2px, transparent 2px 18px);
  opacity: 0.72;
}

.session-item-manager > *,
.session-panel > *,
.session-modal__card > * {
  position: relative;
  z-index: 1;
}

.session-item-manager__hero,
.session-toolbar__top,
.session-toolbar__bottom,
.session-panel__head,
.session-preview-card__row,
.session-folder-card__button {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
}

.session-item-manager__hero {
  align-items: flex-start;
}

.session-item-manager__eyebrow,
.session-panel__eyebrow {
  margin: 0;
  font-size: 0.72rem;
  letter-spacing: 0.22em;
  text-transform: uppercase;
  color: rgba(126, 200, 227, 0.62);
}

.session-item-manager__title,
.session-panel__title {
  margin: 0.55rem 0 0;
  font-family: Cinzel, serif;
  font-size: clamp(1.45rem, 2vw, 2.15rem);
  font-weight: 700;
  color: #f6f7fb;
}

.session-item-manager__description,
.session-panel__description,
.session-empty-copy,
.session-inspector__empty p,
.session-preview-card__copy p,
.session-folder-card__meta {
  margin: 0.7rem 0 0;
  font-size: 0.9rem;
  line-height: 1.65;
  color: rgba(216, 220, 231, 0.68);
}

.session-item-manager__actions,
.session-badge-row,
.session-breadcrumbs,
.session-toolbar__filters,
.session-inspector__stats,
.session-inspector__split,
.session-form-grid,
.session-item-manager__stats,
.session-stack,
.session-workspace,
.session-modal__preview,
.session-modal__form,
.session-scroll,
.session-tree-list,
.session-item-grid,
.session-folder-grid {
  display: grid;
}

.session-item-manager__actions,
.session-badge-row,
.session-breadcrumbs {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
}

.session-item-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.55rem;
  min-height: 2.8rem;
  padding: 0.85rem 1.15rem;
  border: 1px solid rgba(126, 200, 227, 0.18);
  border-radius: 1rem;
  background: rgba(13, 21, 37, 0.78);
  color: #f6f7fb;
  font-size: 0.9rem;
  font-weight: 600;
  cursor: pointer;
  transition: border-color 0.2s ease, transform 0.2s ease, background 0.2s ease;
}

.session-item-button:hover {
  transform: translateY(-1px);
}

.session-item-button--ghost {
  background: rgba(126, 200, 227, 0.08);
}

.session-item-button--accent {
  border-color: rgba(233, 69, 96, 0.26);
  background: rgba(233, 69, 96, 0.14);
  color: #ffe0e7;
}

.session-item-button:disabled {
  cursor: not-allowed;
  opacity: 0.58;
  transform: none;
}

.session-item-manager__stats {
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0.85rem;
  margin-top: 1.2rem;
}

.session-item-stat,
.session-mini-card,
.session-panel,
.session-folder-card,
.session-block,
.session-preview-card,
.session-tree-root,
.session-sidebar__dropzone,
.session-tree-row__body,
.session-modal__card {
  position: relative;
  overflow: hidden;
  border: 1px solid rgba(126, 200, 227, 0.14);
  border-radius: 1.25rem;
  background: rgba(11, 20, 36, 0.76);
}

.session-item-stat,
.session-mini-card {
  padding: 1rem 1.05rem;
}

.session-item-stat span,
.session-mini-card span,
.session-block__head span,
.session-form-field span,
.session-filter-field span {
  font-size: 0.72rem;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: rgba(126, 200, 227, 0.54);
}

.session-item-stat strong,
.session-mini-card strong {
  display: block;
  margin-top: 0.35rem;
  font-size: 1.28rem;
  color: #f6f7fb;
}

.session-item-manager__shell {
  display: grid;
  grid-template-columns: 280px minmax(0, 1fr) 360px;
  gap: 1rem;
  margin-top: 1rem;
}

.session-panel {
  border-radius: 1.75rem;
}

.session-panel__head {
  padding: 1.1rem 1.15rem 0;
}

.session-panel__head--compact {
  padding-bottom: 0.5rem;
}

.session-pill,
.session-rarity-badge,
.session-preview-card__chip {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 1.8rem;
  padding: 0.2rem 0.7rem;
  border: 1px solid rgba(126, 200, 227, 0.16);
  border-radius: 999px;
  background: rgba(126, 200, 227, 0.08);
  color: rgba(216, 220, 231, 0.88);
  font-size: 0.75rem;
}

.session-sidebar {
  padding-bottom: 1rem;
}

.session-tree-root,
.session-sidebar__dropzone {
  margin: 1rem 1rem 0;
  padding: 1rem;
}

.session-tree-root {
  display: flex;
  align-items: center;
  justify-content: space-between;
  cursor: pointer;
}

.session-tree-root__label {
  display: block;
  font-size: 0.95rem;
  font-weight: 700;
  color: #f6f7fb;
}

.session-tree-root__meta,
.session-tree-row__meta {
  display: block;
  margin-top: 0.3rem;
  font-size: 0.78rem;
  color: rgba(126, 200, 227, 0.56);
}

.session-tree-root--active,
.session-tree-row__body--active {
  border-color: rgba(233, 69, 96, 0.32);
  background: rgba(233, 69, 96, 0.12);
}

.session-tree-root--drop,
.session-tree-row__body--drop,
.session-folder-card--drop,
.session-sidebar__dropzone--active,
.session-item-surface--drop {
  border-color: rgba(233, 69, 96, 0.34) !important;
  background: rgba(233, 69, 96, 0.1) !important;
}

.session-tree-list {
  gap: 0.35rem;
  max-height: 38rem;
  margin: 0.95rem 0.55rem 0;
  padding: 0 0.45rem 0 0.35rem;
}

.session-tree-row {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 0.35rem;
  align-items: center;
  padding-left: calc(var(--tree-depth, 0) * 0.95rem);
}

.session-tree-row__toggle,
.session-search-field__clear,
.session-modal__close,
.session-breadcrumbs button,
.session-reset-button,
.session-folder-card__button,
.session-item-card,
.session-tree-row__body {
  border: none;
  background: transparent;
}

.session-tree-row__toggle,
.session-modal__close,
.session-search-field__clear {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: rgba(126, 200, 227, 0.62);
  cursor: pointer;
}

.session-tree-row__toggle {
  width: 1.7rem;
  height: 1.7rem;
  border-radius: 0.75rem;
}

.session-tree-row__toggle--placeholder {
  cursor: default;
}

.session-tree-row__body {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  width: 100%;
  min-height: 2.55rem;
  padding: 0.65rem 0.85rem;
  border-radius: 1rem;
  border: 1px solid rgba(126, 200, 227, 0.12);
  background: rgba(11, 20, 36, 0.64);
  cursor: pointer;
}

.session-tree-row__name,
.session-folder-card__title,
.session-item-card__name,
.session-preview-card__row h4 {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin: 0;
  color: #f6f7fb;
}

.session-tree-row__name,
.session-folder-card__title {
  font-size: 0.95rem;
  font-weight: 700;
}

.session-sidebar__dropzone {
  color: rgba(216, 220, 231, 0.78);
  line-height: 1.6;
}

.session-workspace {
  gap: 1rem;
}

.session-toolbar {
  padding-bottom: 1rem;
}

.session-toolbar__top,
.session-toolbar__bottom {
  padding: 0 1.15rem;
}

.session-toolbar__top {
  margin-top: 1rem;
}

.session-toolbar__filters {
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.75rem;
  padding: 1rem 1.15rem;
}

.session-search-field,
.session-filter-field select,
.session-form-field input,
.session-form-field select,
.session-form-field textarea {
  width: 100%;
  border: 1px solid rgba(126, 200, 227, 0.14);
  border-radius: 1rem;
  background: rgba(11, 20, 36, 0.88);
  color: #f6f7fb;
  outline: none;
}

.session-search-field {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 0.75rem;
  min-height: 3rem;
  padding: 0 1rem;
}

.session-search-field input {
  border: none;
  background: transparent;
  color: #f6f7fb;
}

.session-filter-field,
.session-form-field {
  display: grid;
  gap: 0.45rem;
}

.session-filter-field select,
.session-form-field input,
.session-form-field select,
.session-form-field textarea {
  min-height: 2.8rem;
  padding: 0.8rem 0.95rem;
}

.session-form-field textarea {
  min-height: 7rem;
  resize: vertical;
}

.session-filter-field--sort {
  min-width: 13rem;
}

.session-breadcrumbs button,
.session-reset-button {
  padding: 0.55rem 0.85rem;
  border-radius: 999px;
  border: 1px solid rgba(126, 200, 227, 0.14);
  background: rgba(126, 200, 227, 0.06);
  color: rgba(216, 220, 231, 0.88);
  cursor: pointer;
}

.session-folder-grid,
.session-item-grid {
  gap: 0.9rem;
  padding: 0.25rem 1rem 1rem;
}

.session-folder-grid {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.session-item-grid {
  grid-template-columns: repeat(auto-fill, minmax(168px, 1fr));
  max-height: 45rem;
}

.session-folder-card {
  padding: 0.85rem;
}

.session-folder-card__meta {
  margin-top: 0.35rem;
}

.session-item-card {
  display: grid;
  gap: 0.7rem;
  padding: 0.8rem;
  border-radius: 1.35rem;
  border: 1px solid rgba(126, 200, 227, 0.14);
  background: rgba(10, 18, 33, 0.76);
  text-align: left;
  cursor: pointer;
}

.session-item-card--selected {
  border-color: rgba(233, 69, 96, 0.32);
  box-shadow: inset 0 0 0 1px rgba(233, 69, 96, 0.16);
}

.session-item-card__art,
.session-preview-card__media {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border: 2px solid rgba(255, 255, 255, 0.24);
  border-radius: 1rem;
  background: linear-gradient(180deg, rgba(27, 49, 93, 0.98), rgba(13, 24, 47, 1));
}

.session-item-card__art {
  aspect-ratio: 1 / 1;
}

.session-preview-card__media {
  width: 5.8rem;
  height: 5.8rem;
  flex-shrink: 0;
  font-family: Cinzel, serif;
  font-size: 2rem;
  font-weight: 700;
  color: #f6f7fb;
}

.session-item-card__art img,
.session-preview-card__media img {
  width: 100%;
  height: 100%;
  object-fit: contain;
  padding: 0.55rem;
}

.session-item-card__glyph {
  font-family: Cinzel, serif;
  font-size: 2rem;
  font-weight: 700;
  color: #f6f7fb;
}

.session-item-card__size {
  position: absolute;
  top: 0.55rem;
  right: 0.55rem;
  min-height: 1.6rem;
  padding: 0.1rem 0.55rem;
  border-radius: 999px;
  background: rgba(8, 12, 24, 0.82);
  border: 1px solid rgba(255, 255, 255, 0.14);
  font-size: 0.72rem;
  font-weight: 700;
  color: #f8fafc;
}

.session-item-card__body {
  display: grid;
  gap: 0.2rem;
}

.session-item-card__name {
  font-size: 0.95rem;
}

.session-item-card__meta {
  font-size: 0.8rem;
  color: rgba(216, 220, 231, 0.62);
}

.session-item-card__art--common,
.session-preview-card__media--common {
  border-color: rgba(255, 255, 255, 0.7);
}

.session-item-card__art--uncommon,
.session-preview-card__media--uncommon {
  border-color: rgba(250, 204, 21, 0.8);
}

.session-item-card__art--rare,
.session-preview-card__media--rare {
  border-color: rgba(96, 165, 250, 0.8);
}

.session-item-card__art--epic,
.session-preview-card__media--epic {
  border-color: rgba(192, 132, 252, 0.82);
}

.session-item-card__art--masterwork,
.session-preview-card__media--masterwork {
  border-color: rgba(251, 146, 60, 0.84);
}

.session-item-card__art--legendary,
.session-preview-card__media--legendary {
  border-color: rgba(74, 222, 128, 0.82);
}

.session-item-card__art--unique,
.session-preview-card__media--unique {
  border-color: rgba(248, 113, 113, 0.84);
}

.session-rarity-badge {
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.session-rarity-badge--common {
  border-color: rgba(255, 255, 255, 0.24);
  background: rgba(255, 255, 255, 0.08);
  color: #f8fafc;
}

.session-rarity-badge--uncommon {
  border-color: rgba(250, 204, 21, 0.28);
  background: rgba(202, 138, 4, 0.16);
  color: #fde68a;
}

.session-rarity-badge--rare {
  border-color: rgba(96, 165, 250, 0.28);
  background: rgba(37, 99, 235, 0.16);
  color: #93c5fd;
}

.session-rarity-badge--epic {
  border-color: rgba(192, 132, 252, 0.3);
  background: rgba(126, 34, 206, 0.18);
  color: #e9d5ff;
}

.session-rarity-badge--masterwork {
  border-color: rgba(251, 146, 60, 0.3);
  background: rgba(194, 65, 12, 0.18);
  color: #fdba74;
}

.session-rarity-badge--legendary {
  border-color: rgba(74, 222, 128, 0.3);
  background: rgba(21, 128, 61, 0.18);
  color: #86efac;
}

.session-rarity-badge--unique {
  border-color: rgba(248, 113, 113, 0.3);
  background: rgba(153, 27, 27, 0.18);
  color: #fca5a5;
}

.session-inspector {
  padding-bottom: 1rem;
}

.session-inspector__content {
  display: grid;
  gap: 1rem;
  padding: 0.25rem 1rem 0;
}

.session-preview-card {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 1rem;
  padding: 1rem;
}

.session-mini-card--wide strong {
  font-size: 1rem;
  line-height: 1.6;
}

.session-inspector__stats,
.session-inspector__split,
.session-form-grid {
  gap: 0.8rem;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.session-stack {
  gap: 0.55rem;
}

.session-stack__item {
  padding: 0.7rem 0.8rem;
  border: 1px solid rgba(126, 200, 227, 0.12);
  border-radius: 0.95rem;
  background: rgba(126, 200, 227, 0.05);
  color: #f6f7fb;
  font-size: 0.86rem;
}

.session-inspector__empty {
  display: grid;
  place-items: center;
  min-height: 18rem;
  padding: 1rem;
  text-align: center;
}

.session-empty-copy {
  padding: 0.75rem 1rem 1rem;
}

.session-empty-copy--tight {
  padding: 0;
  margin-top: 0;
}

.session-modal {
  position: fixed;
  inset: 0;
  z-index: 11000;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1.25rem;
}

.session-modal__scrim {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.72);
  backdrop-filter: blur(10px);
}

.session-modal__card {
  width: min(100%, 76rem);
  max-height: min(92vh, 68rem);
  display: flex;
  flex-direction: column;
  border-radius: 2rem;
  background: linear-gradient(180deg, rgba(12, 18, 33, 0.98), rgba(8, 12, 24, 0.98));
  box-shadow: 0 30px 100px rgba(0, 0, 0, 0.46);
}

.session-modal__card--narrow {
  width: min(100%, 34rem);
}

.session-modal__header,
.session-modal__footer {
  padding: 1.25rem 1.35rem;
}

.session-modal__header {
  padding-right: 4rem;
}

.session-modal__close {
  position: absolute;
  top: 1.15rem;
  right: 1.15rem;
  width: 2.15rem;
  height: 2.15rem;
  border-radius: 999px;
}

.session-modal__layout {
  display: grid;
  grid-template-columns: minmax(0, 1.35fr) 23rem;
  gap: 1rem;
  min-height: 0;
  padding: 0 1.35rem 1.35rem;
}

.session-modal__form {
  gap: 1rem;
  min-height: 0;
  padding-right: 0.35rem;
}

.session-modal__form--compact {
  padding: 0 1.35rem 1.35rem;
}

.session-modal__preview {
  gap: 0.9rem;
  align-content: start;
}

.session-modal__footer {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  border-top: 1px solid rgba(126, 200, 227, 0.1);
}

.session-create-error {
  margin: 0 1.35rem;
  padding: 0.85rem 1rem;
  border: 1px solid rgba(248, 113, 113, 0.18);
  border-radius: 1rem;
  background: rgba(127, 29, 29, 0.18);
  color: #fecaca;
  font-size: 0.9rem;
}

.session-scroll {
  overflow: auto;
}

.session-scroll::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

.session-scroll::-webkit-scrollbar-thumb {
  border-radius: 999px;
  background: rgba(126, 200, 227, 0.24);
}

.session-scroll::-webkit-scrollbar-track {
  background: transparent;
}

@media (max-width: 1535px) {
  .session-item-manager__shell {
    grid-template-columns: 260px minmax(0, 1fr);
  }

  .session-inspector {
    grid-column: 1 / -1;
  }
}

@media (max-width: 1279px) {
  .session-item-manager {
    padding: 1.2rem;
  }

  .session-item-manager__hero,
  .session-toolbar__top,
  .session-toolbar__bottom {
    flex-direction: column;
    align-items: stretch;
  }

  .session-item-manager__stats,
  .session-item-manager__shell,
  .session-toolbar__filters,
  .session-folder-grid,
  .session-inspector__stats,
  .session-inspector__split,
  .session-form-grid,
  .session-modal__layout {
    grid-template-columns: 1fr;
  }

  .session-filter-field--sort {
    min-width: 0;
  }
}

@media (max-width: 767px) {
  .session-item-manager__stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .session-item-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .session-preview-card {
    grid-template-columns: 1fr;
  }

  .session-modal {
    padding: 0.75rem;
  }

  .session-modal__header,
  .session-modal__footer,
  .session-modal__layout,
  .session-modal__form--compact {
    padding-left: 1rem;
    padding-right: 1rem;
  }
}
</style>