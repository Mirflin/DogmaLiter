<script setup>
import {
  ArrowLeft as ArrowLeftIcon,
  ArrowLeftRight as ArrowLeftRightIcon,
  Check as CheckIcon,
  ChevronRight as ChevronRightIcon,
  ContactRound as ContactRoundIcon,
  Eye as EyeIcon,
  FolderOpen as FolderOpenIcon,
  LayoutGrid as LayoutGridIcon,
  MessageSquareText as MessageSquareTextIcon,
  Package as PackageIcon,
  RefreshCw as RefreshCwIcon,
  Settings as SettingsIcon,
  TriangleAlert as TriangleAlertIcon,
  UserRound as UserRoundIcon,
  Users as UsersIcon,
  X as XIcon,
} from '@lucide/vue'
import { API_URL } from '@/api'
import SessionCharacterPickerModal from '@/components/session/SessionCharacterPickerModal.vue'
import SessionGMCharacterCreateModal from '@/components/session/SessionGMCharacterCreateModal.vue'
import SessionGMCharacterEditorModal from '@/components/session/SessionGMCharacterEditorModal.vue'
import SessionInventoryBoard from '@/components/session/SessionInventoryBoard.vue'
import SessionItemCompendium from '@/components/session/SessionItemCompendium.vue'
import GameSettingsModal from '@/components/GameSettingsModal.vue'
import DataTable from '@/components/DataTable.vue'
import { getErrorMessage, notify } from '@/notify'
import { useAuthStore } from '@/stores/auth'
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useGameSocket } from '@/composables/useGameSocket'
import { useRoute, useRouter } from 'vue-router'

const CHAT_POLL_INTERVAL = 15000
const CHARACTER_POLL_INTERVAL = 8000
const CHAT_MESSAGE_LIMIT = 40
const MAX_PORTRAIT_SIZE = 5 * 1024 * 1024
const ALLOWED_PORTRAIT_TYPES = ['image/jpeg', 'image/png', 'image/webp']
const tabIcons = {
  sheet: UserRoundIcon,
  inventory: PackageIcon,
  characters: ContactRoundIcon,
  players: UsersIcon,
  items: FolderOpenIcon,
  manage: SettingsIcon,
}

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()

const session = ref(null)
const activeCharacter = ref(null)
const activeCharacterId = ref('')
const activeTab = ref('sheet')
const loading = ref(true)
const refreshing = ref(false)
const characterLoading = ref(false)
const creatingCharacter = ref(false)
const chatSending = ref(false)
const pickerVisible = ref(false)
const chatCollapsed = ref(true)
const profileEditMode = ref(false)
const gmCharacterCreateVisible = ref(false)
const gmCharacterEditorVisible = ref(false)
const error = ref(null)
const characterError = ref(null)
const characterSaveError = ref('')
const characterSaveNotice = ref('')
const chatError = ref(null)
const chatDraft = ref('')
const chatMessagesRef = ref(null)
const chatInputRef = ref(null)
const portraitInputRef = ref(null)
const characterSavePending = ref(false)
const gmCharacterCreateSaving = ref(false)
const gmCharacterEditorSaving = ref(false)
const portraitFile = ref(null)
const portraitPreviewUrl = ref('')
const backstoryExpanded = ref(false)
const gmRosterSearch = ref('')
const gmRosterMemberFilter = ref('all')
const gmRosterRoleFilter = ref('all')
const characterForm = ref(createCharacterFormState())
const gmCharacterEditorTarget = ref(null)
const gmCharacterCreateError = ref('')
const gmCharacterEditorError = ref('')
const gmCharacterCreateRestorePicker = ref(false)

let chatPollHandle = null
let characterPollHandle = null
let managePollHandle = null
let tradePollHandle = null
let characterRequestId = 0
const MANAGE_POLL_INTERVAL = 15000
const TRADE_POLL_INTERVAL = 12000

const gameId = computed(() => route.params.id)
const game = computed(() => session.value?.game ?? null)
const viewer = computed(() => session.value?.viewer ?? {
  user_id: null,
  is_gm: false,
  can_create_character: false,
  owned_character_count: 0,
  character_limit: 0,
})
const isGM = computed(() => Boolean(viewer.value?.is_gm))
const playerNeedsCharacterSelection = computed(() => !isGM.value && !activeCharacterId.value)
const characters = computed(() => session.value?.characters ?? [])
const rosterCharacters = computed(() => {
  if (!isGM.value) return characters.value
  return characters.value.filter(character => character.user_id === viewer.value?.user_id)
})
const itemTags = computed(() => session.value?.item_tags ?? [])
const chatMessages = computed(() => session.value?.messages ?? [])
const selectedCharacterSummary = computed(() => characters.value.find(character => character.id === activeCharacterId.value) ?? null)
const characterSnapshot = computed(() => activeCharacter.value ?? selectedCharacterSummary.value ?? null)
const canEditCharacter = computed(() => {
  if (!characterSnapshot.value) return false
  return isGM.value || characterSnapshot.value.user_id === viewer.value?.user_id
})
const STANDARD_ATTR_KEYS = ['strength', 'dexterity', 'constitution', 'intelligence', 'wisdom', 'charisma']
const STANDARD_ATTR_SET = new Set(STANDARD_ATTR_KEYS)
// Which base attributes the game keeps enabled. Falls back to the legacy single
// boolean for payloads from an older backend.
const enabledStandardAttrs = computed(() => {
  const list = game.value?.enabled_standard_attrs
  if (Array.isArray(list)) return new Set(list)
  return game.value?.show_standard_attrs === false ? new Set() : new Set(STANDARD_ATTR_KEYS)
})
const attributesEnabled = computed(() => enabledStandardAttrs.value.size > 0)
const enableHealth = computed(() => Boolean(game.value?.enable_health))
const enableArmorClass = computed(() => Boolean(game.value?.enable_armor_class))
// A standard attribute that the GM turned off — hidden everywhere it would show.
function isStandardAttrDisabled(name) {
  return STANDARD_ATTR_SET.has(name) && !enabledStandardAttrs.value.has(name)
}
const disabledStandardAttrs = computed(() => STANDARD_ATTR_KEYS.filter((key) => !enabledStandardAttrs.value.has(key)))
const attributeCards = computed(() => {
  if (!characterSnapshot.value?.base_attributes) return []

  const definitions = [
    { key: 'strength', label: 'Strength' },
    { key: 'dexterity', label: 'Dexterity' },
    { key: 'constitution', label: 'Constitution' },
    { key: 'intelligence', label: 'Intelligence' },
    { key: 'wisdom', label: 'Wisdom' },
    { key: 'charisma', label: 'Charisma' },
  ]

  return definitions
    .filter(({ key }) => enabledStandardAttrs.value.has(key))
    .map(({ key, label }) => {
      const base = baseAttributeMap.value[key] ?? 0
      const total = effectiveAttributeMap.value[key] ?? base
      return { key, label, base, value: total, bonus: total - base, contributions: attributeContributions.value[key] ?? [] }
    })
})
const inventoryItems = computed(() => activeCharacter.value?.inventory ?? [])
const equipment = computed(() => activeCharacter.value?.equipment ?? [])
const customAttributes = computed(() => characterSnapshot.value?.custom_attributes ?? [])
const customAttributeCards = computed(() => customAttributes.value.map((attribute) => {
  const name = normalizeAttributeName(attribute?.name)
  const base = Number(attribute?.value) || 0
  const total = effectiveAttributeMap.value[name] ?? base
  return { id: attribute?.id, name: attribute?.name, base, value: total, bonus: total - base, contributions: attributeContributions.value[name] ?? [] }
}))
const baseAttributeMap = computed(() => {
  const map = {}
  const base = characterSnapshot.value?.base_attributes
  if (base) {
    for (const [key, value] of Object.entries(base)) {
      const name = normalizeAttributeName(key)
      if (isStandardAttrDisabled(name)) continue
      map[name] = Number(value) || 0
    }
  }
  for (const attribute of customAttributes.value) {
    const name = normalizeAttributeName(attribute?.name)
    if (name) {
      map[name] = Number(attribute?.value) || 0
    }
  }
  // Health and Armor Class flow through the same modifier pipeline so items can
  // affect them, but they are gated by their own per-game toggles.
  if (enableHealth.value) {
    map.health = Number(characterSnapshot.value?.max_health) || 0
  }
  if (enableArmorClass.value) {
    map.armor_class = Number(characterSnapshot.value?.armor_class) || 0
  }
  return map
})
const equipmentModifiers = computed(() => {
  const acc = {}
  for (const slot of equipment.value) {
    const modifiers = slot?.inventory_item?.item?.attribute_modifiers ?? []
    for (const modifier of modifiers) {
      const name = normalizeAttributeName(modifier?.attribute_name)
      if (!name) continue
      if (isStandardAttrDisabled(name)) continue
      if (!acc[name]) acc[name] = { flat: 0, percent: 0 }
      if (modifier?.is_percentage) acc[name].percent += Number(modifier?.modifier_value) || 0
      else acc[name].flat += Number(modifier?.modifier_value) || 0
    }
  }
  return acc
})
const effectiveAttributeMap = computed(() => {
  const result = {}
  const base = baseAttributeMap.value
  const modifiers = equipmentModifiers.value
  const names = new Set([...Object.keys(base), ...Object.keys(modifiers)])
  for (const name of names) {
    const baseValue = base[name] ?? 0
    const mod = modifiers[name] ?? { flat: 0, percent: 0 }
    result[name] = baseValue + mod.flat + Math.round(baseValue * mod.percent / 100)
  }
  return result
})
// Vitals: base value is GM-set, effective value includes item modifiers.
const baseMaxHealth = computed(() => Number(characterSnapshot.value?.max_health) || 0)
const effectiveMaxHealth = computed(() => effectiveAttributeMap.value.health ?? baseMaxHealth.value)
const currentHealth = computed(() => Number(characterSnapshot.value?.current_health) || 0)
const healthPercent = computed(() => {
  const max = effectiveMaxHealth.value
  if (max <= 0) return 0
  return Math.max(0, Math.min(100, (currentHealth.value / max) * 100))
})
const baseArmorClass = computed(() => Number(characterSnapshot.value?.armor_class) || 0)
const effectiveArmorClass = computed(() => effectiveAttributeMap.value.armor_class ?? baseArmorClass.value)
const armorClassBonus = computed(() => effectiveArmorClass.value - baseArmorClass.value)
const statTooltip = ref({ visible: false, left: 0, top: 0, label: '', base: 0, total: 0, contributions: [] })
function showStatTooltip(event, attribute) {
  const rect = event.currentTarget.getBoundingClientRect()
  const width = 240
  const estHeight = 96 + (attribute.contributions.length * 22)
  let top = rect.bottom + 8
  if (top + estHeight > window.innerHeight - 8) {
    top = rect.top - estHeight - 8
  }
  top = Math.max(8, top)
  const left = Math.max(8, Math.min(rect.left, window.innerWidth - width - 8))
  statTooltip.value = {
    visible: true,
    left,
    top,
    label: attribute.label || attribute.name,
    base: attribute.base,
    total: attribute.value,
    contributions: attribute.contributions,
  }
}
function hideStatTooltip() {
  statTooltip.value = { ...statTooltip.value, visible: false }
}
const attributeContributions = computed(() => {
  const map = {}
  for (const slot of equipment.value) {
    const itemName = slot?.inventory_item?.item?.name || 'Equipped item'
    const modifiers = slot?.inventory_item?.item?.attribute_modifiers ?? []
    for (const modifier of modifiers) {
      const name = normalizeAttributeName(modifier?.attribute_name)
      if (!name) continue
      if (isStandardAttrDisabled(name)) continue
      if (!map[name]) map[name] = []
      map[name].push({ source: itemName, value: Number(modifier?.modifier_value) || 0, percent: Boolean(modifier?.is_percentage) })
    }
  }
  return map
})
function normalizeAttributeName(value) {
  return String(value || '')
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9\s_-]+/g, '')
    .replace(/[\s-]+/g, '_')
}
async function persistInventoryLayout(layout) {
  const characterId = activeCharacterId.value
  if (!characterId || !gameId.value) return

  try {
    const data = await auth.updateCharacterInventory(gameId.value, characterId, layout)
    if (data?.character && activeCharacter.value?.id === data.character.id) {
      activeCharacter.value = data.character
    }
  } catch (error) {
    notify.error({
      title: 'Inventory not saved',
      message: getErrorMessage(error, 'Failed to save inventory changes'),
    })
  }
}
async function handleInventoryItemUpdate(payload) {
  const characterId = activeCharacterId.value
  if (!characterId || !gameId.value || !payload?.id) return

  try {
    const data = await auth.updateInventoryItem(gameId.value, characterId, payload.id, {
      durability: payload.durability,
      max_durability: payload.max_durability,
    })
    if (data?.character && activeCharacter.value?.id === data.character.id) {
      activeCharacter.value = data.character
    }
  } catch (error) {
    notify.error({
      title: 'Item not updated',
      message: getErrorMessage(error, 'Failed to update item'),
    })
  }
}
async function handleInventoryItemSplit(inventoryItemId) {
  const characterId = activeCharacterId.value
  if (!characterId || !gameId.value || !inventoryItemId) return

  try {
    const data = await auth.splitInventoryItem(gameId.value, characterId, inventoryItemId)
    if (data?.character && activeCharacter.value?.id === data.character.id) {
      activeCharacter.value = data.character
    }
  } catch (error) {
    notify.error({
      title: 'Could not unstack',
      message: getErrorMessage(error, 'Failed to unstack item'),
    })
  }
}

async function handleInventoryItemDelete(inventoryItemId) {
  const characterId = activeCharacterId.value
  if (!characterId || !gameId.value || !inventoryItemId) return

  try {
    const data = await auth.deleteInventoryItem(gameId.value, characterId, inventoryItemId)
    if (data?.character && activeCharacter.value?.id === data.character.id) {
      activeCharacter.value = data.character
    }
    notify.success({ title: 'Item removed', message: 'The item was removed from the inventory.' })
  } catch (error) {
    notify.error({
      title: 'Item not removed',
      message: getErrorMessage(error, 'Failed to remove item'),
    })
  }
}
const characterPendingDeletion = ref(null)
const deletingCharacter = ref(false)
function canDeleteCharacter(character) {
  if (!character) return false
  return isGM.value || character.user_id === viewer.value?.user_id
}
function requestCharacterDeletion(character) {
  if (!canDeleteCharacter(character)) return
  characterPendingDeletion.value = character
}
function cancelCharacterDeletion() {
  if (deletingCharacter.value) return
  characterPendingDeletion.value = null
}
async function confirmCharacterDeletion() {
  const character = characterPendingDeletion.value
  if (!character || deletingCharacter.value) return

  deletingCharacter.value = true
  try {
    await auth.deleteGameCharacter(gameId.value, character.id)
    if (session.value?.characters) {
      session.value.characters = session.value.characters.filter((entry) => entry.id !== character.id)
    }
    if (activeCharacterId.value === character.id) {
      activeCharacterId.value = ''
      activeCharacter.value = null
    }
    notify.success({ title: 'Character deleted', message: `${character.name || 'Character'} was removed.` })
    characterPendingDeletion.value = null
  } catch (error) {
    notify.error({
      title: 'Failed to delete character',
      message: getErrorMessage(error, 'Failed to delete character'),
    })
  } finally {
    deletingCharacter.value = false
  }
}
const SHARED_ITEM_PREFIX = 'ITEMLINK::'
const sharedItem = ref(null)
function parseSharedItem(content) {
  if (typeof content !== 'string' || !content.startsWith(SHARED_ITEM_PREFIX)) return null
  try {
    return JSON.parse(content.slice(SHARED_ITEM_PREFIX.length))
  } catch {
    return null
  }
}
function openSharedItem(item) {
  if (item) sharedItem.value = item
}

function viewEquipmentItem(entry) {
  const item = entry?.item
  if (!item) return
  openSharedItem({
    name: item.name,
    rarity: item.rarity,
    category: item.category,
    description: item.description,
    grid_width: item.grid_width,
    grid_height: item.grid_height,
    equip_slot: item.equip_slot || null,
    image_id: item.image_id || null,
    required_attributes: item.required_attributes || [],
    attribute_modifiers: item.attribute_modifiers || [],
    quantity: entry.quantity,
    durability: entry.durability,
    max_durability: entry.max_durability,
    enchantment: entry.enchantment,
    owner: '',
  })
}
function closeSharedItem() {
  sharedItem.value = null
}
function formatAttrLabel(value) {
  return String(value || '').split('_').filter(Boolean).map((part) => part.charAt(0).toUpperCase() + part.slice(1)).join(' ')
}
async function shareInventoryItem(entry) {
  const item = entry?.item
  if (!item) return
  if (!game.value?.enable_chat) {
    notify.warning({ title: 'Chat is disabled', message: 'Enable chat in game settings to share items.' })
    return
  }

  const payload = {
    name: item.name,
    rarity: item.rarity,
    category: item.category,
    description: String(item.description || '').slice(0, 200),
    grid_width: item.grid_width,
    grid_height: item.grid_height,
    equip_slot: item.equip_slot || null,
    image_id: item.image_id || null,
    required_attributes: (item.required_attributes || []).map((requirement) => ({
      attribute_name: requirement.attribute_name,
      min_value: requirement.min_value,
    })),
    attribute_modifiers: (item.attribute_modifiers || []).map((modifier) => ({
      attribute_name: modifier.attribute_name,
      modifier_value: modifier.modifier_value,
      is_percentage: modifier.is_percentage,
    })),
    quantity: entry.quantity,
    durability: entry.durability,
    max_durability: entry.max_durability,
    enchantment: entry.enchantment,
    owner: characterSnapshot.value?.name || '',
  }

  const content = `${SHARED_ITEM_PREFIX}${JSON.stringify(payload)}`
  if (content.length > 2000) {
    notify.error({ title: 'Cannot share item', message: 'This item has too much data to share in chat.' })
    return
  }

  try {
    const data = await auth.sendGameChatMessage(gameId.value, content)
    if (session.value) {
      session.value = {
        ...session.value,
        messages: [...chatMessages.value, data.message].slice(-CHAT_MESSAGE_LIMIT),
      }
    }
    notify.success({ title: 'Item shared', message: 'The item was posted in chat.' })
    scrollChatToBottom()
  } catch (err) {
    notify.error({ title: 'Failed to share item', message: err.response?.data?.error || 'Failed to share item' })
  }
}
const inventoryWidth = computed(() => characterSnapshot.value?.inventory_width ?? 0)
const inventoryHeight = computed(() => characterSnapshot.value?.inventory_height ?? 0)
const inventoryCapacity = computed(() => inventoryWidth.value * inventoryHeight.value)
const occupiedInventoryCells = computed(() => inventoryItems.value.reduce((total, entry) => {
  const width = entry.item?.grid_width ?? 1
  const height = entry.item?.grid_height ?? 1
  return total + (width * height)
}, 0))
const gmRosterMembers = computed(() => {
  const members = game.value?.members ?? []
  const searchQuery = gmRosterSearch.value.trim().toLowerCase()

  return members
    .filter(member => {
      if (gmRosterMemberFilter.value !== 'all' && member.user_id !== gmRosterMemberFilter.value) {
        return false
      }

      if (gmRosterRoleFilter.value === 'all') {
        return true
      }

      if (gmRosterRoleFilter.value === 'gm_team') {
        return member.role === 'gm' || member.role === 'assistant_gm'
      }

      return member.role === gmRosterRoleFilter.value
    })
    .map(member => {
      const allCharacters = characters.value.filter(character => character.user_id === member.user_id)
      const matchesMember = !searchQuery || [member.username, formatRole(member.role)]
        .filter(Boolean)
        .some(value => value.toLowerCase().includes(searchQuery))

      const filteredCharacters = searchQuery
        ? allCharacters.filter(character => [
          character.name,
          character.backstory,
          character.owner?.username,
          member.username,
        ]
          .filter(Boolean)
          .some(value => value.toLowerCase().includes(searchQuery)))
        : allCharacters

      if (searchQuery && !matchesMember && !filteredCharacters.length) {
        return null
      }

      return {
        ...member,
        filteredCharacters,
        totalCharacterCount: allCharacters.length,
      }
    })
    .filter(Boolean)
})
const gmRosterVisibleCharacterCount = computed(() => gmRosterMembers.value.reduce((total, member) => total + member.filteredCharacters.length, 0))
const gmRosterHasFilters = computed(() => Boolean(
  gmRosterSearch.value.trim()
  || gmRosterMemberFilter.value !== 'all'
  || gmRosterRoleFilter.value !== 'all',
))
const currencyCards = computed(() => {
  const snapshot = characterSnapshot.value

  return [
    { key: 'gold', label: 'Gold', value: snapshot?.currency_gold ?? 0 },
    { key: 'silver', label: 'Silver', value: snapshot?.currency_silver ?? 0 },
    { key: 'copper', label: 'Bronze', value: snapshot?.currency_copper ?? 0 },
  ]
})
const currentPortraitUrl = computed(() => portraitPreviewUrl.value || avatarUrl(characterSnapshot.value?.portrait_id))
const rosterTitle = computed(() => (isGM.value ? 'GM Character Vault' : 'Available Characters'))
const rosterDescription = computed(() => {
  if (playerNeedsCharacterSelection.value) {
    return 'Choose an existing character or create one to unlock the rest of the session.'
  }

  if (isGM.value) {
    return 'Only your GM-owned characters live here. Use Configure to transfer them to players or other GMs.'
  }

  return 'Use this roster for fast switching. The entry modal uses the same list and creation flow.'
})
const tabs = computed(() => {
  const rosterTab = {
    id: 'characters',
    label: playerNeedsCharacterSelection.value ? 'Choose Character' : 'Roster',
    icon: 'characters',
    description: playerNeedsCharacterSelection.value
      ? 'Select or create a character before the rest of the session opens.'
      : (isGM.value
          ? 'Your GM-only character vault for staging, editing, and taking control of characters.'
          : 'Quick switching between characters available to this viewer.'),
  }

  if (playerNeedsCharacterSelection.value) {
    return [rosterTab]
  }

  const baseTabs = [
    { id: 'sheet', label: 'Character', icon: 'sheet', description: 'Portrait, identity, profile editing, and read-only character stats.' },
    { id: 'inventory', label: 'Inventory', icon: 'inventory', description: 'Reserved inventory workspace with currency and storage stats.' },
    rosterTab,
  ]

  if (isGM.value) {
    baseTabs.push(
      { id: 'players', label: 'Players', icon: 'players', description: 'GM roster view for members and their accessible characters.' },
      { id: 'items', label: 'Items', icon: 'items', description: 'Campaign compendium for item browsing, tagging, and creation.' },
      { id: 'manage', label: 'Manage', icon: 'manage', description: 'GM control panel: chat history, player roster, and game settings.' },
    )
  }

  return baseTabs
})
const activeTabMeta = computed(() => tabs.value.find(tab => tab.id === activeTab.value) ?? tabs.value[0] ?? null)

const showGameSettings = ref(false)
const managePanel = ref('players')
const isGameOwner = computed(() => (game.value?.members ?? []).some(member => member.user_id === viewer.value?.user_id && member.role === 'gm'))
const playerColumns = [
  { key: 'username', label: 'Player' },
  { key: 'role', label: 'Role', filterable: true, filterOptions: [{ value: 'gm', label: 'GM' }, { value: 'assistant_gm', label: 'Assistant GM' }, { value: 'player', label: 'Player' }] },
  { key: 'joined_at', label: 'Joined' },
  { key: 'characterCount', label: 'Characters', align: 'right', sortable: true },
  { key: 'actions', label: 'Actions', align: 'right' },
]
const chatColumns = [
  { key: 'time', label: 'Time' },
  { key: 'author', label: 'Author', filterable: true },
  { key: 'text', label: 'Message' },
]
const activityColumns = [
  { key: 'time', label: 'Time' },
  { key: 'player', label: 'Player', filterable: true },
  { key: 'character', label: 'Character', filterable: true },
  { key: 'action', label: 'Action', filterable: true },
  { key: 'details', label: 'Details' },
]
const manageActivity = ref([])
const manageActivityLoading = ref(false)
const manageActivityLoaded = ref(false)
const activityRows = computed(() => (manageActivity.value ?? []).map((entry) => ({
  id: entry.id,
  time: formatDateTime(entry.created_at),
  player: entry.user?.username || 'Unknown',
  character: entry.character_name || '—',
  action: entry.action || '',
  details: entry.details || '',
})))
const managePlayers = computed(() => (game.value?.members ?? []).map(member => ({
  ...member,
  characterCount: characters.value.filter(character => character.user_id === member.user_id).length,
})))
const manageChatMessages = ref([])
const manageChatLoading = ref(false)
const manageChatLoaded = ref(false)
const memberPendingRemoval = ref(null)
const removingMember = ref(false)
const chatHistoryRows = computed(() => (manageChatMessages.value ?? []).map((message) => {
  const shared = parseSharedItem(message.content)
  return {
    id: message.id,
    time: formatDateTime(message.created_at),
    author: message.user?.username || 'Unknown user',
    isItem: Boolean(shared),
    text: shared ? (shared.name || 'Shared item') : message.content,
  }
}))
function canRemoveMember(member) {
  return member?.role !== 'gm' && member?.user_id !== viewer.value?.user_id
}

async function loadManageChat({ silent = false } = {}) {
  if (manageChatLoading.value) return
  if (!silent || !manageChatLoaded.value) manageChatLoading.value = true
  try {
    const data = await auth.getGameChatMessages(gameId.value)
    manageChatMessages.value = data.messages || []
    manageChatLoaded.value = true
  } catch (err) {
    if (!silent) notify.error(err, 'Failed to load chat history')
  } finally {
    manageChatLoading.value = false
  }
}

async function loadManageActivity({ silent = false } = {}) {
  if (manageActivityLoading.value) return
  if (!silent || !manageActivityLoaded.value) manageActivityLoading.value = true
  try {
    const data = await auth.getGameActivity(gameId.value)
    manageActivity.value = data.activity || []
    manageActivityLoaded.value = true
  } catch (err) {
    if (!silent) notify.error(err, 'Failed to load activity log')
  } finally {
    manageActivityLoading.value = false
  }
}

async function refreshManageMembers({ silent = false } = {}) {
  try {
    const data = await auth.getGameSession(gameId.value)
    if (session.value && data?.game) {
      session.value = {
        ...session.value,
        game: { ...session.value.game, members: data.game.members ?? session.value.game?.members },
        characters: data.characters ?? session.value.characters,
      }
    }
  } catch (err) {
    if (!silent) notify.error(err, 'Failed to refresh players')
  }
}

function refreshManagePanel(panel = managePanel.value, options = {}) {
  if (panel === 'chat') return loadManageChat(options)
  if (panel === 'activity') return loadManageActivity(options)
  if (panel === 'players') return refreshManageMembers(options)
  return Promise.resolve()
}

const manageRefreshing = ref(false)
async function refreshCurrentManagePanel() {
  if (manageRefreshing.value) return
  manageRefreshing.value = true
  try {
    await refreshManagePanel(managePanel.value)
  } finally {
    manageRefreshing.value = false
  }
}
const gameItemsCache = ref([])
let gameItemsLoaded = false
const ACTIVITY_ITEM_ACTIONS = new Set(['Updated item', 'Removed item', 'Unstacked item'])
function activityItemName(row) {
  return ACTIVITY_ITEM_ACTIONS.has(row?.action) && row?.details ? row.details : ''
}
async function ensureGameItems() {
  if (gameItemsLoaded) return
  try {
    const data = await auth.getGameItems(gameId.value, { per_page: 200 })
    gameItemsCache.value = data.items || []
    gameItemsLoaded = true
  } catch {  }
}
async function openActivityItem(name) {
  if (!name) return
  await ensureGameItems()
  const match = gameItemsCache.value.find(item => String(item.name).toLowerCase() === String(name).toLowerCase())
  if (match) {
    openSharedItem({ ...match, owner: '' })
  } else {
    openSharedItem({ name, description: 'This item is no longer available in the compendium.', required_attributes: [], attribute_modifiers: [] })
  }
}
const viewMember = ref(null)
const viewMemberLoading = ref(false)
const viewMemberCharacters = computed(() => {
  if (!viewMember.value) return []
  return (characters.value ?? []).filter(character => character.user_id === viewMember.value.user_id)
})
const viewMemberMessages = computed(() => {
  if (!viewMember.value) return []
  return (manageChatMessages.value ?? [])
    .filter(message => (message.user_id || message.user?.id) === viewMember.value.user_id)
    .slice()
    .sort((a, b) => new Date(b.created_at) - new Date(a.created_at))
    .slice(0, 6)
    .map((message) => {
      const shared = parseSharedItem(message.content)
      return {
        id: message.id,
        time: formatDateTime(message.created_at),
        text: shared ? `Shared item: ${shared.name || 'item'}` : message.content,
        isItem: Boolean(shared),
      }
    })
})
async function openMemberView(member) {
  if (!member) return
  viewMember.value = member
  if (!manageChatLoaded.value && !manageChatLoading.value) {
    viewMemberLoading.value = true
    try {
      await loadManageChat({ silent: true })
    } finally {
      viewMemberLoading.value = false
    }
  }
}
function closeMemberView() {
  viewMember.value = null
}
const inventoryViewCharacter = ref(null)
const inventoryViewLoading = ref(false)
function buildEffectiveAttributeMap(character) {
  const base = {}
  const src = character?.base_attributes
  if (src) {
    for (const [key, value] of Object.entries(src)) {
      const name = normalizeAttributeName(key)
      if (isStandardAttrDisabled(name)) continue
      base[name] = Number(value) || 0
    }
  }
  for (const attribute of (character?.custom_attributes ?? [])) {
    const name = normalizeAttributeName(attribute?.name)
    if (name) base[name] = Number(attribute?.value) || 0
  }
  const modifiers = {}
  for (const slot of (character?.equipment ?? [])) {
    for (const modifier of (slot?.inventory_item?.item?.attribute_modifiers ?? [])) {
      const name = normalizeAttributeName(modifier?.attribute_name)
      if (!name) continue
      if (isStandardAttrDisabled(name)) continue
      if (!modifiers[name]) modifiers[name] = { flat: 0, percent: 0 }
      if (modifier?.is_percentage) modifiers[name].percent += Number(modifier?.modifier_value) || 0
      else modifiers[name].flat += Number(modifier?.modifier_value) || 0
    }
  }
  const result = {}
  for (const name of new Set([...Object.keys(base), ...Object.keys(modifiers)])) {
    const baseValue = base[name] ?? 0
    const mod = modifiers[name] ?? { flat: 0, percent: 0 }
    result[name] = baseValue + mod.flat + Math.round(baseValue * mod.percent / 100)
  }
  return result
}
const inventoryViewCurrency = computed(() => {
  const character = inventoryViewCharacter.value
  return [
    { key: 'gold', label: 'Gold', value: character?.currency_gold ?? 0 },
    { key: 'silver', label: 'Silver', value: character?.currency_silver ?? 0 },
    { key: 'copper', label: 'Bronze', value: character?.currency_copper ?? 0 },
  ]
})
const inventoryViewAttributes = computed(() => buildEffectiveAttributeMap(inventoryViewCharacter.value))
async function openCharacterInventory(character) {
  if (!character?.id) return
  inventoryViewLoading.value = true
  inventoryViewCharacter.value = { ...character, inventory: [], equipment: [] }
  try {
    const data = await auth.getGameCharacter(gameId.value, character.id)
    if (data?.character) inventoryViewCharacter.value = data.character
  } catch (err) {
    notify.error(err, 'Failed to load inventory')
    inventoryViewCharacter.value = null
  } finally {
    inventoryViewLoading.value = false
  }
}
function closeCharacterInventory() {
  inventoryViewCharacter.value = null
}
const inspectChoice = ref(null)
function openInspectChoice(character) {
  inspectChoice.value = character
}
function closeInspectChoice() {
  inspectChoice.value = null
}
function inspectCharacterSheet() {
  const character = inspectChoice.value
  closeInspectChoice()
  if (character) switchCharacter(character.id, { nextTab: 'sheet' })
}
function inspectCharacterInventory() {
  const character = inspectChoice.value
  closeInspectChoice()
  if (character) openCharacterInventory(character)
}
const showTradeModal = ref(false)
const tradePanel = ref('send')
const tradeIncoming = ref([])
const tradeOutgoing = ref([])
const tradeTargets = ref([])
const tradeRecipientId = ref('')
const tradeRecipientSearch = ref('')
const tradeRecipientOpen = ref(false)
const tradeSelectedIds = ref([])
const tradeBusy = ref(false)
const tradeLoading = ref(false)

const tradingEnabled = computed(() => game.value?.enable_item_trading !== false)
const incomingTradeCount = computed(() => tradeIncoming.value.length)
const tradeTargetOptions = computed(() => tradeTargets.value.filter(target => target.id !== activeCharacterId.value))
const tradeRecipientFiltered = computed(() => {
  const query = tradeRecipientSearch.value.trim().toLowerCase()
  if (!query) return tradeTargetOptions.value
  return tradeTargetOptions.value.filter(target => `${target.name} ${target.owner}`.toLowerCase().includes(query))
})
const tradeSelectedSet = computed(() => new Set(tradeSelectedIds.value))
const tradeOfferableItems = computed(() => {
  const equippedIds = new Set((activeCharacter.value?.equipment ?? []).map(entry => entry.inventory_item_id || entry.inventory_item?.id).filter(Boolean))
  return (activeCharacter.value?.inventory ?? []).filter(entry => !equippedIds.has(entry.id))
})

async function loadTrades({ silent = false } = {}) {
  if (tradeLoading.value) return
  if (!silent) tradeLoading.value = true
  try {
    const data = await auth.getTrades(gameId.value)
    tradeIncoming.value = data.incoming || []
    tradeOutgoing.value = data.outgoing || []
    tradeTargets.value = data.targets || []
  } catch (err) {
    if (!silent) notify.error(err, 'Failed to load trades')
  } finally {
    tradeLoading.value = false
  }
}

function openTradeModal(panel = 'send') {
  tradePanel.value = panel
  tradeRecipientId.value = ''
  tradeRecipientSearch.value = ''
  tradeRecipientOpen.value = false
  tradeSelectedIds.value = []
  showTradeModal.value = true
  loadTrades()
}
function selectTradeRecipient(target) {
  tradeRecipientId.value = target.id
  tradeRecipientSearch.value = `${target.name} — ${target.owner}`
  tradeRecipientOpen.value = false
}
function onRecipientInput() {
  tradeRecipientOpen.value = true
  tradeRecipientId.value = ''
}
function closeRecipientDropdown() {
  window.setTimeout(() => { tradeRecipientOpen.value = false }, 120)
}
function closeTradeModal() {
  if (tradeBusy.value) return
  showTradeModal.value = false
}
function toggleTradeItem(entryId) {
  if (tradeSelectedSet.value.has(entryId)) {
    tradeSelectedIds.value = tradeSelectedIds.value.filter(id => id !== entryId)
  } else {
    tradeSelectedIds.value = [...tradeSelectedIds.value, entryId]
  }
}

async function submitTrade() {
  if (!tradeRecipientId.value || !tradeSelectedIds.value.length || !activeCharacterId.value || tradeBusy.value) return
  tradeBusy.value = true
  try {
    await auth.createTrade(gameId.value, {
      from_character_id: activeCharacterId.value,
      to_character_id: tradeRecipientId.value,
      inventory_item_ids: tradeSelectedIds.value,
    })
    notify.success({ title: 'Offer sent', message: 'Your trade offer is waiting for a response.' })
    tradeSelectedIds.value = []
    tradeRecipientId.value = ''
    tradeRecipientSearch.value = ''
    await Promise.all([pollActiveCharacter(), loadTrades({ silent: true })])
    tradePanel.value = 'outgoing'
  } catch (err) {
    notify.error(err, 'Failed to send trade offer')
  } finally {
    tradeBusy.value = false
  }
}

async function acceptTradeOffer(offer) {
  if (tradeBusy.value) return
  tradeBusy.value = true
  try {
    await auth.acceptTrade(gameId.value, offer.id)
    notify.success({ title: 'Trade accepted', message: 'The items were added to your inventory.' })
    await Promise.all([pollActiveCharacter(), loadTrades({ silent: true })])
  } catch (err) {
    notify.error(err, 'Failed to accept trade')
  } finally {
    tradeBusy.value = false
  }
}

async function declineTradeOffer(offer) {
  if (tradeBusy.value) return
  tradeBusy.value = true
  const mine = offer.from_user_id === viewer.value?.user_id
  try {
    await auth.declineTrade(gameId.value, offer.id)
    notify.success({ title: mine ? 'Trade cancelled' : 'Trade declined' })
    await Promise.all([pollActiveCharacter(), loadTrades({ silent: true })])
  } catch (err) {
    notify.error(err, 'Failed to update trade')
  } finally {
    tradeBusy.value = false
  }
}

function startTradePolling() {
  stopTradePolling()
  loadTrades({ silent: true })
  tradePollHandle = window.setInterval(() => loadTrades({ silent: true }), TRADE_POLL_INTERVAL)
}
function stopTradePolling() {
  if (tradePollHandle) {
    clearInterval(tradePollHandle)
    tradePollHandle = null
  }
}

function startManagePolling() {
  stopManagePolling()
  managePollHandle = setInterval(() => {
    if (activeTab.value === 'manage' && isGM.value) refreshManagePanel(managePanel.value, { silent: true })
  }, MANAGE_POLL_INTERVAL)
}

function stopManagePolling() {
  if (managePollHandle) {
    clearInterval(managePollHandle)
    managePollHandle = null
  }
}

// Realtime: the server pushes invalidation signals over WebSocket; we react by
// refreshing the matching slice using the same functions polling already uses.
// Polling stays active as a fallback for when the socket is down.
const gameSocket = useGameSocket(() => gameId.value, handleRealtimeEvent)

function handleRealtimeEvent(type, data) {
  switch (type) {
    case 'characters_changed':
      // A character was created or deleted — refresh the roster (this is what
      // makes a newly created character appear in the GM's list immediately).
      loadSession({ preserveCharacter: true, promptSelection: false })
      break
    case 'character_updated':
      if (!data?.character_id || data.character_id === activeCharacterId.value) {
        pollActiveCharacter()
      }
      if (isGM.value && activeTab.value === 'manage') {
        refreshManagePanel(managePanel.value, { silent: true })
      }
      break
    case 'trades_changed':
      loadTrades({ silent: true })
      break
    case 'chat_message':
      refreshChatMessages()
      break
    case 'activity_changed':
      if (isGM.value && activeTab.value === 'manage') {
        refreshManagePanel(managePanel.value, { silent: true })
      }
      break
  }
}

const CLEAR_PERIODS = [
  { label: 'Older than 24h', hours: 24 },
  { label: 'Older than 7 days', hours: 168 },
  { label: 'Older than 30 days', hours: 720 },
  { label: 'Everything', hours: 0 },
]
const chatClearHours = ref(24)
const activityClearHours = ref(24)

async function clearChatHistory() {
  try {
    await auth.clearGameChat(gameId.value, chatClearHours.value)
    manageChatLoaded.value = false
    await loadManageChat()
    notify.success({ title: 'Chat cleared', message: 'Chat history was cleared.' })
  } catch (err) {
    notify.error(err, 'Failed to clear chat history')
  }
}

async function clearActivity() {
  try {
    await auth.clearGameActivity(gameId.value, activityClearHours.value)
    manageActivityLoaded.value = false
    await loadManageActivity()
    notify.success({ title: 'Activity cleared', message: 'Activity log was cleared.' })
  } catch (err) {
    notify.error(err, 'Failed to clear activity log')
  }
}

function requestRemoveMember(member) {
  if (!canRemoveMember(member)) return
  memberPendingRemoval.value = member
}

function cancelRemoveMember() {
  if (removingMember.value) return
  memberPendingRemoval.value = null
}

async function confirmRemoveMember() {
  const member = memberPendingRemoval.value
  if (!member || removingMember.value) return
  removingMember.value = true
  try {
    await auth.removeGameMember(gameId.value, member.user_id)
    if (session.value) {
      const nextMembers = (session.value.game?.members ?? []).filter(entry => entry.user_id !== member.user_id)
      const nextCharacters = (session.value.characters ?? []).filter(character => character.user_id !== member.user_id)
      session.value = {
        ...session.value,
        game: { ...session.value.game, members: nextMembers },
        characters: nextCharacters,
      }
    }
    if (activeCharacter.value && activeCharacter.value.user_id === member.user_id) {
      clearActiveCharacter()
    }
    notify.success({ title: 'Player removed', message: `${member.username} was removed from the game.` })
    memberPendingRemoval.value = null
  } catch (err) {
    notify.error(err, 'Failed to remove player')
  } finally {
    removingMember.value = false
  }
}
watch(managePanel, (panel) => {
  if (activeTab.value === 'manage') refreshManagePanel(panel)
})
watch(activeTab, (tab) => {
  if (tab === 'manage') refreshManagePanel()
})

function openGameSettings() {
  showGameSettings.value = true
}

function handleGameSettingsUpdated() {
  loadSession({ preserveCharacter: true, promptSelection: false })
}

function handleGameDeleted() {
  router.push('/games')
}

async function updateMemberRole(member, role) {
  if (!isGameOwner.value || !member) return
  try {
    await auth.updateGameMemberRole(gameId.value, member.user_id, role)
    if (session.value) {
      const nextMembers = (session.value.game?.members ?? []).map(entry => (entry.user_id === member.user_id ? { ...entry, role } : entry))
      session.value = { ...session.value, game: { ...session.value.game, members: nextMembers } }
    }
    notify.success({ title: 'Role updated', message: `${member.username} is now ${role === 'assistant_gm' ? 'a GM' : 'a player'}.` })
  } catch (err) {
    notify.error(err, 'Failed to update role')
  }
}

function avatarUrl(uploadId) {
  if (!uploadId) return null
  return `${API_URL}/api/uploads/${uploadId}`
}

function initials(value) {
  if (!value) return '?'
  return value
    .split(' ')
    .filter(Boolean)
    .slice(0, 2)
    .map(part => part.charAt(0).toUpperCase())
    .join('')
}

function formatDateTime(value) {
  if (!value) return '—'
  return new Intl.DateTimeFormat('en-US', {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(value))
}

function formatRole(role) {
  if (!role) return 'Player'
  return role
    .split('_')
    .filter(Boolean)
    .map(part => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ')
}

function rarityClasses(rarity) {
  const variants = {
    common: 'border-[rgba(255,255,255,0.32)] bg-[rgba(255,255,255,0.08)] text-[#f8fafc]',
    uncommon: 'border-[rgba(250,204,21,0.35)] bg-[rgba(202,138,4,0.16)] text-[#fde68a]',
    rare: 'border-[rgba(96,165,250,0.35)] bg-[rgba(37,99,235,0.16)] text-[#93c5fd]',
    epic: 'border-[rgba(192,132,252,0.35)] bg-[rgba(126,34,206,0.18)] text-[#e9d5ff]',
    masterwork: 'border-[rgba(251,146,60,0.35)] bg-[rgba(194,65,12,0.18)] text-[#fdba74]',
    legendary: 'border-[rgba(74,222,128,0.35)] bg-[rgba(21,128,61,0.18)] text-[#86efac]',
    unique: 'border-[rgba(248,113,113,0.4)] bg-[rgba(153,27,27,0.18)] text-[#fca5a5]',
  }

  return variants[rarity] ?? variants.common
}

function createCharacterFormState(snapshot = null) {
  return {
    name: snapshot?.name ?? '',
    backstory: snapshot?.backstory ?? '',
    gold: snapshot?.currency_gold ?? 0,
    silver: snapshot?.currency_silver ?? 0,
    copper: snapshot?.currency_copper ?? 0,
    currentHealth: snapshot?.current_health ?? 0,
  }
}

function normalizeCurrencyValue(value) {
  const parsed = Number.parseInt(value, 10)
  if (Number.isNaN(parsed)) return 0
  return Math.min(999999999, Math.max(0, parsed))
}

function normalizeHealthValue(value) {
  const parsed = Number.parseInt(value, 10)
  if (Number.isNaN(parsed)) return 0
  return Math.min(9999999, Math.max(0, parsed))
}

function resetPortraitSelection() {
  if (portraitPreviewUrl.value) {
    URL.revokeObjectURL(portraitPreviewUrl.value)
  }

  portraitPreviewUrl.value = ''
  portraitFile.value = null

  if (portraitInputRef.value) {
    portraitInputRef.value.value = ''
  }
}

function syncCharacterForm(snapshot) {
  characterForm.value = createCharacterFormState(snapshot)
  characterSaveError.value = ''
  characterSaveNotice.value = ''
  profileEditMode.value = false
  resetPortraitSelection()
}

function clearActiveCharacter() {
  activeCharacterId.value = ''
  activeCharacter.value = null
  characterError.value = null
  persistActiveCharacter('')
  syncCharacterForm(null)
}

function goBack() {
  router.push(`/games/${gameId.value}`)
}

function memberEmptyStateMessage(member) {
  return member?.role === 'player'
    ? 'No characters assigned to this player yet.'
    : 'No GM characters assigned to this member yet.'
}

function gmRosterEmptyStateMessage(member) {
  if (member?.totalCharacterCount && gmRosterHasFilters.value) {
    return 'No characters match the current search or filters for this member.'
  }

  return memberEmptyStateMessage(member)
}

function resetGMRosterFilters() {
  gmRosterSearch.value = ''
  gmRosterMemberFilter.value = 'all'
  gmRosterRoleFilter.value = 'all'
}

function findCharacterById(characterId) {
  if (!characterId) return null
  if (activeCharacter.value?.id === characterId) return activeCharacter.value
  return characters.value.find(character => character.id === characterId) ?? null
}

function openCharacterPicker() {
  if (!characters.value.length && !viewer.value?.can_create_character) return
  pickerVisible.value = true
}

function openGMCharacterCreateModal() {
  if (!isGM.value || !viewer.value?.can_create_character) return

  gmCharacterCreateRestorePicker.value = pickerVisible.value
  pickerVisible.value = false
  gmCharacterCreateError.value = ''
  gmCharacterCreateVisible.value = true
}

function closeGMCharacterCreateModal() {
  if (gmCharacterCreateSaving.value) return

  const shouldRestorePicker = Boolean(
    gmCharacterCreateRestorePicker.value
      && !activeCharacterId.value
      && (characters.value.length || viewer.value?.can_create_character),
  )

  gmCharacterCreateVisible.value = false
  gmCharacterCreateError.value = ''
  gmCharacterCreateRestorePicker.value = false

  if (shouldRestorePicker) {
    pickerVisible.value = true
  }
}

function requestCharacterCreation() {
  if (isGM.value) {
    openGMCharacterCreateModal()
    return
  }

  createCharacter()
}

function openProfileEditor() {
  if (!canEditCharacter.value) return
  characterSaveError.value = ''
  characterSaveNotice.value = ''
  profileEditMode.value = true
}

function openGMCharacterEditor(target = activeCharacterId.value) {
  if (!isGM.value) return

  const characterId = typeof target === 'string' ? target : target?.id
  const snapshot = typeof target === 'object' && target ? findCharacterById(target.id) ?? target : findCharacterById(characterId)
  if (!snapshot) return

  gmCharacterEditorTarget.value = snapshot
  gmCharacterEditorError.value = ''
  gmCharacterEditorVisible.value = true
}

function closeGMCharacterEditor() {
  if (gmCharacterEditorSaving.value) return
  gmCharacterEditorVisible.value = false
  gmCharacterEditorTarget.value = null
  gmCharacterEditorError.value = ''
}

function cancelProfileEditor() {
  syncCharacterForm(characterSnapshot.value)
}

function focusChatInput() {
  chatCollapsed.value = false
  nextTick(() => {
    chatInputRef.value?.focus()
    scrollChatToBottom()
  })
}

function stopChatPolling() {
  if (chatPollHandle) {
    window.clearInterval(chatPollHandle)
    chatPollHandle = null
  }
}

function startChatPolling() {
  stopChatPolling()
  if (!game.value?.enable_chat) return

  chatPollHandle = window.setInterval(() => {
    refreshChatMessages()
  }, CHAT_POLL_INTERVAL)
}

function activeCharacterStorageKey() {
  return `dl:active-character:${gameId.value}`
}

function readStoredCharacterId() {
  try {
    return localStorage.getItem(activeCharacterStorageKey()) || ''
  } catch {
    return ''
  }
}

function persistActiveCharacter(characterId) {
  try {
    if (characterId) {
      localStorage.setItem(activeCharacterStorageKey(), characterId)
    } else {
      localStorage.removeItem(activeCharacterStorageKey())
    }
  } catch {}
}

function characterSignature(character) {
  if (!character) return ''
  return JSON.stringify({
    inv: (character.inventory || []).map(i => [i.id, i.grid_x, i.grid_y, i.is_rotated, i.quantity, i.durability, i.max_durability, i.enchantment, i.item?.id, i.item?.updated_at]),
    eq: (character.equipment || []).map(e => [e.slot, e.inventory_item_id]),
    cur: [character.currency_gold, character.currency_silver, character.currency_copper],
    base: character.base_attributes,
    custom: (character.custom_attributes || []).map(a => [a.id, a.name, a.value]),
    name: character.name,
    backstory: character.backstory,
    portrait: character.portrait_id,
    iw: character.inventory_width,
    ih: character.inventory_height,
  })
}

function stopCharacterPolling() {
  if (characterPollHandle) {
    window.clearInterval(characterPollHandle)
    characterPollHandle = null
  }
}

function startCharacterPolling() {
  stopCharacterPolling()
  characterPollHandle = window.setInterval(() => {
    pollActiveCharacter()
  }, CHARACTER_POLL_INTERVAL)
}

async function pollActiveCharacter() {
  const characterId = activeCharacterId.value
  if (!characterId || profileEditMode.value || characterLoading.value) return

  try {
    const data = await auth.getGameCharacter(gameId.value, characterId)
    const next = data?.character
    if (!next || next.id !== activeCharacterId.value) return
    if (characterSignature(next) !== characterSignature(activeCharacter.value)) {
      mergeUpdatedCharacterIntoSession(next)
    }
  } catch {}
}

function isChatNearBottom() {
  const node = chatMessagesRef.value
  if (!node) return true
  return node.scrollHeight - node.scrollTop - node.clientHeight < 80
}

async function scrollChatToBottom() {
  await nextTick()
  if (!chatMessagesRef.value) return
  chatMessagesRef.value.scrollTop = chatMessagesRef.value.scrollHeight
}

function handlePortraitSelected(event) {
  const file = event.target.files?.[0]
  if (!file) return

  if (!ALLOWED_PORTRAIT_TYPES.includes(file.type)) {
    characterSaveError.value = 'Only JPEG, PNG, and WebP images are allowed'
    return
  }

  if (file.size > MAX_PORTRAIT_SIZE) {
    characterSaveError.value = 'Portrait file must be under 5MB'
    return
  }

  resetPortraitSelection()
  portraitFile.value = file
  portraitPreviewUrl.value = URL.createObjectURL(file)
  characterSaveError.value = ''
}

async function loadCharacter(characterId) {
  if (!characterId) {
    clearActiveCharacter()
    return
  }

  const requestId = ++characterRequestId
  characterLoading.value = true
  characterError.value = null

  try {
    const data = await auth.getGameCharacter(gameId.value, characterId)
    if (requestId !== characterRequestId) return
    activeCharacter.value = data.character
    syncCharacterForm(data.character)
  } catch (err) {
    if (requestId !== characterRequestId) return
    clearActiveCharacter()
    characterError.value = err.response?.data?.error || 'Failed to load the selected character'
  } finally {
    if (requestId === characterRequestId) {
      characterLoading.value = false
    }
  }
}

async function switchCharacter(characterId, { nextTab = 'sheet', prefetchedCharacter = null } = {}) {
  if (!characterId) {
    clearActiveCharacter()
    return
  }

  activeCharacterId.value = characterId
  persistActiveCharacter(characterId)
  activeTab.value = nextTab
  pickerVisible.value = false

  if (prefetchedCharacter) {
    activeCharacter.value = prefetchedCharacter
    characterError.value = null
    characterLoading.value = false
    syncCharacterForm(prefetchedCharacter)
    return
  }

  if (activeCharacter.value?.id === characterId) {
    characterError.value = null
    syncCharacterForm(activeCharacter.value)
    return
  }

  await loadCharacter(characterId)
}

function mergeCreatedCharacterIntoSession(character) {
  if (!session.value || !character) return

  const existingCharacters = session.value.characters ?? []
  const hasCharacter = existingCharacters.some(entry => entry.id === character.id)
  const currentOwnedCount = viewer.value?.owned_character_count ?? 0
  const ownedCountDelta = !hasCharacter && character.user_id === viewer.value?.user_id ? 1 : 0
  const nextOwnedCount = currentOwnedCount + ownedCountDelta
  const characterLimit = viewer.value?.character_limit ?? 0

  session.value = {
    ...session.value,
    characters: hasCharacter ? existingCharacters : [character, ...existingCharacters],
    viewer: {
      ...session.value.viewer,
      owned_character_count: nextOwnedCount,
      can_create_character: viewer.value?.is_gm || characterLimit < 0 || nextOwnedCount < characterLimit,
    },
  }
}

function mergeUpdatedCharacterIntoSession(character) {
  if (!session.value || !character) return

  const existingCharacters = session.value.characters ?? []
  const hasCharacter = existingCharacters.some(entry => entry.id === character.id)

  session.value = {
    ...session.value,
    characters: hasCharacter
      ? existingCharacters.map(entry => (entry.id === character.id ? { ...entry, ...character } : entry))
      : [character, ...existingCharacters],
  }

  if (activeCharacterId.value === character.id) {
    activeCharacter.value = character
  }
}

async function loadSession({ preserveCharacter = true, promptSelection = false } = {}) {
  const previousCharacterId = preserveCharacter ? activeCharacterId.value : ''
  error.value = null

  if (session.value) {
    refreshing.value = true
  } else {
    loading.value = true
  }

  try {
    const data = await auth.getGameSession(gameId.value)
    session.value = data

    if (!tabs.value.some(tab => tab.id === activeTab.value)) {
      activeTab.value = tabs.value[0]?.id ?? 'characters'
    }

    const availableCharacters = data.characters ?? []
    const candidateCharacterId = previousCharacterId || (preserveCharacter ? readStoredCharacterId() : '')
    const keepCurrentCharacter = Boolean(candidateCharacterId && availableCharacters.some(character => character.id === candidateCharacterId))

    if (promptSelection) {
      clearActiveCharacter()
      pickerVisible.value = Boolean(availableCharacters.length || data.viewer?.can_create_character)
    } else if (keepCurrentCharacter) {
      pickerVisible.value = false
      activeCharacterId.value = candidateCharacterId
      persistActiveCharacter(candidateCharacterId)
      if (activeCharacter.value?.id !== candidateCharacterId) {
        await loadCharacter(candidateCharacterId)
      }
    } else {
      clearActiveCharacter()
      pickerVisible.value = Boolean(availableCharacters.length || data.viewer?.can_create_character)
    }

    startChatPolling()

    if (data.messages?.length) {
      scrollChatToBottom()
    }
  } catch (err) {
    stopChatPolling()
    error.value = err.response?.data?.error || 'Failed to load the game session'
  } finally {
    loading.value = false
    refreshing.value = false
  }
}

async function createCharacter(payload = null, { silent = false } = {}) {
  if (creatingCharacter.value || !viewer.value?.can_create_character) return

  creatingCharacter.value = true

  try {
    const data = await auth.createGameCharacter(gameId.value, payload ?? undefined)
    if (data?.character) {
      mergeCreatedCharacterIntoSession(data.character)
      await switchCharacter(data.character.id, {
        nextTab: 'sheet',
        prefetchedCharacter: data.character,
      })
    }

    await loadSession({ preserveCharacter: true, promptSelection: false })
    return data?.character ?? null
  } catch (err) {
    if (!silent) {
      notify.error(err, 'Failed to create a new character')
      return null
    }

    throw err
  } finally {
    creatingCharacter.value = false
  }
}

async function saveGMCharacterCreate(payload) {
  if (!isGM.value || gmCharacterCreateSaving.value) return

  gmCharacterCreateSaving.value = true
  gmCharacterCreateError.value = ''
  let shouldClose = false

  try {
    const character = await createCharacter(payload, { silent: true })
    if (character) {
      notify.success({
        title: 'Character created',
        message: 'The new character was added to the session roster.',
      })
      shouldClose = true
    }
  } catch (err) {
    gmCharacterCreateError.value = err.response?.data?.error || 'Failed to create GM character'
  } finally {
    gmCharacterCreateSaving.value = false
    if (shouldClose) {
      closeGMCharacterCreateModal()
    }
  }
}

async function persistCharacterChanges(snapshot, payload, nextPortraitFile = null) {
  let latestCharacter = snapshot

  if (payload) {
    const data = await auth.updateGameCharacter(gameId.value, snapshot.id, payload)
    if (data?.character) {
      latestCharacter = data.character
      mergeUpdatedCharacterIntoSession(latestCharacter)
    }
  }

  if (nextPortraitFile) {
    const portraitResponse = await auth.uploadCharacterPortrait(gameId.value, snapshot.id, nextPortraitFile)
    if (portraitResponse?.character) {
      latestCharacter = portraitResponse.character
      mergeUpdatedCharacterIntoSession(latestCharacter)
    }
  }

  return latestCharacter
}

async function saveCharacterProfile() {
  if (!characterSnapshot.value || !canEditCharacter.value || characterSavePending.value) return

  const payload = {
    name: characterForm.value.name.trim(),
    backstory: characterForm.value.backstory.trim(),
    currency_gold: normalizeCurrencyValue(characterForm.value.gold),
    currency_silver: normalizeCurrencyValue(characterForm.value.silver),
    currency_copper: normalizeCurrencyValue(characterForm.value.copper),
  }

  // Players may adjust current health when the feature is enabled.
  if (enableHealth.value) {
    payload.current_health = normalizeHealthValue(characterForm.value.currentHealth)
  }

  if (!payload.name) {
    characterSaveError.value = 'Character name cannot be empty'
    characterSaveNotice.value = ''
    return
  }

  const snapshot = characterSnapshot.value
  const hasProfileChanges = payload.name !== (snapshot.name ?? '')
    || payload.backstory !== (snapshot.backstory ?? '')
    || payload.currency_gold !== (snapshot.currency_gold ?? 0)
    || payload.currency_silver !== (snapshot.currency_silver ?? 0)
    || payload.currency_copper !== (snapshot.currency_copper ?? 0)
    || (enableHealth.value && payload.current_health !== (snapshot.current_health ?? 0))

  if (!hasProfileChanges && !portraitFile.value) {
    profileEditMode.value = false
    characterSaveNotice.value = 'No changes to save'
    characterSaveError.value = ''
    return
  }

  characterSavePending.value = true
  characterSaveError.value = ''
  characterSaveNotice.value = ''

  try {
    const latestCharacter = await persistCharacterChanges(snapshot, hasProfileChanges ? payload : null, portraitFile.value)

    syncCharacterForm(latestCharacter)
    characterSaveNotice.value = 'Profile saved'
    notify.success({
      title: 'Profile saved',
      message: 'Character changes were updated.',
    })
  } catch (err) {
    characterSaveError.value = err.response?.data?.error || 'Failed to save character profile'
  } finally {
    characterSavePending.value = false
  }
}

async function saveGMCharacterEditor({ payload, portraitFile: nextPortraitFile }) {
  if (!isGM.value || !gmCharacterEditorTarget.value || gmCharacterEditorSaving.value) return

  gmCharacterEditorSaving.value = true
  gmCharacterEditorError.value = ''
  let shouldClose = false

  try {
    const latestCharacter = await persistCharacterChanges(gmCharacterEditorTarget.value, payload, nextPortraitFile)
    gmCharacterEditorTarget.value = latestCharacter

    await loadSession({ preserveCharacter: true, promptSelection: false })

    if (activeCharacterId.value === latestCharacter?.id) {
      syncCharacterForm(activeCharacter.value ?? latestCharacter)
    }

    notify.success({
      title: 'Character updated',
      message: 'GM changes were saved.',
    })
    shouldClose = true
  } catch (err) {
    gmCharacterEditorError.value = err.response?.data?.error || 'Failed to save GM character settings'
  } finally {
    gmCharacterEditorSaving.value = false
    if (shouldClose) {
      closeGMCharacterEditor()
    }
  }
}

async function refreshChatMessages() {
  if (!game.value?.enable_chat) return

  try {
    const keepPinnedToBottom = isChatNearBottom()
    const data = await auth.getGameChatMessages(gameId.value)
    if (!session.value) return

    session.value = {
      ...session.value,
      messages: data.messages ?? [],
    }

    if (keepPinnedToBottom) {
      scrollChatToBottom()
    }
  } catch {}
}

async function sendChatMessage() {
  const content = chatDraft.value.trim()
  if (!content || chatSending.value || !game.value?.enable_chat) return

  chatSending.value = true
  chatError.value = null

  try {
    const data = await auth.sendGameChatMessage(gameId.value, content)
    if (!session.value) return

    session.value = {
      ...session.value,
      messages: [...chatMessages.value, data.message].slice(-CHAT_MESSAGE_LIMIT),
    }
    chatDraft.value = ''
    scrollChatToBottom()
    nextTick(() => chatInputRef.value?.focus())
  } catch (err) {
    chatError.value = err.response?.data?.error || 'Failed to send the message'
  } finally {
    chatSending.value = false
  }
}

watch(activeCharacterId, () => {
  backstoryExpanded.value = false

  if (!activeCharacterId.value) {
    syncCharacterForm(null)
  }
})

watch(
  tabs,
  nextTabs => {
    if (!nextTabs.some(tab => tab.id === activeTab.value)) {
      activeTab.value = nextTabs[0]?.id ?? 'characters'
    }
  },
  { immediate: true },
)

onMounted(() => {
  loadSession({ preserveCharacter: true, promptSelection: false })
  startCharacterPolling()
  startManagePolling()
  startTradePolling()
  gameSocket.connect()
})

onBeforeUnmount(() => {
  stopChatPolling()
  stopCharacterPolling()
  stopManagePolling()
  stopTradePolling()
  gameSocket.disconnect()
  resetPortraitSelection()
})
</script>

<template>
  <div class="session-stage min-h-screen w-screen overflow-x-hidden text-[#e8e8f0]">
    <div v-if="loading" class="flex min-h-screen flex-col items-center justify-center px-6 text-center">
      <div class="h-11 w-11 animate-spin rounded-full border-2 border-[#e94560] border-t-transparent"></div>
      <p class="mt-4 text-[14px] text-[#7ec8e3]/45">Loading session workspace...</p>
    </div>

    <div v-else-if="error" class="flex min-h-screen flex-col items-center justify-center px-6 text-center">
      <TriangleAlertIcon class="mb-4 h-16 w-16 text-[#e94560]" :stroke-width="1.5" />
      <p class="mb-5 text-[16px] font-semibold text-[#e94560]">{{ error }}</p>
      <div class="flex flex-wrap justify-center gap-3">
        <button
          @click="loadSession({ preserveCharacter: false, promptSelection: true })"
          class="cursor-pointer rounded-xl border border-[rgba(126,200,227,0.18)] bg-[rgba(126,200,227,0.08)] px-5 py-2.5 text-[14px] text-[#e8e8f0] transition-all duration-200 hover:border-[rgba(126,200,227,0.35)]"
        >
          Retry
        </button>
        <button
          @click="router.push('/games')"
          class="cursor-pointer rounded-xl border border-[rgba(233,69,96,0.22)] bg-transparent px-5 py-2.5 text-[14px] text-[#7ec8e3] transition-all duration-200 hover:border-[#e94560] hover:text-[#e94560]"
        >
          Back to Games
        </button>
      </div>
    </div>

    <div v-else-if="game" class="min-h-screen">
      <SessionCharacterPickerModal
        :visible="pickerVisible"
        :characters="characters"
        :viewer="viewer"
        :creating="creatingCharacter"
        @select="switchCharacter"
        @create="requestCharacterCreation"
      />

      <SessionGMCharacterCreateModal
        :visible="gmCharacterCreateVisible"
        :saving="gmCharacterCreateSaving"
        :error="gmCharacterCreateError"
        :members="game.members ?? []"
        :viewer-user-id="viewer.user_id"
        @close="closeGMCharacterCreateModal"
        @save="saveGMCharacterCreate"
      />

      <SessionGMCharacterEditorModal
        :visible="gmCharacterEditorVisible"
        :character="gmCharacterEditorTarget"
        :saving="gmCharacterEditorSaving"
        :error="gmCharacterEditorError"
        :members="game.members ?? []"
        :disabled-standard-attrs="disabledStandardAttrs"
        :enable-health="enableHealth"
        :enable-armor-class="enableArmorClass"
        @close="closeGMCharacterEditor"
        @save="saveGMCharacterEditor"
      />

      <header class="session-header sticky top-0 z-30 border-b border-[rgba(126,200,227,0.08)] bg-[rgba(7,17,31,0.82)] backdrop-blur-xl">
        <div class="mx-auto flex max-w-[1920px] flex-col gap-4 px-4 py-4 pl-[4.75rem] sm:px-6 sm:pl-[5.75rem] lg:flex-row lg:items-center lg:justify-between xl:pl-[7rem]">
          <div class="flex min-w-0 items-center gap-3">
            <button
              @click="goBack"
              title="Exit session"
              class="flex h-10 w-10 shrink-0 cursor-pointer items-center justify-center rounded-xl border border-[rgba(126,200,227,0.14)] bg-[rgba(126,200,227,0.06)] text-[#e8e8f0]/70 transition-all duration-200 hover:border-[#e94560] hover:text-[#e94560]"
            >
              <ArrowLeftIcon class="h-5 w-5" :stroke-width="2" />
            </button>

            <div class="min-w-0">
              <div class="flex flex-wrap items-center gap-2">
                <h1 class="truncate font-[Cinzel] text-[18px] font-bold tracking-wide text-[#f6f7fb] sm:text-[22px]">{{ game.title }}</h1>
                <span
                  class="session-mode-pill rounded-full border px-2.5 py-1 text-[10px] font-semibold uppercase tracking-[0.18em]"
                  :class="isGM
                    ? 'border-[rgba(143,79,51,0.42)] bg-[rgba(143,79,51,0.18)] text-[#efc7a2]'
                    : 'border-[rgba(101,128,136,0.3)] bg-[rgba(101,128,136,0.14)] text-[#b6d0d7]'"
                >
                  {{ isGM ? 'GM Console' : 'Player View' }}
                </span>
              </div>
            </div>
          </div>

          <div class="flex flex-wrap items-center gap-3">
            <div class="session-count-pill rounded-full border border-[rgba(126,200,227,0.12)] bg-[rgba(15,15,35,0.64)] px-3 py-2 text-[12px] text-[#7ec8e3]/65">
              {{ game.members?.length ?? 0 }} / {{ game.max_players }} players
            </div>
          </div>
        </div>
      </header>

      <nav class="session-tab-rail" aria-label="Session tabs">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          :aria-label="tab.label"
          @click="activeTab = tab.id"
          class="session-tab-button relative flex h-14 w-14 cursor-pointer items-center justify-center rounded-[1.35rem] border transition-all duration-200"
          :class="activeTab === tab.id
            ? 'border-[rgba(197,138,56,0.42)] bg-[rgba(143,79,51,0.18)] text-[#fff4de] shadow-[0_14px_30px_rgba(61,37,20,0.32)]'
            : 'border-[rgba(98,120,128,0.12)] bg-[rgba(15,18,22,0.74)] text-[#d4c6ab]/72 hover:border-[rgba(197,138,56,0.24)] hover:bg-[rgba(197,138,56,0.08)] hover:text-[#fff4de]'"
        >
          <component :is="tabIcons[tab.icon] || UserRoundIcon" class="h-5 w-5" :stroke-width="1.8" />

          <span v-if="activeTab === tab.id" class="absolute -right-1 h-2.5 w-2.5 rounded-full bg-[#c58a38]"></span>
          <span class="session-tooltip">{{ tab.label }}</span>
        </button>
      </nav>

      <div class="mx-auto max-w-[1920px] session-content-shell" :class="{ 'chat-collapsed': chatCollapsed || !game?.enable_chat }">
        <main class="min-w-0">
          <article class="session-command-deck mb-6 rounded-[1.75rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(11,20,36,0.88)] px-4 py-4 shadow-[0_24px_60px_rgba(0,0,0,0.22)] sm:px-5">
            <div class="flex flex-wrap items-center justify-between gap-4">
              <div class="min-w-0">
                <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">{{ activeTabMeta?.label }}</p>
                <div class="mt-2 flex flex-wrap items-center gap-3">
                  <h2 class="truncate font-[Cinzel] text-[28px] font-bold text-[#f6f7fb] sm:text-[32px]">{{ characterSnapshot?.name || activeTabMeta?.label }}</h2>
                  <span class="session-owner-pill rounded-full border border-[rgba(126,200,227,0.14)] bg-[rgba(126,200,227,0.06)] px-3 py-1.5 text-[11px] uppercase tracking-[0.16em] text-[#8fd7ef]">
                    {{ characterSnapshot?.owner?.username || 'Choose character' }}
                  </span>
                </div>
                
              </div>

              <div class="flex flex-wrap items-center gap-3">
                <select
                  v-if="characters.length"
                  :value="activeCharacterId"
                  @change="switchCharacter($event.target.value, { nextTab: activeTab })"
                  class="session-input min-w-[14rem] rounded-2xl border border-[rgba(126,200,227,0.14)] bg-[rgba(7,17,31,0.72)] px-4 py-3 text-[14px] text-[#f6f7fb] outline-none transition-all duration-200 focus:border-[rgba(233,69,96,0.34)]"
                >
                  <option value="">Choose a character</option>
                  <option v-for="character in characters" :key="character.id" :value="character.id">
                    {{ character.name }} · {{ character.owner?.username }}
                  </option>
                </select>

                <button
                  @click="openCharacterPicker"
                  class="cursor-pointer rounded-xl border border-[rgba(233,69,96,0.24)] bg-[rgba(233,69,96,0.12)] px-4 py-2.5 text-[13px] font-semibold text-[#ffe0e7] transition-all duration-200 hover:border-[rgba(233,69,96,0.45)]"
                >
                  Open Picker
                </button>
              </div>
            </div>
          </article>

          <div v-if="characterError" class="session-banner mb-6 rounded-[1.5rem] border border-[rgba(233,69,96,0.28)] bg-[rgba(233,69,96,0.12)] px-5 py-4 text-[14px] text-[#ffb3c1]">
            {{ characterError }}
          </div>

          <section v-if="activeTab === 'sheet'" class="session-sheet-layout space-y-7">
            <article class="session-sheet-frame rounded-[2rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(11,20,36,0.88)] p-5 shadow-[0_24px_60px_rgba(0,0,0,0.22)] sm:p-6">
              <template v-if="characterSnapshot">
                <div class="session-sheet-grid grid gap-6 xl:grid-cols-[minmax(360px,430px)_minmax(0,1fr)]">
                  <article class="session-dossier-panel overflow-hidden rounded-[1.9rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(8,16,30,0.78)]">
                    <div class="flex h-[24rem] items-center justify-center bg-[linear-gradient(180deg,rgba(16,32,52,0.94),rgba(8,16,30,0.94))] sm:h-[28rem]">
                      <img
                        v-if="currentPortraitUrl"
                        :src="currentPortraitUrl"
                        :alt="characterSnapshot.name"
                        class="h-full w-full object-cover object-top"
                      />
                      <span v-else class="font-[Cinzel] text-[44px] font-bold text-[#ff8fa3]">{{ initials(characterSnapshot.name) }}</span>
                    </div>

                    <div class="border-t border-[rgba(126,200,227,0.08)] p-5">
                      <div class="flex items-start justify-between gap-3">
                        <div class="min-w-0">
                          <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Character</p>
                          <h3 class="mt-2 truncate font-[Cinzel] text-[30px] font-bold text-[#f6f7fb]">{{ characterSnapshot.name }}</h3>
                          <p class="mt-2 text-[13px] text-[#d8dce7]/62">{{ characterSnapshot.owner?.username || 'Unknown owner' }}</p>
                        </div>

                        <div v-if="characterLoading" class="inline-flex items-center gap-2 rounded-full border border-[rgba(126,200,227,0.16)] bg-[rgba(126,200,227,0.08)] px-3 py-1.5 text-[10px] uppercase tracking-[0.18em] text-[#8fd7ef]">
                          <span class="h-2 w-2 animate-pulse rounded-full bg-[#8fd7ef]"></span>
                          Loading
                        </div>
                      </div>

                      <div class="mt-5 grid gap-3 sm:grid-cols-3 xl:grid-cols-1 2xl:grid-cols-3">
                        <div
                          v-for="currency in currencyCards"
                          :key="currency.key"
                          class="rounded-2xl border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] px-4 py-3"
                        >
                          <p class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">{{ currency.label }}</p>
                          <p class="mt-2 text-[20px] font-semibold text-[#f6f7fb]">{{ currency.value }}</p>
                        </div>
                      </div>
                    </div>
                  </article>

                  <article class="session-profile-panel rounded-[1.9rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(8,16,30,0.78)] p-5 sm:p-6">
                    <div class="flex flex-wrap items-start justify-between gap-4">
                      <div>
                        <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Profile</p>
                      </div>

                      <div class="flex flex-wrap gap-3">
                        <button
                          v-if="isGM"
                          @click="openGMCharacterEditor(characterSnapshot)"
                          class="cursor-pointer rounded-xl border border-[rgba(197,138,56,0.24)] bg-[rgba(143,79,51,0.16)] px-4 py-2.5 text-[13px] font-semibold text-[#fff4de] transition-all duration-200 hover:border-[rgba(197,138,56,0.4)]"
                        >
                          GM Configure
                        </button>
                        <button
                          v-if="canEditCharacter && !profileEditMode"
                          @click="openProfileEditor"
                          class="cursor-pointer rounded-xl border border-[rgba(233,69,96,0.24)] bg-[rgba(233,69,96,0.12)] px-4 py-2.5 text-[13px] font-semibold text-[#ffe0e7] transition-all duration-200 hover:border-[rgba(233,69,96,0.42)]"
                        >
                          Edit Profile
                        </button>
                        <button
                          v-if="profileEditMode"
                          @click="cancelProfileEditor"
                          class="cursor-pointer rounded-xl border border-[rgba(126,200,227,0.16)] bg-[rgba(126,200,227,0.08)] px-4 py-2.5 text-[13px] font-semibold text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(126,200,227,0.3)]"
                        >
                          Cancel
                        </button>
                      </div>
                    </div>

                    <p v-if="!canEditCharacter" class="mt-5 rounded-[1.3rem] border border-[rgba(126,200,227,0.14)] bg-[rgba(126,200,227,0.06)] px-4 py-3 text-[13px] text-[#d8dce7]/68">
                      This character is read-only for your current access.
                    </p>

                    <div v-if="profileEditMode && canEditCharacter" class="mt-6 space-y-5">
                      <div class="session-profile-editor-grid grid gap-5 xl:grid-cols-[minmax(260px,320px)_minmax(0,1fr)]">
                        <div class="space-y-4 rounded-[1.5rem] border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] p-4">
                          <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Portrait</p>
                          <div class="overflow-hidden rounded-[1.4rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.66)]">
                            <div class="flex h-64 items-center justify-center bg-[linear-gradient(180deg,rgba(16,32,52,0.94),rgba(8,16,30,0.94))]">
                              <img v-if="currentPortraitUrl" :src="currentPortraitUrl" :alt="characterSnapshot.name" class="h-full w-full object-cover object-top" />
                              <span v-else class="font-[Cinzel] text-[34px] font-bold text-[#8fd7ef]">{{ initials(characterSnapshot.name) }}</span>
                            </div>
                          </div>

                          <input
                            ref="portraitInputRef"
                            type="file"
                            accept="image/jpeg,image/png,image/webp"
                            class="hidden"
                            @change="handlePortraitSelected"
                          />

                          <div class="flex flex-wrap gap-3">
                            <button
                              @click="portraitInputRef?.click()"
                              type="button"
                              class="cursor-pointer rounded-xl border border-[rgba(126,200,227,0.16)] bg-[rgba(126,200,227,0.08)] px-4 py-2.5 text-[13px] font-semibold text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(126,200,227,0.3)]"
                            >
                              {{ portraitFile ? 'Replace Portrait' : 'Upload Portrait' }}
                            </button>
                            <button
                              v-if="portraitFile"
                              @click="resetPortraitSelection"
                              type="button"
                              class="cursor-pointer rounded-xl border border-[rgba(233,69,96,0.18)] bg-[rgba(233,69,96,0.08)] px-4 py-2.5 text-[13px] font-semibold text-[#ffe0e7] transition-all duration-200 hover:border-[rgba(233,69,96,0.35)]"
                            >
                              Clear Selection
                            </button>
                          </div>
                          <p class="text-[12px] text-[#d8dce7]/54">JPEG, PNG, WebP. Up to 5MB.</p>
                        </div>

                        <div class="space-y-5">
                          <label class="block">
                            <span class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Name</span>
                            <input
                              v-model="characterForm.name"
                              type="text"
                              maxlength="100"
                              :disabled="characterSavePending"
                              class="session-input mt-2 w-full rounded-[1.25rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.72)] px-4 py-3 text-[15px] text-[#f6f7fb] outline-none transition-all duration-200 placeholder:text-[#7ec8e3]/35 focus:border-[rgba(233,69,96,0.34)] disabled:cursor-not-allowed disabled:opacity-60"
                            />
                          </label>

                          <label class="block">
                            <span class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Backstory</span>
                            <textarea
                              v-model="characterForm.backstory"
                              rows="6"
                              :disabled="characterSavePending"
                              class="session-input mt-2 w-full resize-none rounded-[1.25rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.72)] px-4 py-3 text-[15px] text-[#f6f7fb] outline-none transition-all duration-200 placeholder:text-[#7ec8e3]/35 focus:border-[rgba(233,69,96,0.34)] disabled:cursor-not-allowed disabled:opacity-60"
                            ></textarea>
                          </label>

                          <div>
                            <div class="flex items-center justify-between gap-3">
                              <span class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Currency</span>
                            </div>
                            <div class="mt-4 grid gap-3 sm:grid-cols-3">
                              <label class="rounded-[1.3rem] border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] px-4 py-3">
                                <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Gold</span>
                                <input v-model.number="characterForm.gold" type="number" min="0" class="session-input mt-3 pl-2 w-full border-0 bg-transparent p-0 text-[24px] font-semibold text-[#f6f7fb] outline-none" />
                              </label>
                              <label class="rounded-[1.3rem] border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] px-4 py-3">
                                <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Silver</span>
                                <input v-model.number="characterForm.silver" type="number" min="0" class="session-input mt-3 pl-2 w-full border-0 bg-transparent p-0 text-[24px] font-semibold text-[#f6f7fb] outline-none" />
                              </label>
                              <label class="rounded-[1.3rem] border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] px-4 py-3">
                                <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Bronze</span>
                                <input v-model.number="characterForm.copper" type="number" min="0" class="session-input mt-3 pl-2 w-full border-0 bg-transparent p-0 text-[24px] font-semibold text-[#f6f7fb] outline-none" />
                              </label>
                            </div>
                          </div>

                          <div v-if="enableHealth">
                            <span class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Health</span>
                            <label class="mt-4 block rounded-[1.3rem] border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] px-4 py-3">
                              <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Current HP (max {{ effectiveMaxHealth }})</span>
                              <input v-model.number="characterForm.currentHealth" type="number" min="0" :max="effectiveMaxHealth" class="session-input mt-3 pl-2 w-full border-0 bg-transparent p-0 text-[24px] font-semibold text-[#f6f7fb] outline-none" />
                            </label>
                          </div>
                        </div>
                      </div>

                      <div class="flex flex-wrap gap-3">
                        <button
                          @click="saveCharacterProfile"
                          :disabled="characterSavePending"
                          class="cursor-pointer rounded-xl border border-[rgba(233,69,96,0.28)] bg-[linear-gradient(135deg,rgba(233,69,96,0.9),rgba(194,49,82,0.9))] px-4 py-2.5 text-[13px] font-semibold text-white transition-all duration-200 hover:-translate-y-0.5 hover:shadow-[0_12px_30px_rgba(233,69,96,0.24)] disabled:cursor-not-allowed disabled:opacity-60"
                        >
                          {{ characterSavePending ? 'Saving...' : 'Save Profile' }}
                        </button>
                      </div>
                    </div>

                    <div v-else class="mt-6 space-y-5">
                      <div class="rounded-[1.5rem] border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] p-5">
                        <div class="flex items-center justify-between gap-3">
                          <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Backstory</p>
                          <button
                            v-if="characterSnapshot.backstory"
                            type="button"
                            @click="backstoryExpanded = !backstoryExpanded"
                            class="cursor-pointer rounded-full border border-[rgba(126,200,227,0.16)] bg-[rgba(126,200,227,0.08)] px-3 py-1.5 text-[11px] font-semibold uppercase tracking-[0.14em] text-[#d8f4ff] transition-all duration-200 hover:border-[rgba(126,200,227,0.32)]"
                          >
                            {{ backstoryExpanded ? 'Collapse' : 'Expand' }}
                          </button>
                        </div>
                        <p
                          class="mt-4 whitespace-pre-wrap break-words pr-1 text-[15px] leading-relaxed text-[#e8e8f0]/78 [overflow-wrap:anywhere] transition-all duration-200"
                          :class="backstoryExpanded
                            ? 'max-h-[18rem] overflow-y-auto pr-2 sm:max-h-[24rem]'
                            : 'max-h-[8.5rem] overflow-hidden'"
                        >
                          {{ characterSnapshot.backstory || 'This character starts as a blank record. Enter edit mode to shape the profile.' }}
                        </p>
                      </div>

                      <div class="grid gap-3 sm:grid-cols-3">
                        <div
                          v-for="currency in currencyCards"
                          :key="currency.key"
                          class="rounded-[1.3rem] border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] px-4 py-3"
                        >
                          <p class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">{{ currency.label }}</p>
                          <p class="mt-2 text-[24px] font-semibold text-[#f6f7fb]">{{ currency.value }}</p>
                        </div>
                      </div>

                      <div v-if="enableHealth || enableArmorClass" class="rounded-[1.5rem] border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] p-5">
                        <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Vitals</p>
                        <div class="mt-4 grid gap-3 sm:grid-cols-2">
                          <div v-if="enableHealth" class="rounded-[1.3rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.62)] px-4 py-4">
                            <div class="flex items-center justify-between">
                              <p class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Health</p>
                              <p class="text-[13px] font-semibold text-[#f6f7fb]">{{ currentHealth }} / {{ effectiveMaxHealth }}</p>
                            </div>
                            <div class="mt-3 h-2.5 w-full overflow-hidden rounded-full bg-[rgba(7,17,31,0.8)]">
                              <div class="h-full rounded-full bg-[linear-gradient(90deg,#ef4444,#f87171)] transition-all duration-300" :style="{ width: `${healthPercent}%` }"></div>
                            </div>
                          </div>
                          <div v-if="enableArmorClass" class="rounded-[1.3rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.62)] px-4 py-4">
                            <p class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Armor Class</p>
                            <div class="mt-2 flex items-baseline gap-2">
                              <p class="text-[24px] font-bold text-[#f6f7fb]">{{ effectiveArmorClass }}</p>
                              <span v-if="armorClassBonus" class="text-[12px] font-semibold" :class="armorClassBonus > 0 ? 'text-[#86efac]' : 'text-[#fca5a5]'">
                                {{ baseArmorClass }} ({{ armorClassBonus > 0 ? '+' : '' }}{{ armorClassBonus }})
                              </span>
                            </div>
                          </div>
                        </div>
                      </div>

                      <div v-if="attributesEnabled" class="rounded-[1.5rem] border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] p-5">
                        <div class="flex items-center justify-between gap-4">
                          <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Base Attributes</p>
                          <span class="text-[12px] text-[#d8dce7]/45">Read-only</span>
                        </div>
                        <div class="mt-4 grid gap-3 sm:grid-cols-2 2xl:grid-cols-3">
                          <div
                            v-for="attribute in attributeCards"
                            :key="attribute.key"
                            class="rounded-[1.3rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.62)] px-4 py-4"
                            @mouseenter="showStatTooltip($event, attribute)"
                            @mouseleave="hideStatTooltip"
                          >
                            <p class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">{{ attribute.label }}</p>
                            <div class="mt-2 flex items-baseline gap-2">
                              <p class="text-[24px] font-bold text-[#f6f7fb]">{{ attribute.value }}</p>
                              <span v-if="attribute.bonus" class="text-[12px] font-semibold" :class="attribute.bonus > 0 ? 'text-[#86efac]' : 'text-[#fca5a5]'">
                                {{ attribute.base }} ({{ attribute.bonus > 0 ? '+' : '' }}{{ attribute.bonus }})
                              </span>
                            </div>
                          </div>
                        </div>
                      </div>
                    </div>
                  </article>
                </div>

                <div class="session-sheet-support-grid mt-6 grid gap-6 xl:grid-cols-[minmax(0,1.2fr)_minmax(320px,0.8fr)]">
                  <article class="session-attribute-panel rounded-[1.75rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(8,16,30,0.78)] p-5">
                    <div class="flex items-center justify-between gap-4">
                      <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Custom Attributes</p>
                      <span class="text-[12px] text-[#e8e8f0]/45">{{ customAttributes.length }} tracked</span>
                    </div>

                    <div v-if="customAttributeCards.length" class="mt-4 grid gap-3 md:grid-cols-2">
                      <div
                        v-for="attribute in customAttributeCards"
                        :key="attribute.id"
                        class="rounded-2xl border border-[rgba(233,69,96,0.12)] bg-[rgba(233,69,96,0.08)] px-4 py-3"
                        @mouseenter="showStatTooltip($event, attribute)"
                        @mouseleave="hideStatTooltip"
                      >
                        <p class="text-[11px] uppercase tracking-[0.18em] text-[#ff8fa3]/60">{{ attribute.name }}</p>
                        <div class="mt-2 flex items-baseline gap-2">
                          <p class="text-[20px] font-semibold text-[#f6f7fb]">{{ attribute.value }}</p>
                          <span v-if="attribute.bonus" class="text-[12px] font-semibold" :class="attribute.bonus > 0 ? 'text-[#86efac]' : 'text-[#fca5a5]'">
                            {{ attribute.base }} ({{ attribute.bonus > 0 ? '+' : '' }}{{ attribute.bonus }})
                          </span>
                        </div>
                      </div>
                    </div>

                    <p v-else class="mt-4 text-[14px] text-[#d8dce7]/56">No custom attributes were configured for this character.</p>
                  </article>

                  <article class="session-equipment-panel rounded-[1.75rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(8,16,30,0.78)] p-5">
                    <div class="flex items-center justify-between gap-4">
                      <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Equipment</p>
                      <span class="text-[12px] text-[#e8e8f0]/45">{{ equipment.length }} slots filled</span>
                    </div>

                    <div v-if="equipment.length" class="mt-4 max-h-[12rem] space-y-3 overflow-y-auto pr-1">
                      <div
                        v-for="slot in equipment"
                        :key="`${slot.slot}-${slot.inventory_item_id}`"
                        @click="viewEquipmentItem(slot.inventory_item)"
                        class="cursor-pointer rounded-2xl border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] px-4 py-3 transition-all duration-200 hover:border-[rgba(126,200,227,0.3)] hover:bg-[rgba(126,200,227,0.1)]"
                      >
                        <div class="flex items-start justify-between gap-3">
                          <div>
                            <p class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">{{ formatRole(slot.slot) }}</p>
                            <p class="mt-2 text-[15px] font-semibold text-[#f6f7fb]">{{ slot.inventory_item?.item?.name || 'Unknown item' }}</p>
                          </div>
                          <span class="rounded-full px-2.5 py-1 text-[10px] font-semibold uppercase tracking-[0.14em]" :class="rarityClasses(slot.inventory_item?.item?.rarity)">
                            {{ slot.inventory_item?.item?.rarity || 'common' }}
                          </span>
                        </div>
                      </div>
                    </div>

                    <p v-else class="mt-4 text-[14px] text-[#d8dce7]/56">Nothing is equipped yet.</p>
                  </article>
                </div>
              </template>

              <div v-else class="flex min-h-[520px] flex-col items-center justify-center text-center">
                <h2 class="font-[Cinzel] text-[28px] font-bold text-[#f6f7fb]">Choose A Character First</h2>
                <p class="mt-3 max-w-[30rem] text-[14px] leading-relaxed text-[#d8dce7]/58">The session opens through the character picker. Select an existing character or create a new blank one with a random name.</p>
                <button
                  @click="openCharacterPicker"
                  class="mt-5 cursor-pointer rounded-xl border border-[rgba(233,69,96,0.22)] bg-[rgba(233,69,96,0.08)] px-5 py-3 text-[14px] font-semibold text-[#ffe0e7] transition-all duration-200 hover:border-[rgba(233,69,96,0.45)]"
                >
                  Open Character Picker
                </button>
              </div>
            </article>
          </section>

          <section v-else-if="activeTab === 'inventory'">
            <div v-if="activeCharacterId && (tradingEnabled || incomingTradeCount)" class="mb-4 flex flex-wrap items-center justify-between gap-3 rounded-[1.5rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(8,16,30,0.7)] px-5 py-3.5">
              <div class="flex items-center gap-2.5">
                <ArrowLeftRightIcon class="h-5 w-5 text-[#8fd7ef]" :stroke-width="2" />
                <div>
                  <p class="text-[14px] font-semibold text-[#f6f7fb]">Player Trading</p>
                  <p class="text-[12px] text-[#7ec8e3]/50">Offer items to another character and let them accept.</p>
                </div>
              </div>
              <button
                type="button"
                @click="openTradeModal(incomingTradeCount ? 'incoming' : 'send')"
                class="relative inline-flex cursor-pointer items-center gap-2 rounded-xl border border-[rgba(126,200,227,0.22)] bg-[rgba(126,200,227,0.1)] px-4 py-2.5 text-[13px] font-semibold text-[#8fd7ef] transition-all duration-200 hover:border-[rgba(126,200,227,0.45)]"
              >
                <ArrowLeftRightIcon class="h-4 w-4" :stroke-width="2" />
                Trade
                <span v-if="incomingTradeCount" class="ml-1 inline-flex h-5 min-w-[1.25rem] items-center justify-center rounded-full bg-[#e94560] px-1.5 text-[11px] font-bold text-white">{{ incomingTradeCount }}</span>
              </button>
            </div>
            <SessionInventoryBoard
              :character-name="characterSnapshot?.name || ''"
              :inventory-items="inventoryItems"
              :equipment="equipment"
              :currency-cards="currencyCards"
              :inventory-width="inventoryWidth"
              :inventory-height="inventoryHeight"
              :character-attributes="effectiveAttributeMap"
              :attributes-enabled="attributesEnabled"
              :disabled-standard-attrs="disabledStandardAttrs"
              :character-id="activeCharacterId"
              :can-edit="canEditCharacter"
              @persist="persistInventoryLayout"
              @update-item="handleInventoryItemUpdate"
              @delete-item="handleInventoryItemDelete"
              @share="shareInventoryItem"
              @split="handleInventoryItemSplit"
            />
          </section>

          <section v-else-if="activeTab === 'characters'" class="space-y-5">
            <article class="rounded-[2rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(11,20,36,0.88)] p-5 shadow-[0_24px_60px_rgba(0,0,0,0.22)] sm:p-6">
              <div class="flex flex-wrap items-start justify-between gap-4">
                <div>
                  <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Character Switcher</p>
                  <h3 class="mt-2 font-[Cinzel] text-[30px] font-bold text-[#f6f7fb]">{{ rosterTitle }}</h3>
                </div>

                <div class="flex flex-wrap gap-3">
                  <button
                    v-if="viewer.can_create_character"
                    @click="requestCharacterCreation"
                    :disabled="creatingCharacter"
                    class="cursor-pointer rounded-xl border border-[rgba(233,69,96,0.24)] bg-[rgba(233,69,96,0.12)] px-4 py-2.5 text-[13px] font-semibold text-[#ffe0e7] transition-all duration-200 hover:border-[rgba(233,69,96,0.45)] disabled:cursor-not-allowed disabled:opacity-60"
                  >
                    {{ creatingCharacter ? 'Creating...' : (isGM ? 'Create GM Character' : 'Create New Character') }}
                  </button>
                </div>
              </div>
            </article>

            <div class="grid gap-4 md:grid-cols-2 2xl:grid-cols-3">
              <article
                v-for="character in rosterCharacters"
                :key="character.id"
                class="rounded-[1.8rem] border p-5 transition-all duration-200"
                :class="character.id === activeCharacterId
                  ? 'border-[rgba(233,69,96,0.5)] bg-[rgba(233,69,96,0.14)] shadow-[0_20px_50px_rgba(233,69,96,0.18)] ring-1 ring-[rgba(255,173,189,0.35)]'
                  : 'border-[rgba(126,200,227,0.12)] bg-[rgba(11,20,36,0.88)] shadow-[0_20px_50px_rgba(0,0,0,0.18)]'"
              >
                <div class="flex items-start justify-between gap-4">
                  <div class="flex min-w-0 items-center gap-4">
                    <div class="flex h-14 w-14 items-center justify-center overflow-hidden rounded-2xl border border-[rgba(126,200,227,0.12)] bg-[rgba(126,200,227,0.08)]">
                      <img v-if="avatarUrl(character.portrait_id)" :src="avatarUrl(character.portrait_id)" :alt="character.name" class="h-full w-full object-cover" />
                      <span v-else class="font-[Cinzel] text-[20px] font-bold text-[#8fd7ef]">{{ initials(character.name) }}</span>
                    </div>

                    <div class="min-w-0">
                      <h3 class="truncate font-[Cinzel] text-[24px] font-bold text-[#f6f7fb]">{{ character.name }}</h3>
                      <p class="mt-1 text-[13px] text-[#7ec8e3]/58">{{ character.owner?.username }}</p>
                    </div>
                  </div>

                  <div class="flex shrink-0 flex-col items-end gap-2">
                    <span
                      v-if="character.id === activeCharacterId"
                      class="rounded-full border border-[rgba(255,173,189,0.34)] bg-[rgba(233,69,96,0.2)] px-2.5 py-1 text-[10px] font-semibold uppercase tracking-[0.16em] text-[#ffe0e7]"
                    >
                      Active
                    </span>
                    <span class="rounded-full border border-[rgba(126,200,227,0.14)] bg-[rgba(126,200,227,0.08)] px-2.5 py-1 text-[10px] font-semibold uppercase tracking-[0.16em] text-[#8fd7ef]">
                      {{ character.inventory_width }}x{{ character.inventory_height }}
                    </span>
                  </div>
                </div>

                <p class="mt-4 line-clamp-3 text-[14px] leading-relaxed text-[#d8dce7]/68">
                  {{ character.backstory || 'Blank dossier. Select this character to continue editing and session play.' }}
                </p>

                <div class="mt-6 flex flex-wrap gap-3">
                  <button
                    @click="switchCharacter(character.id, { nextTab: activeTab })"
                    :disabled="character.id === activeCharacterId"
                    class="rounded-xl border px-4 py-2.5 text-[13px] font-semibold transition-all duration-200 disabled:cursor-not-allowed"
                    :class="character.id === activeCharacterId
                      ? 'border-[rgba(255,173,189,0.34)] bg-[rgba(233,69,96,0.2)] text-[#fff2f5]'
                      : 'cursor-pointer border-[rgba(233,69,96,0.24)] bg-[rgba(233,69,96,0.12)] text-[#ffe0e7] hover:border-[rgba(233,69,96,0.45)]'"
                  >
                    {{ character.id === activeCharacterId ? 'Active Character' : 'Make Active' }}
                  </button>
                  <button
                    @click="switchCharacter(character.id, { nextTab: 'sheet' })"
                    class="cursor-pointer rounded-xl border border-[rgba(126,200,227,0.16)] bg-[rgba(126,200,227,0.08)] px-4 py-2.5 text-[13px] font-semibold text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(126,200,227,0.3)]"
                  >
                    Open Character Tab
                  </button>
                  <button
                    v-if="isGM"
                    @click="openGMCharacterEditor(character)"
                    class="cursor-pointer rounded-xl border border-[rgba(197,138,56,0.24)] bg-[rgba(143,79,51,0.16)] px-4 py-2.5 text-[13px] font-semibold text-[#fff4de] transition-all duration-200 hover:border-[rgba(197,138,56,0.4)]"
                  >
                    Configure
                  </button>
                  <button
                    v-if="canDeleteCharacter(character)"
                    @click="requestCharacterDeletion(character)"
                    class="cursor-pointer rounded-xl border border-[rgba(248,113,113,0.24)] bg-[rgba(248,113,113,0.12)] px-4 py-2.5 text-[13px] font-semibold text-[#fecaca] transition-all duration-200 hover:border-[rgba(248,113,113,0.45)]"
                  >
                    Delete
                  </button>
                </div>
              </article>

              <article v-if="!rosterCharacters.length" class="md:col-span-2 2xl:col-span-3 rounded-[2rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(11,20,36,0.88)] p-8 text-center">
                <h3 class="font-[Cinzel] text-[24px] font-bold text-[#f6f7fb]">{{ isGM ? 'No GM characters yet' : 'No characters available' }}</h3>
                <p class="mt-3 text-[14px] text-[#d8dce7]/58">
                  {{ isGM
                    ? 'Create a named character for your GM vault, then transfer it whenever the campaign needs it.'
                    : 'Create a character or choose one from the roster to enter the session.' }}
                </p>
              </article>
            </div>
          </section>

          <section v-else-if="activeTab === 'players' && isGM" class="w-full space-y-5">
            <article class="rounded-[2rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(11,20,36,0.88)] p-5 shadow-[0_24px_60px_rgba(0,0,0,0.22)] sm:p-6">
              <div class="flex items-center justify-between gap-4">
                <div>
                  <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Member Manager</p>
                  <h3 class="mt-2 font-[Cinzel] text-[30px] font-bold text-[#f6f7fb]">Campaign Roster</h3>
                </div>
                <span class="rounded-full border border-[rgba(126,200,227,0.14)] bg-[rgba(126,200,227,0.08)] px-3 py-1.5 text-[11px] uppercase tracking-[0.16em] text-[#8fd7ef]">
                  {{ game.members?.length ?? 0 }} members
                </span>
              </div>

              <div class="mt-6 rounded-[1.5rem] border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] p-4">
                <div class="flex flex-wrap items-end gap-3">
                  <label class="block min-w-[18rem] flex-1">
                    <span class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Search</span>
                    <input
                      v-model="gmRosterSearch"
                      type="text"
                      placeholder="Character, player, role, or backstory"
                      class="session-input mt-2 w-full rounded-[1.25rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.72)] px-4 py-3 text-[14px] text-[#f6f7fb] outline-none transition-all duration-200 placeholder:text-[#7ec8e3]/35 focus:border-[rgba(233,69,96,0.34)]"
                    />
                  </label>

                  <label class="block min-w-[14rem]">
                    <span class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Member</span>
                    <select
                      v-model="gmRosterMemberFilter"
                      class="session-input mt-2 w-full rounded-[1.25rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.72)] px-4 py-3 text-[14px] text-[#f6f7fb] outline-none transition-all duration-200 focus:border-[rgba(233,69,96,0.34)]"
                    >
                      <option value="all">All members</option>
                      <option v-for="member in game.members" :key="member.user_id" :value="member.user_id">
                        {{ member.username }}
                      </option>
                    </select>
                  </label>

                  <label class="block min-w-[13rem]">
                    <span class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Role</span>
                    <select
                      v-model="gmRosterRoleFilter"
                      class="session-input mt-2 w-full rounded-[1.25rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.72)] px-4 py-3 text-[14px] text-[#f6f7fb] outline-none transition-all duration-200 focus:border-[rgba(233,69,96,0.34)]"
                    >
                      <option value="all">All roles</option>
                      <option value="player">Players</option>
                      <option value="gm_team">GM Team</option>
                      <option value="assistant_gm">Assistant GMs</option>
                    </select>
                  </label>

                  <button
                    v-if="gmRosterHasFilters"
                    @click="resetGMRosterFilters"
                    class="cursor-pointer rounded-xl border border-[rgba(233,69,96,0.18)] bg-[rgba(233,69,96,0.08)] px-4 py-3 text-[12px] font-semibold uppercase tracking-[0.14em] text-[#ffe0e7] transition-all duration-200 hover:border-[rgba(233,69,96,0.35)]"
                  >
                    Reset
                  </button>
                </div>

              </div>

              <div class="mt-5 space-y-3">
                <div
                  v-for="member in gmRosterMembers"
                  :key="member.user_id"
                  class="rounded-[1.5rem] border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] p-4"
                >
                  <div class="flex flex-wrap items-start justify-between gap-4">
                    <div class="flex items-center gap-3">
                      <div class="flex h-12 w-12 items-center justify-center overflow-hidden rounded-2xl border border-[rgba(126,200,227,0.12)] bg-[rgba(126,200,227,0.08)]">
                        <img v-if="avatarUrl(member.avatar_id)" :src="avatarUrl(member.avatar_id)" :alt="member.username" class="h-full w-full object-cover" />
                        <span v-else class="font-[Cinzel] text-[18px] font-bold text-[#8fd7ef]">{{ initials(member.username) }}</span>
                      </div>

                      <div>
                        <p class="text-[15px] font-semibold text-[#f6f7fb]">{{ member.username }}</p>
                        <p class="mt-1 text-[12px] text-[#7ec8e3]/58">{{ formatRole(member.role) }} · joined {{ formatDateTime(member.joined_at) }}</p>
                      </div>
                    </div>

                    <div class="flex flex-wrap gap-2 text-[11px] text-[#d8dce7]/60">
                      <span class="rounded-full border border-[rgba(126,200,227,0.14)] px-2.5 py-1">{{ member.totalCharacterCount }} characters</span>
                    </div>
                  </div>

                  <div v-if="member.filteredCharacters.length" class="mt-5 grid gap-4 md:grid-cols-2 2xl:grid-cols-3">
                    <article
                      v-for="character in member.filteredCharacters"
                      :key="character.id"
                      class="rounded-[1.8rem] border p-5 transition-all duration-200"
                      :class="character.id === activeCharacterId
                        ? 'border-[rgba(233,69,96,0.5)] bg-[rgba(233,69,96,0.14)] shadow-[0_20px_50px_rgba(233,69,96,0.18)] ring-1 ring-[rgba(255,173,189,0.35)]'
                        : 'border-[rgba(126,200,227,0.12)] bg-[rgba(11,20,36,0.88)] shadow-[0_20px_50px_rgba(0,0,0,0.18)]'"
                    >
                      <div class="flex items-start justify-between gap-4">
                        <div class="flex min-w-0 items-center gap-4">
                          <div class="flex h-14 w-14 items-center justify-center overflow-hidden rounded-2xl border border-[rgba(126,200,227,0.12)] bg-[rgba(126,200,227,0.08)]">
                            <img v-if="avatarUrl(character.portrait_id)" :src="avatarUrl(character.portrait_id)" :alt="character.name" class="h-full w-full object-cover" />
                            <span v-else class="font-[Cinzel] text-[20px] font-bold text-[#8fd7ef]">{{ initials(character.name) }}</span>
                          </div>

                          <div class="min-w-0">
                            <h4 class="truncate font-[Cinzel] text-[24px] font-bold text-[#f6f7fb]">{{ character.name }}</h4>
                            <p class="mt-1 text-[13px] text-[#7ec8e3]/58">{{ character.owner?.username || member.username }}</p>
                          </div>
                        </div>

                        <div class="flex shrink-0 flex-col items-end gap-2">
                          <span
                            v-if="character.id === activeCharacterId"
                            class="rounded-full border border-[rgba(255,173,189,0.34)] bg-[rgba(233,69,96,0.2)] px-2.5 py-1 text-[10px] font-semibold uppercase tracking-[0.16em] text-[#ffe0e7]"
                          >
                            Active
                          </span>
                          <span class="rounded-full border border-[rgba(126,200,227,0.14)] bg-[rgba(126,200,227,0.08)] px-2.5 py-1 text-[10px] font-semibold uppercase tracking-[0.16em] text-[#8fd7ef]">
                            {{ character.inventory_width }}x{{ character.inventory_height }}
                          </span>
                        </div>
                      </div>

                      <p class="mt-4 line-clamp-3 text-[14px] leading-relaxed text-[#d8dce7]/68">
                        {{ character.backstory || 'Blank dossier.' }}
                      </p>

                      <div class="mt-6 flex flex-wrap gap-3">
                        <button
                          @click="openInspectChoice(character)"
                          class="inline-flex cursor-pointer items-center gap-2 rounded-xl border border-[rgba(126,200,227,0.16)] bg-[rgba(126,200,227,0.08)] px-4 py-2.5 text-[13px] font-semibold text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(126,200,227,0.32)]"
                        >
                          <EyeIcon class="h-4 w-4" :stroke-width="2" />
                          Inspect
                        </button>
                        <button
                          @click="openGMCharacterEditor(character)"
                          class="cursor-pointer rounded-xl border border-[rgba(197,138,56,0.24)] bg-[rgba(143,79,51,0.16)] px-4 py-2.5 text-[13px] font-semibold text-[#fff4de] transition-all duration-200 hover:border-[rgba(197,138,56,0.4)]"
                        >
                          Configure
                        </button>
                        <button
                          @click="requestCharacterDeletion(character)"
                          class="cursor-pointer rounded-xl border border-[rgba(248,113,113,0.24)] bg-[rgba(248,113,113,0.12)] px-4 py-2.5 text-[13px] font-semibold text-[#fecaca] transition-all duration-200 hover:border-[rgba(248,113,113,0.45)]"
                        >
                          Delete
                        </button>
                      </div>
                    </article>
                  </div>

                  <p v-else class="mt-4 text-[13px] text-[#d8dce7]/52">{{ gmRosterEmptyStateMessage(member) }}</p>
                </div>

                <article v-if="!gmRosterMembers.length" class="rounded-[1.75rem] border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] p-8 text-center">
                  <h3 class="font-[Cinzel] text-[24px] font-bold text-[#f6f7fb]">
                    {{ gmRosterHasFilters ? 'Nothing matched the current filters' : 'No members available yet' }}
                  </h3>
                  <p class="mt-3 text-[14px] text-[#d8dce7]/58">
                    {{ gmRosterHasFilters
                      ? 'Adjust the search or filters to broaden the roster view.'
                      : 'Members will appear here once players or assistant GMs join the campaign.' }}
                  </p>
                </article>
              </div>
            </article>
          </section>

          <SessionItemCompendium v-else-if="activeTab === 'items' && isGM" :characters="characters" :available-tags="itemTags" :game-id="gameId" :disabled-standard-attrs="disabledStandardAttrs" :enable-health="enableHealth" :enable-armor-class="enableArmorClass" @created="loadSession({ preserveCharacter: true, promptSelection: false })" />

          <section v-else-if="activeTab === 'manage' && isGM" class="space-y-6">
            <article class="rounded-[2rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(11,20,36,0.88)] p-5 shadow-[0_24px_60px_rgba(0,0,0,0.22)] sm:p-6">
              <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">GM Control Panel</p>
              <h3 class="mt-2 font-[Cinzel] text-[30px] font-bold text-[#f6f7fb]">Manage Game</h3>

              <div class="mt-5 flex flex-wrap items-center justify-between gap-3">
                <div class="inline-flex flex-wrap gap-1.5 rounded-2xl border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.6)] p-1.5">
                  <div
                    v-for="panel in [{ id: 'players', label: 'Players' }, { id: 'chat', label: 'Chat History' }, { id: 'activity', label: 'Activity' }]"
                    :key="panel.id"
                    role="button"
                    tabindex="0"
                    @click="managePanel = panel.id"
                    @keydown.enter="managePanel = panel.id"
                    class="cursor-pointer select-none rounded-xl px-4 py-2 text-[13px] font-semibold transition-all duration-200"
                    :class="managePanel === panel.id
                      ? 'bg-[linear-gradient(135deg,rgba(233,69,96,0.95),rgba(194,49,82,0.95))] text-white shadow-[0_8px_20px_rgba(233,69,96,0.3)]'
                      : 'text-[#d8dce7]/70 hover:bg-[rgba(126,200,227,0.06)] hover:text-[#f6f7fb]'"
                  >
                    {{ panel.label }}
                  </div>
                </div>
                <div class="flex items-center gap-2">
                  <div
                    role="button"
                    tabindex="0"
                    @click="refreshCurrentManagePanel"
                    @keydown.enter="refreshCurrentManagePanel"
                    title="Refresh"
                    class="inline-flex cursor-pointer select-none items-center justify-center rounded-xl border border-[rgba(126,200,227,0.2)] bg-[rgba(126,200,227,0.08)] p-2.5 text-[#8fd7ef] transition-all duration-200 hover:border-[rgba(126,200,227,0.4)]"
                  >
                    <RefreshCwIcon class="h-4 w-4" :class="{ 'animate-spin': manageRefreshing }" :stroke-width="2" />
                  </div>
                  <button
                    type="button"
                    @click="openGameSettings"
                    class="inline-flex cursor-pointer items-center gap-2 rounded-xl border border-[rgba(126,200,227,0.2)] bg-[rgba(126,200,227,0.08)] px-4 py-2.5 text-[13px] font-semibold text-[#8fd7ef] transition-all duration-200 hover:border-[rgba(126,200,227,0.4)]"
                  >
                    <SettingsIcon class="h-4 w-4" :stroke-width="2" />
                    Game Settings
                  </button>
                </div>
              </div>
            </article>

            <!-- Players table -->
            <div v-show="managePanel === 'players'">
              <DataTable
                :columns="playerColumns"
                :rows="managePlayers"
                row-key="user_id"
                :search-keys="['username', 'role']"
                search-placeholder="Search players"
                :page-size="10"
                min-width="640px"
                empty-text="No members yet"
              >
                <template #cell-username="{ row }"><span class="font-semibold text-[#f6f7fb]">{{ row.username }}</span></template>
                <template #cell-role="{ row }"><span class="text-[#d8dce7]/70">{{ formatRole(row.role) }}</span></template>
                <template #cell-joined_at="{ row }"><span class="text-[#7ec8e3]/45">{{ formatDateTime(row.joined_at) }}</span></template>
                <template #cell-characterCount="{ row }"><span class="text-[#f6f7fb]">{{ row.characterCount }}</span></template>
                <template #cell-actions="{ row }">
                  <div class="flex flex-wrap justify-end gap-2">
                    <button type="button" @click="openMemberView(row)" class="inline-flex cursor-pointer items-center gap-1.5 rounded-lg border border-[rgba(126,200,227,0.2)] bg-[rgba(126,200,227,0.08)] px-3 py-1.5 text-[12px] font-semibold text-[#8fd7ef] transition-all duration-200 hover:border-[rgba(126,200,227,0.4)]">
                      <EyeIcon class="h-3.5 w-3.5" :stroke-width="2" /> View
                    </button>
                    <button v-if="isGameOwner && row.role === 'player'" type="button" @click="updateMemberRole(row, 'assistant_gm')" class="cursor-pointer rounded-lg border border-[rgba(197,138,56,0.28)] bg-[rgba(143,79,51,0.16)] px-3 py-1.5 text-[12px] font-semibold text-[#fff4de] transition-all duration-200 hover:border-[rgba(197,138,56,0.45)]">
                      Make GM
                    </button>
                    <button v-else-if="isGameOwner && row.role === 'assistant_gm'" type="button" @click="updateMemberRole(row, 'player')" class="cursor-pointer rounded-lg border border-[rgba(126,200,227,0.2)] bg-[rgba(126,200,227,0.08)] px-3 py-1.5 text-[12px] font-semibold text-[#d8dce7]/80 transition-all duration-200 hover:border-[rgba(126,200,227,0.4)]">
                      Revoke GM
                    </button>
                    <button v-if="canRemoveMember(row)" type="button" @click="requestRemoveMember(row)" class="cursor-pointer rounded-lg border border-[rgba(248,113,113,0.24)] bg-[rgba(248,113,113,0.1)] px-3 py-1.5 text-[12px] font-semibold text-[#fecaca] transition-all duration-200 hover:border-[rgba(248,113,113,0.45)]">
                      Remove
                    </button>
                  </div>
                </template>
              </DataTable>
            </div>

            <!-- Chat history table -->
            <div v-show="managePanel === 'chat'">
              <DataTable
                :columns="chatColumns"
                :rows="chatHistoryRows"
                :search-keys="['author', 'text']"
                search-placeholder="Search chat history"
                :page-size="10"
                min-width="560px"
                min-height="18rem"
                :empty-text="manageChatLoading ? 'Loading...' : 'No chat messages yet'"
              >
                <template #toolbar>
                  <div class="flex items-center gap-2">
                    <select v-model.number="chatClearHours" class="rounded-lg border border-[rgba(126,200,227,0.16)] bg-[rgba(7,17,31,0.72)] px-2.5 py-1.5 text-[12px] text-[#f6f7fb] outline-none">
                      <option v-for="period in CLEAR_PERIODS" :key="period.hours" :value="period.hours">{{ period.label }}</option>
                    </select>
                    <div role="button" tabindex="0" @click="clearChatHistory" @keydown.enter="clearChatHistory" class="cursor-pointer select-none rounded-lg border border-[rgba(248,113,113,0.28)] bg-[rgba(248,113,113,0.12)] px-3 py-1.5 text-[12px] font-semibold text-[#fecaca] transition-all duration-200 hover:border-[rgba(248,113,113,0.5)]">Clear</div>
                  </div>
                </template>
                <template #cell-time="{ row }"><span class="whitespace-nowrap text-[#7ec8e3]/45">{{ row.time }}</span></template>
                <template #cell-author="{ row }"><span class="whitespace-nowrap font-semibold text-[#f6f7fb]">{{ row.author }}</span></template>
                <template #cell-text="{ row }">
                  <span v-if="row.isItem" class="inline-flex items-center gap-1.5 rounded-lg border border-[rgba(126,200,227,0.2)] bg-[rgba(126,200,227,0.08)] px-2 py-1 text-[12px] text-[#8fd7ef]">
                    <PackageIcon class="h-3.5 w-3.5" :stroke-width="2" /> Shared item: {{ row.text }}
                  </span>
                  <span v-else class="inline-block max-w-[34rem] truncate align-middle text-[#e8e8f0]/78" :title="row.text">{{ row.text }}</span>
                </template>
              </DataTable>
            </div>

            <!-- Activity log table -->
            <div v-show="managePanel === 'activity'">
              <DataTable
                :columns="activityColumns"
                :rows="activityRows"
                :search-keys="['player', 'character', 'action', 'details']"
                search-placeholder="Search activity"
                :page-size="10"
                min-width="720px"
                min-height="18rem"
                :empty-text="manageActivityLoading ? 'Loading...' : 'No recorded activity yet'"
              >
                <template #toolbar>
                  <div class="flex items-center gap-2">
                    <select v-model.number="activityClearHours" class="rounded-lg border border-[rgba(126,200,227,0.16)] bg-[rgba(7,17,31,0.72)] px-2.5 py-1.5 text-[12px] text-[#f6f7fb] outline-none">
                      <option v-for="period in CLEAR_PERIODS" :key="period.hours" :value="period.hours">{{ period.label }}</option>
                    </select>
                    <div role="button" tabindex="0" @click="clearActivity" @keydown.enter="clearActivity" class="cursor-pointer select-none rounded-lg border border-[rgba(248,113,113,0.28)] bg-[rgba(248,113,113,0.12)] px-3 py-1.5 text-[12px] font-semibold text-[#fecaca] transition-all duration-200 hover:border-[rgba(248,113,113,0.5)]">Clear</div>
                  </div>
                </template>
                <template #cell-time="{ row }"><span class="whitespace-nowrap text-[#7ec8e3]/45">{{ row.time }}</span></template>
                <template #cell-player="{ row }"><span class="whitespace-nowrap font-semibold text-[#f6f7fb]">{{ row.player }}</span></template>
                <template #cell-character="{ row }"><span class="text-[#d8dce7]/70">{{ row.character }}</span></template>
                <template #cell-action="{ row }">
                  <span class="rounded-full border border-[rgba(126,200,227,0.18)] bg-[rgba(126,200,227,0.08)] px-2.5 py-1 text-[11px] font-semibold text-[#8fd7ef]">{{ row.action }}</span>
                </template>
                <template #cell-details="{ row }">
                  <button
                    v-if="activityItemName(row)"
                    type="button"
                    @click="openActivityItem(activityItemName(row))"
                    class="inline-flex max-w-[28rem] cursor-pointer items-center gap-1.5 truncate rounded-lg border border-[rgba(126,200,227,0.2)] bg-[rgba(126,200,227,0.08)] px-2.5 py-1 align-middle text-[12px] font-medium text-[#8fd7ef] transition-all duration-200 hover:border-[rgba(126,200,227,0.45)]"
                    :title="row.details"
                  >
                    <PackageIcon class="h-3.5 w-3.5 shrink-0" :stroke-width="2" /> {{ row.details }}
                  </button>
                  <span v-else class="inline-block max-w-[28rem] truncate align-middle text-[#e8e8f0]/78" :title="row.details">{{ row.details }}</span>
                </template>
              </DataTable>
            </div>
          </section>
        </main>
      </div>

      <Teleport to="body">
        <div v-if="memberPendingRemoval" class="fixed inset-0 z-[12560] flex items-center justify-center p-4">
          <div class="absolute inset-0 bg-[rgba(5,8,12,0.82)] backdrop-blur-md" @click="cancelRemoveMember"></div>
          <div class="relative w-full max-w-[28rem] overflow-hidden rounded-[1.6rem] border border-[rgba(248,113,113,0.24)] bg-[linear-gradient(180deg,rgba(9,18,34,0.98),rgba(5,10,22,0.98))] p-6 shadow-[0_40px_120px_rgba(0,0,0,0.55)]">
            <p class="text-[11px] uppercase tracking-[0.24em] text-[#fca5a5]/70">Remove Player</p>
            <h3 class="mt-2 break-words font-[Cinzel] text-[22px] font-bold leading-tight text-[#f6f7fb] [overflow-wrap:anywhere]">{{ memberPendingRemoval.username }}</h3>
            <p class="mt-3 text-[14px] leading-relaxed text-[#d8dce7]/68">
              This removes the player from the game and deletes their characters here (with inventory and equipment). This action cannot be undone.
            </p>
            <div class="mt-6 flex flex-wrap justify-end gap-3">
              <button type="button" @click="cancelRemoveMember" :disabled="removingMember" class="cursor-pointer rounded-xl border border-[rgba(126,200,227,0.16)] bg-[rgba(126,200,227,0.08)] px-4 py-2.5 text-[13px] font-semibold text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(126,200,227,0.3)] disabled:cursor-not-allowed disabled:opacity-60">
                Cancel
              </button>
              <button type="button" @click="confirmRemoveMember" :disabled="removingMember" class="inline-flex cursor-pointer items-center gap-2 rounded-xl border border-[rgba(248,113,113,0.28)] bg-[linear-gradient(135deg,rgba(248,113,113,0.9),rgba(220,38,38,0.9))] px-4 py-2.5 text-[13px] font-semibold text-white transition-all duration-200 hover:-translate-y-0.5 disabled:cursor-not-allowed disabled:opacity-60">
                {{ removingMember ? 'Removing...' : 'Remove Player' }}
              </button>
            </div>
          </div>
        </div>
      </Teleport>

      <GameSettingsModal
        :visible="showGameSettings"
        :game-id="gameId"
        @close="showGameSettings = false"
        @updated="handleGameSettingsUpdated"
        @deleted="handleGameDeleted"
      />

      <Teleport to="body">
        <div v-if="characterPendingDeletion" class="fixed inset-0 z-[12550] flex items-center justify-center p-4">
          <div class="absolute inset-0 bg-[rgba(5,8,12,0.82)] backdrop-blur-md" @click="cancelCharacterDeletion"></div>
          <div class="relative w-full max-w-[28rem] overflow-hidden rounded-[1.6rem] border border-[rgba(248,113,113,0.24)] bg-[linear-gradient(180deg,rgba(9,18,34,0.98),rgba(5,10,22,0.98))] p-6 shadow-[0_40px_120px_rgba(0,0,0,0.55)]">
            <p class="text-[11px] uppercase tracking-[0.24em] text-[#fca5a5]/70">Delete Character</p>
            <h3 class="mt-2 break-words font-[Cinzel] text-[22px] font-bold leading-tight text-[#f6f7fb] [overflow-wrap:anywhere]">{{ characterPendingDeletion.name || 'Character' }}</h3>
            <p class="mt-3 text-[14px] leading-relaxed text-[#d8dce7]/68">
              This permanently removes the character along with its inventory, equipment, and attributes. This action cannot be undone.
            </p>
            <div class="mt-6 flex flex-wrap justify-end gap-3">
              <button type="button" @click="cancelCharacterDeletion" :disabled="deletingCharacter" class="cursor-pointer rounded-xl border border-[rgba(126,200,227,0.16)] bg-[rgba(126,200,227,0.08)] px-4 py-2.5 text-[13px] font-semibold text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(126,200,227,0.3)] disabled:cursor-not-allowed disabled:opacity-60">
                Cancel
              </button>
              <button type="button" @click="confirmCharacterDeletion" :disabled="deletingCharacter" class="inline-flex cursor-pointer items-center gap-2 rounded-xl border border-[rgba(248,113,113,0.28)] bg-[linear-gradient(135deg,rgba(248,113,113,0.9),rgba(220,38,38,0.9))] px-4 py-2.5 text-[13px] font-semibold text-white transition-all duration-200 hover:-translate-y-0.5 disabled:cursor-not-allowed disabled:opacity-60">
                {{ deletingCharacter ? 'Deleting...' : 'Delete Character' }}
              </button>
            </div>
          </div>
        </div>
      </Teleport>

      <Teleport to="body">
        <div v-if="sharedItem" class="fixed inset-0 z-[12550] flex items-center justify-center p-4">
          <div class="absolute inset-0 bg-[rgba(5,8,12,0.82)] backdrop-blur-md" @click="closeSharedItem"></div>
          <div class="relative flex max-h-full w-full max-w-[34rem] flex-col overflow-hidden rounded-[1.6rem] border border-[rgba(126,200,227,0.18)] bg-[linear-gradient(180deg,rgba(9,18,34,0.98),rgba(5,10,22,0.98))] shadow-[0_40px_120px_rgba(0,0,0,0.55)]">
            <button type="button" @click="closeSharedItem" aria-label="Close item details" class="absolute right-4 top-4 z-10 flex h-9 w-9 cursor-pointer items-center justify-center rounded-xl border border-[rgba(126,200,227,0.14)] bg-[rgba(126,200,227,0.08)] text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(233,69,96,0.32)]">
              <XIcon class="h-5 w-5" :stroke-width="2" />
            </button>
            <div class="min-h-0 flex-1 overflow-y-auto p-5 sm:p-6">
              <div class="flex gap-4">
                <div class="h-24 w-24 shrink-0 overflow-hidden rounded-[1.2rem] border-2 border-[rgba(126,200,227,0.2)] bg-[rgba(7,17,31,0.66)]">
                  <img v-if="avatarUrl(sharedItem.image_id)" :src="avatarUrl(sharedItem.image_id)" :alt="sharedItem.name" class="h-full w-full object-cover" />
                  <div v-else class="flex h-full w-full items-center justify-center font-[Cinzel] text-[28px] font-bold text-[#7ec8e3]/40">{{ (sharedItem.name || '?').charAt(0).toUpperCase() }}</div>
                </div>
                <div class="min-w-0 flex-1 pr-8">
                  <h3 class="break-words text-[20px] font-bold leading-tight text-[#f6f7fb] [overflow-wrap:anywhere]">{{ sharedItem.name || 'Unnamed item' }}</h3>
                  <div class="mt-1.5 flex flex-wrap items-center gap-2">
                    <span class="rounded-full px-2.5 py-1 text-[10px] font-semibold uppercase tracking-[0.14em]" :class="rarityClasses(sharedItem.rarity)">{{ sharedItem.rarity || 'common' }}</span>
                    <span class="text-[11px] uppercase tracking-[0.16em] text-[#7ec8e3]/45">{{ formatAttrLabel(sharedItem.category || 'other') }}</span>
                    <span class="text-[11px] uppercase tracking-[0.16em] text-[#7ec8e3]/45">{{ sharedItem.grid_width || 1 }}x{{ sharedItem.grid_height || 1 }}</span>
                  </div>
                  <p v-if="sharedItem.owner" class="mt-1 text-[12px] text-[#d8dce7]/60">Shared by {{ sharedItem.owner }}</p>
                </div>
              </div>

              <div class="mt-4">
                <p class="text-[10px] uppercase tracking-[0.18em] text-[#7ec8e3]/55">Description</p>
                <p class="mt-1 break-words whitespace-pre-line text-[13px] leading-relaxed text-[#d8dce7]/72 [overflow-wrap:anywhere]">{{ sharedItem.description?.trim() || 'No description provided.' }}</p>
              </div>

              <div v-if="attributesEnabled && sharedItem.required_attributes?.length" class="mt-4">
                <p class="text-[10px] uppercase tracking-[0.18em] text-[#7ec8e3]/55">Requirements</p>
                <ul class="mt-2 grid gap-1.5 sm:grid-cols-2">
                  <li v-for="(requirement, index) in sharedItem.required_attributes" :key="`sreq-${index}`" class="flex items-center justify-between gap-2 rounded-lg border border-[rgba(126,200,227,0.12)] bg-[rgba(126,200,227,0.05)] px-3 py-1.5 text-[12px] text-[#d8dce7]/82">
                    <span>{{ formatAttrLabel(requirement.attribute_name) }}</span>
                    <span class="font-semibold text-[#f6f7fb]">{{ requirement.min_value }}</span>
                  </li>
                </ul>
              </div>

              <div v-if="attributesEnabled && sharedItem.attribute_modifiers?.length" class="mt-4">
                <p class="text-[10px] uppercase tracking-[0.18em] text-[#7ec8e3]/55">Grants</p>
                <ul class="mt-2 grid gap-1.5 sm:grid-cols-2">
                  <li v-for="(modifier, index) in sharedItem.attribute_modifiers" :key="`smod-${index}`" class="flex items-center justify-between gap-2 rounded-lg border border-[rgba(126,200,227,0.12)] bg-[rgba(126,200,227,0.05)] px-3 py-1.5 text-[12px] text-[#d8dce7]/82">
                    <span>{{ formatAttrLabel(modifier.attribute_name) }}</span>
                    <span class="font-semibold" :class="modifier.modifier_value >= 0 ? 'text-[#8fd7ef]' : 'text-[#fca5a5]'">{{ modifier.modifier_value >= 0 ? '+' : '' }}{{ modifier.modifier_value }}{{ modifier.is_percentage ? '%' : '' }}</span>
                  </li>
                </ul>
              </div>

              <div class="mt-4 flex flex-wrap gap-3 text-[12px] text-[#d8dce7]/70">
                <span>Quantity: <span class="font-semibold text-[#f6f7fb]">{{ sharedItem.quantity || 1 }}</span></span>
                <span v-if="sharedItem.enchantment">Enchantment: <span class="font-semibold text-[#8fd7ef]">+{{ sharedItem.enchantment }}</span></span>
                <span v-if="sharedItem.max_durability != null">Durability: <span class="font-semibold text-[#f6f7fb]">{{ sharedItem.durability ?? sharedItem.max_durability }} / {{ sharedItem.max_durability }}</span></span>
              </div>
            </div>
          </div>
        </div>
      </Teleport>

      <Teleport to="body">
        <div v-if="viewMember" class="fixed inset-0 z-[12540] flex items-center justify-center p-4">
          <div class="absolute inset-0 bg-[rgba(5,8,12,0.85)] backdrop-blur-md" @click="closeMemberView"></div>
          <div class="relative flex max-h-[92vh] w-full max-w-[52rem] flex-col overflow-hidden rounded-[1.6rem] border border-[rgba(126,200,227,0.18)] bg-[linear-gradient(180deg,rgba(9,18,34,0.98),rgba(5,10,22,0.98))] shadow-[0_40px_120px_rgba(0,0,0,0.55)]">
            <button type="button" @click="closeMemberView" aria-label="Close player overview" class="absolute right-4 top-4 z-10 flex h-9 w-9 cursor-pointer items-center justify-center rounded-xl border border-[rgba(126,200,227,0.14)] bg-[rgba(126,200,227,0.08)] text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(233,69,96,0.32)]">
              <XIcon class="h-5 w-5" :stroke-width="2" />
            </button>

            <div class="flex items-center gap-4 border-b border-[rgba(126,200,227,0.1)] px-6 py-5 pr-16">
              <div class="h-16 w-16 shrink-0 overflow-hidden rounded-2xl border-2 border-[rgba(126,200,227,0.2)] bg-[rgba(7,17,31,0.66)]">
                <img v-if="avatarUrl(viewMember.avatar_id)" :src="avatarUrl(viewMember.avatar_id)" :alt="viewMember.username" class="h-full w-full object-cover" />
                <div v-else class="flex h-full w-full items-center justify-center font-[Cinzel] text-[22px] font-bold text-[#7ec8e3]/40">{{ (viewMember.username || '?').charAt(0).toUpperCase() }}</div>
              </div>
              <div class="min-w-0 flex-1">
                <p class="text-[11px] uppercase tracking-[0.24em] text-[#7ec8e3]/58">Player Overview</p>
                <h2 class="mt-1 break-words font-[Cinzel] text-[24px] font-bold leading-tight text-[#f6f7fb] [overflow-wrap:anywhere]">{{ viewMember.username }}</h2>
                <div class="mt-2 flex flex-wrap items-center gap-2 text-[12px]">
                  <span class="rounded-full border border-[rgba(126,200,227,0.18)] bg-[rgba(126,200,227,0.08)] px-3 py-1 font-semibold uppercase tracking-[0.14em] text-[#8fd7ef]">{{ formatRole(viewMember.role) }}</span>
                  <span class="text-[#7ec8e3]/45">Joined {{ formatDateTime(viewMember.joined_at) }}</span>
                  <span class="text-[#7ec8e3]/45">· {{ viewMemberCharacters.length }} character(s)</span>
                </div>
              </div>
            </div>

            <div class="min-h-0 flex-1 overflow-y-auto p-6">
              <p class="text-[10px] uppercase tracking-[0.18em] text-[#7ec8e3]/55">Characters ({{ viewMemberCharacters.length }})</p>
              <div v-if="viewMemberCharacters.length" class="mt-3 grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-4">
                <div v-for="character in viewMemberCharacters" :key="character.id" class="flex flex-col items-center rounded-[1.1rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(8,16,30,0.7)] p-3 text-center">
                  <div class="h-16 w-16 overflow-hidden rounded-[0.9rem] border-2 border-[rgba(126,200,227,0.2)] bg-[rgba(7,17,31,0.66)]">
                    <img v-if="avatarUrl(character.portrait_id)" :src="avatarUrl(character.portrait_id)" :alt="character.name" class="h-full w-full object-cover" />
                    <div v-else class="flex h-full w-full items-center justify-center font-[Cinzel] text-[20px] font-bold text-[#7ec8e3]/40">{{ (character.name || '?').charAt(0).toUpperCase() }}</div>
                  </div>
                  <p class="mt-2 line-clamp-2 break-words text-[13px] font-semibold text-[#f6f7fb] [overflow-wrap:anywhere]">{{ character.name }}</p>
                </div>
              </div>
              <p v-else class="mt-2 text-[13px] text-[#7ec8e3]/45">This player has no characters in this game.</p>

              <p class="mt-6 text-[10px] uppercase tracking-[0.18em] text-[#7ec8e3]/55">Recent messages</p>
              <div v-if="viewMemberLoading" class="mt-3 flex justify-center py-6">
                <div class="h-6 w-6 animate-spin rounded-full border-2 border-[#e94560] border-t-transparent"></div>
              </div>
              <ul v-else-if="viewMemberMessages.length" class="mt-3 space-y-2">
                <li v-for="message in viewMemberMessages" :key="message.id" class="rounded-xl border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.04)] px-3 py-2">
                  <div class="flex items-start justify-between gap-3">
                    <div class="min-w-0 flex-1">
                      <span v-if="message.isItem" class="inline-flex items-center gap-1.5 text-[12px] text-[#8fd7ef]"><PackageIcon class="h-3.5 w-3.5 shrink-0" :stroke-width="2" /> {{ message.text }}</span>
                      <span v-else class="break-words text-[13px] text-[#e8e8f0]/82 [overflow-wrap:anywhere]">{{ message.text }}</span>
                    </div>
                    <span class="shrink-0 text-[11px] text-[#7ec8e3]/40">{{ message.time }}</span>
                  </div>
                </li>
              </ul>
              <p v-else class="mt-2 text-[13px] text-[#7ec8e3]/45">{{ game?.enable_chat ? 'No messages from this player yet.' : 'Chat is disabled for this game.' }}</p>
            </div>
          </div>
        </div>
      </Teleport>

      <Teleport to="body">
        <div v-if="showTradeModal" class="fixed inset-0 z-[12555] flex items-center justify-center p-4">
          <div class="absolute inset-0 bg-[rgba(5,8,12,0.85)] backdrop-blur-md" @click="closeTradeModal"></div>
          <div class="relative flex max-h-[92vh] w-full max-w-[56rem] flex-col overflow-hidden rounded-[1.6rem] border border-[rgba(126,200,227,0.18)] bg-[linear-gradient(180deg,rgba(9,18,34,0.98),rgba(5,10,22,0.98))] shadow-[0_40px_120px_rgba(0,0,0,0.55)]">
            <div class="flex items-center justify-between gap-3 border-b border-[rgba(126,200,227,0.1)] px-6 py-4">
              <div class="flex items-center gap-2.5">
                <ArrowLeftRightIcon class="h-5 w-5 text-[#8fd7ef]" :stroke-width="2" />
                <h2 class="font-[Cinzel] text-[20px] font-bold text-[#f6f7fb]">Player Trading</h2>
              </div>
              <button type="button" @click="closeTradeModal" :disabled="tradeBusy" aria-label="Close" class="flex h-9 w-9 cursor-pointer items-center justify-center rounded-xl border border-[rgba(126,200,227,0.14)] bg-[rgba(126,200,227,0.08)] text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(233,69,96,0.32)] disabled:opacity-50">
                <XIcon class="h-5 w-5" :stroke-width="2" />
              </button>
            </div>

            <div class="flex gap-1.5 border-b border-[rgba(126,200,227,0.1)] px-6 py-3">
              <button
                v-for="panel in [{ id: 'send', label: 'New offer' }, { id: 'incoming', label: 'Incoming' }, { id: 'outgoing', label: 'Sent' }]"
                :key="panel.id"
                type="button"
                @click="tradePanel = panel.id"
                class="relative cursor-pointer rounded-xl px-4 py-2 text-[13px] font-semibold transition-all duration-200"
                :class="tradePanel === panel.id ? 'bg-[linear-gradient(135deg,rgba(233,69,96,0.95),rgba(194,49,82,0.95))] text-white' : 'text-[#d8dce7]/70 hover:bg-[rgba(126,200,227,0.06)] hover:text-[#f6f7fb]'"
              >
                {{ panel.label }}
                <span v-if="panel.id === 'incoming' && incomingTradeCount" class="ml-1.5 inline-flex h-4 min-w-[1rem] items-center justify-center rounded-full bg-[#e94560] px-1 text-[10px] font-bold text-white">{{ incomingTradeCount }}</span>
                <span v-if="panel.id === 'outgoing' && tradeOutgoing.length" class="ml-1.5 inline-flex h-4 min-w-[1rem] items-center justify-center rounded-full bg-[rgba(126,200,227,0.25)] px-1 text-[10px] font-bold text-[#8fd7ef]">{{ tradeOutgoing.length }}</span>
              </button>
            </div>

            <div class="min-h-[26rem] flex-1 overflow-y-auto p-6">
              <!-- New offer -->
              <div v-if="tradePanel === 'send'">
                <p v-if="!tradingEnabled" class="mb-4 rounded-xl border border-[rgba(248,113,113,0.24)] bg-[rgba(248,113,113,0.08)] px-4 py-3 text-[13px] text-[#fca5a5]">Item trading is disabled for this game. You can still respond to existing offers.</p>
                <label class="block text-[11px] uppercase tracking-[0.16em] text-[#7ec8e3]/55">Send to</label>
                <div class="relative mt-1.5">
                  <input
                    v-model="tradeRecipientSearch"
                    @focus="tradeRecipientOpen = true"
                    @input="onRecipientInput"
                    @blur="closeRecipientDropdown"
                    type="text"
                    placeholder="Search character by name…"
                    class="w-full rounded-lg border border-[rgba(126,200,227,0.2)] bg-[rgba(7,17,31,0.72)] px-3 py-2.5 text-[14px] text-[#f6f7fb] outline-none focus:border-[rgba(126,200,227,0.45)] placeholder:text-[#7ec8e3]/30"
                  />
                  <div v-if="tradeRecipientOpen" class="absolute z-20 mt-1 max-h-56 w-full overflow-y-auto rounded-lg border border-[rgba(126,200,227,0.18)] bg-[linear-gradient(180deg,rgba(9,18,34,0.99),rgba(6,11,22,0.99))] py-1 shadow-[0_24px_60px_rgba(0,0,0,0.55)]">
                    <button
                      v-for="target in tradeRecipientFiltered"
                      :key="target.id"
                      type="button"
                      @mousedown.prevent="selectTradeRecipient(target)"
                      class="flex w-full cursor-pointer items-center justify-between gap-2 px-3 py-2 text-left text-[13px] transition-colors hover:bg-[rgba(126,200,227,0.08)]"
                      :class="tradeRecipientId === target.id ? 'text-[#ff8aa0]' : 'text-[#e8e8f0]/82'"
                    >
                      <span class="font-semibold">{{ target.name }}</span>
                      <span class="text-[12px] text-[#7ec8e3]/45">{{ target.owner }}</span>
                    </button>
                    <p v-if="!tradeRecipientFiltered.length" class="px-3 py-2 text-[12px] text-[#7ec8e3]/40">No matching characters</p>
                  </div>
                </div>

                <p class="mt-5 text-[11px] uppercase tracking-[0.16em] text-[#7ec8e3]/55">Items to offer ({{ tradeSelectedIds.length }} selected)</p>
                <p v-if="!tradeOfferableItems.length" class="mt-3 text-[13px] text-[#7ec8e3]/45">No tradeable items. Unequip items first to offer them.</p>
                <div v-else class="mt-3 grid grid-cols-2 gap-2.5 sm:grid-cols-3">
                  <button
                    v-for="entry in tradeOfferableItems"
                    :key="entry.id"
                    type="button"
                    @click="toggleTradeItem(entry.id)"
                    class="flex items-center gap-2.5 rounded-xl border p-2.5 text-left transition-all duration-200"
                    :class="tradeSelectedSet.has(entry.id) ? 'border-[rgba(233,69,96,0.5)] bg-[rgba(233,69,96,0.12)]' : 'border-[rgba(126,200,227,0.12)] bg-[rgba(126,200,227,0.04)] hover:border-[rgba(126,200,227,0.3)]'"
                  >
                    <div class="h-10 w-10 shrink-0 overflow-hidden rounded-lg border border-[rgba(126,200,227,0.16)] bg-[rgba(7,17,31,0.66)]">
                      <img v-if="avatarUrl(entry.item?.image_id)" :src="avatarUrl(entry.item?.image_id)" :alt="entry.item?.name" class="h-full w-full object-cover" />
                    </div>
                    <div class="min-w-0 flex-1">
                      <p class="truncate text-[13px] font-semibold text-[#f6f7fb]">{{ entry.item?.name }}</p>
                      <p class="text-[11px] text-[#7ec8e3]/45">{{ entry.item?.grid_width }}x{{ entry.item?.grid_height }}<span v-if="(entry.quantity || 1) > 1"> · x{{ entry.quantity }}</span></p>
                      <p v-if="entry.max_durability != null" class="text-[11px]" :class="(entry.durability ?? entry.max_durability) <= entry.max_durability * 0.3 ? 'text-[#fca5a5]' : 'text-[#86efac]/80'">
                        Dur {{ entry.durability ?? entry.max_durability }}/{{ entry.max_durability }}
                      </p>
                    </div>
                    <CheckIcon v-if="tradeSelectedSet.has(entry.id)" class="h-4 w-4 shrink-0 text-[#ff6b81]" :stroke-width="3" />
                  </button>
                </div>

                <div class="mt-6 flex justify-end">
                  <button
                    type="button"
                    @click="submitTrade"
                    :disabled="tradeBusy || !tradingEnabled || !tradeRecipientId || !tradeSelectedIds.length"
                    class="inline-flex cursor-pointer items-center gap-2 rounded-xl border border-[rgba(233,69,96,0.3)] bg-[linear-gradient(135deg,rgba(233,69,96,0.92),rgba(194,49,82,0.92))] px-5 py-2.5 text-[13px] font-semibold text-white transition-all duration-200 hover:-translate-y-0.5 disabled:cursor-not-allowed disabled:opacity-50 disabled:hover:translate-y-0"
                  >
                    <ArrowLeftRightIcon class="h-4 w-4" :stroke-width="2" />
                    {{ tradeBusy ? 'Sending…' : 'Send offer' }}
                  </button>
                </div>
              </div>

              <!-- Incoming -->
              <div v-else-if="tradePanel === 'incoming'">
                <p v-if="!tradeIncoming.length" class="py-10 text-center text-[13px] text-[#7ec8e3]/45">No incoming offers.</p>
                <div v-else class="space-y-3">
                  <article v-for="offer in tradeIncoming" :key="offer.id" class="rounded-[1.3rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(8,16,30,0.7)] p-4">
                    <p class="text-[13px] text-[#d8dce7]/72"><span class="font-semibold text-[#f6f7fb]">{{ offer.from_username }}</span> ({{ offer.from_character_name }}) → <span class="font-semibold text-[#f6f7fb]">{{ offer.to_character_name }}</span></p>
                    <div class="mt-3 flex flex-wrap gap-2">
                      <span v-for="item in offer.items" :key="item.id" class="inline-flex items-center gap-1.5 rounded-lg border border-[rgba(126,200,227,0.14)] bg-[rgba(126,200,227,0.06)] px-2 py-1 text-[12px] text-[#d8dce7]/82">
                        <img v-if="avatarUrl(item.item?.image_id)" :src="avatarUrl(item.item?.image_id)" :alt="item.item?.name" class="h-5 w-5 rounded object-cover" />
                        {{ item.item?.name }}<span v-if="(item.quantity || 1) > 1" class="text-[#7ec8e3]/50"> x{{ item.quantity }}</span>
                      </span>
                    </div>
                    <div class="mt-4 flex justify-end gap-2">
                      <button type="button" @click="declineTradeOffer(offer)" :disabled="tradeBusy" class="cursor-pointer rounded-lg border border-[rgba(248,113,113,0.24)] bg-[rgba(248,113,113,0.1)] px-3.5 py-2 text-[12px] font-semibold text-[#fecaca] transition-all duration-200 hover:border-[rgba(248,113,113,0.45)] disabled:opacity-50">Decline</button>
                      <button type="button" @click="acceptTradeOffer(offer)" :disabled="tradeBusy" class="inline-flex cursor-pointer items-center gap-1.5 rounded-lg border border-[rgba(74,222,128,0.3)] bg-[rgba(74,222,128,0.14)] px-3.5 py-2 text-[12px] font-semibold text-[#86efac] transition-all duration-200 hover:border-[rgba(74,222,128,0.5)] disabled:opacity-50"><CheckIcon class="h-3.5 w-3.5" :stroke-width="2.5" /> Accept</button>
                    </div>
                  </article>
                </div>
              </div>

              <!-- Sent -->
              <div v-else>
                <p v-if="!tradeOutgoing.length" class="py-10 text-center text-[13px] text-[#7ec8e3]/45">No pending sent offers.</p>
                <div v-else class="space-y-3">
                  <article v-for="offer in tradeOutgoing" :key="offer.id" class="rounded-[1.3rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(8,16,30,0.7)] p-4">
                    <p class="text-[13px] text-[#d8dce7]/72">To <span class="font-semibold text-[#f6f7fb]">{{ offer.to_username }}</span> ({{ offer.to_character_name }})</p>
                    <div class="mt-3 flex flex-wrap gap-2">
                      <span v-for="item in offer.items" :key="item.id" class="inline-flex items-center gap-1.5 rounded-lg border border-[rgba(126,200,227,0.14)] bg-[rgba(126,200,227,0.06)] px-2 py-1 text-[12px] text-[#d8dce7]/82">
                        <img v-if="avatarUrl(item.item?.image_id)" :src="avatarUrl(item.item?.image_id)" :alt="item.item?.name" class="h-5 w-5 rounded object-cover" />
                        {{ item.item?.name }}<span v-if="(item.quantity || 1) > 1" class="text-[#7ec8e3]/50"> x{{ item.quantity }}</span>
                      </span>
                    </div>
                    <div class="mt-4 flex justify-end">
                      <button type="button" @click="declineTradeOffer(offer)" :disabled="tradeBusy" class="cursor-pointer rounded-lg border border-[rgba(248,113,113,0.24)] bg-[rgba(248,113,113,0.1)] px-3.5 py-2 text-[12px] font-semibold text-[#fecaca] transition-all duration-200 hover:border-[rgba(248,113,113,0.45)] disabled:opacity-50">Cancel offer</button>
                    </div>
                  </article>
                </div>
              </div>
            </div>
          </div>
        </div>
      </Teleport>

      <Teleport to="body">
        <div v-if="inspectChoice" class="fixed inset-0 z-[12555] flex items-center justify-center p-4">
          <div class="absolute inset-0 bg-[rgba(5,8,12,0.82)] backdrop-blur-md" @click="closeInspectChoice"></div>
          <div class="relative w-full max-w-[24rem] overflow-hidden rounded-[1.6rem] border border-[rgba(126,200,227,0.18)] bg-[linear-gradient(180deg,rgba(9,18,34,0.98),rgba(5,10,22,0.98))] p-6 shadow-[0_40px_120px_rgba(0,0,0,0.55)]">
            <button type="button" @click="closeInspectChoice" aria-label="Close" class="absolute right-4 top-4 flex h-9 w-9 cursor-pointer items-center justify-center rounded-xl border border-[rgba(126,200,227,0.14)] bg-[rgba(126,200,227,0.08)] text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(233,69,96,0.32)]">
              <XIcon class="h-5 w-5" :stroke-width="2" />
            </button>
            <p class="text-[11px] uppercase tracking-[0.24em] text-[#7ec8e3]/58">Inspect</p>
            <h3 class="mt-1 break-words pr-10 font-[Cinzel] text-[20px] font-bold leading-tight text-[#f6f7fb] [overflow-wrap:anywhere]">{{ inspectChoice.name }}</h3>
            <p class="mt-2 text-[13px] text-[#d8dce7]/62">What would you like to open?</p>
            <div class="mt-5 grid grid-cols-2 gap-3">
              <button type="button" @click="inspectCharacterSheet" class="flex cursor-pointer flex-col items-center gap-2 rounded-2xl border border-[rgba(126,200,227,0.16)] bg-[rgba(126,200,227,0.08)] px-4 py-5 text-[13px] font-semibold text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(126,200,227,0.4)]">
                <UserRoundIcon class="h-6 w-6 text-[#8fd7ef]" :stroke-width="2" />
                Character
              </button>
              <button type="button" @click="inspectCharacterInventory" class="flex cursor-pointer flex-col items-center gap-2 rounded-2xl border border-[rgba(126,200,227,0.16)] bg-[rgba(126,200,227,0.08)] px-4 py-5 text-[13px] font-semibold text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(126,200,227,0.4)]">
                <LayoutGridIcon class="h-6 w-6 text-[#8fd7ef]" :stroke-width="2" />
                Inventory
              </button>
            </div>
          </div>
        </div>
      </Teleport>

      <Teleport to="body">
        <div v-if="inventoryViewCharacter" class="fixed inset-0 z-[12560] flex items-center justify-center p-4">
          <div class="absolute inset-0 bg-[rgba(5,8,12,0.88)] backdrop-blur-md" @click="closeCharacterInventory"></div>
          <div class="relative flex max-h-[94vh] w-full max-w-[80rem] flex-col overflow-hidden rounded-[1.6rem] border border-[rgba(126,200,227,0.18)] bg-[linear-gradient(180deg,rgba(9,18,34,0.98),rgba(5,10,22,0.98))] shadow-[0_40px_120px_rgba(0,0,0,0.55)]">
            <div class="flex items-center justify-between gap-3 border-b border-[rgba(126,200,227,0.1)] px-6 py-4">
              <div class="min-w-0">
                <p class="text-[11px] uppercase tracking-[0.24em] text-[#7ec8e3]/58">Inventory (read-only)</p>
                <h2 class="mt-1 break-words font-[Cinzel] text-[22px] font-bold leading-tight text-[#f6f7fb] [overflow-wrap:anywhere]">{{ inventoryViewCharacter.name }}</h2>
              </div>
              <button type="button" @click="closeCharacterInventory" aria-label="Close inventory" class="flex h-9 w-9 shrink-0 cursor-pointer items-center justify-center rounded-xl border border-[rgba(126,200,227,0.14)] bg-[rgba(126,200,227,0.08)] text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(233,69,96,0.32)]">
                <XIcon class="h-5 w-5" :stroke-width="2" />
              </button>
            </div>

            <div class="min-h-0 flex-1 overflow-auto p-5 sm:p-6">
              <div v-if="inventoryViewLoading" class="py-20 text-center">
                <div class="mx-auto h-8 w-8 animate-spin rounded-full border-2 border-[#e94560] border-t-transparent"></div>
              </div>
              <SessionInventoryBoard
                v-else
                :character-name="inventoryViewCharacter.name || ''"
                :inventory-items="inventoryViewCharacter.inventory ?? []"
                :equipment="inventoryViewCharacter.equipment ?? []"
                :currency-cards="inventoryViewCurrency"
                :inventory-width="inventoryViewCharacter.inventory_width ?? 0"
                :inventory-height="inventoryViewCharacter.inventory_height ?? 0"
                :character-attributes="inventoryViewAttributes"
                :attributes-enabled="attributesEnabled"
                :disabled-standard-attrs="disabledStandardAttrs"
                :character-id="inventoryViewCharacter.id"
                :can-edit="false"
              />
            </div>
          </div>
        </div>
      </Teleport>

      <Teleport to="body">
        <div
          v-if="statTooltip.visible"
          class="pointer-events-none fixed z-[12000] w-[15rem] rounded-xl border border-[rgba(126,200,227,0.2)] bg-[linear-gradient(180deg,rgba(9,18,34,0.98),rgba(5,10,22,0.98))] p-3 text-left shadow-[0_20px_50px_rgba(0,0,0,0.5)]"
          :style="{ left: `${statTooltip.left}px`, top: `${statTooltip.top}px` }"
        >
          <p class="text-[10px] uppercase tracking-[0.18em] text-[#7ec8e3]/55">{{ statTooltip.label }} breakdown</p>
          <div class="mt-2 flex items-center justify-between text-[12px] text-[#d8dce7]/75">
            <span>Base</span><span class="font-semibold">{{ statTooltip.base }}</span>
          </div>
          <div
            v-for="(contribution, index) in statTooltip.contributions"
            :key="index"
            class="mt-1 flex items-center justify-between gap-2 text-[12px]"
            :class="contribution.value >= 0 ? 'text-[#86efac]' : 'text-[#fca5a5]'"
          >
            <span class="truncate pr-2">{{ contribution.source }}</span>
            <span class="shrink-0 font-semibold">{{ contribution.value >= 0 ? '+' : '' }}{{ contribution.value }}{{ contribution.percent ? '%' : '' }}</span>
          </div>
          <p v-if="!statTooltip.contributions.length" class="mt-1 text-[11px] text-[#d8dce7]/50">No equipment bonuses</p>
          <div class="mt-2 flex items-center justify-between border-t border-[rgba(126,200,227,0.12)] pt-2 text-[12px] font-semibold text-[#f6f7fb]">
            <span>Total</span><span>{{ statTooltip.total }}</span>
          </div>
        </div>
      </Teleport>

      <button
        v-if="game?.enable_chat && chatCollapsed"
        @click="focusChatInput"
        class="session-chat-launcher border-[rgba(197,138,56,0.24)] bg-[rgba(16,19,23,0.96)] text-[#f3ead9]"
      >
        <MessageSquareTextIcon class="h-5 w-5" :stroke-width="1.8" />
        <span class="text-[11px] font-semibold uppercase tracking-[0.18em]">Chat</span>
        <span class="rounded-full border border-[rgba(126,200,227,0.14)] px-2 py-0.5 text-[10px] font-semibold">{{ chatMessages.length }}</span>
      </button>

      <aside v-else-if="game?.enable_chat" class="session-chat-dock">
        <div class="session-chat-dock__shell">
          <div class="flex items-center justify-between gap-4 border-b border-[rgba(126,200,227,0.08)] px-4 py-4">
            <div>
              <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Session Chat</p>
              <p class="mt-1 truncate text-[14px] text-[#f6f7fb]">Latest 40 messages</p>
            </div>

            <div class="flex items-center gap-3">
              <span
                class="rounded-full border px-2.5 py-1 text-[10px] font-semibold uppercase tracking-[0.16em]"
                :class="game.enable_chat
                  ? 'border-[rgba(110,231,183,0.28)] bg-[rgba(22,163,74,0.14)] text-[#86efac]'
                  : 'border-[rgba(248,113,113,0.28)] bg-[rgba(185,28,28,0.14)] text-[#fca5a5]'"
              >
                {{ game.enable_chat ? 'Online' : 'Disabled' }}
              </span>
              <button
                @click="chatCollapsed = true"
                aria-label="Collapse chat dock"
                class="flex h-10 w-10 shrink-0 cursor-pointer items-center justify-center rounded-xl border border-[rgba(126,200,227,0.14)] bg-[rgba(126,200,227,0.08)] text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(233,69,96,0.34)] hover:text-[#ffe0e7]"
              >
                <ChevronRightIcon class="h-5 w-5" :stroke-width="2" />
              </button>
            </div>
          </div>

          <div ref="chatMessagesRef" class="session-scroll flex-1 overflow-y-auto px-4 py-4">
            <div v-if="!game.enable_chat" class="rounded-[1.5rem] border border-[rgba(248,113,113,0.22)] bg-[rgba(127,29,29,0.18)] px-4 py-4 text-[14px] text-[#fecaca]">
              Chat is disabled for this game. The launcher stays visible, but sending is blocked by settings.
            </div>

            <div v-else-if="!chatMessages.length" class="rounded-[1.5rem] border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] px-4 py-4 text-[14px] text-[#d8dce7]/62">
              No messages yet. Start the conversation here without leaving the session.
            </div>

            <div v-else class="space-y-3">
              <article
                v-for="message in chatMessages"
                :key="message.id"
                class="session-chat-message rounded-[1.4rem] border px-4 py-3"
                :class="message.user_id === viewer.user_id
                  ? 'border-[rgba(233,69,96,0.24)] bg-[rgba(233,69,96,0.12)]'
                  : 'border-[rgba(126,200,227,0.12)] bg-[rgba(126,200,227,0.05)]'"
              >
                <div class="flex items-start gap-3">
                  <div class="flex h-10 w-10 shrink-0 items-center justify-center overflow-hidden rounded-2xl border border-[rgba(126,200,227,0.12)] bg-[rgba(126,200,227,0.08)]">
                    <img v-if="avatarUrl(message.user?.avatar_id)" :src="avatarUrl(message.user?.avatar_id)" :alt="message.user?.username" class="h-full w-full object-cover" />
                    <span v-else class="font-[Cinzel] text-[14px] font-bold text-[#8fd7ef]">{{ initials(message.user?.username) }}</span>
                  </div>

                  <div class="min-w-0 flex-1">
                    <div class="flex flex-wrap items-center justify-between gap-3">
                      <p class="truncate text-[13px] font-semibold text-[#f6f7fb]">{{ message.user?.username || 'Unknown user' }}</p>
                      <span class="text-[11px] text-[#d8dce7]/45">{{ formatDateTime(message.created_at) }}</span>
                    </div>
                    <button
                      v-if="parseSharedItem(message.content)"
                      type="button"
                      @click="openSharedItem(parseSharedItem(message.content))"
                      class="mt-2 inline-flex max-w-full cursor-pointer items-center gap-2 rounded-xl border border-[rgba(126,200,227,0.22)] bg-[rgba(126,200,227,0.08)] px-3 py-2 text-left transition-all duration-200 hover:border-[rgba(233,69,96,0.4)] hover:bg-[rgba(233,69,96,0.1)]"
                    >
                      <PackageIcon class="h-4 w-4 shrink-0 text-[#8fd7ef]" :stroke-width="2" />
                      <span class="min-w-0">
                        <span class="block truncate text-[13px] font-semibold text-[#f6f7fb]">{{ parseSharedItem(message.content).name || 'Item' }}</span>
                        <span class="block text-[11px] uppercase tracking-[0.14em] text-[#8fd7ef]/70">Shared item · view</span>
                      </span>
                    </button>
                    <p v-else class="mt-2 whitespace-pre-wrap break-words text-[14px] leading-relaxed text-[#e8e8f0]/78">{{ message.content }}</p>
                  </div>
                </div>
              </article>
            </div>
          </div>

          <div class="border-t border-[rgba(126,200,227,0.08)] px-4 py-4">
            <p v-if="chatError" class="mb-3 rounded-xl border border-[rgba(248,113,113,0.24)] bg-[rgba(127,29,29,0.18)] px-3 py-2 text-[12px] text-[#fecaca]">
              {{ chatError }}
            </p>

            <textarea
              ref="chatInputRef"
              v-model="chatDraft"
              rows="3"
              :disabled="!game.enable_chat || chatSending"
              @keydown.enter.exact.prevent="sendChatMessage"
              placeholder="Write a message for the table..."
              class="session-input w-full resize-none rounded-[1.4rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.72)] px-4 py-3 text-[14px] text-[#f6f7fb] outline-none transition-all duration-200 placeholder:text-[#7ec8e3]/35 focus:border-[rgba(233,69,96,0.34)] disabled:cursor-not-allowed disabled:opacity-60"
            ></textarea>

            <div class="mt-3 flex items-center justify-between gap-3">
              <p class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Enter to send</p>
              <button
                @click="sendChatMessage"
                :disabled="!chatDraft.trim() || !game.enable_chat || chatSending"
                class="cursor-pointer rounded-xl border border-[rgba(233,69,96,0.28)] bg-[linear-gradient(135deg,rgba(233,69,96,0.9),rgba(194,49,82,0.9))] px-4 py-2.5 text-[13px] font-semibold text-white transition-all duration-200 hover:-translate-y-0.5 hover:shadow-[0_12px_30px_rgba(233,69,96,0.24)] disabled:cursor-not-allowed disabled:opacity-60"
              >
                {{ chatSending ? 'Sending...' : 'Send Message' }}
              </button>
            </div>
          </div>
        </div>
      </aside>
    </div>
  </div>
</template>

<style scoped>
.session-stage {
  position: relative;
  isolation: isolate;
  color: var(--dl-brand-text);
}

.session-stage::before {
  content: '';
  position: fixed;
  inset: 0;
  z-index: -2;
  pointer-events: none;
  background:
    linear-gradient(90deg, rgba(126, 200, 227, 0.032) 1px, transparent 1px),
    linear-gradient(0deg, rgba(126, 200, 227, 0.024) 1px, transparent 1px),
    linear-gradient(180deg, #0a0a1a 0%, #09101d 56%, #060912 100%);
  background-size: 88px 88px, 88px 88px, auto;
}

.session-stage::after {
  content: '';
  position: fixed;
  inset: 14px;
  z-index: -1;
  pointer-events: none;
  border: 1px solid rgba(126, 200, 227, 0.07);
}

.session-header {
  position: sticky;
  background: linear-gradient(180deg, rgba(14, 18, 33, 0.96), rgba(8, 11, 22, 0.94)) !important;
  border-bottom-color: rgba(126, 200, 227, 0.14) !important;
  box-shadow: 0 16px 40px rgba(0, 0, 0, 0.34);
}

.session-header::before {
  content: '';
  position: absolute;
  inset: 0;
  pointer-events: none;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.035), transparent 68%),
    repeating-linear-gradient(90deg, rgba(255, 255, 255, 0.03) 0 1px, transparent 1px 72px);
  opacity: 0.55;
}

.session-header > div {
  position: relative;
  z-index: 1;
}

.session-scroll::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

.session-scroll::-webkit-scrollbar-thumb {
  background: rgba(126, 200, 227, 0.22);
  border-radius: 999px;
}

.session-scroll::-webkit-scrollbar-track {
  background: transparent;
}

.session-stage button:not(.session-tab-button):not(.session-chat-launcher) {
  border-color: var(--dl-panel-border) !important;
  background: linear-gradient(180deg, rgba(21, 29, 50, 0.92), rgba(12, 18, 31, 0.96)) !important;
  color: var(--dl-brand-text) !important;
  border-radius: 0.55rem !important;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.04), 0 12px 24px rgba(0, 0, 0, 0.16);
}

.session-stage button:not(.session-tab-button):not(.session-chat-launcher):hover {
  border-color: rgba(233, 69, 96, 0.32) !important;
  color: #ffffff !important;
}

.session-stage button:not(.session-tab-button):not(.session-chat-launcher):disabled {
  box-shadow: none;
}

.session-stage select,
.session-stage textarea,
.session-stage input:not([type='file']):not(.border-0),
.session-input {
  border-radius: 0.5rem !important;
  border-color: var(--dl-panel-border) !important;
  background: rgba(11, 17, 31, 0.88) !important;
  color: var(--dl-brand-text) !important;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.03), 0 1px 0 rgba(0, 0, 0, 0.3);
}

.session-stage select:focus,
.session-stage textarea:focus,
.session-stage input:not([type='file']):not(.border-0):focus,
.session-input:focus {
  border-color: rgba(233, 69, 96, 0.4) !important;
  box-shadow: 0 0 0 1px rgba(233, 69, 96, 0.18), inset 0 1px 0 rgba(255, 255, 255, 0.03);
}

.session-stage article:not(.session-chat-message) {
  position: relative;
  overflow: hidden;
  border-radius: 0.75rem !important;
  border-color: rgba(126, 200, 227, 0.14) !important;
  background: var(--dl-panel-bg) !important;
  box-shadow: 0 24px 60px rgba(0, 0, 0, 0.26), inset 0 1px 0 rgba(255, 255, 255, 0.04) !important;
}

.session-stage article:not(.session-chat-message)::before {
  content: '';
  position: absolute;
  inset: 0;
  pointer-events: none;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.03), transparent 26%),
    repeating-linear-gradient(135deg, rgba(255, 255, 255, 0.015) 0 2px, transparent 2px 18px);
  opacity: 0.65;
}

.session-stage article:not(.session-chat-message)::after {
  content: '';
  position: absolute;
  inset: 9px;
  pointer-events: none;
  border: 1px solid rgba(126, 200, 227, 0.1);
  clip-path: polygon(0 14px, 14px 0, 100% 0, 100% calc(100% - 14px), calc(100% - 14px) 100%, 0 100%);
}

.session-stage article:not(.session-chat-message) > * {
  position: relative;
  z-index: 1;
}

.session-command-deck {
  border-color: rgba(233, 69, 96, 0.24) !important;
  background: linear-gradient(135deg, rgba(39, 18, 31, 0.96), rgba(15, 23, 43, 0.96) 45%, rgba(10, 14, 26, 0.98)) !important;
}

.session-command-deck::before {
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.045), transparent 32%),
    repeating-linear-gradient(90deg, rgba(255, 255, 255, 0.04) 0 1px, transparent 1px 72px),
    linear-gradient(180deg, rgba(255, 255, 255, 0.04), transparent 28%);
}

.session-mode-pill,
.session-count-pill,
.session-owner-pill {
  position: relative;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.04);
}

.session-banner {
  border-color: rgba(233, 69, 96, 0.28) !important;
  background: linear-gradient(180deg, rgba(62, 20, 34, 0.72), rgba(35, 14, 23, 0.78)) !important;
  color: #ffb8c5 !important;
}

.line-clamp-3 {
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.session-content-shell {
  padding: 1.5rem 1rem 22rem 5rem;
  transition: padding-right 0.28s ease, padding-bottom 0.28s ease;
}

.session-content-shell.chat-collapsed {
  padding-bottom: 5rem;
}

.session-tab-rail {
  position: fixed;
  left: 0.75rem;
  top: 50%;
  z-index: 45;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding: 0.5rem;
  transform: translateY(-50%);
  overflow: hidden;
  border: 1px solid rgba(126, 200, 227, 0.14);
  border-radius: 0.8rem;
  background: linear-gradient(180deg, rgba(14, 18, 33, 0.96), rgba(9, 13, 25, 0.98));
  box-shadow: 0 24px 60px rgba(0, 0, 0, 0.3);
}

.session-tab-rail::before {
  content: '';
  position: absolute;
  inset: 6px;
  pointer-events: none;
  border: 1px solid rgba(126, 200, 227, 0.08);
}

.session-tab-button {
  isolation: isolate;
  overflow: hidden;
  border-radius: 0.5rem !important;
  clip-path: polygon(0 12px, 12px 0, 100% 0, 100% calc(100% - 12px), calc(100% - 12px) 100%, 0 100%);
}

.session-tab-button::before {
  content: '';
  position: absolute;
  inset: 0;
  z-index: -1;
  background: linear-gradient(135deg, rgba(233, 69, 96, 0.12), transparent 38%, rgba(126, 200, 227, 0.08));
  opacity: 0;
  transition: opacity 0.2s ease;
}

.session-tab-button:hover::before,
.session-tab-button:focus-visible::before {
  opacity: 1;
}

.session-tab-button:hover {
  transform: translateX(2px);
}

.session-tooltip {
  pointer-events: none;
  position: absolute;
  left: calc(100% + 0.75rem);
  top: 50%;
  transform: translateY(-50%) translateX(-0.25rem);
  opacity: 0;
  white-space: nowrap;
  border: 1px solid rgba(126, 200, 227, 0.14);
  border-radius: 0.5rem;
  background: rgba(11, 16, 29, 0.96);
  padding: 0.45rem 0.85rem;
  font-size: 0.7rem;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: rgba(232, 232, 240, 0.92);
  transition: opacity 0.18s ease, transform 0.18s ease;
}

.session-tab-button:hover .session-tooltip {
  opacity: 1;
  transform: translateY(-50%) translateX(0);
  transition-delay: 0.5s;
}

.session-tab-button:focus-visible .session-tooltip {
  opacity: 1;
  transform: translateY(-50%) translateX(0);
}

.session-chat-launcher {
  position: fixed;
  right: 1rem;
  bottom: 1rem;
  z-index: 40;
  display: inline-flex;
  align-items: center;
  gap: 0.7rem;
  border-radius: 0.7rem;
  padding: 0.9rem 1rem;
  backdrop-filter: blur(18px);
  box-shadow: 0 24px 60px rgba(0, 0, 0, 0.28);
  clip-path: polygon(0 12px, 12px 0, 100% 0, 100% calc(100% - 12px), calc(100% - 12px) 100%, 0 100%);
}

.session-chat-dock {
  position: fixed;
  left: 0.5rem;
  right: 0.5rem;
  bottom: 0.5rem;
  z-index: 40;
  height: 20rem;
  overflow: hidden;
  border: 1px solid rgba(126, 200, 227, 0.14);
  border-radius: 0.8rem;
  background:
    linear-gradient(180deg, rgba(14, 18, 33, 0.98), rgba(9, 13, 25, 0.99)),
    repeating-linear-gradient(90deg, rgba(126, 200, 227, 0.04) 0 1px, transparent 1px 72px);
  backdrop-filter: blur(24px);
  box-shadow: 0 35px 80px rgba(0, 0, 0, 0.38);
}

.session-chat-dock::before {
  content: '';
  position: absolute;
  inset: 8px;
  pointer-events: none;
  border: 1px solid rgba(126, 200, 227, 0.1);
}

.session-chat-dock__shell {
  display: flex;
  height: 100%;
  overflow: hidden;
  border-radius: inherit;
  flex-direction: column;
}

.session-chat-message {
  border-radius: 0.65rem !important;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.03);
}

.session-empty-bay {
  border-color: rgba(126, 200, 227, 0.18) !important;
  background:
    repeating-linear-gradient(90deg, rgba(126, 200, 227, 0.08) 0 1px, transparent 1px 56px),
    repeating-linear-gradient(0deg, rgba(126, 200, 227, 0.08) 0 1px, transparent 1px 56px),
    linear-gradient(180deg, rgba(10, 16, 29, 0.84), rgba(7, 11, 22, 0.92)) !important;
}

.session-sheet-frame {
  padding: 1.75rem 1.75rem 2rem !important;
}

.session-sheet-grid {
  align-items: start;
}

.session-sheet-support-grid,
.session-inventory-layout {
  align-items: stretch;
}

.session-dossier-panel,
.session-profile-panel,
.session-attribute-panel,
.session-equipment-panel,
.session-inventory-board,
.session-inventory-summary,
.session-inventory-meta {
  min-height: 100%;
}

.session-dossier-panel {
  display: flex;
  flex-direction: column;
}

.session-profile-panel {
  padding: 1.75rem !important;
  align-self: start;
}

.session-profile-editor-grid {
  align-items: start;
}

.session-sheet-support-grid {
  margin-top: 1.75rem;
}

.session-inventory-board {
  padding: 2rem !important;
}

.session-empty-bay--inventory {
  position: relative;
  min-height: 34rem;
  overflow: hidden;
}

.session-inventory-grid-phantom {
  position: absolute;
  inset: 1.5rem;
  display: grid;
  grid-template-columns: repeat(8, minmax(0, 1fr));
  gap: 0.85rem;
  pointer-events: none;
}

.session-inventory-grid-phantom__cell {
  border: 1px solid rgba(126, 200, 227, 0.14);
  border-radius: 0.75rem;
  background: linear-gradient(180deg, rgba(18, 27, 49, 0.66), rgba(8, 13, 24, 0.74));
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.03);
}

.session-inventory-grid-copy {
  position: relative;
  z-index: 1;
  display: flex;
  min-height: 29rem;
  max-width: 24rem;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  margin-left: auto;
  margin-right: auto;
  text-align: center;
  padding: 2rem;
  backdrop-filter: blur(3px);
}

.session-inventory-sidebar {
  display: grid;
  gap: 1.25rem;
}

.session-inventory-summary,
.session-inventory-meta {
  padding: 1.5rem !important;
}

@media (min-width: 640px) {
  .session-content-shell {
    padding-left: 5.75rem;
    padding-right: 1.5rem;
  }

  .session-chat-dock {
    left: auto;
    width: 22rem;
  }

  .session-inventory-grid-phantom {
    grid-template-columns: repeat(10, minmax(0, 1fr));
  }
}

@media (min-width: 1280px) {
  .session-content-shell {
    padding-bottom: 2rem;
    padding-left: 7rem;
    padding-right: 29rem;
  }

  .session-content-shell.chat-collapsed {
    padding-bottom: 2rem;
    padding-right: 6rem;
  }

  .session-chat-dock {
    top: 5.85rem;
    bottom: 1.5rem;
    right: 1.5rem;
    left: auto;
    width: 24rem;
    height: auto;
  }

  .session-chat-launcher {
    right: 1.5rem;
    bottom: 1.5rem;
  }

  .session-sheet-grid {
    grid-template-columns: minmax(390px, 450px) minmax(0, 1fr);
    gap: 1.75rem;
  }

  .session-dossier-panel {
    align-self: start;
  }

  .session-profile-panel {
    min-height: 0;
  }

  .session-sheet-support-grid {
    grid-template-columns: minmax(0, 1.35fr) minmax(340px, 0.85fr);
  }

  .session-inventory-layout {
    grid-template-columns: minmax(0, 1fr) 22rem;
    gap: 1.75rem;
  }

  .session-inventory-sidebar {
    position: sticky;
    top: 7.35rem;
    align-self: start;
  }
}

@media (min-width: 1536px) {
  .session-content-shell {
    padding-right: 31rem;
  }

  .session-sheet-frame {
    padding: 2rem 2rem 2.25rem !important;
  }

  .session-empty-bay--inventory {
    min-height: 38rem;
  }
}
</style>