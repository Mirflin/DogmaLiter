<script setup>
import { computed, ref, watch } from 'vue'

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

const RARITY_OPTIONS = ['common', 'uncommon', 'rare', 'epic', 'legendary', 'artifact']
const SORT_OPTIONS = [
  { value: 'name-asc', label: 'Name A-Z' },
  { value: 'name-desc', label: 'Name Z-A' },
  { value: 'rarity', label: 'Rarity' },
  { value: 'recent', label: 'Recently Updated' },
]
const EQUIP_SLOT_OPTIONS = ['head', 'chest', 'gloves', 'belt', 'boots', 'main_hand', 'off_hand', 'ring_1', 'ring_2', 'amulet']

const currentFolderId = ref(null)
const searchQuery = ref('')
const sortMode = ref('name-asc')
const folders = ref([])
const itemLocations = ref({})
const localItems = ref([])
const draggingEntry = ref(null)
const dragTargetKey = ref('')
const showCreateFolderModal = ref(false)
const folderNameDraft = ref('')
const folderParentDraft = ref(null)
const showCreateItemModal = ref(false)
const selectedItem = ref(null)
const saveStateReady = ref(false)
const itemDraft = ref(createEmptyItemDraft())

const storageKey = computed(() => props.gameId ? `dogmaliter:item-manager:${props.gameId}` : '')
const folderMap = computed(() => Object.fromEntries(folders.value.map(folder => [folder.id, folder])))
const searchNeedle = computed(() => searchQuery.value.trim().toLowerCase())
const allItems = computed(() => [
  ...(props.items ?? []).map(item => ({ ...item, source: 'server' })),
  ...localItems.value.map(item => ({ ...item, source: 'local' })),
])
const folderTreeRows = computed(() => flattenFolderRows())
const breadcrumbs = computed(() => {
  const trail = [{ id: null, name: 'Item Library' }]
  if (!currentFolderId.value) return trail

  const chain = []
  let cursorId = currentFolderId.value
  while (cursorId && folderMap.value[cursorId]) {
    const folder = folderMap.value[cursorId]
    chain.unshift({ id: folder.id, name: folder.name })
    cursorId = folder.parentId ?? null
  }

  return [...trail, ...chain]
})
const visibleFolders = computed(() => {
  const pool = searchNeedle.value
    ? folders.value.filter(folder => folderMatchesSearch(folder))
    : childFoldersOf(currentFolderId.value)

  return [...pool].sort((left, right) => left.name.localeCompare(right.name))
})
const visibleItems = computed(() => {
  const pool = searchNeedle.value
    ? allItems.value.filter(item => itemMatchesSearch(item))
    : allItems.value.filter(item => normalizeFolderId(itemLocations.value[item.id]) === normalizeFolderId(currentFolderId.value))

  return [...pool].sort(compareItems)
})

watch(() => props.gameId, () => {
  loadState()
}, { immediate: true })

watch([folders, itemLocations, localItems], () => {
  saveState()
}, { deep: true })

function createEmptyItemDraft() {
  return {
    name: '',
    description: '',
    rarity: 'common',
    gridWidth: 1,
    gridHeight: 1,
    typesText: '',
    equippable: false,
    equipSlot: '',
    requirementsText: '',
    modifiersText: '',
  }
}

function normalizeFolderId(folderId) {
  return folderId ?? null
}

function createId(prefix) {
  if (window.crypto?.randomUUID) {
    return `${prefix}-${window.crypto.randomUUID()}`
  }

  return `${prefix}-${Date.now()}-${Math.random().toString(16).slice(2)}`
}

function childFoldersOf(folderId) {
  const normalizedParentId = normalizeFolderId(folderId)
  return folders.value.filter(folder => normalizeFolderId(folder.parentId) === normalizedParentId)
}

function flattenFolderRows(parentId = null, depth = 0) {
  const rows = []

  for (const folder of childFoldersOf(parentId).sort((left, right) => left.name.localeCompare(right.name))) {
    rows.push({ ...folder, depth })
    rows.push(...flattenFolderRows(folder.id, depth + 1))
  }

  return rows
}

function folderMatchesSearch(folder) {
  if (!searchNeedle.value) return true
  const pathText = buildFolderPath(folder.id).map(segment => segment.name).join(' / ').toLowerCase()
  return pathText.includes(searchNeedle.value)
}

function itemMatchesSearch(item) {
  const text = [
    item.name,
    item.description,
    item.rarity,
    ...(item.types ?? []),
  ]
    .filter(Boolean)
    .join(' ')
    .toLowerCase()

  return text.includes(searchNeedle.value)
}

function buildFolderPath(folderId) {
  const chain = []
  let cursorId = folderId

  while (cursorId && folderMap.value[cursorId]) {
    const folder = folderMap.value[cursorId]
    chain.unshift(folder)
    cursorId = folder.parentId ?? null
  }

  return chain
}

function folderItemCount(folderId) {
  return allItems.value.filter(item => normalizeFolderId(itemLocations.value[item.id]) === normalizeFolderId(folderId)).length
}

function folderChildCount(folderId) {
  return childFoldersOf(folderId).length
}

function folderPathLabel(folderId) {
  const path = buildFolderPath(folderId)
  if (!path.length) return 'Item Library'
  return `Item Library / ${path.map(folder => folder.name).join(' / ')}`
}

function rarityRank(rarity) {
  const order = {
    common: 0,
    uncommon: 1,
    rare: 2,
    epic: 3,
    legendary: 4,
    artifact: 5,
  }

  return order[rarity] ?? 0
}

function compareItems(left, right) {
  if (sortMode.value === 'name-desc') {
    return right.name.localeCompare(left.name)
  }

  if (sortMode.value === 'rarity') {
    const rarityDelta = rarityRank(right.rarity) - rarityRank(left.rarity)
    if (rarityDelta !== 0) return rarityDelta
    return left.name.localeCompare(right.name)
  }

  if (sortMode.value === 'recent') {
    const leftTime = Date.parse(left.updated_at || left.created_at || 0)
    const rightTime = Date.parse(right.updated_at || right.created_at || 0)
    if (rightTime !== leftTime) return rightTime - leftTime
    return left.name.localeCompare(right.name)
  }

  return left.name.localeCompare(right.name)
}

function loadState() {
  saveStateReady.value = false
  currentFolderId.value = null
  searchQuery.value = ''
  sortMode.value = 'name-asc'
  folders.value = []
  itemLocations.value = {}
  localItems.value = []

  if (!storageKey.value) {
    saveStateReady.value = true
    return
  }

  try {
    const raw = window.localStorage.getItem(storageKey.value)
    if (raw) {
      const parsed = JSON.parse(raw)
      folders.value = Array.isArray(parsed.folders) ? parsed.folders : []
      itemLocations.value = parsed.itemLocations && typeof parsed.itemLocations === 'object' ? parsed.itemLocations : {}
      localItems.value = Array.isArray(parsed.localItems) ? parsed.localItems : []
    }
  } catch {
    folders.value = []
    itemLocations.value = {}
    localItems.value = []
  }

  if (currentFolderId.value && !folderMap.value[currentFolderId.value]) {
    currentFolderId.value = null
  }

  saveStateReady.value = true
}

function saveState() {
  if (!saveStateReady.value || !storageKey.value) return

  window.localStorage.setItem(storageKey.value, JSON.stringify({
    folders: folders.value,
    itemLocations: itemLocations.value,
    localItems: localItems.value,
  }))
}

function selectFolder(folderId) {
  currentFolderId.value = normalizeFolderId(folderId)
}

function openCreateFolderModal(parentId = currentFolderId.value) {
  folderNameDraft.value = ''
  folderParentDraft.value = normalizeFolderId(parentId)
  showCreateFolderModal.value = true
}

function createFolder() {
  const name = folderNameDraft.value.trim()
  if (!name) return

  const folder = {
    id: createId('folder'),
    name,
    parentId: normalizeFolderId(folderParentDraft.value),
    created_at: new Date().toISOString(),
  }

  folders.value = [...folders.value, folder]
  currentFolderId.value = folder.id
  showCreateFolderModal.value = false
}

function openCreateItemModal() {
  itemDraft.value = createEmptyItemDraft()
  showCreateItemModal.value = true
}

function parseTypes(text) {
  return text
    .split(',')
    .map(value => value.trim())
    .filter(Boolean)
}

function parseRequirements(text) {
  return text
    .split('\n')
    .map(line => line.trim())
    .filter(Boolean)
    .map(line => {
      const [attributeName, minValue] = line.split(':').map(part => part.trim())
      return {
        attribute_name: attributeName,
        min_value: Number(minValue) || 0,
      }
    })
    .filter(entry => entry.attribute_name)
}

function parseModifiers(text) {
  return text
    .split('\n')
    .map(line => line.trim())
    .filter(Boolean)
    .map(line => {
      const [attributeName, rawValue] = line.split(':').map(part => part.trim())
      const isPercentage = rawValue?.endsWith('%') ?? false
      const parsedValue = Number((rawValue || '').replace('%', '')) || 0
      return {
        attribute_name: attributeName,
        modifier_value: parsedValue,
        is_percentage: isPercentage,
      }
    })
    .filter(entry => entry.attribute_name)
}

function createItem() {
  const name = itemDraft.value.name.trim()
  if (!name) return

  const now = new Date().toISOString()
  const item = {
    id: createId('local-item'),
    game_id: props.gameId,
    created_by_id: 'local-workspace',
    name,
    description: itemDraft.value.description.trim(),
    image_id: null,
    rarity: itemDraft.value.rarity,
    grid_width: Math.max(1, Number(itemDraft.value.gridWidth) || 1),
    grid_height: Math.max(1, Number(itemDraft.value.gridHeight) || 1),
    is_equippable: itemDraft.value.equippable,
    equip_slot: itemDraft.value.equippable && itemDraft.value.equipSlot ? itemDraft.value.equipSlot : null,
    types: parseTypes(itemDraft.value.typesText),
    required_attributes: parseRequirements(itemDraft.value.requirementsText),
    attribute_modifiers: parseModifiers(itemDraft.value.modifiersText),
    created_at: now,
    updated_at: now,
  }

  localItems.value = [item, ...localItems.value]
  itemLocations.value = {
    ...itemLocations.value,
    [item.id]: normalizeFolderId(currentFolderId.value),
  }

  selectedItem.value = { ...item, source: 'local' }
  showCreateItemModal.value = false
}

function openItemDetails(item) {
  selectedItem.value = item
}

function deleteLocalItem(itemId) {
  localItems.value = localItems.value.filter(item => item.id !== itemId)
  const nextLocations = { ...itemLocations.value }
  delete nextLocations[itemId]
  itemLocations.value = nextLocations
  selectedItem.value = null
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

function folderBadgeClass(folderId) {
  if (normalizeFolderId(folderId) === normalizeFolderId(currentFolderId.value)) {
    return 'border-[rgba(233,69,96,0.36)] bg-[rgba(233,69,96,0.16)] text-[#ffe0e7]'
  }

  return 'border-[rgba(126,200,227,0.12)] bg-[rgba(126,200,227,0.05)] text-[#dbe4f0] hover:border-[rgba(126,200,227,0.24)] hover:bg-[rgba(126,200,227,0.08)]'
}

function rarityBadgeClass(rarity) {
  const variants = {
    common: 'border-[rgba(148,163,184,0.28)] bg-[rgba(148,163,184,0.12)] text-[#e2e8f0]',
    uncommon: 'border-[rgba(74,222,128,0.28)] bg-[rgba(22,163,74,0.14)] text-[#86efac]',
    rare: 'border-[rgba(96,165,250,0.28)] bg-[rgba(37,99,235,0.16)] text-[#93c5fd]',
    epic: 'border-[rgba(244,114,182,0.3)] bg-[rgba(190,24,93,0.16)] text-[#f9a8d4]',
    legendary: 'border-[rgba(251,191,36,0.32)] bg-[rgba(217,119,6,0.18)] text-[#fde68a]',
    artifact: 'border-[rgba(248,113,113,0.32)] bg-[rgba(153,27,27,0.18)] text-[#fca5a5]',
  }

  return variants[rarity] ?? variants.common
}

function formatFolderDescription(folder) {
  return `${folderChildCount(folder.id)} folders · ${folderItemCount(folder.id)} items`
}
</script>

<template>
  <section class="session-item-manager rounded-[2rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(11,20,36,0.88)] p-5 shadow-[0_24px_60px_rgba(0,0,0,0.2)] sm:p-6">
    <div class="session-item-manager__header flex flex-wrap items-start justify-between gap-4">
      <div>
        <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Item Manager</p>
        <h2 class="mt-2 font-[Cinzel] text-[28px] font-bold text-[#f6f7fb]">Campaign Library</h2>
        <p class="mt-2 max-w-[42rem] text-[14px] leading-relaxed text-[#d8dce7]/62">
          Organize items like files: build folder structures, drag entries between folders, search the whole library, and open any item for a full detail view.
        </p>
      </div>

      <div class="flex flex-wrap gap-3">
        <button
          @click="openCreateFolderModal()"
          class="cursor-pointer rounded-xl border border-[rgba(126,200,227,0.16)] bg-[rgba(126,200,227,0.08)] px-4 py-2.5 text-[13px] font-semibold text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(126,200,227,0.3)]"
        >
          New Folder
        </button>
        <button
          @click="openCreateItemModal()"
          class="cursor-pointer rounded-xl border border-[rgba(233,69,96,0.24)] bg-[rgba(233,69,96,0.12)] px-4 py-2.5 text-[13px] font-semibold text-[#ffe0e7] transition-all duration-200 hover:border-[rgba(233,69,96,0.42)]"
        >
          Create Item
        </button>
      </div>
    </div>

    <div class="session-item-manager__layout mt-6 grid gap-5 xl:grid-cols-[340px_minmax(0,1fr)] 2xl:grid-cols-[380px_minmax(0,1fr)]">
      <aside class="session-item-tree rounded-[1.75rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.76)] p-4">
        <div class="flex items-center justify-between gap-3">
          <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Folders</p>
          <span class="rounded-full border border-[rgba(126,200,227,0.14)] px-2.5 py-1 text-[10px] uppercase tracking-[0.16em] text-[#7ec8e3]/60">
            {{ folders.length }} total
          </span>
        </div>

        <button
          @click="selectFolder(null)"
          @dragover.prevent="handleFolderDragOver(null)"
          @dragenter.prevent="handleFolderDragOver(null)"
          @drop.prevent="dropIntoFolder(null)"
          class="mt-4 flex w-full items-center justify-between rounded-2xl border px-3 py-3 text-left text-[13px] font-semibold transition-all duration-200"
          :class="dragTargetActive('folder:root')
            ? 'border-[rgba(233,69,96,0.4)] bg-[rgba(233,69,96,0.14)] text-[#ffe0e7]'
            : folderBadgeClass(null)"
        >
          <span>Item Library</span>
          <span class="text-[11px] text-[#7ec8e3]/55">{{ allItems.length }}</span>
        </button>

        <div v-if="folderTreeRows.length" class="session-item-scroll mt-3 max-h-[540px] space-y-1 overflow-y-auto pr-1">
          <button
            v-for="folder in folderTreeRows"
            :key="folder.id"
            @click="selectFolder(folder.id)"
            draggable="true"
            @dragstart="startFolderDrag(folder.id)"
            @dragover.prevent="handleFolderDragOver(folder.id)"
            @dragenter.prevent="handleFolderDragOver(folder.id)"
            @drop.prevent="dropIntoFolder(folder.id)"
            @dragend="stopDrag"
            class="flex w-full items-center justify-between rounded-2xl border px-3 py-2.5 text-left text-[13px] transition-all duration-200"
            :class="dragTargetActive(`folder:${folder.id}`)
              ? 'border-[rgba(233,69,96,0.4)] bg-[rgba(233,69,96,0.14)] text-[#ffe0e7]'
              : folderBadgeClass(folder.id)"
            :style="{ paddingLeft: `${0.75 + (folder.depth * 0.85)}rem` }"
          >
            <span class="truncate">{{ folder.name }}</span>
            <span class="text-[10px] uppercase tracking-[0.14em] text-[#7ec8e3]/45">{{ folderItemCount(folder.id) }}</span>
          </button>
        </div>

        <p v-else class="mt-4 text-[13px] leading-relaxed text-[#d8dce7]/56">
          No folders yet. Create the first one to start structuring the campaign library.
        </p>
      </aside>

      <div class="session-item-workbench space-y-5">
        <div class="session-item-toolbar rounded-[1.75rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.76)] p-4">
          <div class="flex flex-wrap items-center gap-3">
            <div class="min-w-[220px] flex-1">
              <label class="block text-[11px] uppercase tracking-[0.2em] text-[#7ec8e3]/45">Search</label>
              <input
                v-model="searchQuery"
                type="text"
                placeholder="Search items, descriptions, tags, rarities"
                class="mt-2 w-full rounded-2xl border border-[rgba(126,200,227,0.14)] bg-[rgba(11,20,36,0.9)] px-4 py-3 text-[14px] text-[#f6f7fb] outline-none transition-all duration-200 placeholder:text-[#7ec8e3]/30 focus:border-[rgba(233,69,96,0.34)]"
              />
            </div>

            <div class="w-full min-w-[180px] sm:w-auto">
              <label class="block text-[11px] uppercase tracking-[0.2em] text-[#7ec8e3]/45">Sort</label>
              <select
                v-model="sortMode"
                class="mt-2 w-full rounded-2xl border border-[rgba(126,200,227,0.14)] bg-[rgba(11,20,36,0.9)] px-4 py-3 text-[14px] text-[#f6f7fb] outline-none transition-all duration-200 focus:border-[rgba(233,69,96,0.34)]"
              >
                <option v-for="option in SORT_OPTIONS" :key="option.value" :value="option.value">{{ option.label }}</option>
              </select>
            </div>
          </div>

          <div class="mt-4 flex flex-wrap items-center gap-2 text-[12px] text-[#d8dce7]/60">
            <button
              v-for="segment in breadcrumbs"
              :key="segment.id ?? 'root'"
              @click="selectFolder(segment.id)"
              class="cursor-pointer rounded-full border border-[rgba(126,200,227,0.14)] px-3 py-1.5 transition-all duration-200 hover:border-[rgba(126,200,227,0.3)] hover:text-[#f6f7fb]"
            >
              {{ segment.name }}
            </button>
            <span v-if="searchNeedle" class="rounded-full border border-[rgba(233,69,96,0.16)] px-3 py-1.5 text-[#ffb3c1]">
              {{ visibleItems.length + visibleFolders.length }} search hits
            </span>
          </div>
        </div>

        <div
          class="session-item-dropzone rounded-[1.75rem] border border-dashed border-[rgba(126,200,227,0.18)] bg-[rgba(126,200,227,0.04)] p-4"
          :class="dragTargetActive('folder:root') ? 'border-[rgba(233,69,96,0.4)] bg-[rgba(233,69,96,0.08)]' : ''"
          @dragover.prevent="handleFolderDragOver(currentFolderId)"
          @dragenter.prevent="handleFolderDragOver(currentFolderId)"
          @drop.prevent="dropIntoFolder(currentFolderId)"
        >
          <p class="text-[12px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">
            {{ searchNeedle ? 'Search results across the whole library' : folderPathLabel(currentFolderId) }}
          </p>
          <p class="mt-2 text-[14px] text-[#d8dce7]/62">
            Drop items or folders here to move them into the currently open directory.
          </p>
        </div>

        <div class="session-item-folder-stage rounded-[1.75rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(11,20,36,0.84)] p-4">
          <div class="flex items-center justify-between gap-4">
            <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Folders</p>
            <span class="text-[12px] text-[#d8dce7]/50">{{ visibleFolders.length }} visible</span>
          </div>

          <div v-if="visibleFolders.length" class="session-item-folder-grid mt-4 grid gap-4 sm:grid-cols-2 2xl:grid-cols-3">
            <button
              v-for="folder in visibleFolders"
              :key="folder.id"
              @click="selectFolder(folder.id)"
              draggable="true"
              @dragstart="startFolderDrag(folder.id)"
              @dragover.prevent="handleFolderDragOver(folder.id)"
              @dragenter.prevent="handleFolderDragOver(folder.id)"
              @drop.prevent="dropIntoFolder(folder.id)"
              @dragend="stopDrag"
              class="cursor-pointer rounded-[1.5rem] border p-4 text-left transition-all duration-200"
              :class="dragTargetActive(`folder:${folder.id}`)
                ? 'border-[rgba(233,69,96,0.4)] bg-[rgba(233,69,96,0.14)]'
                : 'border-[rgba(126,200,227,0.12)] bg-[rgba(126,200,227,0.05)] hover:border-[rgba(126,200,227,0.25)]'"
            >
              <p class="truncate text-[15px] font-semibold text-[#f6f7fb]">{{ folder.name }}</p>
              <p class="mt-2 text-[12px] text-[#7ec8e3]/55">{{ formatFolderDescription(folder) }}</p>
            </button>
          </div>

          <p v-else class="mt-4 text-[14px] text-[#d8dce7]/56">
            {{ searchNeedle ? 'No folders match the current search.' : 'No subfolders in this location yet.' }}
          </p>
        </div>

        <div class="session-item-catalog rounded-[1.75rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(11,20,36,0.84)] p-4">
          <div class="flex items-center justify-between gap-4">
            <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Items</p>
            <span class="text-[12px] text-[#d8dce7]/50">{{ visibleItems.length }} visible</span>
          </div>

          <div v-if="visibleItems.length" class="session-item-scroll mt-4 max-h-[720px] overflow-y-auto pr-1">
            <div class="session-item-grid grid gap-4 xl:grid-cols-2 2xl:grid-cols-3">
              <article
                v-for="item in visibleItems"
                :key="item.id"
                draggable="true"
                @dragstart="startItemDrag(item.id)"
                @dragend="stopDrag"
                class="session-item-card cursor-pointer rounded-[1.5rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(126,200,227,0.05)] p-4 transition-all duration-200 hover:border-[rgba(126,200,227,0.28)]"
                @click="openItemDetails(item)"
              >
                <div class="flex flex-wrap items-start justify-between gap-4">
                  <div class="min-w-0 flex-1">
                    <div class="flex flex-wrap items-center gap-2">
                      <p class="truncate text-[15px] font-semibold text-[#f6f7fb]">{{ item.name }}</p>
                      <span v-if="item.source === 'local'" class="rounded-full border border-[rgba(233,69,96,0.16)] px-2.5 py-1 text-[10px] uppercase tracking-[0.14em] text-[#ffb3c1]">
                        Local Draft
                      </span>
                    </div>
                    <p class="mt-2 text-[13px] leading-relaxed text-[#d8dce7]/60">{{ item.description || 'No description provided yet.' }}</p>
                  </div>

                  <div class="flex flex-wrap items-center gap-2">
                    <span class="rounded-full border px-2.5 py-1 text-[10px] font-semibold uppercase tracking-[0.14em]" :class="rarityBadgeClass(item.rarity)">
                      {{ item.rarity }}
                    </span>
                    <span class="rounded-full border border-[rgba(126,200,227,0.14)] px-2.5 py-1 text-[10px] uppercase tracking-[0.14em] text-[#7ec8e3]/60">
                      {{ item.grid_width }}x{{ item.grid_height }}
                    </span>
                  </div>
                </div>

                <div class="mt-3 flex flex-wrap gap-2 text-[11px] text-[#d8dce7]/55">
                  <span class="rounded-full border border-[rgba(126,200,227,0.14)] px-2.5 py-1">{{ folderPathLabel(itemLocations[item.id]) }}</span>
                  <span v-for="type in item.types" :key="`${item.id}-${type}`" class="rounded-full border border-[rgba(126,200,227,0.14)] px-2.5 py-1">
                    {{ type }}
                  </span>
                  <span v-if="item.is_equippable" class="rounded-full border border-[rgba(233,69,96,0.16)] px-2.5 py-1">
                    {{ item.equip_slot || 'equippable' }}
                  </span>
                </div>
              </article>
            </div>
          </div>

          <p v-else class="mt-4 text-[14px] text-[#d8dce7]/56">
            {{ searchNeedle ? 'No items match the current search.' : 'There are no items in this folder yet.' }}
          </p>
        </div>
      </div>
    </div>

    <Teleport to="body">
      <div v-if="showCreateFolderModal" class="session-item-modal fixed inset-0 z-[11000] flex items-center justify-center px-4" @click.self="showCreateFolderModal = false">
        <div class="absolute inset-0 bg-black/70 backdrop-blur-sm"></div>
        <div class="session-item-modal__card relative w-full max-w-[460px] rounded-[1.8rem] border border-[rgba(126,200,227,0.14)] bg-[#081220] p-6 shadow-[0_24px_80px_rgba(0,0,0,0.45)]">
          <button @click="showCreateFolderModal = false" class="absolute right-4 top-4 cursor-pointer border-none bg-transparent text-[24px] leading-none text-[#7ec8e3]/45 transition-colors duration-200 hover:text-[#f6f7fb]">&times;</button>
          <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">New Folder</p>
          <h3 class="mt-2 font-[Cinzel] text-[24px] font-bold text-[#f6f7fb]">Create Folder</h3>
          <p class="mt-2 text-[14px] text-[#d8dce7]/60">Parent: {{ folderPathLabel(folderParentDraft) }}</p>

          <label class="mt-5 block text-[11px] uppercase tracking-[0.2em] text-[#7ec8e3]/45">Folder Name</label>
          <input
            v-model="folderNameDraft"
            type="text"
            placeholder="Bestiary, Loot Drops, Boss Gear"
            class="mt-2 w-full rounded-2xl border border-[rgba(126,200,227,0.14)] bg-[rgba(11,20,36,0.94)] px-4 py-3 text-[14px] text-[#f6f7fb] outline-none transition-all duration-200 placeholder:text-[#7ec8e3]/30 focus:border-[rgba(233,69,96,0.34)]"
          />

          <div class="mt-6 flex justify-end gap-3">
            <button @click="showCreateFolderModal = false" class="cursor-pointer rounded-xl border border-[rgba(126,200,227,0.14)] bg-transparent px-4 py-2.5 text-[13px] font-semibold text-[#d8dce7]/70 transition-all duration-200 hover:border-[rgba(126,200,227,0.3)] hover:text-[#f6f7fb]">Cancel</button>
            <button @click="createFolder" :disabled="!folderNameDraft.trim()" class="cursor-pointer rounded-xl border border-[rgba(233,69,96,0.24)] bg-[rgba(233,69,96,0.12)] px-4 py-2.5 text-[13px] font-semibold text-[#ffe0e7] transition-all duration-200 hover:border-[rgba(233,69,96,0.42)] disabled:cursor-not-allowed disabled:opacity-60">Create</button>
          </div>
        </div>
      </div>
    </Teleport>

    <Teleport to="body">
      <div v-if="showCreateItemModal" class="session-item-modal fixed inset-0 z-[11000] flex items-center justify-center px-4 py-8" @click.self="showCreateItemModal = false">
        <div class="absolute inset-0 bg-black/70 backdrop-blur-sm"></div>
        <div class="session-item-modal__card session-item-scroll relative w-full max-w-[760px] overflow-y-auto rounded-[1.8rem] border border-[rgba(126,200,227,0.14)] bg-[#081220] p-6 shadow-[0_24px_80px_rgba(0,0,0,0.45)]">
          <button @click="showCreateItemModal = false" class="absolute right-4 top-4 cursor-pointer border-none bg-transparent text-[24px] leading-none text-[#7ec8e3]/45 transition-colors duration-200 hover:text-[#f6f7fb]">&times;</button>
          <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Create Item</p>
          <h3 class="mt-2 font-[Cinzel] text-[24px] font-bold text-[#f6f7fb]">New Library Entry</h3>
          <p class="mt-2 text-[14px] text-[#d8dce7]/60">The new item will be placed in {{ folderPathLabel(currentFolderId) }}.</p>

          <div class="mt-6 grid gap-4 md:grid-cols-2">
            <div class="md:col-span-2">
              <label class="block text-[11px] uppercase tracking-[0.2em] text-[#7ec8e3]/45">Name</label>
              <input v-model="itemDraft.name" type="text" class="mt-2 w-full rounded-2xl border border-[rgba(126,200,227,0.14)] bg-[rgba(11,20,36,0.94)] px-4 py-3 text-[14px] text-[#f6f7fb] outline-none transition-all duration-200 focus:border-[rgba(233,69,96,0.34)]" />
            </div>

            <div class="md:col-span-2">
              <label class="block text-[11px] uppercase tracking-[0.2em] text-[#7ec8e3]/45">Description</label>
              <textarea v-model="itemDraft.description" rows="4" class="mt-2 w-full rounded-2xl border border-[rgba(126,200,227,0.14)] bg-[rgba(11,20,36,0.94)] px-4 py-3 text-[14px] text-[#f6f7fb] outline-none transition-all duration-200 focus:border-[rgba(233,69,96,0.34)]"></textarea>
            </div>

            <div>
              <label class="block text-[11px] uppercase tracking-[0.2em] text-[#7ec8e3]/45">Rarity</label>
              <select v-model="itemDraft.rarity" class="mt-2 w-full rounded-2xl border border-[rgba(126,200,227,0.14)] bg-[rgba(11,20,36,0.94)] px-4 py-3 text-[14px] text-[#f6f7fb] outline-none transition-all duration-200 focus:border-[rgba(233,69,96,0.34)]">
                <option v-for="option in RARITY_OPTIONS" :key="option" :value="option">{{ option }}</option>
              </select>
            </div>

            <div>
              <label class="block text-[11px] uppercase tracking-[0.2em] text-[#7ec8e3]/45">Types</label>
              <input v-model="itemDraft.typesText" type="text" placeholder="weapon, consumable, relic" class="mt-2 w-full rounded-2xl border border-[rgba(126,200,227,0.14)] bg-[rgba(11,20,36,0.94)] px-4 py-3 text-[14px] text-[#f6f7fb] outline-none transition-all duration-200 placeholder:text-[#7ec8e3]/30 focus:border-[rgba(233,69,96,0.34)]" />
            </div>

            <div>
              <label class="block text-[11px] uppercase tracking-[0.2em] text-[#7ec8e3]/45">Grid Width</label>
              <input v-model="itemDraft.gridWidth" type="number" min="1" class="mt-2 w-full rounded-2xl border border-[rgba(126,200,227,0.14)] bg-[rgba(11,20,36,0.94)] px-4 py-3 text-[14px] text-[#f6f7fb] outline-none transition-all duration-200 focus:border-[rgba(233,69,96,0.34)]" />
            </div>

            <div>
              <label class="block text-[11px] uppercase tracking-[0.2em] text-[#7ec8e3]/45">Grid Height</label>
              <input v-model="itemDraft.gridHeight" type="number" min="1" class="mt-2 w-full rounded-2xl border border-[rgba(126,200,227,0.14)] bg-[rgba(11,20,36,0.94)] px-4 py-3 text-[14px] text-[#f6f7fb] outline-none transition-all duration-200 focus:border-[rgba(233,69,96,0.34)]" />
            </div>

            <label class="flex items-center gap-3 rounded-2xl border border-[rgba(126,200,227,0.14)] bg-[rgba(126,200,227,0.05)] px-4 py-3 text-[14px] text-[#f6f7fb]">
              <input v-model="itemDraft.equippable" type="checkbox" class="h-4 w-4 accent-[#e94560]" />
              Equippable item
            </label>

            <div>
              <label class="block text-[11px] uppercase tracking-[0.2em] text-[#7ec8e3]/45">Equip Slot</label>
              <select v-model="itemDraft.equipSlot" :disabled="!itemDraft.equippable" class="mt-2 w-full rounded-2xl border border-[rgba(126,200,227,0.14)] bg-[rgba(11,20,36,0.94)] px-4 py-3 text-[14px] text-[#f6f7fb] outline-none transition-all duration-200 focus:border-[rgba(233,69,96,0.34)] disabled:cursor-not-allowed disabled:opacity-60">
                <option value="">Select slot</option>
                <option v-for="slot in EQUIP_SLOT_OPTIONS" :key="slot" :value="slot">{{ slot }}</option>
              </select>
            </div>

            <div>
              <label class="block text-[11px] uppercase tracking-[0.2em] text-[#7ec8e3]/45">Requirements</label>
              <textarea v-model="itemDraft.requirementsText" rows="4" placeholder="strength:12&#10;dexterity:10" class="mt-2 w-full rounded-2xl border border-[rgba(126,200,227,0.14)] bg-[rgba(11,20,36,0.94)] px-4 py-3 text-[14px] text-[#f6f7fb] outline-none transition-all duration-200 placeholder:text-[#7ec8e3]/30 focus:border-[rgba(233,69,96,0.34)]"></textarea>
            </div>

            <div>
              <label class="block text-[11px] uppercase tracking-[0.2em] text-[#7ec8e3]/45">Modifiers</label>
              <textarea v-model="itemDraft.modifiersText" rows="4" placeholder="strength:2&#10;damage:5%" class="mt-2 w-full rounded-2xl border border-[rgba(126,200,227,0.14)] bg-[rgba(11,20,36,0.94)] px-4 py-3 text-[14px] text-[#f6f7fb] outline-none transition-all duration-200 placeholder:text-[#7ec8e3]/30 focus:border-[rgba(233,69,96,0.34)]"></textarea>
            </div>
          </div>

          <div class="mt-6 flex justify-end gap-3">
            <button @click="showCreateItemModal = false" class="cursor-pointer rounded-xl border border-[rgba(126,200,227,0.14)] bg-transparent px-4 py-2.5 text-[13px] font-semibold text-[#d8dce7]/70 transition-all duration-200 hover:border-[rgba(126,200,227,0.3)] hover:text-[#f6f7fb]">Cancel</button>
            <button @click="createItem" :disabled="!itemDraft.name.trim()" class="cursor-pointer rounded-xl border border-[rgba(233,69,96,0.24)] bg-[rgba(233,69,96,0.12)] px-4 py-2.5 text-[13px] font-semibold text-[#ffe0e7] transition-all duration-200 hover:border-[rgba(233,69,96,0.42)] disabled:cursor-not-allowed disabled:opacity-60">Save Item</button>
          </div>
        </div>
      </div>
    </Teleport>

    <Teleport to="body">
      <div v-if="selectedItem" class="session-item-modal fixed inset-0 z-[11000] flex items-center justify-center px-4 py-8" @click.self="selectedItem = null">
        <div class="absolute inset-0 bg-black/70 backdrop-blur-sm"></div>
        <div class="session-item-modal__card session-item-scroll relative w-full max-w-[760px] overflow-y-auto rounded-[1.8rem] border border-[rgba(126,200,227,0.14)] bg-[#081220] p-6 shadow-[0_24px_80px_rgba(0,0,0,0.45)]">
          <button @click="selectedItem = null" class="absolute right-4 top-4 cursor-pointer border-none bg-transparent text-[24px] leading-none text-[#7ec8e3]/45 transition-colors duration-200 hover:text-[#f6f7fb]">&times;</button>

          <div class="flex flex-wrap items-start justify-between gap-4">
            <div>
              <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Item Details</p>
              <h3 class="mt-2 font-[Cinzel] text-[28px] font-bold text-[#f6f7fb]">{{ selectedItem.name }}</h3>
              <p class="mt-2 text-[14px] text-[#d8dce7]/62">{{ selectedItem.description || 'No item description provided.' }}</p>
            </div>

            <div class="flex flex-wrap gap-2">
              <span class="rounded-full border px-2.5 py-1 text-[10px] font-semibold uppercase tracking-[0.14em]" :class="rarityBadgeClass(selectedItem.rarity)">
                {{ selectedItem.rarity }}
              </span>
              <span v-if="selectedItem.source === 'local'" class="rounded-full border border-[rgba(233,69,96,0.16)] px-2.5 py-1 text-[10px] uppercase tracking-[0.14em] text-[#ffb3c1]">
                Local Draft
              </span>
            </div>
          </div>

          <div class="mt-6 grid gap-4 md:grid-cols-2">
            <div class="rounded-[1.5rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(126,200,227,0.05)] p-4">
              <p class="text-[11px] uppercase tracking-[0.2em] text-[#7ec8e3]/45">Library Location</p>
              <p class="mt-2 text-[15px] font-semibold text-[#f6f7fb]">{{ folderPathLabel(itemLocations[selectedItem.id]) }}</p>
            </div>

            <div class="rounded-[1.5rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(126,200,227,0.05)] p-4">
              <p class="text-[11px] uppercase tracking-[0.2em] text-[#7ec8e3]/45">Footprint</p>
              <p class="mt-2 text-[15px] font-semibold text-[#f6f7fb]">{{ selectedItem.grid_width }}x{{ selectedItem.grid_height }}</p>
            </div>

            <div class="rounded-[1.5rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(126,200,227,0.05)] p-4">
              <p class="text-[11px] uppercase tracking-[0.2em] text-[#7ec8e3]/45">Types</p>
              <div class="mt-3 flex flex-wrap gap-2 text-[11px] text-[#d8dce7]/60">
                <span v-for="type in selectedItem.types" :key="`${selectedItem.id}-${type}`" class="rounded-full border border-[rgba(126,200,227,0.14)] px-2.5 py-1">{{ type }}</span>
                <span v-if="!selectedItem.types?.length" class="rounded-full border border-[rgba(126,200,227,0.14)] px-2.5 py-1">No tags</span>
              </div>
            </div>

            <div class="rounded-[1.5rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(126,200,227,0.05)] p-4">
              <p class="text-[11px] uppercase tracking-[0.2em] text-[#7ec8e3]/45">Equip State</p>
              <p class="mt-2 text-[15px] font-semibold text-[#f6f7fb]">{{ selectedItem.is_equippable ? selectedItem.equip_slot || 'Equippable' : 'Not equippable' }}</p>
            </div>
          </div>

          <div class="mt-6 grid gap-4 md:grid-cols-2">
            <div class="rounded-[1.5rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(126,200,227,0.05)] p-4">
              <p class="text-[11px] uppercase tracking-[0.2em] text-[#7ec8e3]/45">Requirements</p>
              <div v-if="selectedItem.required_attributes?.length" class="mt-3 space-y-2 text-[13px] text-[#f6f7fb]">
                <div v-for="requirement in selectedItem.required_attributes" :key="`${selectedItem.id}-${requirement.attribute_name}`" class="rounded-xl border border-[rgba(126,200,227,0.14)] px-3 py-2">
                  {{ requirement.attribute_name }} >= {{ requirement.min_value }}
                </div>
              </div>
              <p v-else class="mt-3 text-[13px] text-[#d8dce7]/56">No requirements specified.</p>
            </div>

            <div class="rounded-[1.5rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(126,200,227,0.05)] p-4">
              <p class="text-[11px] uppercase tracking-[0.2em] text-[#7ec8e3]/45">Modifiers</p>
              <div v-if="selectedItem.attribute_modifiers?.length" class="mt-3 space-y-2 text-[13px] text-[#f6f7fb]">
                <div v-for="modifier in selectedItem.attribute_modifiers" :key="`${selectedItem.id}-${modifier.attribute_name}`" class="rounded-xl border border-[rgba(126,200,227,0.14)] px-3 py-2">
                  {{ modifier.attribute_name }} {{ modifier.modifier_value > 0 ? '+' : '' }}{{ modifier.modifier_value }}{{ modifier.is_percentage ? '%' : '' }}
                </div>
              </div>
              <p v-else class="mt-3 text-[13px] text-[#d8dce7]/56">No modifiers specified.</p>
            </div>
          </div>

          <div class="mt-6 flex flex-wrap items-center justify-between gap-3">
            <div class="text-[12px] text-[#d8dce7]/50">
              Updated {{ selectedItem.updated_at || selectedItem.created_at || 'just now' }}
            </div>

            <button
              v-if="selectedItem.source === 'local'"
              @click="deleteLocalItem(selectedItem.id)"
              class="cursor-pointer rounded-xl border border-[rgba(248,113,113,0.22)] bg-[rgba(127,29,29,0.18)] px-4 py-2.5 text-[13px] font-semibold text-[#fecaca] transition-all duration-200 hover:border-[rgba(248,113,113,0.4)]"
            >
              Delete Draft Item
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
  border-color: rgba(126, 200, 227, 0.14) !important;
  background: var(--dl-panel-bg) !important;
}

.session-item-manager::before,
.session-item-modal__card::before {
  content: '';
  position: absolute;
  inset: 0;
  pointer-events: none;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.03), transparent 24%),
    repeating-linear-gradient(135deg, rgba(255, 255, 255, 0.015) 0 2px, transparent 2px 18px);
  opacity: 0.7;
}

.session-item-manager::after,
.session-item-modal__card::after {
  content: '';
  position: absolute;
  inset: 9px;
  pointer-events: none;
  border: 1px solid rgba(126, 200, 227, 0.1);
  clip-path: polygon(0 14px, 14px 0, 100% 0, 100% calc(100% - 14px), calc(100% - 14px) 100%, 0 100%);
}

.session-item-manager > *,
.session-item-modal__card > * {
  position: relative;
  z-index: 1;
}

.session-item-manager__header {
  align-items: end;
}

.session-item-manager__layout,
.session-item-folder-grid,
.session-item-grid {
  align-items: stretch;
}

.session-item-tree,
.session-item-toolbar,
.session-item-folder-stage,
.session-item-catalog,
.session-item-modal__card {
  position: relative;
  overflow: hidden;
  border-color: rgba(126, 200, 227, 0.14) !important;
  background: var(--dl-panel-bg-soft) !important;
  box-shadow: var(--dl-shadow-soft);
}

.session-item-tree {
  min-height: 30rem;
}

.session-item-workbench {
  display: grid;
  gap: 1.25rem;
}

.session-item-dropzone {
  position: relative;
  overflow: hidden;
  min-height: 7.5rem;
  border-color: rgba(126, 200, 227, 0.18) !important;
  background:
    linear-gradient(90deg, rgba(126, 200, 227, 0.05) 1px, transparent 1px),
    linear-gradient(0deg, rgba(126, 200, 227, 0.05) 1px, transparent 1px),
    linear-gradient(180deg, rgba(11, 17, 31, 0.9), rgba(8, 12, 24, 0.94)) !important;
  background-size: 74px 74px, 74px 74px, auto;
}

.session-item-folder-stage,
.session-item-catalog {
  min-height: 18rem;
}

.session-item-card {
  display: flex;
  min-height: 13.5rem;
  flex-direction: column;
  justify-content: space-between;
  border-color: rgba(126, 200, 227, 0.14) !important;
  background: linear-gradient(180deg, rgba(18, 27, 49, 0.7), rgba(9, 14, 27, 0.84)) !important;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.03);
}

.session-item-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 18px 40px rgba(0, 0, 0, 0.22), inset 0 1px 0 rgba(255, 255, 255, 0.03);
}

.session-item-manager input,
.session-item-manager select,
.session-item-manager textarea,
.session-item-modal__card input,
.session-item-modal__card select,
.session-item-modal__card textarea {
  border-color: rgba(126, 200, 227, 0.14) !important;
  background: rgba(11, 17, 31, 0.88) !important;
  color: var(--dl-brand-text) !important;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.03);
}

.session-item-manager input:focus,
.session-item-manager select:focus,
.session-item-manager textarea:focus,
.session-item-modal__card input:focus,
.session-item-modal__card select:focus,
.session-item-modal__card textarea:focus {
  border-color: rgba(233, 69, 96, 0.34) !important;
  box-shadow: 0 0 0 1px rgba(233, 69, 96, 0.16), inset 0 1px 0 rgba(255, 255, 255, 0.03);
}

.session-item-modal__card {
  max-height: min(90vh, 58rem);
  border-color: rgba(126, 200, 227, 0.16) !important;
  background: linear-gradient(180deg, rgba(13, 18, 33, 0.98), rgba(8, 12, 24, 0.98)) !important;
}

.session-item-scroll::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

.session-item-scroll::-webkit-scrollbar-thumb {
  background: rgba(126, 200, 227, 0.22);
  border-radius: 999px;
}

.session-item-scroll::-webkit-scrollbar-track {
  background: transparent;
}

@media (min-width: 1280px) {
  .session-item-tree {
    position: sticky;
    top: 7.35rem;
    align-self: start;
    min-height: 42rem;
  }

  .session-item-catalog {
    min-height: 30rem;
  }
}

@media (min-width: 1536px) {
  .session-item-manager {
    padding: 1.75rem 1.75rem 2rem !important;
  }

  .session-item-grid {
    gap: 1rem;
  }
}
</style>