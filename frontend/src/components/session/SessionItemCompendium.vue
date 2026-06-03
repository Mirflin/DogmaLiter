<script setup>
import {
  Check,
  Eye,
  FilePlus2,
  Package,
  Pencil,
  Plus,
  Search,
  Send,
  ShoppingCart,
  Trash2,
  User,
  X,
} from '@lucide/vue'
import { API_URL } from '@/api'
import { getErrorMessage, notify } from '@/notify'
import { useAuthStore } from '@/stores/auth'
import { computed, onBeforeUnmount, ref, watch } from 'vue'

const emit = defineEmits(['created'])

const props = defineProps({
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

const ITEM_PAGE_SIZE = 18
const ITEM_NAME_LIMIT = 20
const ITEM_DESCRIPTION_LIMIT = 100
const ITEM_CARD_DESCRIPTION_PREVIEW_LIMIT = 100
const ITEM_DETAIL_DESCRIPTION_PREVIEW_LIMIT = 100
const MAX_ITEM_IMAGE_SIZE = 5 * 1024 * 1024
const ALLOWED_ITEM_IMAGE_TYPES = ['image/jpeg', 'image/png', 'image/webp']
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
const dateFormatter = new Intl.DateTimeFormat('en-US', {
  dateStyle: 'medium',
  timeStyle: 'short',
})

const searchQuery = ref('')
const debouncedSearchQuery = ref('')
const rarityFilter = ref('all')
const categoryFilter = ref('all')
const slotFilter = ref('all')
const tagFilter = ref('all')
const sortMode = ref('recent')
const currentPage = ref(1)
const itemsLoading = ref(false)
const itemsError = ref('')
const rawItems = ref([])
const pagination = ref({
  page: 1,
  perPage: ITEM_PAGE_SIZE,
  totalItems: 0,
  totalPages: 0,
  hasPrev: false,
  hasNext: false,
})
const selectedItemId = ref('')
const hoveredItemId = ref('')
const showCreateItemModal = ref(false)
const showItemDetailModal = ref(false)
const createItemSubmitting = ref(false)
const detailItemSubmitting = ref(false)
const detailDeleteSubmitting = ref(false)
const itemDetailMode = ref('view')
const detailItemId = ref('')
const confirmItemDeletion = ref(false)
const newTagDraft = ref('')
const itemDraft = ref(createEmptyItemDraft())
const detailNewTagDraft = ref('')
const detailItemDraft = ref(createEmptyItemDraft())
const itemImageInputRef = ref(null)
const detailItemImageInputRef = ref(null)
const itemImageFile = ref(null)
const detailItemImageFile = ref(null)
const itemImagePreviewUrl = ref('')
const detailItemImagePreviewUrl = ref('')
const itemContextMenu = ref(createClosedItemContextMenu())
const deleteCandidateId = ref('')
const deleteCandidateSubmitting = ref(false)
const cartItems = ref([])
const showCartModal = ref(false)
const cartCharacterSearch = ref('')
const cartTargetCharacterIds = ref([])
const cartSubmitting = ref(false)

let searchDebounceHandle = null
let itemListRequestId = 0

const allItems = computed(() => (rawItems.value ?? []).map(item => normalizeItem(item)))
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
const activeFilterBadges = computed(() => {
  const badges = []
  if (debouncedSearchQuery.value) {
    badges.push(`Query: ${debouncedSearchQuery.value}`)
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
  totalItems: pagination.value.totalItems,
  visibleItems: allItems.value.length,
  currentPage: pagination.value.page,
  totalPages: pagination.value.totalPages,
  totalTags: availableTagNames.value.length,
}))
const itemResultsLabel = computed(() => {
  if (!pagination.value.totalItems || !allItems.value.length) {
    return 'No items loaded yet.'
  }

  const start = ((pagination.value.page - 1) * pagination.value.perPage) + 1
  const end = Math.min(pagination.value.totalItems, start + allItems.value.length - 1)
  return `Showing ${start}-${end} of ${pagination.value.totalItems}`
})
const inspectorItem = computed(() => itemById.value[hoveredItemId.value]
  ?? itemById.value[selectedItemId.value]
  ?? allItems.value[0]
  ?? null)
const selectedTagSuggestions = computed(() => availableTagNames.value.filter(tag => !itemDraft.value.tags
  .some(selectedTag => selectedTag.toLowerCase() === tag.toLowerCase())))
const itemImageMetaLabel = computed(() => {
  if (!itemImageFile.value) {
    return 'No image selected yet.'
  }
  return `${itemImageFile.value.name} · ${formatFileSize(itemImageFile.value.size)}`
})
const detailItem = computed(() => itemById.value[detailItemId.value] ?? null)
const detailItemImageMetaLabel = computed(() => {
  if (detailItemImageFile.value) {
    return `${detailItemImageFile.value.name} · ${formatFileSize(detailItemImageFile.value.size)}`
  }
  if (detailItem.value?.image_id) {
    return 'Using the currently linked image.'
  }
  return 'No image selected yet.'
})
const detailTagSuggestions = computed(() => availableTagNames.value.filter(tag => !detailItemDraft.value.tags
  .some(selectedTag => selectedTag.toLowerCase() === tag.toLowerCase())))
const itemNameLength = computed(() => itemDraft.value.name.length)
const itemDescriptionLength = computed(() => itemDraft.value.description.length)
const detailFormDisabled = computed(() => itemDetailMode.value !== 'edit' || detailItemSubmitting.value || detailDeleteSubmitting.value)
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
const detailPreviewItem = computed(() => {
  if (itemDetailMode.value !== 'edit') {
    return detailItem.value
  }

  return normalizeItem({
    ...(detailItem.value ?? {}),
    id: 'detail-preview',
    name: detailItemDraft.value.name || detailItem.value?.name || 'Untitled Item',
    description: detailItemDraft.value.description,
    rarity: detailItemDraft.value.rarity,
    category: detailItemDraft.value.category,
    grid_width: normalizePositiveInteger(detailItemDraft.value.gridWidth, 1),
    grid_height: normalizePositiveInteger(detailItemDraft.value.gridHeight, 1),
    equip_slot: detailItemDraft.value.category === 'equipment' ? detailItemDraft.value.equipSlot : null,
    tags: detailItemDraft.value.tags,
    required_attributes: detailItemDraft.value.requiredAttributes,
    attribute_modifiers: detailItemDraft.value.attributeModifiers,
    updated_at: new Date().toISOString(),
  })
})
const contextMenuItem = computed(() => itemById.value[itemContextMenu.value.itemId] ?? null)
const deleteCandidateItem = computed(() => itemById.value[deleteCandidateId.value] ?? null)
const cartItemIds = computed(() => new Set(cartItems.value.map(line => line.item.id)))
const cartCount = computed(() => cartItems.value.length)
const cartCharacterOptions = computed(() => {
  const query = cartCharacterSearch.value.trim().toLowerCase()
  const list = (props.characters ?? []).filter(character => character?.id)
  if (!query) {
    return list
  }
  return list.filter(character => String(character.name || '').toLowerCase().includes(query))
})
const cartTargetCharacterIdSet = computed(() => new Set(cartTargetCharacterIds.value))
const cartTargetCharacters = computed(() => (props.characters ?? [])
  .filter(character => cartTargetCharacterIdSet.value.has(character?.id)))
const itemContextMenuStyle = computed(() => ({
  left: `${itemContextMenu.value.x}px`,
  top: `${itemContextMenu.value.y}px`,
}))

watch(searchQuery, (value) => {
  if (searchDebounceHandle) {
    clearTimeout(searchDebounceHandle)
  }

  searchDebounceHandle = setTimeout(() => {
    debouncedSearchQuery.value = value.trim()
  }, 220)
})

watch(
  () => [props.gameId, debouncedSearchQuery.value, rarityFilter.value, categoryFilter.value, slotFilter.value, tagFilter.value, sortMode.value],
  () => {
    if (currentPage.value !== 1) {
      currentPage.value = 1
      return
    }

    fetchItems()
  },
  { immediate: true },
)

watch(currentPage, () => {
  fetchItems()
})

watch(() => itemDraft.value.description, (value) => {
  const sanitizedValue = sanitizeDescriptionInput(value)
  if (sanitizedValue !== value) {
    itemDraft.value.description = sanitizedValue
  }
})

watch(() => detailItemDraft.value.description, (value) => {
  const sanitizedValue = sanitizeDescriptionInput(value)
  if (sanitizedValue !== value) {
    detailItemDraft.value.description = sanitizedValue
  }
})

watch(
  () => allItems.value.map(item => item.id),
  (visibleIds) => {
    if (visibleIds.includes(selectedItemId.value)) {
      return
    }

    selectedItemId.value = visibleIds[0] ?? ''
  },
  { immediate: true },
)

onBeforeUnmount(() => {
  if (searchDebounceHandle) {
    clearTimeout(searchDebounceHandle)
  }
  clearItemImageSelection()
  clearDetailItemImageSelection()
})

async function fetchItems() {
  if (!props.gameId) {
    rawItems.value = []
    pagination.value = createEmptyPaginationState()
    itemsError.value = ''
    return
  }

  const requestId = ++itemListRequestId
  itemsLoading.value = true
  itemsError.value = ''

  try {
    const data = await auth.getGameItems(props.gameId, {
      page: currentPage.value,
      per_page: ITEM_PAGE_SIZE,
      search: debouncedSearchQuery.value || undefined,
      rarity: rarityFilter.value !== 'all' ? rarityFilter.value : undefined,
      category: categoryFilter.value !== 'all' ? categoryFilter.value : undefined,
      slot: slotFilter.value !== 'all' ? slotFilter.value : undefined,
      tag: tagFilter.value !== 'all' ? tagFilter.value : undefined,
      sort: sortMode.value,
    })

    if (requestId !== itemListRequestId) {
      return
    }

    rawItems.value = Array.isArray(data?.items) ? data.items : []
    pagination.value = normalizePagination(data?.pagination)
  } catch (error) {
    if (requestId !== itemListRequestId) {
      return
    }

    rawItems.value = []
    pagination.value = createEmptyPaginationState()
    itemsError.value = getErrorMessage(error, 'Failed to load compendium items')
  } finally {
    if (requestId === itemListRequestId) {
      itemsLoading.value = false
    }
  }
}

function createEmptyPaginationState() {
  return {
    page: 1,
    perPage: ITEM_PAGE_SIZE,
    totalItems: 0,
    totalPages: 0,
    hasPrev: false,
    hasNext: false,
  }
}

function normalizePagination(value) {
  return {
    page: normalizePositiveInteger(value?.page, currentPage.value),
    perPage: normalizePositiveInteger(value?.per_page, ITEM_PAGE_SIZE),
    totalItems: Math.max(0, Number.parseInt(value?.total_items, 10) || 0),
    totalPages: Math.max(0, Number.parseInt(value?.total_pages, 10) || 0),
    hasPrev: Boolean(value?.has_prev),
    hasNext: Boolean(value?.has_next),
  }
}

function normalizeItem(item) {
  const rarity = RARITY_OPTIONS.includes(String(item?.rarity || '').toLowerCase())
    ? String(item.rarity).toLowerCase()
    : 'common'
  const category = CATEGORY_OPTIONS.includes(String(item?.category || '').toLowerCase())
    ? String(item.category).toLowerCase()
    : 'other'
  const equipSlot = normalizeEquipSlot(item?.equip_slot)
  const tags = uniqueLabels(Array.isArray(item?.tags) ? item.tags : [])
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
    id: String(item?.id || 'item-preview'),
    name: String(item?.name || 'Untitled Item').trim() || 'Untitled Item',
    description: String(item?.description || '').trim(),
    image_id: item?.image_id ? String(item.image_id) : '',
    rarity,
    category,
    equip_slot: category === 'equipment' ? equipSlot : '',
    grid_width: normalizePositiveInteger(item?.grid_width, 1),
    grid_height: normalizePositiveInteger(item?.grid_height, 1),
    tags,
    required_attributes: requiredAttributes,
    attribute_modifiers: attributeModifiers,
    created_at: item?.created_at || '',
    updated_at: item?.updated_at || item?.created_at || '',
  }
}

function clearFilters() {
  searchQuery.value = ''
  debouncedSearchQuery.value = ''
  rarityFilter.value = 'all'
  categoryFilter.value = 'all'
  slotFilter.value = 'all'
  tagFilter.value = 'all'
  sortMode.value = 'recent'
}

function changePage(nextPage) {
  if (itemsLoading.value) {
    return
  }
  if (nextPage < 1 || nextPage > Math.max(1, pagination.value.totalPages)) {
    return
  }
  if (nextPage === currentPage.value) {
    return
  }
  currentPage.value = nextPage
}

function selectItem(itemId) {
  selectedItemId.value = itemId
}

function createClosedItemContextMenu() {
  return {
    visible: false,
    itemId: '',
    x: 0,
    y: 0,
  }
}

function openItemContextMenu(event, item) {
  selectItem(item.id)

  const menuWidth = 220
  const menuHeight = 176
  const viewportWidth = window.innerWidth
  const viewportHeight = window.innerHeight

  itemContextMenu.value = {
    visible: true,
    itemId: item.id,
    x: Math.max(12, Math.min(event.clientX, viewportWidth - menuWidth - 12)),
    y: Math.max(12, Math.min(event.clientY, viewportHeight - menuHeight - 12)),
  }
}

function closeItemContextMenu() {
  itemContextMenu.value = createClosedItemContextMenu()
}

function requestItemDeletion(item) {
  closeItemContextMenu()
  selectItem(item.id)
  deleteCandidateId.value = item.id
}

function cancelItemDeletion() {
  if (deleteCandidateSubmitting.value) {
    return
  }
  deleteCandidateId.value = ''
}

async function confirmCandidateDeletion() {
  if (deleteCandidateSubmitting.value || !props.gameId || !deleteCandidateId.value) {
    return
  }

  const deletedItemId = deleteCandidateId.value
  const deletedItemName = deleteCandidateItem.value?.name || 'Item'
  deleteCandidateSubmitting.value = true

  try {
    await auth.deleteGameItem(props.gameId, deletedItemId)
    deleteCandidateId.value = ''

    if (detailItemId.value === deletedItemId) {
      closeItemDetailModal(true)
    }

    if (currentPage.value > 1 && allItems.value.length === 1) {
      currentPage.value -= 1
    } else {
      await fetchItems()
    }

    emit('created')
    notify.success({
      title: 'Item deleted',
      message: `${deletedItemName} was removed from the compendium and every character inventory.`,
    })
  } catch (error) {
    notify.error({
      title: 'Failed to delete item',
      message: getErrorMessage(error, 'Failed to delete item'),
    })
  } finally {
    deleteCandidateSubmitting.value = false
  }
}

function isInCart(item) {
  return cartItemIds.value.has(item.id)
}

function createCartLine(item) {
  return {
    item,
    quantity: 1,
    durability: 100,
    maxDurability: 100,
    hasEnchantment: false,
    enchantment: 0,
  }
}

function toggleCartCharacter(characterId) {
  if (cartTargetCharacterIds.value.includes(characterId)) {
    cartTargetCharacterIds.value = cartTargetCharacterIds.value.filter(id => id !== characterId)
    return
  }
  cartTargetCharacterIds.value = [...cartTargetCharacterIds.value, characterId]
}

function toggleCartItem(item) {
  const normalized = normalizeItem(item)
  const index = cartItems.value.findIndex(line => line.item.id === normalized.id)
  if (index >= 0) {
    cartItems.value.splice(index, 1)
    return
  }
  cartItems.value.push(createCartLine(normalized))
}

function removeCartLine(itemId) {
  cartItems.value = cartItems.value.filter(line => line.item.id !== itemId)
}

function clearCart() {
  cartItems.value = []
}

function openCartModal() {
  if (!cartItems.value.length) {
    notify.warning({ title: 'Cart is empty', message: 'Select items from the compendium first.' })
    return
  }
  closeItemContextMenu()
  cartCharacterSearch.value = ''
  cartTargetCharacterIds.value = []
  showCartModal.value = true
}

function closeCartModal(force = false) {
  if (cartSubmitting.value && !force) {
    return
  }
  showCartModal.value = false
}

function clampInt(value, min, max, fallback) {
  const parsed = Number.parseInt(value, 10)
  if (Number.isNaN(parsed)) {
    return fallback
  }
  return Math.min(max, Math.max(min, parsed))
}

async function deliverCart() {
  if (cartSubmitting.value) {
    return
  }
  if (!cartItems.value.length) {
    notify.warning({ title: 'Cart is empty', message: 'Add at least one item before delivering.' })
    return
  }
  if (!cartTargetCharacterIds.value.length) {
    notify.warning({ title: 'Choose a character', message: 'Select at least one character to receive these items.' })
    return
  }

  const payload = cartItems.value.map((line) => {
    const maxDurability = clampInt(line.maxDurability, 1, 1000000, 100)
    const entry = {
      item_id: line.item.id,
      quantity: clampInt(line.quantity, 1, 9999, 1),
      max_durability: maxDurability,
      durability: clampInt(line.durability, 0, maxDurability, maxDurability),
    }
    if (line.hasEnchantment) {
      entry.enchantment = clampInt(line.enchantment, -999, 999, 0)
    }
    return entry
  })

  const targets = cartTargetCharacters.value
  const succeeded = []
  const succeededIds = []
  const failed = []
  cartSubmitting.value = true

  try {
    for (const character of targets) {
      try {
        await auth.giveItemsToCharacter(props.gameId, character.id, payload)
        succeeded.push(character.name || 'Unnamed')
        succeededIds.push(character.id)
      } catch (error) {
        failed.push(`${character.name || 'Unnamed'}: ${getErrorMessage(error, 'delivery failed')}`)
      }
    }
  } finally {
    cartSubmitting.value = false
  }

  if (succeeded.length) {
    emit('created')
    notify.success({
      title: 'Items delivered',
      message: `${payload.length} item(s) delivered to ${succeeded.join(', ')}.`,
    })
  }

  if (failed.length) {
    cartTargetCharacterIds.value = cartTargetCharacterIds.value.filter(id => !succeededIds.includes(id))
    notify.error({
      title: 'Some deliveries failed',
      message: failed.join(' · '),
    })
    return
  }

  clearCart()
  closeCartModal(true)
}

function openItemDetailModal(item, mode = 'view') {
  closeItemContextMenu()
  selectItem(item.id)
  detailItemId.value = item.id
  detailItemDraft.value = createItemDraftFromItem(item)
  detailNewTagDraft.value = ''
  confirmItemDeletion.value = false
  clearDetailItemImageSelection()
  itemDetailMode.value = mode
  showItemDetailModal.value = true
}

function closeItemDetailModal(force = false) {
  if ((detailItemSubmitting.value || detailDeleteSubmitting.value) && !force) {
    return
  }

  showItemDetailModal.value = false
  itemDetailMode.value = 'view'
  detailItemId.value = ''
  detailItemDraft.value = createEmptyItemDraft()
  detailNewTagDraft.value = ''
  confirmItemDeletion.value = false
  clearDetailItemImageSelection()
}

function beginDetailItemEdit() {
  if (!detailItem.value) {
    return
  }

  detailItemDraft.value = createItemDraftFromItem(detailItem.value)
  detailNewTagDraft.value = ''
  confirmItemDeletion.value = false
  clearDetailItemImageSelection()
  itemDetailMode.value = 'edit'
}

function cancelDetailItemEdit() {
  if (detailItemSubmitting.value) {
    return
  }

  if (detailItem.value) {
    detailItemDraft.value = createItemDraftFromItem(detailItem.value)
  }
  detailNewTagDraft.value = ''
  confirmItemDeletion.value = false
  clearDetailItemImageSelection()
  itemDetailMode.value = 'view'
}

function openCreateItemModal() {
  closeItemContextMenu()
  itemDraft.value = createEmptyItemDraft()
  newTagDraft.value = ''
  clearItemImageSelection()
  showCreateItemModal.value = true
}

function closeCreateItemModal(force = false) {
  if (createItemSubmitting.value && !force) {
    return
  }

  showCreateItemModal.value = false
  newTagDraft.value = ''
  clearItemImageSelection()
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

function addDetailRequirementRow() {
  detailItemDraft.value.requiredAttributes.push(createEmptyRequirementRow(defaultAttributeName()))
}

function removeDetailRequirementRow(index) {
  detailItemDraft.value.requiredAttributes.splice(index, 1)
}

function addDetailModifierRow() {
  detailItemDraft.value.attributeModifiers.push(createEmptyModifierRow(defaultAttributeName()))
}

function removeDetailModifierRow(index) {
  detailItemDraft.value.attributeModifiers.splice(index, 1)
}

function toggleDraftTag(tag) {
  const lookupKey = tag.toLowerCase()
  if (itemDraft.value.tags.some(entry => entry.toLowerCase() === lookupKey)) {
    itemDraft.value.tags = itemDraft.value.tags.filter(entry => entry.toLowerCase() !== lookupKey)
    return
  }

  if (itemDraft.value.tags.length >= 20) {
    notifyCreateItemValidation('Each item can have up to 20 tags.')
    return
  }

  itemDraft.value.tags = [...itemDraft.value.tags, tag].sort((left, right) => left.localeCompare(right))
}

function removeDraftTag(tag) {
  itemDraft.value.tags = itemDraft.value.tags.filter(entry => entry.toLowerCase() !== tag.toLowerCase())
}

function toggleDetailDraftTag(tag) {
  const lookupKey = tag.toLowerCase()
  if (detailItemDraft.value.tags.some(entry => entry.toLowerCase() === lookupKey)) {
    detailItemDraft.value.tags = detailItemDraft.value.tags.filter(entry => entry.toLowerCase() !== lookupKey)
    return
  }

  if (detailItemDraft.value.tags.length >= 20) {
    notifyCreateItemValidation('Each item can have up to 20 tags.')
    return
  }

  detailItemDraft.value.tags = [...detailItemDraft.value.tags, tag].sort((left, right) => left.localeCompare(right))
}

function removeDetailDraftTag(tag) {
  detailItemDraft.value.tags = detailItemDraft.value.tags.filter(entry => entry.toLowerCase() !== tag.toLowerCase())
}

function addDraftTagFromInput() {
  const normalized = normalizeTagName(newTagDraft.value)
  if (!normalized) {
    newTagDraft.value = ''
    return
  }
  if (normalized.length > 60) {
    notifyCreateItemValidation('Tag names must be 60 characters or fewer.')
    return
  }
  if (itemDraft.value.tags.length >= 20) {
    notifyCreateItemValidation('Each item can have up to 20 tags.')
    return
  }

  if (!itemDraft.value.tags.some(entry => entry.toLowerCase() === normalized.toLowerCase())) {
    itemDraft.value.tags = [...itemDraft.value.tags, normalized].sort((left, right) => left.localeCompare(right))
  }
  newTagDraft.value = ''
}

function addDetailDraftTagFromInput() {
  const normalized = normalizeTagName(detailNewTagDraft.value)
  if (!normalized) {
    detailNewTagDraft.value = ''
    return
  }
  if (normalized.length > 60) {
    notifyCreateItemValidation('Tag names must be 60 characters or fewer.')
    return
  }
  if (detailItemDraft.value.tags.length >= 20) {
    notifyCreateItemValidation('Each item can have up to 20 tags.')
    return
  }

  if (!detailItemDraft.value.tags.some(entry => entry.toLowerCase() === normalized.toLowerCase())) {
    detailItemDraft.value.tags = [...detailItemDraft.value.tags, normalized].sort((left, right) => left.localeCompare(right))
  }
  detailNewTagDraft.value = ''
}

function openItemImagePicker() {
  itemImageInputRef.value?.click()
}

function openDetailItemImagePicker() {
  detailItemImageInputRef.value?.click()
}

function validateItemImageFile(file) {
  if (!ALLOWED_ITEM_IMAGE_TYPES.includes(file.type)) {
    return 'Only JPEG, PNG, and WebP images are allowed.'
  }
  if (file.size > MAX_ITEM_IMAGE_SIZE) {
    return 'Item image must be under 5MB.'
  }

  return ''
}

function handleItemImageSelected(event) {
  const file = event.target?.files?.[0]
  if (!file) {
    return
  }

  const validationMessage = validateItemImageFile(file)
  if (validationMessage) {
    notify.error({
      title: 'Item image problem',
      message: validationMessage,
    })
    clearItemImageSelection()
    return
  }

  if (itemImagePreviewUrl.value) {
    URL.revokeObjectURL(itemImagePreviewUrl.value)
  }

  itemImageFile.value = file
  itemImagePreviewUrl.value = URL.createObjectURL(file)
}

function handleDetailItemImageSelected(event) {
  const file = event.target?.files?.[0]
  if (!file) {
    return
  }

  const validationMessage = validateItemImageFile(file)
  if (validationMessage) {
    notify.error({
      title: 'Item image problem',
      message: validationMessage,
    })
    clearDetailItemImageSelection()
    return
  }

  if (detailItemImagePreviewUrl.value) {
    URL.revokeObjectURL(detailItemImagePreviewUrl.value)
  }

  detailItemImageFile.value = file
  detailItemImagePreviewUrl.value = URL.createObjectURL(file)
}

function clearItemImageSelection() {
  if (itemImagePreviewUrl.value) {
    URL.revokeObjectURL(itemImagePreviewUrl.value)
  }
  itemImagePreviewUrl.value = ''
  itemImageFile.value = null

  if (itemImageInputRef.value) {
    itemImageInputRef.value.value = ''
  }
}

function clearDetailItemImageSelection() {
  if (detailItemImagePreviewUrl.value) {
    URL.revokeObjectURL(detailItemImagePreviewUrl.value)
  }
  detailItemImagePreviewUrl.value = ''
  detailItemImageFile.value = null

  if (detailItemImageInputRef.value) {
    detailItemImageInputRef.value.value = ''
  }
}

async function createItem() {
  if (createItemSubmitting.value) {
    return
  }

  const payload = buildItemPayload(itemDraft.value)
  const validationMessage = validateItemPayload(payload)
  if (validationMessage) {
    notifyCreateItemValidation(validationMessage)
    return
  }

  createItemSubmitting.value = true

  try {
    const response = await auth.createGameItem(props.gameId, payload)
    const createdItem = response?.item ?? null
    let imageUploadWarning = ''

    if (itemImageFile.value && createdItem?.id) {
      try {
        await auth.uploadGameItemImage(props.gameId, createdItem.id, itemImageFile.value)
      } catch (error) {
        imageUploadWarning = getErrorMessage(error, 'Item image upload failed')
        notify.warning({
          title: 'Item created without image',
          message: imageUploadWarning,
        })
      }
    }

    if (currentPage.value !== 1) {
      currentPage.value = 1
    } else {
      await fetchItems()
    }

    closeCreateItemModal(true)
    emit('created')
    notify.success({
      title: 'Item created',
      message: imageUploadWarning
        ? `${payload.name} was added to the compendium. Upload the image again if needed.`
        : `${payload.name} was added to the compendium.`,
    })
  } catch (error) {
    notify.error({
      title: 'Failed to create item',
      message: getErrorMessage(error, 'Failed to create item'),
    })
  } finally {
    createItemSubmitting.value = false
  }
}

async function saveDetailItem() {
  if (detailItemSubmitting.value || !props.gameId || !detailItemId.value) {
    return
  }

  const payload = buildItemPayload(detailItemDraft.value)
  const validationMessage = validateItemPayload(payload)
  if (validationMessage) {
    notifyCreateItemValidation(validationMessage)
    return
  }

  detailItemSubmitting.value = true

  try {
    await auth.updateGameItem(props.gameId, detailItemId.value, payload)
    let imageUploadWarning = ''

    if (detailItemImageFile.value) {
      try {
        await auth.uploadGameItemImage(props.gameId, detailItemId.value, detailItemImageFile.value)
      } catch (error) {
        imageUploadWarning = getErrorMessage(error, 'Item image upload failed')
        notify.warning({
          title: 'Item updated without new image',
          message: imageUploadWarning,
        })
      }
    }

    await fetchItems()
    emit('created')

    const refreshedItem = itemById.value[detailItemId.value]
    if (refreshedItem) {
      detailItemDraft.value = createItemDraftFromItem(refreshedItem)
    }
    clearDetailItemImageSelection()
    detailNewTagDraft.value = ''
    confirmItemDeletion.value = false
    itemDetailMode.value = 'view'

    notify.success({
      title: 'Item updated',
      message: imageUploadWarning
        ? `${payload.name} was updated. Upload the image again if needed.`
        : `${payload.name} was updated.`,
    })
  } catch (error) {
    notify.error({
      title: 'Failed to update item',
      message: getErrorMessage(error, 'Failed to update item'),
    })
  } finally {
    detailItemSubmitting.value = false
  }
}

async function deleteCurrentItem() {
  if (!confirmItemDeletion.value) {
    confirmItemDeletion.value = true
    return
  }
  if (detailDeleteSubmitting.value || !props.gameId || !detailItemId.value) {
    return
  }

  detailDeleteSubmitting.value = true
  const deletedItemId = detailItemId.value
  const deletedItemName = detailItem.value?.name || detailItemDraft.value.name.trim() || 'Item'

  try {
    await auth.deleteGameItem(props.gameId, deletedItemId)
    closeItemDetailModal(true)

    if (currentPage.value > 1 && allItems.value.length === 1) {
      currentPage.value -= 1
    } else {
      await fetchItems()
    }

    emit('created')
    notify.success({
      title: 'Item deleted',
      message: `${deletedItemName} was removed from the compendium and every character inventory.`,
    })
  } catch (error) {
    notify.error({
      title: 'Failed to delete item',
      message: getErrorMessage(error, 'Failed to delete item'),
    })
  } finally {
    detailDeleteSubmitting.value = false
    confirmItemDeletion.value = false
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

function createItemDraftFromItem(item) {
  const normalizedItem = normalizeItem(item)

  return {
    name: normalizedItem.name,
    description: normalizedItem.description,
    rarity: normalizedItem.rarity,
    category: normalizedItem.category,
    gridWidth: normalizedItem.grid_width,
    gridHeight: normalizedItem.grid_height,
    equipSlot: normalizedItem.equip_slot || 'main_hand',
    tags: [...normalizedItem.tags],
    requiredAttributes: normalizedItem.required_attributes.map(entry => ({
      attribute_name: entry.attribute_name,
      min_value: entry.min_value,
    })),
    attributeModifiers: normalizedItem.attribute_modifiers.map(entry => ({
      attribute_name: entry.attribute_name,
      modifier_value: entry.modifier_value,
      is_percentage: Boolean(entry.is_percentage),
    })),
  }
}

function buildItemPayload(draft) {
  return {
    name: String(draft?.name || '').trim(),
    description: sanitizeDescriptionInput(String(draft?.description || '').trim()),
    rarity: draft?.rarity,
    category: draft?.category,
    tags: uniqueLabels(Array.isArray(draft?.tags) ? draft.tags : []),
    grid_width: normalizePositiveInteger(draft?.gridWidth, 1),
    grid_height: normalizePositiveInteger(draft?.gridHeight, 1),
    equip_slot: draft?.category === 'equipment' ? normalizeEquipSlot(draft?.equipSlot) : undefined,
    required_attributes: (draft?.requiredAttributes ?? [])
      .map(entry => ({
        attribute_name: normalizeAttributeName(entry.attribute_name),
        min_value: normalizeInteger(entry.min_value, 0),
      }))
      .filter(entry => entry.attribute_name),
    attribute_modifiers: (draft?.attributeModifiers ?? [])
      .map(entry => ({
        attribute_name: normalizeAttributeName(entry.attribute_name),
        modifier_value: normalizeSignedInteger(entry.modifier_value, 0),
        is_percentage: Boolean(entry.is_percentage),
      }))
      .filter(entry => entry.attribute_name),
  }
}

function validateItemPayload(payload) {
  if (!props.gameId) {
    return 'Game context is missing.'
  }
  if (!payload.name) {
    return 'Item name cannot be empty.'
  }
  if (payload.name.length > ITEM_NAME_LIMIT) {
    return `Item names can be up to ${ITEM_NAME_LIMIT} characters.`
  }
  if (payload.description.length > ITEM_DESCRIPTION_LIMIT) {
    return `Descriptions can be up to ${ITEM_DESCRIPTION_LIMIT} characters.`
  }
  if (countLineBreaks(payload.description) > 1) {
    return 'Descriptions can contain only one line break.'
  }
  if (payload.category === 'equipment' && !payload.equip_slot) {
    return 'Equipment items need an equip slot.'
  }

  return ''
}

function sanitizeDescriptionInput(value) {
  const normalized = String(value || '').replace(/\r\n?/g, '\n')
  const parts = normalized.split('\n')
  const limitedValue = parts.length <= 2
    ? normalized
    : `${parts[0]}\n${parts.slice(1).join(' ')}`

  return limitedValue.slice(0, ITEM_DESCRIPTION_LIMIT)
}

function countLineBreaks(value) {
  return (String(value || '').match(/\n/g) ?? []).length
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

function formatFileSize(value) {
  if (!Number.isFinite(value) || value <= 0) {
    return '0 B'
  }
  if (value >= 1024 * 1024) {
    return `${(value / (1024 * 1024)).toFixed(1)} MB`
  }
  if (value >= 1024) {
    return `${Math.round(value / 1024)} KB`
  }
  return `${value} B`
}

function notifyCreateItemValidation(message) {
  notify.warning({
    title: 'Check item details',
    message,
  })
}

function truncateText(value, limit, fallback = '') {
  const normalized = String(value || '').trim()
  if (!normalized) {
    return fallback
  }
  if (normalized.length <= limit) {
    return normalized
  }
  return `${normalized.slice(0, Math.max(0, limit - 3)).trimEnd()}...`
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

function resolvedItemImageUrl(item) {
  if (item?.id === 'draft-preview') {
    return itemImagePreviewUrl.value
  }
  if (item?.id === 'detail-preview') {
    return detailItemImagePreviewUrl.value || itemImageUrl(detailItem.value)
  }
  return itemImageUrl(item)
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
  <section class="flex min-h-[52rem] flex-col gap-6 xl:h-[calc(100vh-5rem)] xl:min-h-0">
    <article class="overflow-hidden rounded-[1.9rem] border border-[rgba(126,200,227,0.14)] bg-[linear-gradient(180deg,rgba(9,18,34,0.98),rgba(5,10,22,0.98))] p-5 shadow-[0_32px_90px_rgba(0,0,0,0.35)] sm:p-6">
      <div class="flex flex-col gap-5 xl:flex-row xl:items-end xl:justify-between">
        <div>
          <p class="text-[11px] uppercase tracking-[0.24em] text-[#7ec8e3]/58">GM Compendium</p>
          <div class="mt-3 flex flex-wrap items-center gap-3">
            <h2 class="font-[Cinzel] text-[28px] font-bold text-[#f6f7fb] sm:text-[34px]">Item Compendium</h2>
            <span class="rounded-full border border-[rgba(126,200,227,0.16)] bg-[rgba(126,200,227,0.08)] px-3 py-1.5 text-[11px] uppercase tracking-[0.16em] text-[#8fd7ef]">
              {{ compendiumStats.totalItems }} total
            </span>
          </div>
        </div>

        <div class="flex flex-wrap gap-3">
          <button
            type="button"
            @click="openCartModal"
            :disabled="!cartCount"
            class="relative inline-flex cursor-pointer items-center gap-2 rounded-xl border border-[rgba(126,200,227,0.2)] bg-[rgba(126,200,227,0.08)] px-4 py-2.5 text-[13px] font-semibold text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(126,200,227,0.4)] disabled:cursor-not-allowed disabled:opacity-50"
          >
            <ShoppingCart class="h-4 w-4" :stroke-width="2" />
            Transfer Cart
            <span v-if="cartCount" class="flex h-5 min-w-5 items-center justify-center rounded-full bg-[rgba(233,69,96,0.92)] px-1.5 text-[11px] font-semibold text-white">{{ cartCount }}</span>
          </button>
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

    <div class="grid min-h-0 flex-1 content-stretch gap-6 xl:grid-cols-[minmax(0,1.45fr)_390px]">
      <article class="flex min-h-[42rem] flex-col overflow-hidden rounded-[1.75rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(8,16,30,0.78)] xl:h-full xl:min-h-0">
        <div class="flex flex-wrap items-center justify-between gap-3 border-b border-[rgba(126,200,227,0.1)] px-5 py-4 sm:px-6">
          <div>
            <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Compendium Entries</p>
            <p class="mt-2 text-[14px] text-[#d8dce7]/58">{{ itemResultsLabel }}</p>
          </div>

          <div class="flex flex-wrap gap-2">
            <button
              type="button"
              @click="changePage(currentPage - 1)"
              :disabled="itemsLoading || !pagination.hasPrev"
              class="cursor-pointer rounded-xl border border-[rgba(126,200,227,0.16)] bg-[rgba(126,200,227,0.08)] px-3 py-2 text-[12px] font-semibold text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(126,200,227,0.3)] disabled:cursor-not-allowed disabled:opacity-40"
            >
              Prev
            </button>
            <button
              type="button"
              @click="changePage(currentPage + 1)"
              :disabled="itemsLoading || !pagination.hasNext"
              class="cursor-pointer rounded-xl border border-[rgba(126,200,227,0.16)] bg-[rgba(126,200,227,0.08)] px-3 py-2 text-[12px] font-semibold text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(126,200,227,0.3)] disabled:cursor-not-allowed disabled:opacity-40"
            >
              Next
            </button>
          </div>
        </div>

        <p v-if="itemsError" class="mx-5 mt-5 rounded-[1.2rem] border border-[rgba(248,113,113,0.2)] bg-[rgba(127,29,29,0.18)] px-4 py-3 text-[13px] text-[#fecaca] sm:mx-6">
          {{ itemsError }}
        </p>

        <div v-if="allItems.length" class="grid min-h-0 flex-1 content-start auto-rows-min gap-4 overflow-y-auto p-5 sm:grid-cols-2 2xl:grid-cols-3 sm:p-6">
          <button
            v-for="item in allItems"
            :key="item.id"
            type="button"
            @click="selectItem(item.id)"
            @contextmenu.prevent="openItemContextMenu($event, item)"
            @mouseenter="hoveredItemId = item.id"
            @mouseleave="hoveredItemId = ''"
            @focus="selectedItemId = item.id"
            class="group relative cursor-pointer rounded-[1.45rem] border bg-[rgba(7,17,31,0.72)] p-4 text-left transition-all duration-200 hover:-translate-y-0.5 hover:border-[rgba(126,200,227,0.24)]"
            :class="[
              selectedItemId === item.id ? 'border-[rgba(233,69,96,0.32)] shadow-[0_12px_36px_rgba(233,69,96,0.12)]' : 'border-[rgba(126,200,227,0.12)]',
              isInCart(item) ? 'ring-2 ring-[rgba(233,69,96,0.45)]' : '',
            ]"
          >
            <span
              role="checkbox"
              :aria-checked="isInCart(item)"
              :title="isInCart(item) ? 'Remove from transfer cart' : 'Add to transfer cart'"
              @click.stop.prevent="toggleCartItem(item)"
              class="absolute bottom-3 left-3 z-10 flex h-7 w-7 cursor-pointer items-center justify-center rounded-lg border shadow-[0_4px_12px_rgba(0,0,0,0.4)] transition-all duration-200"
              :class="isInCart(item)
                ? 'border-[rgba(233,69,96,0.55)] bg-[rgba(233,69,96,0.92)] text-white'
                : 'border-[rgba(126,200,227,0.35)] bg-[rgba(7,17,31,0.92)] text-transparent hover:border-[rgba(126,200,227,0.7)]'"
            >
              <Check class="h-4 w-4" :stroke-width="3" />
            </span>
            <div class="flex items-start gap-4">
              <div class="relative flex h-20 w-20 shrink-0 items-center justify-center overflow-hidden rounded-[1.3rem] border text-[24px] font-bold uppercase" :class="itemFrameClass(item.rarity)">
                <img v-if="itemImageUrl(item)" :src="itemImageUrl(item)" :alt="item.name" class="h-full w-full object-cover" />
                <span v-else>{{ itemGlyph(item) }}</span>
                <span class="absolute right-2 top-2 rounded-full border border-[rgba(255,255,255,0.16)] bg-[rgba(7,17,31,0.82)] px-2 py-0.5 text-[10px] font-semibold text-[#f6f7fb]">
                  {{ itemSizeLabel(item) }}
                </span>
              </div>

              <div class="min-w-0 flex-1">
                <div class="flex flex-wrap items-start gap-2">
                  <h3 class="min-w-0 flex-1 break-words whitespace-normal text-[16px] font-semibold leading-snug text-[#f6f7fb] line-clamp-2 [overflow-wrap:anywhere]">{{ item.name }}</h3>
                  <span class="shrink-0 rounded-full border px-2.5 py-1 text-[10px] font-semibold uppercase tracking-[0.16em]" :class="rarityBadgeClass(item.rarity)">
                    {{ formatLabel(item.rarity) }}
                  </span>
                </div>

                <p class="mt-2 line-clamp-3 break-words whitespace-pre-line text-[13px] leading-relaxed text-[#d8dce7]/58">
                  {{ truncateText(item.description, ITEM_CARD_DESCRIPTION_PREVIEW_LIMIT, 'No description yet.') }}
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
                </div>
              </div>
            </div>

            <div class="mt-4 flex flex-wrap items-center gap-3 border-t border-[rgba(126,200,227,0.08)] pt-3 text-[12px] text-[#d8dce7]/56">
              <span>{{ item.required_attributes.length }} requirements</span>
              <span>{{ item.attribute_modifiers.length }} modifiers</span>
              <span>Updated {{ formatDateTime(item.updated_at) }}</span>
              <span class="ml-auto text-[#7ec8e3]/38">Right-click</span>
            </div>
          </button>
        </div>

        <div v-else class="flex min-h-[320px] flex-1 flex-col items-center justify-center gap-4 px-6 py-12 text-center">
          <div class="flex h-16 w-16 items-center justify-center rounded-full border border-[rgba(126,200,227,0.14)] bg-[rgba(126,200,227,0.08)] text-[#8fd7ef]">
            <Package class="h-7 w-7" :stroke-width="2" />
          </div>
          <div>
            <h3 class="font-[Cinzel] text-[24px] font-bold text-[#f6f7fb]">{{ itemsLoading ? 'Loading items' : 'No items found' }}</h3>
            <p class="mt-3 max-w-[34rem] text-[14px] leading-relaxed text-[#d8dce7]/58">
              {{ itemsLoading
                ? 'The server is preparing the current page.'
                : (hasActiveFilters
                    ? 'Try broadening the current filters or search query.'
                    : 'Create the first compendium entry for this campaign.') }}
            </p>
          </div>
        </div>
      </article>

      <aside class="space-y-6 min-h-0">

        <article class="flex min-h-[42rem] flex-col overflow-y-auto rounded-[1.75rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(8,16,30,0.78)] p-5 sm:p-6 xl:h-full xl:min-h-0">
          <div>
            <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Inspector</p>
            <p class="mt-2 text-[14px] leading-relaxed text-[#d8dce7]/58">Pinned preview for the currently selected compendium entry.</p>
          </div>

          <div v-if="inspectorItem" class="mt-5 flex flex-1 flex-col gap-5">
            <div class="overflow-hidden rounded-[1.5rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.66)]">
              <div class="flex h-64 items-center justify-center" :class="itemFrameClass(inspectorItem.rarity)">
                <img v-if="resolvedItemImageUrl(inspectorItem)" :src="resolvedItemImageUrl(inspectorItem)" :alt="inspectorItem.name" class="h-full w-full object-cover" />
                <span v-else class="font-[Cinzel] text-[40px] font-bold">{{ itemGlyph(inspectorItem) }}</span>
              </div>
            </div>

            <div>
              <div class="flex flex-wrap items-center gap-2">
                <h3 class="min-w-0 flex-1 break-words whitespace-normal font-[Cinzel] text-[28px] font-bold leading-tight text-[#f6f7fb] line-clamp-2 [overflow-wrap:anywhere]">{{ inspectorItem.name }}</h3>
                <span class="shrink-0 rounded-full border px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.16em]" :class="rarityBadgeClass(inspectorItem.rarity)">
                  {{ formatLabel(inspectorItem.rarity) }}
                </span>
              </div>
              <p class="mt-3 break-words whitespace-pre-line text-[14px] leading-relaxed text-[#d8dce7]/62">{{ truncateText(inspectorItem.description, ITEM_DETAIL_DESCRIPTION_PREVIEW_LIMIT, 'No description yet.') }}</p>
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

            <div class="min-h-[5.75rem]">
              <p class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Custom Tags</p>
              <div v-if="inspectorItem.tags.length" class="mt-3 flex min-h-[2.5rem] flex-wrap content-start gap-2">
                <span v-for="tag in inspectorItem.tags" :key="`${inspectorItem.id}-${tag}`" class="rounded-full border border-[rgba(233,69,96,0.18)] bg-[rgba(233,69,96,0.08)] px-3 py-1.5 text-[12px] text-[#ffe0e7]">
                  {{ tag }}
                </span>
              </div>
              <div v-else class="mt-3 flex min-h-[2.5rem] items-center">
                <p class="text-[14px] text-[#d8dce7]/58">No custom tags assigned to this item.</p>
              </div>
            </div>
          </div>

          <div v-else class="mt-6 flex flex-1 items-center text-[14px] leading-relaxed text-[#d8dce7]/58">Select an item from the current page to inspect it here.</div>
        </article>
      </aside>
    </div>

    <Teleport to="body">
      <div v-if="itemContextMenu.visible && contextMenuItem" class="fixed inset-0 z-[12480]" @click="closeItemContextMenu" @contextmenu.prevent="closeItemContextMenu">
        <div class="fixed z-[12490] w-[220px] overflow-hidden rounded-[1.3rem] border border-[rgba(126,200,227,0.16)] bg-[linear-gradient(180deg,rgba(9,18,34,0.98),rgba(5,10,22,0.98))] p-2 shadow-[0_20px_50px_rgba(0,0,0,0.45)]" :style="itemContextMenuStyle" @click.stop @contextmenu.prevent>
          <button type="button" class="flex w-full cursor-pointer items-center gap-3 rounded-[1rem] px-3 py-3 text-left text-[13px] font-semibold text-[#f6f7fb] transition-all duration-200 hover:bg-[rgba(126,200,227,0.08)]" @click="openItemDetailModal(contextMenuItem, 'view')">
            <Eye class="h-4 w-4 text-[#8fd7ef]" :stroke-width="2" />
            Open Details
          </button>
          <button type="button" class="mt-1 flex w-full cursor-pointer items-center gap-3 rounded-[1rem] px-3 py-3 text-left text-[13px] font-semibold text-[#f6f7fb] transition-all duration-200 hover:bg-[rgba(233,69,96,0.12)]" @click="openItemDetailModal(contextMenuItem, 'edit')">
            <Pencil class="h-4 w-4 text-[#ffe0e7]" :stroke-width="2" />
            Edit Item
          </button>
          <button type="button" class="mt-1 flex w-full cursor-pointer items-center gap-3 rounded-[1rem] px-3 py-3 text-left text-[13px] font-semibold text-[#fecaca] transition-all duration-200 hover:bg-[rgba(248,113,113,0.12)]" @click="requestItemDeletion(contextMenuItem)">
            <Trash2 class="h-4 w-4 text-[#fca5a5]" :stroke-width="2" />
            Delete Item
          </button>
        </div>
      </div>
    </Teleport>

    <Teleport to="body">
      <div v-if="deleteCandidateItem" class="fixed inset-0 z-[12520] flex items-center justify-center p-4">
        <div class="absolute inset-0 bg-[rgba(5,8,12,0.82)] backdrop-blur-md" @click="cancelItemDeletion"></div>

        <div class="relative w-full max-w-[28rem] overflow-hidden rounded-[1.6rem] border border-[rgba(248,113,113,0.24)] bg-[linear-gradient(180deg,rgba(9,18,34,0.98),rgba(5,10,22,0.98))] p-6 shadow-[0_40px_120px_rgba(0,0,0,0.52)]">
          <div class="flex items-center gap-3">
            <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl border border-[rgba(248,113,113,0.24)] bg-[rgba(248,113,113,0.12)] text-[#fca5a5]">
              <Trash2 class="h-6 w-6" :stroke-width="2" />
            </div>
            <div class="min-w-0">
              <p class="text-[11px] uppercase tracking-[0.24em] text-[#fca5a5]/70">Delete Item</p>
              <h3 class="mt-1 break-words font-[Cinzel] text-[22px] font-bold leading-tight text-[#f6f7fb] [overflow-wrap:anywhere]">{{ deleteCandidateItem.name }}</h3>
            </div>
          </div>

          <p class="mt-4 text-[14px] leading-relaxed text-[#d8dce7]/68">
            This permanently removes the item from the compendium and from every character inventory. This action cannot be undone.
          </p>

          <div class="mt-6 flex flex-wrap justify-end gap-3">
            <button type="button" @click="cancelItemDeletion" :disabled="deleteCandidateSubmitting" class="cursor-pointer rounded-xl border border-[rgba(126,200,227,0.16)] bg-[rgba(126,200,227,0.08)] px-4 py-2.5 text-[13px] font-semibold text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(126,200,227,0.3)] disabled:cursor-not-allowed disabled:opacity-60">
              Cancel
            </button>
            <button type="button" @click="confirmCandidateDeletion" :disabled="deleteCandidateSubmitting" class="inline-flex cursor-pointer items-center gap-2 rounded-xl border border-[rgba(248,113,113,0.28)] bg-[linear-gradient(135deg,rgba(248,113,113,0.9),rgba(220,38,38,0.9))] px-4 py-2.5 text-[13px] font-semibold text-white transition-all duration-200 hover:-translate-y-0.5 disabled:cursor-not-allowed disabled:opacity-60">
              <Trash2 class="h-4 w-4" :stroke-width="2" />
              {{ deleteCandidateSubmitting ? 'Deleting...' : 'Delete Item' }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <Teleport to="body">
      <div v-if="showCartModal" class="fixed inset-0 z-[12530] p-3 sm:p-4">
        <div class="absolute inset-0 bg-[rgba(5,8,12,0.82)] backdrop-blur-md" @click="closeCartModal"></div>

        <div class="relative mx-auto flex h-full w-full max-w-[62rem] flex-col overflow-hidden rounded-[2rem] border border-[rgba(126,200,227,0.16)] bg-[linear-gradient(180deg,rgba(9,18,34,0.98),rgba(5,10,22,0.98))] shadow-[0_40px_120px_rgba(0,0,0,0.52)]">
          <button
            type="button"
            @click="closeCartModal"
            :disabled="cartSubmitting"
            class="absolute right-5 top-5 z-10 flex h-10 w-10 cursor-pointer items-center justify-center rounded-xl border border-[rgba(126,200,227,0.14)] bg-[rgba(126,200,227,0.08)] text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(233,69,96,0.32)] disabled:cursor-not-allowed disabled:opacity-60"
            aria-label="Close transfer cart"
          >
            <X class="h-5 w-5" :stroke-width="2" />
          </button>

          <div class="border-b border-[rgba(126,200,227,0.1)] px-5 py-5 pr-16 sm:px-6">
            <p class="text-[11px] uppercase tracking-[0.24em] text-[#7ec8e3]/58">Deliver Items</p>
            <div class="mt-3 flex flex-wrap items-center gap-3">
              <h2 class="font-[Cinzel] text-[26px] font-bold text-[#f6f7fb] sm:text-[32px]">Transfer Cart</h2>
              <span class="rounded-full border border-[rgba(126,200,227,0.16)] bg-[rgba(126,200,227,0.08)] px-3 py-1.5 text-[11px] uppercase tracking-[0.16em] text-[#8fd7ef]">{{ cartCount }} item(s)</span>
            </div>
            <p class="mt-3 max-w-[44rem] text-[13px] leading-relaxed text-[#d8dce7]/58">Tune each item's quantity, enchantment, and optional durability, then choose a character to receive the whole cart.</p>
          </div>

          <div class="min-h-0 flex-1 overflow-y-auto px-5 py-5 sm:px-6">
            <div v-if="cartItems.length" class="space-y-4">
              <article v-for="line in cartItems" :key="line.item.id" class="rounded-[1.4rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(8,16,30,0.78)] p-4">
                <div class="flex items-start gap-4">
                  <div class="relative flex h-16 w-16 shrink-0 items-center justify-center overflow-hidden rounded-[1.1rem] border text-[18px] font-bold uppercase" :class="itemFrameClass(line.item.rarity)">
                    <img v-if="itemImageUrl(line.item)" :src="itemImageUrl(line.item)" :alt="line.item.name" class="h-full w-full object-cover" />
                    <span v-else>{{ itemGlyph(line.item) }}</span>
                  </div>

                  <div class="min-w-0 flex-1">
                    <div class="flex items-start justify-between gap-3">
                      <div class="min-w-0">
                        <h3 class="break-words text-[15px] font-semibold leading-snug text-[#f6f7fb] [overflow-wrap:anywhere]">{{ line.item.name }}</h3>
                        <p class="mt-1 text-[12px] text-[#d8dce7]/55">{{ formatLabel(line.item.rarity) }} · {{ formatLabel(line.item.category) }} · {{ itemSizeLabel(line.item) }}</p>
                      </div>
                      <button type="button" @click="removeCartLine(line.item.id)" :disabled="cartSubmitting" class="flex h-9 w-9 shrink-0 cursor-pointer items-center justify-center rounded-xl border border-[rgba(248,113,113,0.24)] bg-[rgba(248,113,113,0.1)] text-[#fca5a5] transition-all duration-200 hover:border-[rgba(248,113,113,0.4)] disabled:cursor-not-allowed disabled:opacity-60" aria-label="Remove from cart">
                        <Trash2 class="h-4 w-4" :stroke-width="2" />
                      </button>
                    </div>

                    <div class="mt-3 grid gap-3 sm:grid-cols-3">
                      <label class="block">
                        <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Quantity</span>
                        <input v-model.number="line.quantity" type="number" min="1" max="9999" class="session-input mt-1.5 w-full rounded-[1rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.72)] px-3 py-2.5 text-[14px] text-[#f6f7fb] outline-none" />
                      </label>
                      <label class="block">
                        <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Durability</span>
                        <input v-model.number="line.durability" type="number" min="0" :max="line.maxDurability" class="session-input mt-1.5 w-full rounded-[1rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.72)] px-3 py-2.5 text-[14px] text-[#f6f7fb] outline-none" />
                      </label>
                      <label class="block">
                        <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Max durability</span>
                        <input v-model.number="line.maxDurability" type="number" min="1" class="session-input mt-1.5 w-full rounded-[1rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.72)] px-3 py-2.5 text-[14px] text-[#f6f7fb] outline-none" />
                      </label>
                    </div>

                    <div class="mt-3 flex flex-wrap items-center gap-3">
                      <label class="flex cursor-pointer items-center gap-2.5 rounded-[1rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.72)] px-3 py-2.5">
                        <input v-model="line.hasEnchantment" type="checkbox" class="h-4 w-4 cursor-pointer accent-[#e94560]" />
                        <span class="text-[12px] font-semibold text-[#d8dce7]/78">Enchantment</span>
                      </label>
                      <label v-if="line.hasEnchantment" class="block min-w-[8rem] flex-1">
                        <input v-model.number="line.enchantment" type="number" min="-999" max="999" placeholder="Enchantment level" class="session-input w-full rounded-[1rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.72)] px-3 py-2.5 text-[14px] text-[#f6f7fb] outline-none" />
                      </label>
                    </div>
                  </div>
                </div>
              </article>
            </div>

            <div v-else class="flex min-h-[12rem] flex-col items-center justify-center gap-3 text-center">
              <ShoppingCart class="h-8 w-8 text-[#7ec8e3]/40" :stroke-width="1.8" />
              <p class="text-[14px] text-[#d8dce7]/58">Your cart is empty.</p>
            </div>
          </div>

          <div class="border-t border-[rgba(126,200,227,0.1)] px-5 py-5 sm:px-6">
            <div class="grid gap-5 lg:grid-cols-[minmax(0,1fr)_minmax(0,18rem)]">
              <div>
                <div class="flex items-center justify-between gap-3">
                  <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Recipients</span>
                  <span v-if="cartTargetCharacterIds.length" class="text-[11px] font-semibold text-[#8fd7ef]">{{ cartTargetCharacterIds.length }} selected</span>
                </div>
                <div class="mt-2 flex items-center gap-3 rounded-[1.2rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.72)] px-4 py-3">
                  <Search class="h-4 w-4 text-[#7ec8e3]/45" :stroke-width="2" />
                  <input v-model="cartCharacterSearch" type="text" placeholder="Search characters" class="session-input w-full border-0 bg-transparent p-0 text-[14px] text-[#f6f7fb] outline-none placeholder:text-[#7ec8e3]/30" />
                </div>
                <div class="mt-3 max-h-[12rem] overflow-y-auto rounded-[1.2rem] border border-[rgba(126,200,227,0.1)] bg-[rgba(7,17,31,0.5)]">
                  <button
                    v-for="character in cartCharacterOptions"
                    :key="character.id"
                    type="button"
                    @click="toggleCartCharacter(character.id)"
                    class="flex w-full cursor-pointer items-center gap-3 border-b border-[rgba(126,200,227,0.06)] px-4 py-3 text-left transition-all duration-200 last:border-b-0 hover:bg-[rgba(126,200,227,0.06)]"
                    :class="cartTargetCharacterIdSet.has(character.id) ? 'bg-[rgba(233,69,96,0.12)]' : ''"
                  >
                    <span
                      class="flex h-6 w-6 shrink-0 items-center justify-center rounded-md border transition-all duration-200"
                      :class="cartTargetCharacterIdSet.has(character.id)
                        ? 'border-[rgba(233,69,96,0.55)] bg-[rgba(233,69,96,0.92)] text-white'
                        : 'border-[rgba(126,200,227,0.3)] bg-[rgba(7,17,31,0.7)] text-transparent'"
                    >
                      <Check class="h-3.5 w-3.5" :stroke-width="3" />
                    </span>
                    <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl border border-[rgba(126,200,227,0.14)] bg-[rgba(126,200,227,0.08)] text-[#8fd7ef]">
                      <User class="h-4 w-4" :stroke-width="2" />
                    </div>
                    <div class="min-w-0 flex-1">
                      <p class="truncate text-[14px] font-semibold text-[#f6f7fb]">{{ character.name }}</p>
                      <p v-if="character.owner?.username" class="truncate text-[12px] text-[#d8dce7]/52">{{ character.owner.username }}</p>
                    </div>
                  </button>
                  <p v-if="!cartCharacterOptions.length" class="px-4 py-4 text-[13px] text-[#d8dce7]/50">No characters match your search.</p>
                </div>
              </div>

              <div class="flex flex-col justify-end gap-3">
                <p class="text-[13px] leading-relaxed text-[#d8dce7]/58">
                  Delivering <span class="font-semibold text-[#f6f7fb]">{{ cartCount }}</span> item(s) to
                  <span class="font-semibold text-[#f6f7fb]">{{ cartTargetCharacterIds.length }}</span> character(s).
                </p>
                <div class="flex gap-3">
                  <button type="button" @click="clearCart" :disabled="cartSubmitting || !cartCount" class="cursor-pointer rounded-xl border border-[rgba(126,200,227,0.16)] bg-[rgba(126,200,227,0.08)] px-4 py-2.5 text-[13px] font-semibold text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(126,200,227,0.3)] disabled:cursor-not-allowed disabled:opacity-50">
                    Clear
                  </button>
                  <button type="button" @click="deliverCart" :disabled="cartSubmitting || !cartCount || !cartTargetCharacterIds.length" class="inline-flex flex-1 cursor-pointer items-center justify-center gap-2 rounded-xl border border-[rgba(233,69,96,0.28)] bg-[linear-gradient(135deg,rgba(233,69,96,0.92),rgba(194,49,82,0.92))] px-4 py-2.5 text-[13px] font-semibold text-white transition-all duration-200 hover:-translate-y-0.5 disabled:cursor-not-allowed disabled:opacity-60 disabled:hover:translate-y-0">
                    <Send class="h-4 w-4" :stroke-width="2" />
                    {{ cartSubmitting ? 'Delivering...' : 'Deliver Items' }}
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Teleport>

    <Teleport to="body">
      <div v-if="showItemDetailModal && detailPreviewItem" class="fixed inset-0 z-[12510] p-3 sm:p-4">
        <div class="absolute inset-0 bg-[rgba(5,8,12,0.82)] backdrop-blur-md" @click="closeItemDetailModal"></div>

        <div class="relative flex h-full w-full flex-col overflow-hidden rounded-[2rem] border border-[rgba(126,200,227,0.16)] bg-[linear-gradient(180deg,rgba(9,18,34,0.98),rgba(5,10,22,0.98))] shadow-[0_40px_120px_rgba(0,0,0,0.52)]">
          <button
            type="button"
            @click="closeItemDetailModal"
            :disabled="detailItemSubmitting || detailDeleteSubmitting"
            class="absolute right-5 top-5 z-10 flex h-10 w-10 cursor-pointer items-center justify-center rounded-xl border border-[rgba(126,200,227,0.14)] bg-[rgba(126,200,227,0.08)] text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(233,69,96,0.32)] disabled:cursor-not-allowed disabled:opacity-60"
            aria-label="Close item detail modal"
          >
            <X class="h-5 w-5" :stroke-width="2" />
          </button>

          <div class="border-b border-[rgba(126,200,227,0.1)] px-5 py-5 pr-16 sm:px-6">
            <p class="text-[11px] uppercase tracking-[0.24em] text-[#7ec8e3]/58">Compendium Entry</p>
            <div class="mt-3 flex flex-wrap items-center gap-3">
              <h2 class="font-[Cinzel] text-[28px] font-bold text-[#f6f7fb] sm:text-[34px]">{{ itemDetailMode === 'edit' ? 'Edit Compendium Entry' : 'Compendium Entry Details' }}</h2>
              <span class="rounded-full border border-[rgba(126,200,227,0.16)] bg-[rgba(126,200,227,0.08)] px-3 py-1.5 text-[11px] uppercase tracking-[0.16em] text-[#8fd7ef]">
                {{ detailPreviewItem.name }}
              </span>
            </div>
            <div class="mt-4 flex flex-wrap gap-3">
              <button
                type="button"
                @click="itemDetailMode === 'edit' ? cancelDetailItemEdit() : beginDetailItemEdit()"
                :disabled="detailItemSubmitting || detailDeleteSubmitting"
                class="inline-flex cursor-pointer items-center gap-2 rounded-xl border border-[rgba(126,200,227,0.16)] bg-[rgba(126,200,227,0.08)] px-4 py-2.5 text-[13px] font-semibold text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(126,200,227,0.3)] disabled:cursor-not-allowed disabled:opacity-60"
              >
                <Pencil class="h-4 w-4" :stroke-width="2" />
                {{ itemDetailMode === 'edit' ? 'Discard Changes' : 'Edit Item' }}
              </button>
            </div>
          </div>

          <div class="grid min-h-0 flex-1 gap-6 overflow-hidden px-5 pb-5 pt-5 sm:px-6 lg:grid-cols-[390px_minmax(0,1fr)]">
            <aside class="min-h-0 space-y-5 overflow-y-auto pr-1">
              <article class="rounded-[1.6rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(8,16,30,0.78)] p-5">
                <div class="overflow-hidden rounded-[1.5rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.66)]">
                  <div class="flex h-64 items-center justify-center" :class="itemFrameClass(detailPreviewItem.rarity)">
                    <img v-if="resolvedItemImageUrl(detailPreviewItem)" :src="resolvedItemImageUrl(detailPreviewItem)" :alt="detailPreviewItem.name" class="h-full w-full object-cover" />
                    <span v-else class="font-[Cinzel] text-[40px] font-bold">{{ itemGlyph(detailPreviewItem) }}</span>
                  </div>
                </div>

                <div class="mt-5">
                  <div class="flex flex-wrap items-center gap-2">
                    <h3 class="min-w-0 flex-1 break-words whitespace-normal font-[Cinzel] text-[28px] font-bold leading-tight text-[#f6f7fb] line-clamp-2 [overflow-wrap:anywhere]">{{ detailPreviewItem.name }}</h3>
                    <span class="shrink-0 rounded-full border px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.16em]" :class="rarityBadgeClass(detailPreviewItem.rarity)">
                      {{ formatLabel(detailPreviewItem.rarity) }}
                    </span>
                  </div>
                  <p class="mt-3 break-words whitespace-pre-line text-[14px] leading-relaxed text-[#d8dce7]/62">{{ truncateText(detailPreviewItem.description, ITEM_DETAIL_DESCRIPTION_PREVIEW_LIMIT, 'No description yet.') }}</p>
                </div>

                <div class="mt-5 grid gap-3 sm:grid-cols-2">
                  <div class="rounded-[1.2rem] border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] px-4 py-3">
                    <p class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Category</p>
                    <p class="mt-2 text-[15px] font-semibold text-[#f6f7fb]">{{ formatLabel(detailPreviewItem.category) }}</p>
                  </div>
                  <div class="rounded-[1.2rem] border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] px-4 py-3">
                    <p class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Grid Size</p>
                    <p class="mt-2 text-[15px] font-semibold text-[#f6f7fb]">{{ itemSizeLabel(detailPreviewItem) }}</p>
                  </div>
                  <div class="rounded-[1.2rem] border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] px-4 py-3 sm:col-span-2">
                    <p class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Equip Slot</p>
                    <p class="mt-2 text-[15px] font-semibold text-[#f6f7fb]">{{ detailPreviewItem.equip_slot ? formatLabel(detailPreviewItem.equip_slot) : 'Not equippable' }}</p>
                  </div>
                </div>

                <div class="mt-5 min-h-[5.75rem]">
                  <p class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Custom Tags</p>
                  <div v-if="detailPreviewItem.tags.length" class="mt-3 flex min-h-[2.5rem] flex-wrap content-start gap-2">
                    <span v-for="tag in detailPreviewItem.tags" :key="`detail-tag-${tag}`" class="rounded-full border border-[rgba(233,69,96,0.18)] bg-[rgba(233,69,96,0.08)] px-3 py-1.5 text-[12px] text-[#ffe0e7]">
                      {{ tag }}
                    </span>
                  </div>
                  <div v-else class="mt-3 flex min-h-[2.5rem] items-center">
                    <p class="text-[14px] text-[#d8dce7]/58">No custom tags assigned to this item.</p>
                  </div>
                </div>
              </article>
            </aside>

            <section class="min-h-0 space-y-5 overflow-y-auto pr-1">
              <article class="rounded-[1.6rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(8,16,30,0.78)] p-5">
                <div class="grid gap-5 lg:grid-cols-[minmax(0,1fr)_minmax(320px,0.9fr)]">
                  <label class="block">
                    <span class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Name</span>
                    <input
                      v-model="detailItemDraft.name"
                      type="text"
                      :maxlength="ITEM_NAME_LIMIT"
                      :disabled="detailFormDisabled"
                      class="session-input mt-2 w-full rounded-[1.25rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.72)] px-4 py-3 text-[15px] text-[#f6f7fb] outline-none placeholder:text-[#7ec8e3]/30 disabled:cursor-not-allowed disabled:opacity-65"
                    />
                    <p class="mt-2 text-[12px] text-[#d8dce7]/54">{{ detailItemDraft.name.length }}/{{ ITEM_NAME_LIMIT }}</p>
                  </label>

                  <div class="grid gap-4 sm:grid-cols-3">
                    <label class="block">
                      <span class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Quality</span>
                      <select v-model="detailItemDraft.rarity" :disabled="detailFormDisabled" class="session-input mt-2 w-full rounded-[1.25rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.72)] px-4 py-3 text-[14px] text-[#f6f7fb] outline-none disabled:cursor-not-allowed disabled:opacity-65">
                        <option v-for="rarity in RARITY_OPTIONS" :key="rarity" :value="rarity">{{ formatLabel(rarity) }}</option>
                      </select>
                    </label>

                    <label class="block">
                      <span class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Category</span>
                      <select v-model="detailItemDraft.category" :disabled="detailFormDisabled" class="session-input mt-2 w-full rounded-[1.25rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.72)] px-4 py-3 text-[14px] text-[#f6f7fb] outline-none disabled:cursor-not-allowed disabled:opacity-65">
                        <option v-for="category in CATEGORY_OPTIONS" :key="category" :value="category">{{ formatLabel(category) }}</option>
                      </select>
                    </label>

                    <label class="block">
                      <span class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Equip Slot</span>
                      <select v-model="detailItemDraft.equipSlot" :disabled="detailFormDisabled || detailItemDraft.category !== 'equipment'" class="session-input mt-2 w-full rounded-[1.25rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.72)] px-4 py-3 text-[14px] text-[#f6f7fb] outline-none disabled:cursor-not-allowed disabled:opacity-45">
                        <option v-for="slot in EQUIP_SLOT_OPTIONS" :key="slot" :value="slot">{{ formatLabel(slot) }}</option>
                      </select>
                    </label>
                  </div>
                </div>

                <label class="mt-5 block">
                  <span class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Description</span>
                  <textarea
                    v-model="detailItemDraft.description"
                    rows="6"
                    :maxlength="ITEM_DESCRIPTION_LIMIT"
                    :disabled="detailFormDisabled"
                    class="session-input mt-2 w-full resize-y rounded-[1.25rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.72)] px-4 py-3 text-[15px] text-[#f6f7fb] outline-none placeholder:text-[#7ec8e3]/30 disabled:cursor-not-allowed disabled:opacity-65"
                  ></textarea>
                  <p class="mt-2 text-[12px] text-[#d8dce7]/54">{{ detailItemDraft.description.length }}/{{ ITEM_DESCRIPTION_LIMIT }} · 1 line break max</p>
                </label>
              </article>

              <article class="rounded-[1.6rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(8,16,30,0.78)] p-5">
                <div>
                  <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Image Upload</p>
                  <p class="mt-2 text-[14px] leading-relaxed text-[#d8dce7]/58">Select a replacement image to upload after the item details are saved. Leaving this empty keeps the current image.</p>
                </div>

                <input ref="detailItemImageInputRef" type="file" accept="image/jpeg,image/png,image/webp" class="hidden" @change="handleDetailItemImageSelected" />

                <div class="mt-5 flex flex-wrap gap-3">
                  <button type="button" @click="openDetailItemImagePicker" :disabled="detailFormDisabled" class="cursor-pointer rounded-xl border border-[rgba(126,200,227,0.16)] bg-[rgba(126,200,227,0.08)] px-4 py-2.5 text-[13px] font-semibold text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(126,200,227,0.3)] disabled:cursor-not-allowed disabled:opacity-60">
                    {{ detailItemImageFile ? 'Replace Selected Image' : 'Select Replacement Image' }}
                  </button>
                  <button v-if="detailItemImageFile" type="button" @click="clearDetailItemImageSelection" :disabled="detailFormDisabled" class="cursor-pointer rounded-xl border border-[rgba(248,113,113,0.2)] bg-[rgba(248,113,113,0.12)] px-4 py-2.5 text-[13px] font-semibold text-[#fecaca] transition-all duration-200 hover:border-[rgba(248,113,113,0.35)] disabled:cursor-not-allowed disabled:opacity-60">
                    Remove Selection
                  </button>
                </div>

                <div class="mt-4 rounded-[1.2rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.62)] px-4 py-3">
                  <p class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Selected File</p>
                  <p class="mt-2 text-[14px] text-[#f6f7fb]">{{ detailItemImageMetaLabel }}</p>
                  <p class="mt-2 text-[12px] text-[#d8dce7]/54">JPEG, PNG, or WebP up to 5MB.</p>
                </div>
              </article>

              <article class="rounded-[1.6rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(8,16,30,0.78)] p-5">
                <div>
                  <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Footprint</p>
                  <p class="mt-2 text-[14px] leading-relaxed text-[#d8dce7]/58">These values still define the occupied inventory cells for this item.</p>
                </div>

                <div class="mt-5 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
                  <label class="rounded-[1.2rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.62)] px-4 py-3">
                    <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Grid Width</span>
                    <input v-model.number="detailItemDraft.gridWidth" :disabled="detailFormDisabled" type="number" min="1" max="12" class="session-input mt-3 w-full border-0 bg-transparent p-0 text-[24px] font-semibold text-[#f6f7fb] outline-none disabled:cursor-not-allowed disabled:opacity-65" />
                  </label>
                  <label class="rounded-[1.2rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.62)] px-4 py-3">
                    <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Grid Height</span>
                    <input v-model.number="detailItemDraft.gridHeight" :disabled="detailFormDisabled" type="number" min="1" max="12" class="session-input mt-3 w-full border-0 bg-transparent p-0 text-[24px] font-semibold text-[#f6f7fb] outline-none disabled:cursor-not-allowed disabled:opacity-65" />
                  </label>
                  <div class="rounded-[1.2rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.62)] px-4 py-3 sm:col-span-2">
                    <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Preview Summary</span>
                    <p class="mt-3 text-[18px] font-semibold text-[#f6f7fb]">{{ itemSizeLabel(detailPreviewItem) }}</p>
                    <p class="mt-2 text-[13px] leading-relaxed text-[#d8dce7]/56">Adjust width and height to match the intended inventory footprint.</p>
                  </div>
                </div>
              </article>

              <article class="rounded-[1.6rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(8,16,30,0.78)] p-5">
                <div>
                  <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Tags</p>
                  <p class="mt-2 text-[14px] leading-relaxed text-[#d8dce7]/58">Game-scoped tags remain unique. Add a new one or attach an existing campaign tag.</p>
                </div>

                <div class="mt-5 grid gap-4 xl:grid-cols-[minmax(0,1fr)_auto]">
                  <label class="block">
                    <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Create New Tag</span>
                    <input
                      v-model="detailNewTagDraft"
                      type="text"
                      maxlength="60"
                      :disabled="detailFormDisabled"
                      placeholder="Example: Quest Reward"
                      class="session-input mt-2 w-full rounded-[1.25rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.72)] px-4 py-3 text-[14px] text-[#f6f7fb] outline-none placeholder:text-[#7ec8e3]/30 disabled:cursor-not-allowed disabled:opacity-65"
                      @keydown.enter.prevent="addDetailDraftTagFromInput"
                    />
                  </label>

                  <button type="button" @click="addDetailDraftTagFromInput" :disabled="detailFormDisabled" class="inline-flex cursor-pointer items-center justify-center gap-2 self-end rounded-xl border border-[rgba(233,69,96,0.22)] bg-[rgba(233,69,96,0.12)] px-4 py-3 text-[13px] font-semibold text-[#ffe0e7] transition-all duration-200 hover:border-[rgba(233,69,96,0.38)] disabled:cursor-not-allowed disabled:opacity-60">
                    <Plus class="h-4 w-4" :stroke-width="2" />
                    Create Tag
                  </button>
                </div>

                <div class="mt-5">
                  <p class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Selected Tags</p>
                  <div v-if="detailItemDraft.tags.length" class="mt-3 flex flex-wrap gap-2">
                    <button
                      v-for="tag in detailItemDraft.tags"
                      :key="`detail-selected-${tag}`"
                      type="button"
                      @click="removeDetailDraftTag(tag)"
                      :disabled="detailFormDisabled"
                      class="inline-flex cursor-pointer items-center gap-2 rounded-full border border-[rgba(233,69,96,0.24)] bg-[rgba(233,69,96,0.12)] px-3 py-1.5 text-[12px] text-[#ffe0e7] disabled:cursor-not-allowed disabled:opacity-60"
                    >
                      {{ tag }}
                      <X class="h-3.5 w-3.5" :stroke-width="2" />
                    </button>
                  </div>
                  <p v-else class="mt-3 text-[14px] text-[#d8dce7]/58">No tags assigned yet.</p>
                </div>

                <div v-if="detailTagSuggestions.length" class="mt-5">
                  <p class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Existing Game Tags</p>
                  <div class="mt-3 flex flex-wrap gap-2">
                    <button
                      v-for="tag in detailTagSuggestions"
                      :key="`detail-suggestion-${tag}`"
                      type="button"
                      @click="toggleDetailDraftTag(tag)"
                      :disabled="detailFormDisabled"
                      class="cursor-pointer rounded-full border border-[rgba(126,200,227,0.14)] bg-[rgba(126,200,227,0.08)] px-3 py-1.5 text-[12px] text-[#d8dce7]/72 transition-all duration-200 hover:border-[rgba(126,200,227,0.24)] disabled:cursor-not-allowed disabled:opacity-60"
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
                    <p class="mt-2 text-[14px] leading-relaxed text-[#d8dce7]/58">Edit the stat gates that must be met before the item can be used.</p>
                  </div>
                  <button type="button" @click="addDetailRequirementRow" :disabled="detailFormDisabled" class="inline-flex cursor-pointer items-center gap-2 rounded-xl border border-[rgba(233,69,96,0.22)] bg-[rgba(233,69,96,0.12)] px-4 py-2.5 text-[13px] font-semibold text-[#ffe0e7] transition-all duration-200 hover:border-[rgba(233,69,96,0.38)] disabled:cursor-not-allowed disabled:opacity-60">
                    <Plus class="h-4 w-4" :stroke-width="2" />
                    Add Requirement
                  </button>
                </div>

                <div v-if="detailItemDraft.requiredAttributes.length" class="mt-5 space-y-3">
                  <article v-for="(requirement, index) in detailItemDraft.requiredAttributes" :key="`detail-requirement-${index}`" class="rounded-[1.2rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.62)] p-4">
                    <div class="grid gap-3 lg:grid-cols-[minmax(0,1fr)_180px_auto]">
                      <label class="block">
                        <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Attribute</span>
                        <select v-model="requirement.attribute_name" :disabled="detailFormDisabled" class="session-input mt-2 w-full rounded-xl border border-[rgba(126,200,227,0.12)] bg-[rgba(126,200,227,0.08)] px-3 py-2.5 text-[14px] text-[#f6f7fb] outline-none disabled:cursor-not-allowed disabled:opacity-65">
                          <option v-for="option in attributeOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
                        </select>
                      </label>
                      <label class="block">
                        <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Minimum</span>
                        <input v-model.number="requirement.min_value" :disabled="detailFormDisabled" type="number" class="session-input mt-2 w-full rounded-xl border border-[rgba(126,200,227,0.12)] bg-[rgba(126,200,227,0.08)] px-3 py-2.5 text-[14px] text-[#f6f7fb] outline-none disabled:cursor-not-allowed disabled:opacity-65" />
                      </label>
                      <div class="flex items-end justify-start lg:justify-end">
                        <button type="button" @click="removeDetailRequirementRow(index)" :disabled="detailFormDisabled" class="inline-flex cursor-pointer items-center rounded-xl border border-[rgba(248,113,113,0.2)] bg-[rgba(248,113,113,0.12)] px-3 py-2.5 text-[12px] font-semibold text-[#fecaca] transition-all duration-200 hover:border-[rgba(248,113,113,0.35)] disabled:cursor-not-allowed disabled:opacity-60">Remove</button>
                      </div>
                    </div>
                  </article>
                </div>

                <p v-else class="mt-5 text-[14px] text-[#d8dce7]/58">No requirements yet.</p>
              </article>

              <article class="rounded-[1.6rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(8,16,30,0.78)] p-5">
                <div class="flex flex-wrap items-center justify-between gap-3">
                  <div>
                    <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Modifiers</p>
                    <p class="mt-2 text-[14px] leading-relaxed text-[#d8dce7]/58">Tune the bonuses or penalties this item grants.</p>
                  </div>
                  <button type="button" @click="addDetailModifierRow" :disabled="detailFormDisabled" class="inline-flex cursor-pointer items-center gap-2 rounded-xl border border-[rgba(233,69,96,0.22)] bg-[rgba(233,69,96,0.12)] px-4 py-2.5 text-[13px] font-semibold text-[#ffe0e7] transition-all duration-200 hover:border-[rgba(233,69,96,0.38)] disabled:cursor-not-allowed disabled:opacity-60">
                    <Plus class="h-4 w-4" :stroke-width="2" />
                    Add Modifier
                  </button>
                </div>

                <div v-if="detailItemDraft.attributeModifiers.length" class="mt-5 space-y-3">
                  <article v-for="(modifier, index) in detailItemDraft.attributeModifiers" :key="`detail-modifier-${index}`" class="rounded-[1.2rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.62)] p-4">
                    <div class="grid gap-3 xl:grid-cols-[minmax(0,1fr)_160px_180px_auto]">
                      <label class="block">
                        <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Attribute</span>
                        <select v-model="modifier.attribute_name" :disabled="detailFormDisabled" class="session-input mt-2 w-full rounded-xl border border-[rgba(126,200,227,0.12)] bg-[rgba(126,200,227,0.08)] px-3 py-2.5 text-[14px] text-[#f6f7fb] outline-none disabled:cursor-not-allowed disabled:opacity-65">
                          <option v-for="option in attributeOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
                        </select>
                      </label>
                      <label class="block">
                        <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Value</span>
                        <input v-model.number="modifier.modifier_value" :disabled="detailFormDisabled" type="number" class="session-input mt-2 w-full rounded-xl border border-[rgba(126,200,227,0.12)] bg-[rgba(126,200,227,0.08)] px-3 py-2.5 text-[14px] text-[#f6f7fb] outline-none disabled:cursor-not-allowed disabled:opacity-65" />
                      </label>
                      <label class="block">
                        <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Mode</span>
                        <select v-model="modifier.is_percentage" :disabled="detailFormDisabled" class="session-input mt-2 w-full rounded-xl border border-[rgba(126,200,227,0.12)] bg-[rgba(126,200,227,0.08)] px-3 py-2.5 text-[14px] text-[#f6f7fb] outline-none disabled:cursor-not-allowed disabled:opacity-65">
                          <option :value="false">Flat</option>
                          <option :value="true">Percent</option>
                        </select>
                      </label>
                      <div class="flex items-end justify-start xl:justify-end">
                        <button type="button" @click="removeDetailModifierRow(index)" :disabled="detailFormDisabled" class="inline-flex cursor-pointer items-center rounded-xl border border-[rgba(248,113,113,0.2)] bg-[rgba(248,113,113,0.12)] px-3 py-2.5 text-[12px] font-semibold text-[#fecaca] transition-all duration-200 hover:border-[rgba(248,113,113,0.35)] disabled:cursor-not-allowed disabled:opacity-60">Remove</button>
                      </div>
                    </div>
                  </article>
                </div>

                <p v-else class="mt-5 text-[14px] text-[#d8dce7]/58">No modifiers yet.</p>
              </article>
            </section>
          </div>

          <div class="flex flex-col gap-4 border-t border-[rgba(126,200,227,0.1)] px-5 py-4 sm:px-6">
            <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
              <p class="max-w-[48rem] text-[13px] leading-relaxed text-[#d8dce7]/56">
                {{ confirmItemDeletion
                  ? 'Press delete again to confirm. The item will be removed from the compendium and from every character inventory and equipped slot.'
                  : 'Use edit mode to change fields, then save. Deletion is permanent for this game.' }}
              </p>

              <div class="flex flex-wrap gap-3">
                <button type="button" @click="deleteCurrentItem" :disabled="detailItemSubmitting || detailDeleteSubmitting" class="inline-flex cursor-pointer items-center gap-2 rounded-xl border border-[rgba(248,113,113,0.24)] bg-[rgba(248,113,113,0.12)] px-4 py-2.5 text-[13px] font-semibold text-[#fecaca] transition-all duration-200 hover:border-[rgba(248,113,113,0.38)] disabled:cursor-not-allowed disabled:opacity-60">
                  <Trash2 class="h-4 w-4" :stroke-width="2" />
                  {{ detailDeleteSubmitting ? 'Deleting...' : (confirmItemDeletion ? 'Confirm Delete' : 'Delete Item') }}
                </button>
                <button v-if="confirmItemDeletion" type="button" @click="confirmItemDeletion = false" :disabled="detailDeleteSubmitting" class="cursor-pointer rounded-xl border border-[rgba(126,200,227,0.16)] bg-[rgba(126,200,227,0.08)] px-4 py-2.5 text-[13px] font-semibold text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(126,200,227,0.3)] disabled:cursor-not-allowed disabled:opacity-60">
                  Cancel Delete
                </button>
              </div>
            </div>

            <div class="flex flex-wrap justify-end gap-3">
              <button type="button" @click="closeItemDetailModal" :disabled="detailItemSubmitting || detailDeleteSubmitting" class="cursor-pointer rounded-xl border border-[rgba(126,200,227,0.16)] bg-[rgba(126,200,227,0.08)] px-4 py-2.5 text-[13px] font-semibold text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(126,200,227,0.3)] disabled:cursor-not-allowed disabled:opacity-60">
                Close
              </button>
              <button v-if="itemDetailMode === 'edit'" type="button" @click="saveDetailItem" :disabled="detailItemSubmitting || detailDeleteSubmitting" class="cursor-pointer rounded-xl border border-[rgba(233,69,96,0.28)] bg-[linear-gradient(135deg,rgba(233,69,96,0.9),rgba(194,49,82,0.9))] px-4 py-2.5 text-[13px] font-semibold text-white transition-all duration-200 hover:-translate-y-0.5 hover:shadow-[0_12px_30px_rgba(233,69,96,0.24)] disabled:cursor-not-allowed disabled:opacity-60">
                {{ detailItemSubmitting ? 'Saving...' : 'Save Changes' }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </Teleport>

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
            <p class="mt-3 max-w-[52rem] text-[14px] leading-relaxed text-[#d8dce7]/62">
              Upload an image, define the stats, and preview the final item card before it is created.
            </p>
          </div>

          <div class="grid min-h-0 flex-1 gap-6 overflow-hidden px-5 pb-5 pt-5 sm:px-6 lg:grid-cols-[minmax(0,1.5fr)_390px]">
            <section class="min-h-0 space-y-5 overflow-y-auto pr-1">
              <article class="rounded-[1.6rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(8,16,30,0.78)] p-5">
                <div class="grid gap-5 lg:grid-cols-[minmax(0,1fr)_minmax(320px,0.9fr)]">
                  <label class="block">
                    <span class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Name</span>
                    <input
                      v-model="itemDraft.name"
                      type="text"
                      :maxlength="ITEM_NAME_LIMIT"
                      :disabled="createItemSubmitting"
                      class="session-input mt-2 w-full rounded-[1.25rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.72)] px-4 py-3 text-[15px] text-[#f6f7fb] outline-none placeholder:text-[#7ec8e3]/30"
                    />
                    <p class="mt-2 text-[12px] text-[#d8dce7]/54">{{ itemNameLength }}/{{ ITEM_NAME_LIMIT }}</p>
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
                    :maxlength="ITEM_DESCRIPTION_LIMIT"
                    :disabled="createItemSubmitting"
                    class="session-input mt-2 w-full resize-y rounded-[1.25rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.72)] px-4 py-3 text-[15px] text-[#f6f7fb] outline-none placeholder:text-[#7ec8e3]/30"
                  ></textarea>
                  <p class="mt-2 text-[12px] text-[#d8dce7]/54">{{ itemDescriptionLength }}/{{ ITEM_DESCRIPTION_LIMIT }} · 1 line break max</p>
                </label>
              </article>

              <article class="rounded-[1.6rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(8,16,30,0.78)] p-5">
                <div class="flex flex-wrap items-center justify-between gap-3">
                  <div>
                    <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Image Upload</p>
                    <p class="mt-2 text-[14px] leading-relaxed text-[#d8dce7]/58">The item is created first, then the selected image is uploaded and linked automatically.</p>
                  </div>
                </div>

                <input ref="itemImageInputRef" type="file" accept="image/jpeg,image/png,image/webp" class="hidden" @change="handleItemImageSelected" />

                <div class="mt-5 flex flex-wrap gap-3">
                  <button type="button" @click="openItemImagePicker" :disabled="createItemSubmitting" class="cursor-pointer rounded-xl border border-[rgba(126,200,227,0.16)] bg-[rgba(126,200,227,0.08)] px-4 py-2.5 text-[13px] font-semibold text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(126,200,227,0.3)] disabled:cursor-not-allowed disabled:opacity-60">
                    {{ itemImageFile ? 'Replace Image' : 'Select Image' }}
                  </button>
                  <button v-if="itemImageFile" type="button" @click="clearItemImageSelection" :disabled="createItemSubmitting" class="cursor-pointer rounded-xl border border-[rgba(248,113,113,0.2)] bg-[rgba(248,113,113,0.12)] px-4 py-2.5 text-[13px] font-semibold text-[#fecaca] transition-all duration-200 hover:border-[rgba(248,113,113,0.35)] disabled:cursor-not-allowed disabled:opacity-60">
                    Remove Image
                  </button>
                </div>

                <div class="mt-4 rounded-[1.2rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.62)] px-4 py-3">
                  <p class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Selected File</p>
                  <p class="mt-2 text-[14px] text-[#f6f7fb]">{{ itemImageMetaLabel }}</p>
                  <p class="mt-2 text-[12px] text-[#d8dce7]/54">JPEG, PNG, or WebP up to 5MB.</p>
                </div>
              </article>

              <article class="rounded-[1.6rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(8,16,30,0.78)] p-5">
                <div>
                  <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Footprint</p>
                  <p class="mt-2 text-[14px] leading-relaxed text-[#d8dce7]/58">These values still control the inventory footprint, while the preview keeps the classic item card layout.</p>
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
                    <p class="mt-2 text-[13px] leading-relaxed text-[#d8dce7]/56">Adjust width and height to match the intended inventory footprint.</p>
                  </div>
                </div>
              </article>

              <article class="rounded-[1.6rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(8,16,30,0.78)] p-5">
                <div>
                  <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Tags</p>
                  <p class="mt-2 text-[14px] leading-relaxed text-[#d8dce7]/58">Create a new game tag or attach existing ones before the item is saved.</p>
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

                  <button type="button" @click="addDraftTagFromInput" :disabled="createItemSubmitting" class="inline-flex cursor-pointer items-center justify-center gap-2 self-end rounded-xl border border-[rgba(233,69,96,0.22)] bg-[rgba(233,69,96,0.12)] px-4 py-3 text-[13px] font-semibold text-[#ffe0e7] transition-all duration-200 hover:border-[rgba(233,69,96,0.38)] disabled:cursor-not-allowed disabled:opacity-60">
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
                    <p class="mt-2 text-[14px] leading-relaxed text-[#d8dce7]/58">Add explicit stat gates row by row.</p>
                  </div>
                  <button type="button" @click="addRequirementRow" :disabled="createItemSubmitting" class="inline-flex cursor-pointer items-center gap-2 rounded-xl border border-[rgba(233,69,96,0.22)] bg-[rgba(233,69,96,0.12)] px-4 py-2.5 text-[13px] font-semibold text-[#ffe0e7] transition-all duration-200 hover:border-[rgba(233,69,96,0.38)] disabled:cursor-not-allowed disabled:opacity-60">
                    <Plus class="h-4 w-4" :stroke-width="2" />
                    Add Requirement
                  </button>
                </div>

                <div v-if="itemDraft.requiredAttributes.length" class="mt-5 space-y-3">
                  <article v-for="(requirement, index) in itemDraft.requiredAttributes" :key="`requirement-${index}`" class="rounded-[1.2rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.62)] p-4">
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

                <p v-else class="mt-5 text-[14px] text-[#d8dce7]/58">No requirements yet.</p>
              </article>

              <article class="rounded-[1.6rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(8,16,30,0.78)] p-5">
                <div class="flex flex-wrap items-center justify-between gap-3">
                  <div>
                    <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Modifiers</p>
                    <p class="mt-2 text-[14px] leading-relaxed text-[#d8dce7]/58">Configure bonuses and penalties row by row.</p>
                  </div>
                  <button type="button" @click="addModifierRow" :disabled="createItemSubmitting" class="inline-flex cursor-pointer items-center gap-2 rounded-xl border border-[rgba(233,69,96,0.22)] bg-[rgba(233,69,96,0.12)] px-4 py-2.5 text-[13px] font-semibold text-[#ffe0e7] transition-all duration-200 hover:border-[rgba(233,69,96,0.38)] disabled:cursor-not-allowed disabled:opacity-60">
                    <Plus class="h-4 w-4" :stroke-width="2" />
                    Add Modifier
                  </button>
                </div>

                <div v-if="itemDraft.attributeModifiers.length" class="mt-5 space-y-3">
                  <article v-for="(modifier, index) in itemDraft.attributeModifiers" :key="`modifier-${index}`" class="rounded-[1.2rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.62)] p-4">
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

                <p v-else class="mt-5 text-[14px] text-[#d8dce7]/58">No modifiers yet.</p>
              </article>
            </section>

            <aside class="min-h-0 overflow-y-auto">
              <article class="rounded-[1.6rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(8,16,30,0.78)] p-5">
                <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Live Preview</p>
                <div class="mt-5 overflow-hidden rounded-[1.5rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.66)]">
                  <div class="flex h-64 items-center justify-center" :class="itemFrameClass(draftPreviewItem.rarity)">
                    <img v-if="resolvedItemImageUrl(draftPreviewItem)" :src="resolvedItemImageUrl(draftPreviewItem)" :alt="draftPreviewItem.name" class="h-full w-full object-cover" />
                    <span v-else class="font-[Cinzel] text-[40px] font-bold">{{ itemGlyph(draftPreviewItem) }}</span>
                  </div>
                </div>

                <div class="mt-5">
                  <div class="flex flex-wrap items-center gap-2">
                    <h3 class="min-w-0 flex-1 break-words whitespace-normal font-[Cinzel] text-[28px] font-bold leading-tight text-[#f6f7fb] line-clamp-2 [overflow-wrap:anywhere]">{{ draftPreviewItem.name }}</h3>
                    <span class="shrink-0 rounded-full border px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.16em]" :class="rarityBadgeClass(draftPreviewItem.rarity)">
                      {{ formatLabel(draftPreviewItem.rarity) }}
                    </span>
                  </div>
                  <p class="mt-3 break-words whitespace-pre-line text-[14px] leading-relaxed text-[#d8dce7]/62">{{ truncateText(draftPreviewItem.description, ITEM_DETAIL_DESCRIPTION_PREVIEW_LIMIT, 'The description preview appears here as you type.') }}</p>
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
              </article>
            </aside>
          </div>

          <div class="flex flex-col gap-4 border-t border-[rgba(126,200,227,0.1)] px-5 py-4 sm:flex-row sm:items-center sm:justify-between sm:px-6">
            <p class="max-w-[42rem] text-[13px] leading-relaxed text-[#d8dce7]/56">The selected image uploads immediately after the item record is created. If the upload fails, the item still exists and can be retried later.</p>

            <div class="flex flex-wrap gap-3">
              <button type="button" @click="closeCreateItemModal" :disabled="createItemSubmitting" class="cursor-pointer rounded-xl border border-[rgba(126,200,227,0.16)] bg-[rgba(126,200,227,0.08)] px-4 py-2.5 text-[13px] font-semibold text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(126,200,227,0.3)] disabled:cursor-not-allowed disabled:opacity-60">
                Cancel
              </button>
              <button type="button" @click="createItem" :disabled="createItemSubmitting" class="cursor-pointer rounded-xl border border-[rgba(233,69,96,0.28)] bg-[linear-gradient(135deg,rgba(233,69,96,0.9),rgba(194,49,82,0.9))] px-4 py-2.5 text-[13px] font-semibold text-white transition-all duration-200 hover:-translate-y-0.5 hover:shadow-[0_12px_30px_rgba(233,69,96,0.24)] disabled:cursor-not-allowed disabled:opacity-60">
                {{ createItemSubmitting ? 'Creating...' : 'Create Item' }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </Teleport>
  </section>
</template>