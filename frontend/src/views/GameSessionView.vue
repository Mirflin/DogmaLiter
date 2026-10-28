<script setup>
import {
  ArrowLeft as ArrowLeftIcon,
  ArrowLeftRight as ArrowLeftRightIcon,
  Check as CheckIcon,
  ChevronRight as ChevronRightIcon,
  ContactRound as ContactRoundIcon,
  Copy,
  Eye as EyeIcon,
  FolderOpen as FolderOpenIcon,
  LayoutGrid as LayoutGridIcon,
  MessageSquareText as MessageSquareTextIcon,
  Package as PackageIcon,
  Plus as PlusIcon,
  RefreshCw as RefreshCwIcon,
  Settings as SettingsIcon,
  Trash2 as Trash2Icon,
  TriangleAlert as TriangleAlertIcon,
  UserRound as UserRoundIcon,
  Users as UsersIcon,
  X as XIcon,
} from '@lucide/vue'
import { API_URL } from '@/api'
import { ATTRIBUTE_ICONS } from '@/assets/icons/attributeIcons'
import SessionBackdrop from '@/components/session/SessionBackdrop.vue'
import SessionCharacterPickerModal from '@/components/session/SessionCharacterPickerModal.vue'
import SessionGMCharacterCreateModal from '@/components/session/SessionGMCharacterCreateModal.vue'
import SessionGMCharacterEditorModal from '@/components/session/SessionGMCharacterEditorModal.vue'
import SessionInventoryBoard from '@/components/session/SessionInventoryBoard.vue'
import SessionItemCompendium from '@/components/session/SessionItemCompendium.vue'
import SearchableSelect from '@/components/session/SearchableSelect.vue'
import GameSettingsModal from '@/components/GameSettingsModal.vue'
import DataTable from '@/components/DataTable.vue'
import { getErrorMessage, notify } from '@/notify'
import { useAuthStore } from '@/stores/auth'
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useGameSocket } from '@/composables/useGameSocket'
import { useImageFallback } from '@/composables/useImageFallback'
import { useFillHeight } from '@/composables/useFillHeight'
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
const fillHeightSectionRef = ref(null)
const { heightPx: fillSectionHeight } = useFillHeight(fillHeightSectionRef)
const { markFailed: markImageFailed, isFailed: isImageFailed } = useImageFallback()
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
const characterOptions = computed(() => characters.value.map((character) => ({
  value: character.id,
  label: character.owner?.username ? `${character.name} · ${character.owner.username}` : character.name,
})))
const rosterCharacters = computed(() => {
  if (!isGM.value) return characters.value
  return characters.value.filter(character => character.user_id === viewer.value?.user_id)
})
const characterVaultRows = computed(() => rosterCharacters.value.map(character => ({
  ...character,
  sizeLabel: `${character.inventory_width}x${character.inventory_height}`,
})))
const campaignCharacterRows = computed(() => {
  const memberById = new Map((game.value?.members ?? []).map(member => [member.user_id, member]))
  return characters.value.map(character => {
    const member = memberById.get(character.user_id)
    return {
      ...character,
      ownerName: character.owner?.username || member?.username || 'Unknown',
      roleLabel: formatRole(member?.role || ''),
      createdAt: character.created_at || '',
      sizeLabel: `${character.inventory_width}x${character.inventory_height}`,
    }
  })
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
// Stat objects for the hover tooltip (same shape as base-attribute cards), so it
// shows which equipped items contribute to Health / Armor Class.
const healthStat = computed(() => ({
  label: 'Health',
  base: baseMaxHealth.value,
  value: effectiveMaxHealth.value,
  bonus: effectiveMaxHealth.value - baseMaxHealth.value,
  contributions: attributeContributions.value.health ?? [],
}))
const armorClassStat = computed(() => ({
  label: 'Armor Class',
  base: baseArmorClass.value,
  value: effectiveArmorClass.value,
  bonus: armorClassBonus.value,
  contributions: attributeContributions.value.armor_class ?? [],
}))
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
const duplicatingCharacterId = ref('')
async function duplicateCharacter(character) {
  if (!isGM.value || !character || duplicatingCharacterId.value) return

  duplicatingCharacterId.value = character.id
  try {
    const detail = await auth.getGameCharacter(gameId.value, character.id)
    const source = detail?.character
    if (!source) throw new Error('Could not load character to duplicate')

    const created = await createCharacter({
      name: `${source.name} (Copy)`.slice(0, 100),
      backstory: source.backstory || '',
      owner_user_id: source.user_id,
    }, { silent: true })
    if (!created) return

    await persistCharacterChanges(created, {
      name: created.name,
      backstory: source.backstory || '',
      owner_user_id: source.user_id,
      inventory_width: source.inventory_width,
      inventory_height: source.inventory_height,
      currency_gold: source.currency_gold,
      currency_silver: source.currency_silver,
      currency_copper: source.currency_copper,
      max_health: source.max_health,
      current_health: source.current_health,
      armor_class: source.armor_class,
      base_attributes: {
        strength: source.base_attributes?.strength ?? 10,
        dexterity: source.base_attributes?.dexterity ?? 10,
        constitution: source.base_attributes?.constitution ?? 10,
        intelligence: source.base_attributes?.intelligence ?? 10,
        wisdom: source.base_attributes?.wisdom ?? 10,
        charisma: source.base_attributes?.charisma ?? 10,
      },
      custom_attributes: (source.custom_attributes ?? []).map((attribute, index) => ({
        name: attribute.name,
        value: attribute.value ?? 0,
        sort_order: attribute.sort_order ?? index,
      })),
    })

    notify.success({
      title: 'Character duplicated',
      message: `Cloned "${source.name}" as "${created.name}". Inventory and equipment aren't copied.`,
    })
  } catch (error) {
    notify.error({
      title: 'Failed to duplicate character',
      message: getErrorMessage(error, 'Failed to duplicate character'),
    })
  } finally {
    duplicatingCharacterId.value = ''
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
const currencyCards = computed(() => {
  const snapshot = characterSnapshot.value

  return [
    { key: 'gold', label: 'Gold', value: snapshot?.currency_gold ?? 0 },
    { key: 'silver', label: 'Silver', value: snapshot?.currency_silver ?? 0 },
    { key: 'copper', label: 'Bronze', value: snapshot?.currency_copper ?? 0 },
  ]
})
const currentPortraitUrl = computed(() => portraitPreviewUrl.value || avatarUrl(characterSnapshot.value?.portrait_id))
const rosterTitle = computed(() => (isGM.value ? 'Character Vault' : 'Character Vault'))
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

  // Switching characters lives in the header for everyone, so the roster is only a tab while a player
  // still has to pick one. GMs get the whole party, the item compendium and the table's settings.
  const baseTabs = [
    { id: 'sheet', label: 'Character', icon: 'sheet', description: 'Portrait, identity, profile editing, and read-only character stats.' },
    { id: 'inventory', label: 'Inventory', icon: 'inventory', description: 'Equipment, backpack and currency for the active character.' },
  ]

  if (isGM.value) {
    baseTabs.push(
      { id: 'players', label: 'Party', icon: 'players', description: 'Every character in the campaign, with the GM tools for each.' },
      { id: 'items', label: 'Compendium', icon: 'items', description: 'Campaign compendium for item browsing, tagging, and creation.' },
      { id: 'manage', label: 'Table', icon: 'manage', description: 'Members and invites, chat and activity logs, and game settings.' },
    )
  }

  return baseTabs
})
const activeTabMeta = computed(() => tabs.value.find(tab => tab.id === activeTab.value) ?? tabs.value[0] ?? null)
// These GM tabs are game-wide, not tied to a single character, so the
// character-context header (name/owner/picker) has nothing to show.
const CHARACTERLESS_TABS = new Set(['players', 'items', 'manage'])
const showCharacterHeader = computed(() => !CHARACTERLESS_TABS.has(activeTab.value))

// Inventory tab: the board stays centred. The character summary sits beside it when both sides have
// room for it, otherwise it becomes a strip above the board. Measured, because the board's width
// follows the character's backpack size and the chat dock takes space when open.
const deskRef = ref(null)
const boardWrap = ref(null)
const deskMode = ref('stack')
const boardWidth = ref(0)
const SIDE_MIN_PX = 13.5 * 16
const DESK_GAP_PX = 24
let deskObserver = null

function measureDesk() {
  const desk = deskRef.value
  const board = boardWrap.value?.firstElementChild
  if (!desk || !board) return
  boardWidth.value = Math.round(board.getBoundingClientRect().width)
  const side = (desk.clientWidth - boardWidth.value - 2 * DESK_GAP_PX) / 2
  deskMode.value = side >= SIDE_MIN_PX ? 'sides' : 'stack'
}

watch([deskRef, boardWrap], ([desk, wrap]) => {
  deskObserver?.disconnect()
  if (!desk || !wrap || typeof ResizeObserver === 'undefined') return
  deskObserver = new ResizeObserver(() => measureDesk())
  deskObserver.observe(desk)
  if (wrap.firstElementChild) deskObserver.observe(wrap.firstElementChild)
  measureDesk()
})
onBeforeUnmount(() => deskObserver?.disconnect())

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
  const url = `${API_URL}/api/uploads/${uploadId}`
  return isImageFailed(url) ? null : url
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
    common: 'border-ink/32 bg-ink/8 text-ink',
    uncommon: 'border-warning/35 bg-warning/16 text-warning',
    rare: 'border-sky/35 bg-primary/16 text-sky',
    epic: 'border-r-epic/35 bg-r-epic/18 text-r-epic',
    masterwork: 'border-warning/35 bg-warning/18 text-r-masterwork',
    legendary: 'border-success/35 bg-success/18 text-success',
    unique: 'border-danger/40 bg-danger/18 text-danger',
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
  <div class="session-stage min-h-screen w-screen overflow-x-hidden text-ink">
    <SessionBackdrop />
    <div v-if="loading" class="flex min-h-screen flex-col items-center justify-center px-6 text-center">
      <div class="h-11 w-11 animate-spin rounded-full border-2 border-primary border-t-transparent"></div>
      <p class="mt-4 text-[14px] text-faint">Loading session workspace...</p>
    </div>

    <div v-else-if="error" class="flex min-h-screen flex-col items-center justify-center px-6 text-center">
      <TriangleAlertIcon class="mb-4 h-16 w-16 text-primary-ink" :stroke-width="1.5" />
      <p class="mb-5 text-[16px] font-semibold text-primary-ink">{{ error }}</p>
      <div class="flex flex-wrap justify-center gap-3">
        <button
          @click="loadSession({ preserveCharacter: false, promptSelection: true })"
          class="cursor-pointer rounded-xl border border-line bg-n3 px-5 py-2.5 text-[14px] text-ink transition-all duration-200 hover:border-line-strong"
        >
          Retry
        </button>
        <button
          @click="router.push('/games')"
          class="cursor-pointer rounded-xl border border-primary/22 bg-transparent px-5 py-2.5 text-[14px] text-sky transition-all duration-200 hover:border-primary hover:text-primary-ink"
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

      <header class="session-header">
        <div class="session-header__inner">
          <button type="button" class="session-chrome session-icon-btn" title="Leave the session" aria-label="Leave the session" @click="goBack">
            <ArrowLeftIcon class="h-5 w-5" :stroke-width="2" />
          </button>

          <div class="session-header__title">
            <h1>{{ game.title }}</h1>
            <span v-if="isGM" class="session-gm-badge" title="You run this game">GM</span>
          </div>

          <nav class="session-tabs" aria-label="Session sections">
            <button
              v-for="tab in tabs"
              :key="tab.id"
              type="button"
              class="session-tab-button"
              :class="{ 'is-active': activeTab === tab.id }"
              :aria-current="activeTab === tab.id ? 'page' : undefined"
              @click="activeTab = tab.id"
            >
              <component :is="tabIcons[tab.icon] || UserRoundIcon" class="h-4 w-4" :stroke-width="2" />
              {{ tab.label }}
            </button>
          </nav>

          <div class="session-header__actions">
            <div v-if="showCharacterHeader && characters.length" class="session-switcher">
              <SearchableSelect
                :model-value="activeCharacterId"
                :options="characterOptions"
                placeholder="Choose a character"
                class="session-switcher__select"
                @update:model-value="$event && switchCharacter($event, { nextTab: activeTab })"
              />
              <button
                v-if="viewer.can_create_character && !isGM"
                type="button"
                class="session-chrome session-icon-btn"
                :disabled="creatingCharacter"
                title="New character"
                aria-label="New character"
                @click="requestCharacterCreation"
              >
                <PlusIcon class="h-4 w-4" :stroke-width="2.2" />
              </button>
            </div>

            <button
              v-if="activeCharacterId && (tradingEnabled || incomingTradeCount)"
              type="button"
              class="session-chrome session-action-btn"
              :class="{ 'has-news': incomingTradeCount }"
              @click="openTradeModal(incomingTradeCount ? 'incoming' : 'send')"
            >
              <ArrowLeftRightIcon class="h-4 w-4" :stroke-width="2" />
              <span class="session-action-btn__label">Trade</span>
              <span v-if="incomingTradeCount" class="session-count">{{ incomingTradeCount }}</span>
            </button>

            <button
              v-if="game?.enable_chat"
              type="button"
              class="session-chrome session-action-btn"
              :class="{ 'is-on': !chatCollapsed }"
              :aria-pressed="!chatCollapsed"
              @click="chatCollapsed ? focusChatInput() : (chatCollapsed = true)"
            >
              <MessageSquareTextIcon class="h-4 w-4" :stroke-width="2" />
              <span class="session-action-btn__label">Chat</span>
              <span v-if="chatMessages.length" class="session-count session-count--quiet">{{ chatMessages.length }}</span>
            </button>
          </div>
        </div>
      </header>

      <!-- Phones: the same sections as a labelled bar along the bottom edge. -->
      <nav class="session-tabbar" aria-label="Session sections">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          type="button"
          class="session-tab-button"
          :class="{ 'is-active': activeTab === tab.id }"
          :aria-current="activeTab === tab.id ? 'page' : undefined"
          @click="activeTab = tab.id"
        >
          <component :is="tabIcons[tab.icon] || UserRoundIcon" class="h-5 w-5" :stroke-width="1.9" />
          <span>{{ tab.label }}</span>
        </button>
      </nav>

      <div class="mx-auto max-w-[1920px] session-content-shell" :class="{ 'chat-collapsed': chatCollapsed || !game?.enable_chat }">
        <main class="min-w-0 @container">

          <div v-if="characterError" class="session-banner mb-6 rounded-[1.5rem] border border-primary/28 bg-primary/12 px-5 py-4 text-[14px] text-primary-soft">
            {{ characterError }}
          </div>

          <section v-if="activeTab === 'sheet'" class="session-sheet-layout space-y-7">
            <div class="session-sheet-frame">
              <template v-if="characterSnapshot">
                <div v-if="profileEditMode && canEditCharacter" class="session-sheet-grid grid gap-6 xl:grid-cols-[minmax(360px,430px)_minmax(0,1fr)]">
                  <article class="session-dossier-panel overflow-hidden rounded-[1.9rem] border border-line bg-n1/78">
                    <div class="flex items-center justify-center bg-n0" :class="currentPortraitUrl ? 'h-[24rem] sm:h-[28rem]' : 'h-44'">
                      <img
                        v-if="currentPortraitUrl"
                        :src="currentPortraitUrl"
                        :alt="characterSnapshot.name"
                        class="h-full w-full object-cover object-top"
                        @error="markImageFailed(currentPortraitUrl)"
                      />
                      <div v-else class="num flex h-24 w-24 items-center justify-center rounded-[1.4rem] bg-n3 text-[40px] font-bold text-primary-ink">
                        {{ initials(characterSnapshot.name) }}
                      </div>
                    </div>

                    <div class="border-t border-line-soft p-5">
                      <div class="flex items-start justify-between gap-3">
                        <div class="min-w-0">
                          <p class="text-[13px] font-semibold text-dim">Character</p>
                          <h3 class="mt-2 truncate font-heading text-[30px] font-bold text-ink">{{ characterSnapshot.name }}</h3>
                          <p class="mt-2 text-[13px] text-ink-soft/62">{{ characterSnapshot.owner?.username || 'Unknown owner' }}</p>
                        </div>

                        <div v-if="characterLoading" class="inline-flex items-center gap-2 rounded-full border border-line bg-n3 px-3 py-1.5 text-[13px] font-semibold text-sky">
                          <span class="h-2 w-2 animate-pulse rounded-full bg-sky"></span>
                          Loading
                        </div>
                      </div>
                    </div>
                  </article>

                  <article class="session-profile-panel rounded-[1.9rem] border border-line bg-n1/78 p-5 sm:p-6">
                    <div class="flex flex-wrap items-start justify-between gap-4">
                      <div>
                        <p class="text-[13px] font-semibold text-dim">Profile</p>
                      </div>

                      <div class="flex flex-wrap gap-3">
                        <button
                          v-if="isGM"
                          @click="openGMCharacterEditor(characterSnapshot)"
                          class="cursor-pointer rounded-xl border border-warning/24 bg-warning/16 px-4 py-2.5 text-[13px] font-semibold text-ink transition-all duration-200 hover:border-warning/40"
                        >
                          GM Configure
                        </button>
                        <button
                          v-if="canEditCharacter && !profileEditMode"
                          @click="openProfileEditor"
                          class="cursor-pointer rounded-xl border border-primary/24 bg-primary/12 px-4 py-2.5 text-[13px] font-semibold text-primary-soft transition-all duration-200 hover:border-primary/42"
                        >
                          Edit Profile
                        </button>
                        <button
                          v-if="profileEditMode && canDeleteCharacter(characterSnapshot)"
                          @click="requestCharacterDeletion(characterSnapshot)"
                          class="inline-flex cursor-pointer items-center gap-1.5 rounded-xl border border-danger/40 bg-danger/12 px-4 py-2.5 text-[13px] font-semibold text-danger-soft transition-all duration-200 hover:border-danger/65"
                        >
                          <Trash2Icon class="h-4 w-4" :stroke-width="2" /> Delete character
                        </button>
                        <button
                          v-if="profileEditMode"
                          @click="cancelProfileEditor"
                          class="cursor-pointer rounded-xl border border-line bg-n3 px-4 py-2.5 text-[13px] font-semibold text-ink transition-all duration-200 hover:border-line-strong"
                        >
                          Cancel
                        </button>
                      </div>
                    </div>

                    <p v-if="!canEditCharacter" class="mt-5 rounded-[1.3rem] border border-line bg-n3 px-4 py-3 text-[13px] text-ink-soft/68">
                      This character is read-only for your current access.
                    </p>

                    <div v-if="profileEditMode && canEditCharacter" class="mt-6 space-y-5">
                      <div class="session-profile-editor-grid grid gap-5 xl:grid-cols-[minmax(260px,320px)_minmax(0,1fr)]">
                        <div class="space-y-4 rounded-[1.5rem] border border-line-soft bg-n3 p-4">
                          <p class="text-[13px] font-semibold text-dim">Portrait</p>
                          <div class="overflow-hidden rounded-[1.4rem] border border-line bg-n1/66">
                            <div class="flex h-64 items-center justify-center bg-[linear-gradient(180deg,rgba(16,32,52,0.94),rgba(10,16,41,0.94))]">
                              <img v-if="currentPortraitUrl" :src="currentPortraitUrl" :alt="characterSnapshot.name" class="h-full w-full object-cover object-top" @error="markImageFailed(currentPortraitUrl)" />
                              <div v-else class="flex h-28 w-28 items-center justify-center rounded-full border-2 border-line-strong bg-[linear-gradient(135deg,rgba(123,135,184,0.16),rgba(88,101,221,0.06))] font-heading text-[34px] font-bold text-sky">{{ initials(characterSnapshot.name) }}</div>
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
                              class="cursor-pointer rounded-xl border border-line bg-n3 px-4 py-2.5 text-[13px] font-semibold text-ink transition-all duration-200 hover:border-line-strong"
                            >
                              {{ portraitFile ? 'Replace Portrait' : 'Upload Portrait' }}
                            </button>
                            <button
                              v-if="portraitFile"
                              @click="resetPortraitSelection"
                              type="button"
                              class="cursor-pointer rounded-xl border border-primary/18 bg-primary/8 px-4 py-2.5 text-[13px] font-semibold text-primary-soft transition-all duration-200 hover:border-primary/35"
                            >
                              Clear Selection
                            </button>
                          </div>
                          <p class="text-[12px] text-ink-soft/54">JPEG, PNG, WebP. Up to 5MB.</p>
                        </div>

                        <div class="space-y-5">
                          <label class="block">
                            <span class="text-[13px] font-semibold text-dim">Name</span>
                            <input
                              v-model="characterForm.name"
                              type="text"
                              maxlength="100"
                              :disabled="characterSavePending"
                              class="session-input mt-2 w-full rounded-[1.25rem] border border-line bg-n1/72 px-4 py-3 text-[15px] text-ink outline-none transition-all duration-200 placeholder:text-faint focus:border-primary/34 disabled:cursor-not-allowed disabled:opacity-60"
                            />
                          </label>

                          <label class="block">
                            <span class="text-[13px] font-semibold text-dim">Backstory</span>
                            <textarea
                              v-model="characterForm.backstory"
                              rows="6"
                              :disabled="characterSavePending"
                              class="session-input mt-2 w-full resize-none rounded-[1.25rem] border border-line bg-n1/72 px-4 py-3 text-[15px] text-ink outline-none transition-all duration-200 placeholder:text-faint focus:border-primary/34 disabled:cursor-not-allowed disabled:opacity-60"
                            ></textarea>
                          </label>

                          <div>
                            <div class="flex items-center justify-between gap-3">
                              <span class="text-[13px] font-semibold text-dim">Currency</span>
                            </div>
                            <div class="mt-4 grid gap-3 @xl:grid-cols-3">
                              <label class="rounded-[1.3rem] border border-line-soft bg-n3 px-4 py-3">
                                <span class="text-[13px] font-semibold text-faint">Gold</span>
                                <input v-model.number="characterForm.gold" type="number" min="0" class="session-input mt-3 pl-2 w-full border-0 bg-transparent p-0 text-[24px] font-semibold text-ink outline-none" />
                              </label>
                              <label class="rounded-[1.3rem] border border-line-soft bg-n3 px-4 py-3">
                                <span class="text-[13px] font-semibold text-faint">Silver</span>
                                <input v-model.number="characterForm.silver" type="number" min="0" class="session-input mt-3 pl-2 w-full border-0 bg-transparent p-0 text-[24px] font-semibold text-ink outline-none" />
                              </label>
                              <label class="rounded-[1.3rem] border border-line-soft bg-n3 px-4 py-3">
                                <span class="text-[13px] font-semibold text-faint">Bronze</span>
                                <input v-model.number="characterForm.copper" type="number" min="0" class="session-input mt-3 pl-2 w-full border-0 bg-transparent p-0 text-[24px] font-semibold text-ink outline-none" />
                              </label>
                            </div>
                          </div>

                          <div v-if="enableHealth">
                            <span class="text-[13px] font-semibold text-dim">Health</span>
                            <label class="mt-4 block rounded-[1.3rem] border border-line-soft bg-n3 px-4 py-3">
                              <span class="text-[13px] font-semibold text-faint">Current HP (max {{ effectiveMaxHealth }})</span>
                              <input v-model.number="characterForm.currentHealth" type="number" min="0" :max="effectiveMaxHealth" class="session-input mt-3 pl-2 w-full border-0 bg-transparent p-0 text-[24px] font-semibold text-ink outline-none" />
                            </label>
                          </div>
                        </div>
                      </div>

                      <div class="flex flex-wrap gap-3">
                        <button
                          @click="saveCharacterProfile"
                          :disabled="characterSavePending"
                          class="cursor-pointer rounded-xl border border-primary/28 bg-[linear-gradient(135deg,rgba(88,101,221,0.9),rgba(74,86,200,0.9))] px-4 py-2.5 text-[13px] font-semibold text-white transition-all duration-200 disabled:cursor-not-allowed disabled:opacity-60"
                        >
                          {{ characterSavePending ? 'Saving...' : 'Save Profile' }}
                        </button>
                      </div>
                    </div>

                  </article>
                </div>

                <!-- Read-only sheet: three columns that fit one screen on desktop; long lists scroll in place. -->
                <div v-else class="sheet">
                  <section class="sheet__col sheet__who">
                    <div class="sheet__portrait" :class="{ 'has-image': currentPortraitUrl }">
                      <img v-if="currentPortraitUrl" :src="currentPortraitUrl" :alt="characterSnapshot.name" @error="markImageFailed(currentPortraitUrl)" />
                      <span v-else class="num">{{ initials(characterSnapshot.name) }}</span>
                    </div>
                    <div>
                      <h2 class="sheet__name">{{ characterSnapshot.name }}</h2>
                      <p class="sheet__owner">{{ characterSnapshot.owner?.username || 'Unknown owner' }}</p>
                    </div>

                    <div v-if="enableHealth || enableArmorClass" class="sheet__vitals">
                      <div v-if="enableHealth" class="sheet-stat sheet-stat--wide" @mouseenter="showStatTooltip($event, healthStat)" @mouseleave="hideStatTooltip">
                        <component :is="ATTRIBUTE_ICONS.health" class="sheet-stat__icon text-danger" :stroke-width="2" />
                        <span class="sheet-stat__label">Health</span>
                        <span class="sheet-stat__value num">{{ currentHealth }}<small>/{{ effectiveMaxHealth }}</small></span>
                        <span class="sheet-hp" aria-hidden="true"><i :style="{ width: `${healthPercent}%` }" /></span>
                      </div>
                      <div v-if="enableArmorClass" class="sheet-stat" @mouseenter="showStatTooltip($event, armorClassStat)" @mouseleave="hideStatTooltip">
                        <component :is="ATTRIBUTE_ICONS.armor_class" class="sheet-stat__icon text-sky" :stroke-width="2" />
                        <span class="sheet-stat__label">Armor class</span>
                        <span class="sheet-stat__value num">{{ effectiveArmorClass }}<em v-if="armorClassBonus" class="sheet-bonus" :class="{ 'is-neg': armorClassBonus < 0 }">{{ armorClassBonus > 0 ? '+' : '' }}{{ armorClassBonus }}</em></span>
                      </div>
                    </div>

                    <ul class="sheet__coins" aria-label="Currency">
                      <li v-for="coin in currencyCards" :key="coin.key" :class="`inv-coin--${coin.key}`">
                        <span class="num">{{ coin.value }}</span>
                        <span class="sheet-stat__label">{{ coin.label }}</span>
                      </li>
                    </ul>
                  </section>

                  <section class="sheet__col sheet__main">
                    <div class="sheet__head">
                      <h3 class="sheet__title">Attributes</h3>
                      <div class="flex flex-wrap gap-2">
                        <button v-if="isGM" type="button" class="session-chrome session-action-btn" @click="openGMCharacterEditor(characterSnapshot)">GM configure</button>
                        <button v-if="canEditCharacter" type="button" class="session-chrome session-action-btn" @click="openProfileEditor">Edit profile</button>
                      </div>
                    </div>
                    <p v-if="!canEditCharacter" class="sheet__note">You can view this character but not change it.</p>

                    <div class="sheet__scroll">
                      <ul v-if="attributesEnabled && attributeCards.length" class="sheet__attrs" aria-label="Base attributes">
                        <li v-for="attribute in attributeCards" :key="attribute.key" class="sheet-stat" @mouseenter="showStatTooltip($event, attribute)" @mouseleave="hideStatTooltip">
                          <component :is="ATTRIBUTE_ICONS[attribute.key]" class="sheet-stat__icon" :stroke-width="2" />
                          <span class="sheet-stat__label">{{ attribute.label }}</span>
                          <span class="sheet-stat__value num">{{ attribute.value }}<em v-if="attribute.bonus" class="sheet-bonus" :class="{ 'is-neg': attribute.bonus < 0 }">{{ attribute.bonus > 0 ? '+' : '' }}{{ attribute.bonus }}</em></span>
                        </li>
                      </ul>

                      <template v-if="customAttributeCards.length">
                        <h4 class="sheet__subtitle">Campaign attributes</h4>
                        <ul class="sheet__attrs sheet__attrs--custom" aria-label="Campaign attributes">
                          <li v-for="attribute in customAttributeCards" :key="attribute.id" class="sheet-stat sheet-stat--plain" @mouseenter="showStatTooltip($event, attribute)" @mouseleave="hideStatTooltip">
                            <span class="sheet-stat__label">{{ attribute.name }}</span>
                            <span class="sheet-stat__value num">{{ attribute.value }}<em v-if="attribute.bonus" class="sheet-bonus" :class="{ 'is-neg': attribute.bonus < 0 }">{{ attribute.bonus > 0 ? '+' : '' }}{{ attribute.bonus }}</em></span>
                          </li>
                        </ul>
                      </template>
                      <p v-else-if="isGM" class="sheet__empty">No campaign attributes on this character. Add them in GM configure.</p>
                    </div>
                  </section>

                  <section class="sheet__col sheet__side">
                    <div class="sheet__block sheet__block--story">
                      <h3 class="sheet__title">Backstory</h3>
                      <p class="sheet__story" :class="{ 'is-empty': !characterSnapshot.backstory }">{{ characterSnapshot.backstory || 'No backstory yet.' }}<template v-if="!characterSnapshot.backstory && canEditCharacter"> Use Edit profile to write one.</template></p>
                    </div>

                    <div class="sheet__block">
                      <div class="sheet__head">
                        <h3 class="sheet__title">Equipped</h3>
                        <span class="sheet__count num">{{ equipment.length }}</span>
                      </div>
                      <ul v-if="equipment.length" class="sheet__list">
                        <li v-for="slot in equipment" :key="`${slot.slot}-${slot.inventory_item_id}`" class="is-link" :style="{ '--resin': `var(--color-r-${slot.inventory_item?.item?.rarity || 'common'})` }" @click="viewEquipmentItem(slot.inventory_item)">
                          <span class="sheet__list-name"><i class="sheet__dot" />{{ slot.inventory_item?.item?.name || 'Unknown item' }}</span>
                          <span class="sheet__slot">{{ formatRole(slot.slot) }}</span>
                        </li>
                      </ul>
                      <p v-else class="sheet__empty">Nothing equipped. Drag gear onto slots in Inventory.</p>
                    </div>
                  </section>
                </div>
              </template>

              <div v-else class="flex min-h-[520px] flex-col items-center justify-center text-center">
                <h2 class="font-heading text-[28px] font-bold text-ink">Choose A Character First</h2>
                <p class="mt-3 max-w-[30rem] text-[14px] leading-relaxed text-ink-soft/58">The session opens through the character picker. Select an existing character or create a new blank one with a random name.</p>
                <button
                  @click="openCharacterPicker"
                  class="mt-5 cursor-pointer rounded-xl border border-primary/22 bg-primary/8 px-5 py-3 text-[14px] font-semibold text-primary-soft transition-all duration-200 hover:border-primary/45"
                >
                  Open Character Picker
                </button>
              </div>
            </div>
          </section>

          <section v-else-if="activeTab === 'inventory'" ref="deskRef" class="inv-desk" :class="`inv-desk--${deskMode}`" :style="{ '--board-w': `${boardWidth}px` }">
            <!-- The player's side of the table: who they are, what they carry, what is waiting for them. -->
            <aside v-if="characterSnapshot" class="inv-desk__summary">
              <div class="inv-who">
                <div class="inv-who__portrait">
                  <img v-if="currentPortraitUrl" :src="currentPortraitUrl" :alt="characterSnapshot.name" @error="markImageFailed(currentPortraitUrl)" />
                  <span v-else class="num">{{ initials(characterSnapshot.name) }}</span>
                </div>
                <div class="min-w-0">
                  <p class="inv-who__name">{{ characterSnapshot.name }}</p>
                  <p class="inv-who__owner">{{ characterSnapshot.owner?.username || '—' }}</p>
                </div>
              </div>

              <div v-if="enableHealth || enableArmorClass" class="inv-vitals">
                <div v-if="enableHealth" class="inv-vital" @mouseenter="showStatTooltip($event, healthStat)" @mouseleave="hideStatTooltip">
                  <span class="inv-label inline-flex items-center gap-1"><component :is="ATTRIBUTE_ICONS.health" class="h-3.5 w-3.5 text-danger" :stroke-width="2" />Health</span>
                  <span class="num inv-vital__value">{{ currentHealth }}<small>/{{ effectiveMaxHealth }}</small></span>
                  <span class="inv-vital__bar" aria-hidden="true"><i :style="{ width: `${healthPercent}%` }" /></span>
                </div>
                <div v-if="enableArmorClass" class="inv-vital" @mouseenter="showStatTooltip($event, armorClassStat)" @mouseleave="hideStatTooltip">
                  <span class="inv-label inline-flex items-center gap-1"><component :is="ATTRIBUTE_ICONS.armor_class" class="h-3.5 w-3.5 text-sky" :stroke-width="2" />Armor class</span>
                  <span class="num inv-vital__value">{{ effectiveArmorClass }}<em v-if="armorClassBonus" class="inv-bonus">{{ armorClassBonus > 0 ? '+' : '' }}{{ armorClassBonus }}</em></span>
                </div>
              </div>

              <ul v-if="attributeCards.length" class="inv-stats" aria-label="Attributes">
                <li v-for="attribute in attributeCards" :key="attribute.key" :title="attribute.label" @mouseenter="showStatTooltip($event, attribute)" @mouseleave="hideStatTooltip">
                  <span class="inv-label inline-flex items-center gap-1"><component :is="ATTRIBUTE_ICONS[attribute.key]" class="h-3.5 w-3.5" :stroke-width="2" />{{ attribute.label.slice(0, 3).toUpperCase() }}</span>
                  <span class="num inv-stats__value">{{ attribute.value }}</span>
                  <em v-if="attribute.bonus" class="inv-bonus num" :class="{ 'is-neg': attribute.bonus < 0 }">{{ attribute.bonus > 0 ? '+' : '' }}{{ attribute.bonus }}</em>
                </li>
              </ul>

              <ul class="inv-coins" aria-label="Currency">
                <li v-for="coin in currencyCards" :key="coin.key" :class="`inv-coin--${coin.key}`">
                  <span class="num">{{ coin.value }}</span>
                  <span class="inv-label">{{ coin.label }}</span>
                </li>
              </ul>

            </aside>

            <div ref="boardWrap" class="inv-desk__board">
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
            </div>
            <aside v-if="tradeIncoming.length" class="inv-desk__offers inv-offers" aria-label="Trade offers">
                <p class="inv-offers__title">Waiting for you</p>
                <div v-for="offer in tradeIncoming" :key="offer.id" class="inv-offer">
                  <p class="inv-offer__text"><span class="text-orchid font-semibold">{{ offer.from_character_name || offer.from_username }}</span> offers {{ offer.items.map(entry => entry.item?.name).filter(Boolean).join(', ') }}<template v-if="offer.to_character_name"> to {{ offer.to_character_name }}</template>.</p>
                  <div class="inv-offer__actions">
                    <button type="button" class="session-chrome session-action-btn" :disabled="tradeBusy" @click="declineTradeOffer(offer)">Decline</button>
                    <button type="button" class="session-primary-btn" :disabled="tradeBusy" @click="acceptTradeOffer(offer)"><CheckIcon class="h-4 w-4" :stroke-width="2.4" /> Accept</button>
                  </div>
                </div>
              </aside>
          </section>

          <section v-else-if="activeTab === 'characters'" ref="fillHeightSectionRef" class="flex min-h-[42rem] flex-col gap-5 xl:min-h-0" :style="fillSectionHeight ? { height: fillSectionHeight } : {}">
            <article class="shrink-0 rounded-[2rem] border border-line bg-n1/88 p-5 shadow-[0_24px_60px_rgba(0,0,0,0.22)] sm:p-6">
              <div class="flex flex-wrap items-start justify-between gap-4">
                <div>
                  <p class="text-[13px] font-semibold text-dim">Character Switcher</p>
                  <h3 class="mt-2 font-heading text-[30px] font-bold text-ink">{{ rosterTitle }}</h3>
                </div>

                <div class="flex flex-wrap gap-3">
                  <button
                    v-if="viewer.can_create_character"
                    @click="requestCharacterCreation"
                    :disabled="creatingCharacter"
                    class="cursor-pointer rounded-xl border border-primary/24 bg-primary/12 px-4 py-2.5 text-[13px] font-semibold text-primary-soft transition-all duration-200 hover:border-primary/45 disabled:cursor-not-allowed disabled:opacity-60"
                  >
                    {{ creatingCharacter ? 'Creating...' : (isGM ? 'Create GM Character' : 'Create New Character') }}
                  </button>
                </div>
              </div>
            </article>

            <DataTable
              :columns="[
                { key: 'name', label: 'Character', sortable: true },
                { key: 'sizeLabel', label: 'Grid' },
                { key: 'actions', label: '', align: 'right' },
              ]"
              :rows="characterVaultRows"
              row-key="id"
              :search-keys="['name', 'backstory']"
              search-placeholder="Search characters"
              :page-size="20"
              min-width="720px"
              fill-height
              class="flex-1 min-h-0"
              :empty-text="isGM ? 'No GM characters yet — create one to stage it for the campaign.' : 'No characters available yet.'"
            >
              <template #cell-name="{ row }">
                <div class="flex items-center gap-3">
                  <div class="flex h-10 w-10 shrink-0 items-center justify-center overflow-hidden rounded-xl border border-line bg-n3">
                    <img v-if="avatarUrl(row.portrait_id)" :src="avatarUrl(row.portrait_id)" :alt="row.name" class="h-full w-full object-cover" @error="markImageFailed(avatarUrl(row.portrait_id))" />
                    <span v-else class="font-heading text-[13px] font-bold text-sky">{{ initials(row.name) }}</span>
                  </div>
                  <div class="min-w-0 flex items-center gap-2">
                    <span class="truncate font-semibold text-ink">{{ row.name }}</span>
                    <span v-if="row.id === activeCharacterId" class="shrink-0 rounded-full bg-primary/20 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-[0.14em] text-primary-soft">Active</span>
                  </div>
                </div>
              </template>
              <template #cell-sizeLabel="{ row }"><span class="text-dim">{{ row.sizeLabel }}</span></template>
              <template #cell-actions="{ row }">
                <div class="flex flex-wrap justify-end gap-2">
                  <button
                    @click="switchCharacter(row.id, { nextTab: activeTab })"
                    :disabled="row.id === activeCharacterId"
                    class="rounded-lg border px-3 py-1.5 text-[12px] font-semibold transition-all duration-200 disabled:cursor-not-allowed"
                    :class="row.id === activeCharacterId
                      ? 'border-primary/34 bg-primary/20 text-primary-soft'
                      : 'cursor-pointer border-primary/24 bg-primary/12 text-primary-soft hover:border-primary/45'"
                  >
                    {{ row.id === activeCharacterId ? 'Active' : 'Make Active' }}
                  </button>
                  <button
                    @click="switchCharacter(row.id, { nextTab: 'sheet' })"
                    class="cursor-pointer rounded-lg border border-line bg-n3 px-3 py-1.5 text-[12px] font-semibold text-ink transition-all duration-200 hover:border-line-strong"
                  >
                    Open
                  </button>
                  <button
                    v-if="isGM"
                    @click="openGMCharacterEditor(row)"
                    class="cursor-pointer rounded-lg border border-warning/24 bg-warning/16 px-3 py-1.5 text-[12px] font-semibold text-ink transition-all duration-200 hover:border-warning/40"
                  >
                    Configure
                  </button>
                  <button
                    v-if="isGM"
                    @click="duplicateCharacter(row)"
                    :disabled="duplicatingCharacterId === row.id"
                    class="inline-flex cursor-pointer items-center gap-1.5 rounded-lg border border-line bg-n3 px-3 py-1.5 text-[12px] font-semibold text-ink transition-all duration-200 hover:border-line-strong disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    <Copy class="h-3.5 w-3.5" :stroke-width="2" /> {{ duplicatingCharacterId === row.id ? 'Duplicating…' : 'Duplicate' }}
                  </button>
                  <button
                    v-if="canDeleteCharacter(row)"
                    @click="requestCharacterDeletion(row)"
                    class="inline-flex cursor-pointer items-center gap-1.5 rounded-lg border border-danger/40 bg-danger/18 px-3 py-1.5 text-[12px] font-semibold text-danger-soft transition-all duration-200 hover:border-danger/65 hover:bg-danger/28"
                  >
                    <Trash2Icon class="h-3.5 w-3.5" :stroke-width="2" /> Delete
                  </button>
                </div>
              </template>
            </DataTable>
          </section>

          <section v-else-if="activeTab === 'players' && isGM" ref="fillHeightSectionRef" class="flex min-h-[42rem] w-full flex-col xl:min-h-0" :style="fillSectionHeight ? { height: fillSectionHeight } : {}">
            <article class="flex flex-1 flex-col min-h-0 rounded-[2rem] border border-line bg-n1/88 p-5 shadow-[0_24px_60px_rgba(0,0,0,0.22)] sm:p-6">
              <div class="flex shrink-0 flex-wrap items-center justify-between gap-3">
                <div>
                  <h3 class="session-section-title">Party</h3>
                  <p class="mt-1 text-[14px] text-dim">{{ campaignCharacterRows.length }} characters · {{ game.members?.length ?? 0 }} members</p>
                </div>
                <button
                  v-if="viewer.can_create_character"
                  type="button"
                  class="session-primary-btn"
                  :disabled="creatingCharacter"
                  @click="requestCharacterCreation"
                >
                  <PlusIcon class="h-4 w-4" :stroke-width="2.2" />
                  {{ creatingCharacter ? 'Creating…' : 'New character' }}
                </button>
              </div>

              <DataTable
                class="mt-5 flex-1 min-h-0"
                fill-height
                :columns="[
                  { key: 'name', label: 'Character', sortable: true },
                  { key: 'ownerName', label: 'Owner', sortable: true, filterable: true },
                  { key: 'roleLabel', label: 'Role', sortable: true, filterable: true },
                  { key: 'createdAt', label: 'Created', sortable: true },
                  { key: 'sizeLabel', label: 'Grid' },
                  { key: 'actions', label: '', align: 'right' },
                ]"
                :rows="campaignCharacterRows"
                row-key="id"
                :search-keys="['name', 'ownerName', 'roleLabel', 'backstory']"
                search-placeholder="Search character, player, or role"
                :page-size="20"
                min-width="880px"
                empty-text="No characters in this campaign yet"
              >
                <template #cell-name="{ row }">
                  <div class="flex items-center gap-3">
                    <div class="flex h-10 w-10 shrink-0 items-center justify-center overflow-hidden rounded-xl border border-line bg-n3">
                      <img v-if="avatarUrl(row.portrait_id)" :src="avatarUrl(row.portrait_id)" :alt="row.name" class="h-full w-full object-cover" @error="markImageFailed(avatarUrl(row.portrait_id))" />
                      <span v-else class="font-heading text-[13px] font-bold text-sky">{{ initials(row.name) }}</span>
                    </div>
                    <div class="min-w-0 flex items-center gap-2">
                      <span class="truncate font-semibold text-ink">{{ row.name }}</span>
                      <span v-if="row.id === activeCharacterId" class="shrink-0 rounded-full bg-primary/20 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-[0.14em] text-primary-soft">Active</span>
                    </div>
                  </div>
                </template>
                <template #cell-createdAt="{ row }"><span class="text-dim">{{ row.createdAt ? formatDateTime(row.createdAt) : '—' }}</span></template>
                <template #cell-sizeLabel="{ row }"><span class="text-dim">{{ row.sizeLabel }}</span></template>
                <template #cell-actions="{ row }">
                  <div class="flex flex-wrap justify-end gap-2">
                    <button
                      v-if="characters.some(character => character.id === row.id)"
                      @click="switchCharacter(row.id, { nextTab: 'sheet' })"
                      :disabled="row.id === activeCharacterId"
                      class="cursor-pointer rounded-lg border border-primary/24 bg-primary/12 px-3 py-1.5 text-[12px] font-semibold text-primary-soft transition-all duration-200 hover:border-primary/45 disabled:cursor-default disabled:opacity-60"
                    >
                      {{ row.id === activeCharacterId ? 'Active' : 'Play as' }}
                    </button>
                    <button @click="openInspectChoice(row)" class="inline-flex cursor-pointer items-center gap-1.5 rounded-lg border border-line bg-n3 px-3 py-1.5 text-[12px] font-semibold text-sky transition-all duration-200 hover:border-n7">
                      <EyeIcon class="h-3.5 w-3.5" :stroke-width="2" /> View
                    </button>
                    <button @click="openGMCharacterEditor(row)" class="cursor-pointer rounded-lg border border-warning/24 bg-warning/16 px-3 py-1.5 text-[12px] font-semibold text-ink transition-all duration-200 hover:border-warning/40">
                      Configure
                    </button>
                    <button
                      @click="duplicateCharacter(row)"
                      :disabled="duplicatingCharacterId === row.id"
                      class="inline-flex cursor-pointer items-center gap-1.5 rounded-lg border border-line bg-n3 px-3 py-1.5 text-[12px] font-semibold text-ink transition-all duration-200 hover:border-line-strong disabled:cursor-not-allowed disabled:opacity-50"
                    >
                      <Copy class="h-3.5 w-3.5" :stroke-width="2" /> {{ duplicatingCharacterId === row.id ? 'Duplicating…' : 'Duplicate' }}
                    </button>
                    <button @click="requestCharacterDeletion(row)" class="inline-flex cursor-pointer items-center gap-1.5 rounded-lg border border-danger/40 bg-danger/18 px-3 py-1.5 text-[12px] font-semibold text-danger-soft transition-all duration-200 hover:border-danger/65 hover:bg-danger/28">
                      <Trash2Icon class="h-3.5 w-3.5" :stroke-width="2" /> Delete
                    </button>
                  </div>
                </template>
              </DataTable>
            </article>
          </section>

          <SessionItemCompendium v-else-if="activeTab === 'items' && isGM" :characters="characters" :available-tags="itemTags" :game-id="gameId" :disabled-standard-attrs="disabledStandardAttrs" :enable-health="enableHealth" :enable-armor-class="enableArmorClass" @created="loadSession({ preserveCharacter: true, promptSelection: false })" />

          <section v-else-if="activeTab === 'manage' && isGM" ref="fillHeightSectionRef" class="flex min-h-[42rem] flex-col gap-6 xl:min-h-0" :style="fillSectionHeight ? { height: fillSectionHeight } : {}">
            <article class="shrink-0 rounded-[2rem] border border-line bg-n1/88 p-5 shadow-[0_24px_60px_rgba(0,0,0,0.22)] sm:p-6">
              <h3 class="session-section-title">Table</h3>
              <p class="mt-1 text-[14px] text-dim">Who plays, what happened, and how this game runs.</p>

              <div class="mt-5 flex flex-wrap items-center justify-between gap-3">
                <div class="inline-flex flex-wrap gap-1.5 rounded-2xl border border-line bg-n1/60 p-1.5">
                  <div
                    v-for="panel in [{ id: 'players', label: 'Members' }, { id: 'chat', label: 'Chat log' }, { id: 'activity', label: 'Activity' }, { id: 'settings', label: 'Settings' }]"
                    :key="panel.id"
                    role="button"
                    tabindex="0"
                    @click="managePanel = panel.id"
                    @keydown.enter="managePanel = panel.id"
                    class="cursor-pointer select-none rounded-xl px-4 py-2 text-[13px] font-semibold transition-all duration-200"
                    :class="managePanel === panel.id
                      ? 'bg-[linear-gradient(135deg,rgba(88,101,221,0.95),rgba(74,86,200,0.95))] text-white shadow-[0_8px_20px_rgba(88,101,221,0.3)]'
                      : 'text-ink-soft/70 hover:bg-n3 hover:text-ink'"
                  >
                    {{ panel.label }}
                  </div>
                </div>
                <div v-if="managePanel !== 'settings'" class="flex items-center gap-2">
                  <div
                    role="button"
                    tabindex="0"
                    @click="refreshCurrentManagePanel"
                    @keydown.enter="refreshCurrentManagePanel"
                    title="Refresh"
                    aria-label="Refresh"
                    class="inline-flex cursor-pointer select-none items-center justify-center rounded-xl border border-line bg-n3 p-2.5 text-sky transition-all duration-200 hover:border-n7"
                  >
                    <RefreshCwIcon class="h-4 w-4" :class="{ 'animate-spin': manageRefreshing }" :stroke-width="2" />
                  </div>
                </div>
              </div>
            </article>

            <GameSettingsModal
              v-if="managePanel === 'settings'"
              inline
              :game-id="gameId"
              @updated="handleGameSettingsUpdated"
              @deleted="handleGameDeleted"
            />

            <!-- Players table -->
            <div v-show="managePanel === 'players'" class="flex-1 min-h-0 flex flex-col">
              <DataTable
                fill-height
                class="flex-1 min-h-0"
                :columns="playerColumns"
                :rows="managePlayers"
                row-key="user_id"
                :search-keys="['username', 'role']"
                search-placeholder="Search players"
                :page-size="20"
                min-width="640px"
                empty-text="No members yet"
              >
                <template #cell-username="{ row }"><span class="font-semibold text-ink">{{ row.username }}</span></template>
                <template #cell-role="{ row }"><span class="text-ink-soft/70">{{ formatRole(row.role) }}</span></template>
                <template #cell-joined_at="{ row }"><span class="text-faint">{{ formatDateTime(row.joined_at) }}</span></template>
                <template #cell-characterCount="{ row }"><span class="text-ink">{{ row.characterCount }}</span></template>
                <template #cell-actions="{ row }">
                  <div class="flex flex-wrap justify-end gap-2">
                    <button type="button" @click="openMemberView(row)" class="inline-flex cursor-pointer items-center gap-1.5 rounded-lg border border-line bg-n3 px-3 py-1.5 text-[12px] font-semibold text-sky transition-all duration-200 hover:border-n7">
                      <EyeIcon class="h-3.5 w-3.5" :stroke-width="2" /> View
                    </button>
                    <button v-if="isGameOwner && row.role === 'player'" type="button" @click="updateMemberRole(row, 'assistant_gm')" class="cursor-pointer rounded-lg border border-warning/28 bg-warning/16 px-3 py-1.5 text-[12px] font-semibold text-ink transition-all duration-200 hover:border-warning/45">
                      Make GM
                    </button>
                    <button v-else-if="isGameOwner && row.role === 'assistant_gm'" type="button" @click="updateMemberRole(row, 'player')" class="cursor-pointer rounded-lg border border-line bg-n3 px-3 py-1.5 text-[12px] font-semibold text-ink-soft/80 transition-all duration-200 hover:border-n7">
                      Revoke GM
                    </button>
                    <button v-if="canRemoveMember(row)" type="button" @click="requestRemoveMember(row)" class="inline-flex cursor-pointer items-center gap-1.5 rounded-lg border border-danger/40 bg-danger/16 px-3 py-1.5 text-[12px] font-semibold text-danger-soft transition-all duration-200 hover:border-danger/65 hover:bg-danger/26">
                      <Trash2Icon class="h-3.5 w-3.5" :stroke-width="2" /> Remove
                    </button>
                  </div>
                </template>
              </DataTable>
            </div>

            <!-- Chat history table -->
            <div v-show="managePanel === 'chat'" class="flex-1 min-h-0 flex flex-col">
              <DataTable
                fill-height
                class="flex-1 min-h-0"
                :columns="chatColumns"
                :rows="chatHistoryRows"
                :search-keys="['author', 'text']"
                search-placeholder="Search chat history"
                :page-size="20"
                min-width="560px"
                min-height="18rem"
                :empty-text="manageChatLoading ? 'Loading...' : 'No chat messages yet'"
              >
                <template #toolbar>
                  <div class="flex items-center gap-2">
                    <select v-model.number="chatClearHours" class="rounded-lg border border-line bg-n1/72 px-2.5 py-1.5 text-[12px] text-ink outline-none">
                      <option v-for="period in CLEAR_PERIODS" :key="period.hours" :value="period.hours">{{ period.label }}</option>
                    </select>
                    <div role="button" tabindex="0" @click="clearChatHistory" @keydown.enter="clearChatHistory" class="cursor-pointer select-none rounded-lg border border-danger/28 bg-danger/12 px-3 py-1.5 text-[12px] font-semibold text-danger-soft transition-all duration-200 hover:border-danger/50">Clear</div>
                  </div>
                </template>
                <template #cell-time="{ row }"><span class="whitespace-nowrap text-faint">{{ row.time }}</span></template>
                <template #cell-author="{ row }"><span class="whitespace-nowrap font-semibold text-ink">{{ row.author }}</span></template>
                <template #cell-text="{ row }">
                  <span v-if="row.isItem" class="inline-flex items-center gap-1.5 rounded-lg border border-line bg-n3 px-2 py-1 text-[12px] text-sky">
                    <PackageIcon class="h-3.5 w-3.5" :stroke-width="2" /> Shared item: {{ row.text }}
                  </span>
                  <span v-else class="inline-block max-w-[34rem] truncate align-middle text-ink/78" :title="row.text">{{ row.text }}</span>
                </template>
              </DataTable>
            </div>

            <!-- Activity log table -->
            <div v-show="managePanel === 'activity'" class="flex-1 min-h-0 flex flex-col">
              <DataTable
                fill-height
                class="flex-1 min-h-0"
                :columns="activityColumns"
                :rows="activityRows"
                :search-keys="['player', 'character', 'action', 'details']"
                search-placeholder="Search activity"
                :page-size="20"
                min-width="720px"
                min-height="18rem"
                :empty-text="manageActivityLoading ? 'Loading...' : 'No recorded activity yet'"
              >
                <template #toolbar>
                  <div class="flex items-center gap-2">
                    <select v-model.number="activityClearHours" class="rounded-lg border border-line bg-n1/72 px-2.5 py-1.5 text-[12px] text-ink outline-none">
                      <option v-for="period in CLEAR_PERIODS" :key="period.hours" :value="period.hours">{{ period.label }}</option>
                    </select>
                    <div role="button" tabindex="0" @click="clearActivity" @keydown.enter="clearActivity" class="cursor-pointer select-none rounded-lg border border-danger/28 bg-danger/12 px-3 py-1.5 text-[12px] font-semibold text-danger-soft transition-all duration-200 hover:border-danger/50">Clear</div>
                  </div>
                </template>
                <template #cell-time="{ row }"><span class="whitespace-nowrap text-faint">{{ row.time }}</span></template>
                <template #cell-player="{ row }"><span class="whitespace-nowrap font-semibold text-ink">{{ row.player }}</span></template>
                <template #cell-character="{ row }"><span class="text-ink-soft/70">{{ row.character }}</span></template>
                <template #cell-action="{ row }">
                  <span class="rounded-full border border-line bg-n3 px-2.5 py-1 text-[11px] font-semibold text-sky">{{ row.action }}</span>
                </template>
                <template #cell-details="{ row }">
                  <button
                    v-if="activityItemName(row)"
                    type="button"
                    @click="openActivityItem(activityItemName(row))"
                    class="inline-flex max-w-[28rem] cursor-pointer items-center gap-1.5 truncate rounded-lg border border-line bg-n3 px-2.5 py-1 align-middle text-[12px] font-medium text-sky transition-all duration-200 hover:border-n7"
                    :title="row.details"
                  >
                    <PackageIcon class="h-3.5 w-3.5 shrink-0" :stroke-width="2" /> {{ row.details }}
                  </button>
                  <span v-else class="inline-block max-w-[28rem] truncate align-middle text-ink/78" :title="row.details">{{ row.details }}</span>
                </template>
              </DataTable>
            </div>
          </section>
        </main>
      </div>

      <Teleport to="body">
        <div v-if="memberPendingRemoval" class="fixed inset-0 z-[12560] flex items-center justify-center p-4">
          <div class="absolute inset-0 bg-n0/82 backdrop-blur-md" @click="cancelRemoveMember"></div>
          <div class="relative w-full max-w-[28rem] overflow-hidden rounded-[1.6rem] border border-danger/24 bg-[linear-gradient(180deg,rgba(10,16,41,0.98),rgba(6,10,28,0.98))] p-6 shadow-[0_40px_120px_rgba(0,0,0,0.55)]">
            <p class="text-[13px] font-semibold text-danger/70">Remove Player</p>
            <h3 class="mt-2 break-words font-heading text-[22px] font-bold leading-tight text-ink [overflow-wrap:anywhere]">{{ memberPendingRemoval.username }}</h3>
            <p class="mt-3 text-[14px] leading-relaxed text-ink-soft/68">
              This removes the player from the game and deletes their characters here (with inventory and equipment). This action cannot be undone.
            </p>
            <div class="mt-6 flex flex-wrap justify-end gap-3">
              <button type="button" @click="cancelRemoveMember" :disabled="removingMember" class="cursor-pointer rounded-xl border border-line bg-n3 px-4 py-2.5 text-[13px] font-semibold text-ink transition-all duration-200 hover:border-line-strong disabled:cursor-not-allowed disabled:opacity-60">
                Cancel
              </button>
              <button type="button" @click="confirmRemoveMember" :disabled="removingMember" class="inline-flex cursor-pointer items-center gap-2 rounded-xl border border-danger/28 bg-[linear-gradient(135deg,rgba(242,122,133,0.9),rgba(220,38,38,0.9))] px-4 py-2.5 text-[13px] font-semibold text-white transition-all duration-200 disabled:cursor-not-allowed disabled:opacity-60">
                {{ removingMember ? 'Removing...' : 'Remove Player' }}
              </button>
            </div>
          </div>
        </div>
      </Teleport>


      <Teleport to="body">
        <div v-if="characterPendingDeletion" class="fixed inset-0 z-[12550] flex items-center justify-center p-4">
          <div class="absolute inset-0 bg-n0/82 backdrop-blur-md" @click="cancelCharacterDeletion"></div>
          <div class="relative w-full max-w-[28rem] overflow-hidden rounded-[1.6rem] border border-danger/24 bg-[linear-gradient(180deg,rgba(10,16,41,0.98),rgba(6,10,28,0.98))] p-6 shadow-[0_40px_120px_rgba(0,0,0,0.55)]">
            <p class="text-[13px] font-semibold text-danger/70">Delete Character</p>
            <h3 class="mt-2 break-words font-heading text-[22px] font-bold leading-tight text-ink [overflow-wrap:anywhere]">{{ characterPendingDeletion.name || 'Character' }}</h3>
            <p class="mt-3 text-[14px] leading-relaxed text-ink-soft/68">
              This permanently removes the character along with its inventory, equipment, and attributes. This action cannot be undone.
            </p>
            <div class="mt-6 flex flex-wrap justify-end gap-3">
              <button type="button" @click="cancelCharacterDeletion" :disabled="deletingCharacter" class="cursor-pointer rounded-xl border border-line bg-n3 px-4 py-2.5 text-[13px] font-semibold text-ink transition-all duration-200 hover:border-line-strong disabled:cursor-not-allowed disabled:opacity-60">
                Cancel
              </button>
              <button type="button" @click="confirmCharacterDeletion" :disabled="deletingCharacter" class="inline-flex cursor-pointer items-center gap-2 rounded-xl border border-danger/28 bg-[linear-gradient(135deg,rgba(242,122,133,0.9),rgba(220,38,38,0.9))] px-4 py-2.5 text-[13px] font-semibold text-white transition-all duration-200 disabled:cursor-not-allowed disabled:opacity-60">
                {{ deletingCharacter ? 'Deleting...' : 'Delete Character' }}
              </button>
            </div>
          </div>
        </div>
      </Teleport>

      <Teleport to="body">
        <div v-if="sharedItem" class="fixed inset-0 z-[12550] flex items-center justify-center p-4">
          <div class="absolute inset-0 bg-n0/82 backdrop-blur-md" @click="closeSharedItem"></div>
          <div class="relative flex max-h-full w-full max-w-[34rem] flex-col overflow-hidden rounded-[1.6rem] border border-line bg-[linear-gradient(180deg,rgba(10,16,41,0.98),rgba(6,10,28,0.98))] shadow-[0_40px_120px_rgba(0,0,0,0.55)]">
            <button type="button" @click="closeSharedItem" aria-label="Close item details" class="absolute right-4 top-4 z-10 flex h-9 w-9 cursor-pointer items-center justify-center rounded-xl border border-line bg-n3 text-ink transition-all duration-200 hover:border-primary/32">
              <XIcon class="h-5 w-5" :stroke-width="2" />
            </button>
            <div class="min-h-0 flex-1 overflow-y-auto p-5 sm:p-6">
              <div class="flex gap-4">
                <div class="h-24 w-24 shrink-0 overflow-hidden rounded-[1.2rem] border-2 border-line bg-n1/66">
                  <img v-if="avatarUrl(sharedItem.image_id)" :src="avatarUrl(sharedItem.image_id)" :alt="sharedItem.name" class="h-full w-full object-cover" @error="markImageFailed(avatarUrl(sharedItem.image_id))" />
                  <div v-else class="flex h-full w-full items-center justify-center font-heading text-[28px] font-bold text-faint">{{ (sharedItem.name || '?').charAt(0).toUpperCase() }}</div>
                </div>
                <div class="min-w-0 flex-1 pr-8">
                  <h3 class="break-words text-[20px] font-bold leading-tight text-ink [overflow-wrap:anywhere]">{{ sharedItem.name || 'Unnamed item' }}</h3>
                  <div class="mt-1.5 flex flex-wrap items-center gap-2">
                    <span class="rounded-full px-2.5 py-1 text-[10px] font-semibold uppercase tracking-[0.14em]" :class="rarityClasses(sharedItem.rarity)">{{ sharedItem.rarity || 'common' }}</span>
                    <span class="text-[13px] font-semibold text-faint">{{ formatAttrLabel(sharedItem.category || 'other') }}</span>
                    <span class="text-[13px] font-semibold text-faint">{{ sharedItem.grid_width || 1 }}x{{ sharedItem.grid_height || 1 }}</span>
                  </div>
                  <p v-if="sharedItem.owner" class="mt-1 text-[12px] text-ink-soft/60">Shared by {{ sharedItem.owner }}</p>
                </div>
              </div>

              <div class="mt-4">
                <p class="text-[13px] font-semibold text-dim">Description</p>
                <p class="mt-1 break-words whitespace-pre-line text-[13px] leading-relaxed text-ink-soft/72 [overflow-wrap:anywhere]">{{ sharedItem.description?.trim() || 'No description provided.' }}</p>
              </div>

              <div v-if="attributesEnabled && sharedItem.required_attributes?.length" class="mt-4">
                <p class="text-[13px] font-semibold text-dim">Requirements</p>
                <ul class="mt-2 grid gap-1.5 sm:grid-cols-2">
                  <li v-for="(requirement, index) in sharedItem.required_attributes" :key="`sreq-${index}`" class="flex items-center justify-between gap-2 rounded-lg border border-line bg-n3 px-3 py-1.5 text-[12px] text-ink-soft/82">
                    <span>{{ formatAttrLabel(requirement.attribute_name) }}</span>
                    <span class="font-semibold text-ink">{{ requirement.min_value }}</span>
                  </li>
                </ul>
              </div>

              <div v-if="attributesEnabled && sharedItem.attribute_modifiers?.length" class="mt-4">
                <p class="text-[13px] font-semibold text-dim">Grants</p>
                <ul class="mt-2 grid gap-1.5 sm:grid-cols-2">
                  <li v-for="(modifier, index) in sharedItem.attribute_modifiers" :key="`smod-${index}`" class="flex items-center justify-between gap-2 rounded-lg border border-line bg-n3 px-3 py-1.5 text-[12px] text-ink-soft/82">
                    <span>{{ formatAttrLabel(modifier.attribute_name) }}</span>
                    <span class="font-semibold" :class="modifier.modifier_value >= 0 ? 'text-sky' : 'text-danger'">{{ modifier.modifier_value >= 0 ? '+' : '' }}{{ modifier.modifier_value }}{{ modifier.is_percentage ? '%' : '' }}</span>
                  </li>
                </ul>
              </div>

              <div class="mt-4 flex flex-wrap gap-3 text-[12px] text-ink-soft/70">
                <span>Quantity: <span class="font-semibold text-ink">{{ sharedItem.quantity || 1 }}</span></span>
                <span v-if="sharedItem.enchantment">Enchantment: <span class="font-semibold text-sky">+{{ sharedItem.enchantment }}</span></span>
                <span v-if="sharedItem.max_durability != null">Durability: <span class="font-semibold text-ink">{{ sharedItem.durability ?? sharedItem.max_durability }} / {{ sharedItem.max_durability }}</span></span>
              </div>
            </div>
          </div>
        </div>
      </Teleport>

      <Teleport to="body">
        <div v-if="viewMember" class="fixed inset-0 z-[12540] flex items-center justify-center p-4">
          <div class="absolute inset-0 bg-n0/85 backdrop-blur-md" @click="closeMemberView"></div>
          <div class="relative flex max-h-[92vh] w-full max-w-[52rem] flex-col overflow-hidden rounded-[1.6rem] border border-line bg-[linear-gradient(180deg,rgba(10,16,41,0.98),rgba(6,10,28,0.98))] shadow-[0_40px_120px_rgba(0,0,0,0.55)]">
            <button type="button" @click="closeMemberView" aria-label="Close player overview" class="absolute right-4 top-4 z-10 flex h-9 w-9 cursor-pointer items-center justify-center rounded-xl border border-line bg-n3 text-ink transition-all duration-200 hover:border-primary/32">
              <XIcon class="h-5 w-5" :stroke-width="2" />
            </button>

            <div class="flex items-center gap-4 border-b border-line-soft px-6 py-5 pr-16">
              <div class="h-16 w-16 shrink-0 overflow-hidden rounded-2xl border-2 border-line bg-n1/66">
                <img v-if="avatarUrl(viewMember.avatar_id)" :src="avatarUrl(viewMember.avatar_id)" :alt="viewMember.username" class="h-full w-full object-cover" @error="markImageFailed(avatarUrl(viewMember.avatar_id))" />
                <div v-else class="flex h-full w-full items-center justify-center font-heading text-[22px] font-bold text-faint">{{ (viewMember.username || '?').charAt(0).toUpperCase() }}</div>
              </div>
              <div class="min-w-0 flex-1">
                <p class="text-[13px] font-semibold text-dim">Player Overview</p>
                <h2 class="mt-1 break-words font-heading text-[24px] font-bold leading-tight text-ink [overflow-wrap:anywhere]">{{ viewMember.username }}</h2>
                <div class="mt-2 flex flex-wrap items-center gap-2 text-[12px]">
                  <span class="rounded-full border border-line bg-n3 px-3 py-1 font-semibold uppercase tracking-[0.14em] text-sky">{{ formatRole(viewMember.role) }}</span>
                  <span class="text-faint">Joined {{ formatDateTime(viewMember.joined_at) }}</span>
                  <span class="text-faint">· {{ viewMemberCharacters.length }} character(s)</span>
                </div>
              </div>
            </div>

            <div class="min-h-0 flex-1 overflow-y-auto p-6">
              <p class="text-[13px] font-semibold text-dim">Characters ({{ viewMemberCharacters.length }})</p>
              <div v-if="viewMemberCharacters.length" class="mt-3 grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-4">
                <div v-for="character in viewMemberCharacters" :key="character.id" class="flex flex-col items-center rounded-[1.1rem] border border-line bg-n1/70 p-3 text-center">
                  <div class="h-16 w-16 overflow-hidden rounded-[0.9rem] border-2 border-line bg-n1/66">
                    <img v-if="avatarUrl(character.portrait_id)" :src="avatarUrl(character.portrait_id)" :alt="character.name" class="h-full w-full object-cover" @error="markImageFailed(avatarUrl(character.portrait_id))" />
                    <div v-else class="flex h-full w-full items-center justify-center font-heading text-[20px] font-bold text-faint">{{ (character.name || '?').charAt(0).toUpperCase() }}</div>
                  </div>
                  <p class="mt-2 line-clamp-2 break-words text-[13px] font-semibold text-ink [overflow-wrap:anywhere]">{{ character.name }}</p>
                </div>
              </div>
              <p v-else class="mt-2 text-[13px] text-faint">This player has no characters in this game.</p>

              <p class="mt-6 text-[13px] font-semibold text-dim">Recent messages</p>
              <div v-if="viewMemberLoading" class="mt-3 flex justify-center py-6">
                <div class="h-6 w-6 animate-spin rounded-full border-2 border-primary border-t-transparent"></div>
              </div>
              <ul v-else-if="viewMemberMessages.length" class="mt-3 space-y-2">
                <li v-for="message in viewMemberMessages" :key="message.id" class="rounded-xl border border-line-soft bg-n2 px-3 py-2">
                  <div class="flex items-start justify-between gap-3">
                    <div class="min-w-0 flex-1">
                      <span v-if="message.isItem" class="inline-flex items-center gap-1.5 text-[12px] text-sky"><PackageIcon class="h-3.5 w-3.5 shrink-0" :stroke-width="2" /> {{ message.text }}</span>
                      <span v-else class="break-words text-[13px] text-ink/82 [overflow-wrap:anywhere]">{{ message.text }}</span>
                    </div>
                    <span class="shrink-0 text-[11px] text-faint">{{ message.time }}</span>
                  </div>
                </li>
              </ul>
              <p v-else class="mt-2 text-[13px] text-faint">{{ game?.enable_chat ? 'No messages from this player yet.' : 'Chat is disabled for this game.' }}</p>
            </div>
          </div>
        </div>
      </Teleport>

      <Teleport to="body">
        <div v-if="showTradeModal" class="fixed inset-0 z-[12555] flex items-center justify-center p-4">
          <div class="absolute inset-0 bg-n0/85 backdrop-blur-md" @click="closeTradeModal"></div>
          <div class="relative flex max-h-[92vh] w-full max-w-[56rem] flex-col overflow-hidden rounded-[1.6rem] border border-line bg-[linear-gradient(180deg,rgba(10,16,41,0.98),rgba(6,10,28,0.98))] shadow-[0_40px_120px_rgba(0,0,0,0.55)]">
            <div class="flex items-center justify-between gap-3 border-b border-line-soft px-6 py-4">
              <div class="flex items-center gap-2.5">
                <ArrowLeftRightIcon class="h-5 w-5 text-sky" :stroke-width="2" />
                <h2 class="font-heading text-[20px] font-bold text-ink">Player Trading</h2>
              </div>
              <button type="button" @click="closeTradeModal" :disabled="tradeBusy" aria-label="Close" class="flex h-9 w-9 cursor-pointer items-center justify-center rounded-xl border border-line bg-n3 text-ink transition-all duration-200 hover:border-primary/32 disabled:opacity-50">
                <XIcon class="h-5 w-5" :stroke-width="2" />
              </button>
            </div>

            <div class="flex gap-1.5 border-b border-line-soft px-6 py-3">
              <button
                v-for="panel in [{ id: 'send', label: 'New offer' }, { id: 'incoming', label: 'Incoming' }, { id: 'outgoing', label: 'Sent' }]"
                :key="panel.id"
                type="button"
                @click="tradePanel = panel.id"
                class="relative cursor-pointer rounded-xl px-4 py-2 text-[13px] font-semibold transition-all duration-200"
                :class="tradePanel === panel.id ? 'bg-[linear-gradient(135deg,rgba(88,101,221,0.95),rgba(74,86,200,0.95))] text-white' : 'text-ink-soft/70 hover:bg-n3 hover:text-ink'"
              >
                {{ panel.label }}
                <span v-if="panel.id === 'incoming' && incomingTradeCount" class="ml-1.5 inline-flex h-4 min-w-[1rem] items-center justify-center rounded-full bg-primary px-1 text-[10px] font-bold text-white">{{ incomingTradeCount }}</span>
                <span v-if="panel.id === 'outgoing' && tradeOutgoing.length" class="ml-1.5 inline-flex h-4 min-w-[1rem] items-center justify-center rounded-full bg-n5 px-1 text-[10px] font-bold text-sky">{{ tradeOutgoing.length }}</span>
              </button>
            </div>

            <div class="min-h-[26rem] flex-1 overflow-y-auto p-6">
              <!-- New offer -->
              <div v-if="tradePanel === 'send'">
                <p v-if="!tradingEnabled" class="mb-4 rounded-xl border border-danger/24 bg-danger/8 px-4 py-3 text-[13px] text-danger">Item trading is disabled for this game. You can still respond to existing offers.</p>
                <label class="block text-[13px] font-semibold text-dim">Send to</label>
                <div class="relative mt-1.5">
                  <input
                    v-model="tradeRecipientSearch"
                    @focus="tradeRecipientOpen = true"
                    @input="onRecipientInput"
                    @blur="closeRecipientDropdown"
                    type="text"
                    placeholder="Search character by name…"
                    class="w-full rounded-lg border border-line bg-n1/72 px-3 py-2.5 text-[14px] text-ink outline-none focus:border-n7 placeholder:text-faint"
                  />
                  <div v-if="tradeRecipientOpen" class="absolute z-20 mt-1 max-h-56 w-full overflow-y-auto rounded-lg border border-line bg-[linear-gradient(180deg,rgba(10,16,41,0.99),rgba(6,11,22,0.99))] py-1 shadow-[0_24px_60px_rgba(0,0,0,0.55)]">
                    <button
                      v-for="target in tradeRecipientFiltered"
                      :key="target.id"
                      type="button"
                      @mousedown.prevent="selectTradeRecipient(target)"
                      class="flex w-full cursor-pointer items-center justify-between gap-2 px-3 py-2 text-left text-[13px] transition-colors hover:bg-n3"
                      :class="tradeRecipientId === target.id ? 'text-primary-ink' : 'text-ink/82'"
                    >
                      <span class="font-semibold">{{ target.name }}</span>
                      <span class="text-[12px] text-faint">{{ target.owner }}</span>
                    </button>
                    <p v-if="!tradeRecipientFiltered.length" class="px-3 py-2 text-[12px] text-faint">No matching characters</p>
                  </div>
                </div>

                <p class="mt-5 text-[13px] font-semibold text-dim">Items to offer ({{ tradeSelectedIds.length }} selected)</p>
                <p v-if="!tradeOfferableItems.length" class="mt-3 text-[13px] text-faint">No tradeable items. Unequip items first to offer them.</p>
                <div v-else class="mt-3 grid grid-cols-2 gap-2.5 sm:grid-cols-3">
                  <button
                    v-for="entry in tradeOfferableItems"
                    :key="entry.id"
                    type="button"
                    @click="toggleTradeItem(entry.id)"
                    class="flex items-center gap-2.5 rounded-xl border p-2.5 text-left transition-all duration-200"
                    :class="tradeSelectedSet.has(entry.id) ? 'border-primary/50 bg-primary/12' : 'border-line bg-n2 hover:border-line-strong'"
                  >
                    <div class="h-10 w-10 shrink-0 overflow-hidden rounded-lg border border-line bg-n1/66">
                      <img v-if="avatarUrl(entry.item?.image_id)" :src="avatarUrl(entry.item?.image_id)" :alt="entry.item?.name" class="h-full w-full object-cover" @error="markImageFailed(avatarUrl(entry.item?.image_id))" />
                    </div>
                    <div class="min-w-0 flex-1">
                      <p class="truncate text-[13px] font-semibold text-ink">{{ entry.item?.name }}</p>
                      <p class="text-[11px] text-faint">{{ entry.item?.grid_width }}x{{ entry.item?.grid_height }}<span v-if="(entry.quantity || 1) > 1"> · x{{ entry.quantity }}</span></p>
                      <p v-if="entry.max_durability != null" class="text-[11px]" :class="(entry.durability ?? entry.max_durability) <= entry.max_durability * 0.3 ? 'text-danger' : 'text-success/80'">
                        Dur {{ entry.durability ?? entry.max_durability }}/{{ entry.max_durability }}
                      </p>
                    </div>
                    <CheckIcon v-if="tradeSelectedSet.has(entry.id)" class="h-4 w-4 shrink-0 text-danger" :stroke-width="3" />
                  </button>
                </div>

                <div class="mt-6 flex justify-end">
                  <button
                    type="button"
                    @click="submitTrade"
                    :disabled="tradeBusy || !tradingEnabled || !tradeRecipientId || !tradeSelectedIds.length"
                    class="inline-flex cursor-pointer items-center gap-2 rounded-xl border border-primary/30 bg-[linear-gradient(135deg,rgba(88,101,221,0.92),rgba(74,86,200,0.92))] px-5 py-2.5 text-[13px] font-semibold text-white transition-all duration-200 disabled:cursor-not-allowed disabled:opacity-50 disabled:hover:translate-y-0"
                  >
                    <ArrowLeftRightIcon class="h-4 w-4" :stroke-width="2" />
                    {{ tradeBusy ? 'Sending…' : 'Send offer' }}
                  </button>
                </div>
              </div>

              <!-- Incoming -->
              <div v-else-if="tradePanel === 'incoming'">
                <p v-if="!tradeIncoming.length" class="py-10 text-center text-[13px] text-faint">No incoming offers.</p>
                <div v-else class="space-y-3">
                  <article v-for="offer in tradeIncoming" :key="offer.id" class="rounded-[1.3rem] border border-line bg-n1/70 p-4">
                    <p class="text-[13px] text-ink-soft/72"><span class="font-semibold text-ink">{{ offer.from_username }}</span> ({{ offer.from_character_name }}) → <span class="font-semibold text-ink">{{ offer.to_character_name }}</span></p>
                    <div class="mt-3 flex flex-wrap gap-2">
                      <span v-for="item in offer.items" :key="item.id" class="inline-flex items-center gap-1.5 rounded-lg border border-line bg-n3 px-2 py-1 text-[12px] text-ink-soft/82">
                        <img v-if="avatarUrl(item.item?.image_id)" :src="avatarUrl(item.item?.image_id)" :alt="item.item?.name" class="h-5 w-5 rounded object-cover" @error="markImageFailed(avatarUrl(item.item?.image_id))" />
                        {{ item.item?.name }}<span v-if="(item.quantity || 1) > 1" class="text-dim"> x{{ item.quantity }}</span>
                      </span>
                    </div>
                    <div class="mt-4 flex justify-end gap-2">
                      <button type="button" @click="declineTradeOffer(offer)" :disabled="tradeBusy" class="cursor-pointer rounded-lg border border-danger/24 bg-danger/10 px-3.5 py-2 text-[12px] font-semibold text-danger-soft transition-all duration-200 hover:border-danger/45 disabled:opacity-50">Decline</button>
                      <button type="button" @click="acceptTradeOffer(offer)" :disabled="tradeBusy" class="inline-flex cursor-pointer items-center gap-1.5 rounded-lg border border-success/30 bg-success/14 px-3.5 py-2 text-[12px] font-semibold text-success transition-all duration-200 hover:border-success/50 disabled:opacity-50"><CheckIcon class="h-3.5 w-3.5" :stroke-width="2.5" /> Accept</button>
                    </div>
                  </article>
                </div>
              </div>

              <!-- Sent -->
              <div v-else>
                <p v-if="!tradeOutgoing.length" class="py-10 text-center text-[13px] text-faint">No pending sent offers.</p>
                <div v-else class="space-y-3">
                  <article v-for="offer in tradeOutgoing" :key="offer.id" class="rounded-[1.3rem] border border-line bg-n1/70 p-4">
                    <p class="text-[13px] text-ink-soft/72">To <span class="font-semibold text-ink">{{ offer.to_username }}</span> ({{ offer.to_character_name }})</p>
                    <div class="mt-3 flex flex-wrap gap-2">
                      <span v-for="item in offer.items" :key="item.id" class="inline-flex items-center gap-1.5 rounded-lg border border-line bg-n3 px-2 py-1 text-[12px] text-ink-soft/82">
                        <img v-if="avatarUrl(item.item?.image_id)" :src="avatarUrl(item.item?.image_id)" :alt="item.item?.name" class="h-5 w-5 rounded object-cover" @error="markImageFailed(avatarUrl(item.item?.image_id))" />
                        {{ item.item?.name }}<span v-if="(item.quantity || 1) > 1" class="text-dim"> x{{ item.quantity }}</span>
                      </span>
                    </div>
                    <div class="mt-4 flex justify-end">
                      <button type="button" @click="declineTradeOffer(offer)" :disabled="tradeBusy" class="cursor-pointer rounded-lg border border-danger/24 bg-danger/10 px-3.5 py-2 text-[12px] font-semibold text-danger-soft transition-all duration-200 hover:border-danger/45 disabled:opacity-50">Cancel offer</button>
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
          <div class="absolute inset-0 bg-n0/82 backdrop-blur-md" @click="closeInspectChoice"></div>
          <div class="relative w-full max-w-[24rem] overflow-hidden rounded-[1.6rem] border border-line bg-[linear-gradient(180deg,rgba(10,16,41,0.98),rgba(6,10,28,0.98))] p-6 shadow-[0_40px_120px_rgba(0,0,0,0.55)]">
            <button type="button" @click="closeInspectChoice" aria-label="Close" class="absolute right-4 top-4 flex h-9 w-9 cursor-pointer items-center justify-center rounded-xl border border-line bg-n3 text-ink transition-all duration-200 hover:border-primary/32">
              <XIcon class="h-5 w-5" :stroke-width="2" />
            </button>
            <p class="text-[13px] font-semibold text-dim">Inspect</p>
            <h3 class="mt-1 break-words pr-10 font-heading text-[20px] font-bold leading-tight text-ink [overflow-wrap:anywhere]">{{ inspectChoice.name }}</h3>
            <p class="mt-2 text-[13px] text-ink-soft/62">What would you like to open?</p>
            <div class="mt-5 grid grid-cols-2 gap-3">
              <button type="button" @click="inspectCharacterSheet" class="flex cursor-pointer flex-col items-center gap-2 rounded-2xl border border-line bg-n3 px-4 py-5 text-[13px] font-semibold text-ink transition-all duration-200 hover:border-n7">
                <UserRoundIcon class="h-6 w-6 text-sky" :stroke-width="2" />
                Character
              </button>
              <button type="button" @click="inspectCharacterInventory" class="flex cursor-pointer flex-col items-center gap-2 rounded-2xl border border-line bg-n3 px-4 py-5 text-[13px] font-semibold text-ink transition-all duration-200 hover:border-n7">
                <LayoutGridIcon class="h-6 w-6 text-sky" :stroke-width="2" />
                Inventory
              </button>
            </div>
          </div>
        </div>
      </Teleport>

      <Teleport to="body">
        <div v-if="inventoryViewCharacter" class="fixed inset-0 z-[12560] flex items-center justify-center p-4">
          <div class="absolute inset-0 bg-n0/88 backdrop-blur-md" @click="closeCharacterInventory"></div>
          <div class="relative flex max-h-[94vh] w-full max-w-[80rem] flex-col overflow-hidden rounded-[1.6rem] border border-line bg-[linear-gradient(180deg,rgba(10,16,41,0.98),rgba(6,10,28,0.98))] shadow-[0_40px_120px_rgba(0,0,0,0.55)]">
            <div class="flex items-center justify-between gap-3 border-b border-line-soft px-6 py-4">
              <div class="min-w-0">
                <p class="text-[13px] font-semibold text-dim">Inventory (read-only)</p>
                <h2 class="mt-1 break-words font-heading text-[22px] font-bold leading-tight text-ink [overflow-wrap:anywhere]">{{ inventoryViewCharacter.name }}</h2>
              </div>
              <button type="button" @click="closeCharacterInventory" aria-label="Close inventory" class="flex h-9 w-9 shrink-0 cursor-pointer items-center justify-center rounded-xl border border-line bg-n3 text-ink transition-all duration-200 hover:border-primary/32">
                <XIcon class="h-5 w-5" :stroke-width="2" />
              </button>
            </div>

            <div class="min-h-0 flex-1 overflow-auto p-5 sm:p-6">
              <div v-if="inventoryViewLoading" class="py-20 text-center">
                <div class="mx-auto h-8 w-8 animate-spin rounded-full border-2 border-primary border-t-transparent"></div>
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
          class="pointer-events-none fixed z-[12000] w-[15rem] rounded-xl border border-line bg-[linear-gradient(180deg,rgba(10,16,41,0.98),rgba(6,10,28,0.98))] p-3 text-left shadow-[0_20px_50px_rgba(0,0,0,0.5)]"
          :style="{ left: `${statTooltip.left}px`, top: `${statTooltip.top}px` }"
        >
          <p class="text-[13px] font-semibold text-dim">{{ statTooltip.label }} breakdown</p>
          <div class="mt-2 flex items-center justify-between text-[12px] text-ink-soft/75">
            <span>Base</span><span class="font-semibold">{{ statTooltip.base }}</span>
          </div>
          <div
            v-for="(contribution, index) in statTooltip.contributions"
            :key="index"
            class="mt-1 flex items-center justify-between gap-2 text-[12px]"
            :class="contribution.value >= 0 ? 'text-success' : 'text-danger'"
          >
            <span class="truncate pr-2">{{ contribution.source }}</span>
            <span class="shrink-0 font-semibold">{{ contribution.value >= 0 ? '+' : '' }}{{ contribution.value }}{{ contribution.percent ? '%' : '' }}</span>
          </div>
          <p v-if="!statTooltip.contributions.length" class="mt-1 text-[11px] text-ink-soft/50">No equipment bonuses</p>
          <div class="mt-2 flex items-center justify-between border-t border-line pt-2 text-[12px] font-semibold text-ink">
            <span>Total</span><span>{{ statTooltip.total }}</span>
          </div>
        </div>
      </Teleport>


      <aside v-if="game?.enable_chat && !chatCollapsed" class="session-chat-dock">
        <div class="session-chat-dock__shell">
          <div class="flex items-center justify-between gap-4 border-b border-line-soft px-4 py-4">
            <div>
              <p class="text-[13px] font-semibold text-dim">Session Chat</p>
              <p class="mt-1 truncate text-[14px] text-ink">Latest 40 messages</p>
            </div>

            <div class="flex items-center gap-3">
              <span
                class="rounded-full border px-2.5 py-1 text-[10px] font-semibold uppercase tracking-[0.16em]"
                :class="game.enable_chat
                  ? 'border-[rgba(110,231,183,0.28)] bg-[rgba(22,163,74,0.14)] text-success'
                  : 'border-danger/28 bg-[rgba(185,28,28,0.14)] text-danger'"
              >
                {{ game.enable_chat ? 'Online' : 'Disabled' }}
              </span>
              <button
                @click="chatCollapsed = true"
                aria-label="Collapse chat dock"
                class="flex h-10 w-10 shrink-0 cursor-pointer items-center justify-center rounded-xl border border-line bg-n3 text-ink transition-all duration-200 hover:border-primary/34 hover:text-primary-soft"
              >
                <ChevronRightIcon class="h-5 w-5" :stroke-width="2" />
              </button>
            </div>
          </div>

          <div ref="chatMessagesRef" class="session-scroll flex-1 overflow-y-auto px-4 py-4">
            <div v-if="!game.enable_chat" class="rounded-[1.5rem] border border-danger/22 bg-danger/18 px-4 py-4 text-[14px] text-danger-soft">
              Chat is disabled for this game. The launcher stays visible, but sending is blocked by settings.
            </div>

            <div v-else-if="!chatMessages.length" class="rounded-[1.5rem] border border-line-soft bg-n3 px-4 py-4 text-[14px] text-ink-soft/62">
              No messages yet. Start the conversation here without leaving the session.
            </div>

            <div v-else class="space-y-3">
              <article
                v-for="message in chatMessages"
                :key="message.id"
                class="session-chat-message rounded-[1.4rem] border px-4 py-3"
                :class="message.user_id === viewer.user_id
                  ? 'border-primary/24 bg-primary/12'
                  : 'border-line bg-n3'"
              >
                <div class="flex items-start gap-3">
                  <div class="flex h-10 w-10 shrink-0 items-center justify-center overflow-hidden rounded-2xl border border-line bg-n3">
                    <img v-if="avatarUrl(message.user?.avatar_id)" :src="avatarUrl(message.user?.avatar_id)" :alt="message.user?.username" class="h-full w-full object-cover" @error="markImageFailed(avatarUrl(message.user?.avatar_id))" />
                    <span v-else class="font-heading text-[14px] font-bold text-sky">{{ initials(message.user?.username) }}</span>
                  </div>

                  <div class="min-w-0 flex-1">
                    <div class="flex flex-wrap items-center justify-between gap-3">
                      <p class="truncate text-[13px] font-semibold text-ink">{{ message.user?.username || 'Unknown user' }}</p>
                      <span class="text-[11px] text-ink-soft/45">{{ formatDateTime(message.created_at) }}</span>
                    </div>
                    <button
                      v-if="parseSharedItem(message.content)"
                      type="button"
                      @click="openSharedItem(parseSharedItem(message.content))"
                      class="mt-2 inline-flex max-w-full cursor-pointer items-center gap-2 rounded-xl border border-line bg-n3 px-3 py-2 text-left transition-all duration-200 hover:border-primary/40 hover:bg-primary/10"
                    >
                      <PackageIcon class="h-4 w-4 shrink-0 text-sky" :stroke-width="2" />
                      <span class="min-w-0">
                        <span class="block truncate text-[13px] font-semibold text-ink">{{ parseSharedItem(message.content).name || 'Item' }}</span>
                        <span class="block text-[13px] font-semibold text-sky/70">Shared item · view</span>
                      </span>
                    </button>
                    <p v-else class="mt-2 whitespace-pre-wrap break-words text-[14px] leading-relaxed text-ink/78">{{ message.content }}</p>
                  </div>
                </div>
              </article>
            </div>
          </div>

          <div class="border-t border-line-soft px-4 py-4">
            <p v-if="chatError" class="mb-3 rounded-xl border border-danger/24 bg-danger/18 px-3 py-2 text-[12px] text-danger-soft">
              {{ chatError }}
            </p>

            <textarea
              ref="chatInputRef"
              v-model="chatDraft"
              rows="3"
              :disabled="!game.enable_chat || chatSending"
              @keydown.enter.exact.prevent="sendChatMessage"
              placeholder="Write a message for the table..."
              class="session-input w-full resize-none rounded-[1.4rem] border border-line bg-n1/72 px-4 py-3 text-[14px] text-ink outline-none transition-all duration-200 placeholder:text-faint focus:border-primary/34 disabled:cursor-not-allowed disabled:opacity-60"
            ></textarea>

            <div class="mt-3 flex items-center justify-between gap-3">
              <p class="text-[13px] font-semibold text-faint">Enter to send</p>
              <button
                @click="sendChatMessage"
                :disabled="!chatDraft.trim() || !game.enable_chat || chatSending"
                class="cursor-pointer rounded-xl border border-primary/28 bg-[linear-gradient(135deg,rgba(88,101,221,0.9),rgba(74,86,200,0.9))] px-4 py-2.5 text-[13px] font-semibold text-white transition-all duration-200 disabled:cursor-not-allowed disabled:opacity-60"
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

/* One-row header: back, game, sections, then the character and the table's tools. */
.session-header {
  position: sticky;
  top: 0;
  z-index: 30;
  background: rgb(6 10 28 / 0.78);
  backdrop-filter: blur(14px);
  -webkit-backdrop-filter: blur(14px);
}

.session-header__inner {
  max-width: 1920px;
  margin: 0 auto;
  display: flex;
  align-items: center;
  gap: 0.75rem 1.25rem;
  min-height: 4rem;
  padding: 0.6rem 1.5rem;
}

.session-header__title {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  min-width: 0;
}

.session-header__title h1 {
  margin: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 18px;
  font-weight: 700;
  letter-spacing: -0.01em;
  color: var(--color-ink);
}

.session-gm-badge {
  flex-shrink: 0;
  padding: 0.15rem 0.5rem;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 700;
  color: var(--color-warning);
  background: rgb(239 195 94 / 0.14);
}

.session-tabs {
  display: flex;
  gap: 0.25rem;
  margin-right: auto;
}

.session-tabs .session-tab-button {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  padding: 0.5rem 0.85rem;
  border: 0;
  border-radius: 0.65rem;
  background: none;
  font: inherit;
  font-size: 14px;
  font-weight: 500;
  color: var(--color-dim);
  cursor: pointer;
  transition: background-color 0.2s ease, color 0.2s ease;
}

.session-tabs .session-tab-button:hover {
  color: var(--color-ink);
  background: rgb(27 39 80 / 0.7);
}

.session-tab-button.is-active {
  color: var(--color-ink);
  font-weight: 600;
  background: rgb(88 101 221 / 0.22);
  box-shadow: inset 0 0 0 1px rgb(164 174 255 / 0.35);
}

.session-header__actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-left: auto;
}

.session-switcher {
  display: flex;
  align-items: center;
  gap: 0.4rem;
}

.session-switcher__select {
  width: 15rem;
}

.session-icon-btn,
.session-action-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.45rem;
  height: 2.5rem;
  border: 0;
  border-radius: 0.7rem;
  background: var(--color-n3);
  font: inherit;
  font-size: 14px;
  font-weight: 600;
  color: var(--color-ink);
  cursor: pointer;
  transition: background-color 0.2s ease, transform 0.16s ease;
}

.session-icon-btn {
  width: 2.5rem;
  flex-shrink: 0;
}

.session-action-btn {
  padding: 0 0.9rem;
}

.session-icon-btn:hover,
.session-action-btn:hover {
  background: var(--color-n4);
}

.session-icon-btn:active,
.session-action-btn:active {
  transform: scale(0.96);
}

.session-icon-btn:disabled {
  opacity: 0.5;
  cursor: progress;
}

.session-action-btn.is-on {
  background: rgb(88 101 221 / 0.22);
  box-shadow: inset 0 0 0 1px rgb(164 174 255 / 0.35);
}

/* An incoming trade waits for an answer: the button carries the orchid of the logo's moons. */
.session-action-btn.has-news {
  box-shadow: inset 0 0 0 1px rgb(232 144 210 / 0.55);
}

.session-count {
  min-width: 1.35rem;
  padding: 0 0.4rem;
  border-radius: 999px;
  font-family: var(--font-num);
  font-size: 13px;
  line-height: 1.35rem;
  text-align: center;
  color: #fff;
  background: var(--color-orchid);
}

.session-count--quiet {
  color: var(--color-dim);
  background: var(--color-n5);
}

/* ---------- Inventory tab: board centred, summary beside or above it ---------- */
.inv-desk {
  display: grid;
  gap: 1.25rem 1.5rem;
}

.inv-desk__summary,
.inv-desk__offers {
  display: grid;
  gap: 1.1rem;
  padding: 1.25rem;
  border-radius: 1.25rem;
  background: rgb(8 13 36 / 0.82);
  box-shadow: inset 0 0 0 1px rgb(74 90 150 / 0.35);
}

/* Sides: equal outer tracks keep the board on the centre line of the screen. */
.inv-desk--sides {
  grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr);
  align-items: start;
}

.inv-desk--sides .inv-desk__summary,
.inv-desk--sides .inv-desk__offers {
  position: sticky;
  top: 5.5rem;
  width: min(100%, 19rem);
}

.inv-desk--sides .inv-desk__summary {
  grid-column: 1;
  grid-row: 1;
  justify-self: end;
}

.inv-desk--sides .inv-desk__board {
  grid-column: 2;
  grid-row: 1;
}

.inv-desk--sides .inv-desk__offers {
  grid-column: 3;
  grid-row: 1;
  justify-self: start;
}

/* Stack: a compact strip as wide as the board, then the board, then any offers. */
.inv-desk--stack {
  grid-template-columns: minmax(0, 1fr);
  justify-items: center;
}

/* The board centres itself (it is fit-content from lg up, full width below). */
.inv-desk--stack .inv-desk__board {
  justify-self: stretch;
  min-width: 0;
}

.inv-desk--sides .inv-desk__board {
  min-width: 0;
}

.inv-desk--stack .inv-desk__summary,
.inv-desk--stack .inv-desk__offers {
  width: 100%;
  max-width: max(var(--board-w), 20rem);
}

.inv-desk--stack .inv-desk__summary {
  grid-template-columns: auto auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 0.75rem 1.25rem;
  padding: 0.85rem 1rem;
}

.inv-desk--stack .inv-desk__offers {
  order: -1;
}

.inv-desk--stack .inv-vitals {
  display: flex;
  gap: 0.5rem;
}

.inv-desk--stack .inv-vital {
  padding: 0.45rem 0.7rem;
  min-width: 6.5rem;
}

.inv-desk--stack .inv-vital__value {
  font-size: 21px;
}

.inv-desk--stack .inv-stats {
  grid-template-columns: repeat(auto-fit, minmax(3.4rem, 1fr));
}

.inv-desk--stack .inv-stats li {
  padding: 0.35rem 0.2rem;
}

.inv-desk--stack .inv-stats__value {
  font-size: 18px;
}

.inv-desk--stack .inv-coins {
  flex-direction: column;
  gap: 0.1rem;
}

.inv-desk--stack .inv-coins .num {
  font-size: 16px;
}

.inv-desk--stack .inv-who__portrait {
  width: 2.75rem;
  height: 2.75rem;
  font-size: 16px;
}

@media (max-width: 899px) {
  .inv-desk--stack .inv-desk__summary {
    grid-template-columns: minmax(0, 1fr) auto;
  }

  .inv-desk--stack .inv-stats {
    grid-column: 1 / -1;
    order: 3;
  }

  .inv-desk--stack .inv-coins {
    flex-direction: row;
    gap: 0.9rem;
  }
}

@media (max-width: 599px) {
  .inv-desk--stack .inv-desk__summary {
    grid-template-columns: minmax(0, 1fr);
  }

  .inv-desk--stack .inv-vital {
    flex: 1;
    min-width: 0;
  }
}

.inv-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--color-faint);
}

.inv-bonus {
  margin-left: 0.15rem;
  font-style: normal;
  font-size: 13px;
  color: var(--color-orchid);
}

.inv-who {
  display: flex;
  align-items: center;
  gap: 0.85rem;
}

.inv-who__portrait {
  display: grid;
  place-items: center;
  width: 3.5rem;
  height: 3.5rem;
  flex-shrink: 0;
  overflow: hidden;
  border-radius: 0.9rem;
  background: var(--color-n3);
  font-size: 20px;
  font-weight: 700;
  color: var(--color-primary-ink);
}

.inv-who__portrait img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.inv-who__name {
  margin: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 17px;
  font-weight: 700;
  color: var(--color-ink);
}

.inv-who__owner {
  margin: 0.1rem 0 0;
  font-size: 13px;
  color: var(--color-dim);
}

.inv-vitals {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 0.75rem;
}

.inv-vital {
  display: grid;
  gap: 0.15rem;
  padding: 0.65rem 0.8rem;
  border-radius: 0.8rem;
  background: var(--color-n2);
}

.inv-vital__value {
  font-size: 26px;
  font-weight: 700;
  line-height: 1;
  color: var(--color-ink);
}

.inv-vital__value small {
  font-size: 16px;
  color: var(--color-dim);
}

.inv-vital__bar {
  height: 4px;
  margin-top: 0.35rem;
  border-radius: 2px;
  background: var(--color-n4);
}

.inv-vital__bar i {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--color-success);
}

.inv-stats {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 0.4rem;
  margin: 0;
  padding: 0;
  list-style: none;
}

.inv-stats li {
  display: grid;
  justify-items: center;
  padding: 0.5rem 0.25rem;
  border-radius: 0.7rem;
  background: var(--color-n2);
  cursor: default;
}

.inv-stats__value {
  font-size: 22px;
  font-weight: 700;
  line-height: 1.1;
  color: var(--color-ink);
}

.inv-coins {
  display: flex;
  gap: 1rem;
  margin: 0;
  padding: 0;
  list-style: none;
}

.inv-coins li {
  display: flex;
  align-items: baseline;
  gap: 0.3rem;
}

.inv-coins .num {
  font-size: 19px;
  font-weight: 700;
}

.inv-coin--gold .num { color: #f1cf72; }
.inv-coin--silver .num { color: #d3d9ea; }
.inv-coin--copper .num { color: #e0a079; }

.inv-offers {
  gap: 0.6rem;
}

.inv-offers__title {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: var(--color-ink);
}

.inv-offer {
  display: grid;
  gap: 0.6rem;
  padding: 0.8rem;
  border-radius: 0.8rem;
  background: var(--color-n2);
  box-shadow: inset 0 0 0 1px rgb(232 144 210 / 0.4);
}

.inv-offer__text {
  margin: 0;
  font-size: 14px;
  line-height: 1.45;
  color: var(--color-ink-soft);
}

.inv-offer__actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.4rem;
}


/* ---------- Character tab: read-only sheet ---------- */
.sheet {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 1.25rem;
}

.sheet__col {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  min-height: 0;
  padding: 1.25rem;
  border-radius: 1.25rem;
  background: rgb(8 13 36 / 0.82);
  box-shadow: inset 0 0 0 1px rgb(74 90 150 / 0.35);
}

.sheet__portrait {
  display: grid;
  place-items: center;
  flex-shrink: 1;
  min-height: 7rem;
  aspect-ratio: 4 / 3;
  overflow: hidden;
  border-radius: 0.9rem;
  background: var(--color-n0);
  font-size: 44px;
  font-weight: 700;
  color: var(--color-primary-ink);
}

.sheet__portrait.has-image {
  aspect-ratio: 3 / 4;
}

@media (max-width: 899px) {
  .sheet__portrait:not(.has-image) {
    aspect-ratio: auto;
    height: 6rem;
    font-size: 34px;
  }

  .sheet__portrait.has-image {
    max-height: 22rem;
  }
}

.sheet__portrait img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  object-position: top;
}

.sheet__name {
  margin: 0;
  font-size: 24px;
  font-weight: 700;
  letter-spacing: -0.02em;
  line-height: 1.15;
  color: var(--color-ink);
  overflow-wrap: anywhere;
}

.sheet__owner {
  margin: 0.2rem 0 0;
  font-size: 14px;
  color: var(--color-dim);
}

.sheet__vitals {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.5rem;
}

.sheet-stat {
  display: grid;
  grid-template-columns: auto 1fr;
  align-items: center;
  column-gap: 0.5rem;
  row-gap: 0.15rem;
  padding: 0.7rem 0.85rem;
  border-radius: 0.8rem;
  background: var(--color-n2);
  cursor: default;
}

.sheet-stat--wide {
  grid-column: 1 / -1;
}

.sheet-stat__icon {
  width: 1.1rem;
  height: 1.1rem;
  color: var(--color-primary-ink);
}

.sheet-stat__label {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-dim);
}

.sheet-stat__value {
  grid-column: 1 / -1;
  font-size: 26px;
  font-weight: 700;
  line-height: 1.05;
  color: var(--color-ink);
}

.sheet-stat__value small {
  font-size: 16px;
  color: var(--color-dim);
}

.sheet-bonus {
  margin-left: 0.3rem;
  font-style: normal;
  font-size: 15px;
  color: var(--color-orchid);
}

/* Gear can also take a stat down; that reads as a penalty, not a bonus. */
.sheet-bonus.is-neg,
.inv-bonus.is-neg {
  color: var(--color-danger);
}

.sheet-stat--plain {
  grid-template-columns: 1fr;
}

.sheet__subtitle {
  margin: 1.25rem 0 0.6rem;
  font-size: 14px;
  font-weight: 600;
  color: var(--color-dim);
}

.sheet__scroll {
  min-height: 0;
  overflow-y: auto;
  padding-right: 0.2rem;
}

.sheet-hp {
  grid-column: 1 / -1;
  height: 5px;
  margin-top: 0.3rem;
  border-radius: 3px;
  background: var(--color-n4);
}

.sheet-hp i {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--color-success);
  transition: width 0.3s ease;
}

.sheet__coins {
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem 1.1rem;
  margin: 0;
  padding: 0;
  list-style: none;
}

.sheet__coins li {
  display: flex;
  align-items: baseline;
  gap: 0.3rem;
}

.sheet__coins .num {
  font-size: 22px;
  font-weight: 700;
}

.sheet__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
}

.sheet__title {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  color: var(--color-ink);
}

.sheet__count {
  font-size: 15px;
  color: var(--color-dim);
}

.sheet__story {
  margin: 0;
  min-height: 0;
  overflow-y: auto;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  font-size: 15px;
  line-height: 1.6;
  color: var(--color-ink-soft);
}

.sheet__story.is-empty {
  color: var(--color-faint);
}

.sheet__note {
  margin: 0;
  font-size: 14px;
  color: var(--color-faint);
}

.sheet__attrs {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(9.5rem, 1fr));
  gap: 0.5rem;
  margin: 0;
  padding: 0;
  list-style: none;
}

.sheet__block {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  min-height: 0;
}

.sheet__list {
  display: grid;
  align-content: start;
  gap: 0.35rem;
  min-height: 0;
  margin: 0;
  padding: 0 0.2rem 0 0;
  overflow-y: auto;
  list-style: none;
}

.sheet__list li {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 0.55rem 0.75rem;
  border-radius: 0.65rem;
  background: var(--color-n2);
}

.sheet__list li.is-link {
  cursor: pointer;
  transition: background-color 0.2s ease;
}

.sheet__list li.is-link:hover {
  background: var(--color-n3);
}

.sheet__list-name {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  min-width: 0;
  font-size: 14px;
  font-weight: 500;
  color: var(--color-ink);
  overflow-wrap: anywhere;
}

.sheet__list-value {
  flex-shrink: 0;
  font-size: 19px;
  font-weight: 700;
  color: var(--color-ink);
}

.sheet__dot {
  width: 0.6rem;
  height: 0.6rem;
  flex-shrink: 0;
  border-radius: 0.2rem;
  background: color-mix(in oklab, var(--resin) 40%, var(--color-n2));
  box-shadow: inset 0 0 0 1.5px var(--resin);
}

.sheet__slot {
  flex-shrink: 0;
  font-size: 13px;
  color: var(--color-dim);
}

.sheet__empty {
  margin: 0;
  font-size: 14px;
  line-height: 1.5;
  color: var(--color-faint);
}

@media (min-width: 900px) {
  .sheet {
    grid-template-columns: minmax(15rem, 19rem) minmax(0, 1fr);
  }

  .sheet__side {
    grid-column: 1 / -1;
  }
}

/* Desktop: the whole sheet fits one screen under the header; long text and lists scroll inside. */
@media (min-width: 1280px) and (min-height: 640px) {
  /* Columns are as tall as their content, never taller than the screen under the header. */
  .sheet {
    grid-template-columns: minmax(15rem, 19rem) minmax(0, 1fr) minmax(16rem, 21rem);
    align-items: start;
  }

  .sheet__col {
    max-height: calc(100svh - 7.75rem);
  }

  .sheet__side {
    grid-column: auto;
  }

  .sheet__who {
    overflow: hidden;
  }

  .sheet__portrait {
    min-height: 0;
    max-height: 42vh;
  }

  /* Story and gear share the column; each scrolls in its half when long. */
  .sheet__side .sheet__block {
    flex: 0 1 auto;
    max-height: 50%;
  }

  .sheet__main {
    overflow: hidden;
  }
}

.session-section-title {
  margin: 0;
  font-size: 26px;
  font-weight: 700;
  letter-spacing: -0.02em;
  color: var(--color-ink);
}

.session-primary-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  min-height: 2.6rem;
  padding: 0 1rem;
  border: 0;
  border-radius: 0.7rem;
  background: var(--color-primary);
  font: inherit;
  font-size: 14px;
  font-weight: 600;
  color: #fff;
  cursor: pointer;
  transition: background-color 0.2s ease;
}

.session-primary-btn:hover {
  background: var(--color-primary-hover);
}

.session-primary-btn:disabled {
  opacity: 0.6;
  cursor: progress;
}

/* Bottom tab bar, phones only. */
.session-tabbar {
  display: none;
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
  background: rgba(123, 135, 184, 0.22);
  border-radius: 999px;
}

.session-scroll::-webkit-scrollbar-track {
  background: transparent;
}

.session-stage button:not(.session-tab-button):not(.session-chrome):not(.session-primary-btn) {
  border-color: transparent !important;
  background: var(--color-n3) !important;
  color: var(--color-ink) !important;
  border-radius: 0.6rem !important;
  box-shadow: none;
}

.session-stage button:not(.session-tab-button):not(.session-chrome):not(.session-primary-btn):hover {
  background: var(--color-n4) !important;
  color: #ffffff !important;
}

.session-stage button:not(.session-tab-button):not(.session-chrome):not(.session-primary-btn):disabled {
  box-shadow: none;
}

.session-stage select,
.session-stage textarea,
.session-stage input:not([type='file']):not(.border-0),
.session-input {
  border-radius: 0.65rem !important;
  border-color: var(--color-n5) !important;
  background: var(--color-n0) !important;
  color: var(--color-ink) !important;
  box-shadow: none;
}

.session-stage select:focus,
.session-stage textarea:focus,
.session-stage input:not([type='file']):not(.border-0):focus,
.session-input:focus {
  border-color: rgba(88, 101, 221, 0.4) !important;
  box-shadow: 0 0 0 1px rgba(88, 101, 221, 0.18), inset 0 1px 0 rgba(255, 255, 255, 0.03);
}

.session-stage article:not(.session-chat-message) {
  position: relative;
  border-radius: 1.25rem !important;
  border-color: transparent !important;
  background: rgb(8 13 36 / 0.82) !important;
  box-shadow: inset 0 0 0 1px rgb(74 90 150 / 0.35) !important;
}

.session-mode-pill,
.session-count-pill,
.session-owner-pill {
  position: relative;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.04);
}

.session-banner {
  border-color: rgba(88, 101, 221, 0.28) !important;
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
  padding: 1.5rem 1rem 22rem;
  transition: padding-right 0.28s ease, padding-bottom 0.28s ease;
}

.session-content-shell.chat-collapsed {
  padding-bottom: 5rem;
}

.session-chat-dock {
  position: fixed;
  left: 0.5rem;
  right: 0.5rem;
  bottom: 0.5rem;
  z-index: 40;
  height: 20rem;
  overflow: hidden;
  border: 1px solid rgba(123, 135, 184, 0.14);
  border-radius: 0.8rem;
  background:
    linear-gradient(180deg, rgba(14, 18, 33, 0.98), rgba(9, 13, 25, 0.99)),
    repeating-linear-gradient(90deg, rgba(123, 135, 184, 0.04) 0 1px, transparent 1px 72px);
  backdrop-filter: blur(24px);
  box-shadow: 0 35px 80px rgba(0, 0, 0, 0.38);
}

.session-chat-dock::before {
  content: '';
  position: absolute;
  inset: 8px;
  pointer-events: none;
  border: 1px solid rgba(123, 135, 184, 0.1);
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
  border-color: rgba(123, 135, 184, 0.18) !important;
  background:
    repeating-linear-gradient(90deg, rgba(123, 135, 184, 0.08) 0 1px, transparent 1px 56px),
    repeating-linear-gradient(0deg, rgba(123, 135, 184, 0.08) 0 1px, transparent 1px 56px),
    linear-gradient(180deg, rgba(10, 16, 29, 0.84), rgba(7, 11, 22, 0.92)) !important;
}

.session-sheet-frame {
  padding: 0;
}

.session-sheet-grid {
  align-items: start;
}

.session-sheet-support-grid,
.session-inventory-layout {
  align-items: stretch;
}

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
  align-self: start;
}

/* Equipment panel fills the height of the taller Custom Attributes panel so its
   list scrolls within the matched height instead of leaving a fixed gap. */
.session-equipment-panel {
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
  border: 1px solid rgba(123, 135, 184, 0.14);
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

@media (max-width: 639px) {
  .session-content-shell {
    padding: 1.5rem 1rem 27rem 1rem;
  }

  .session-content-shell.chat-collapsed {
    padding-bottom: 9.5rem;
  }

  .session-chat-dock {
    bottom: 4.75rem;
  }
}

/* Below the width where every section fits beside the header tools, sections move to a bottom bar. */
@media (max-width: 1099px) {
  .session-tabs {
    display: none;
  }

  .session-tabbar {
    position: fixed;
    left: 0.5rem;
    right: 0.5rem;
    bottom: 0.5rem;
    z-index: 45;
    display: flex;
    gap: 0.25rem;
    padding: 0.35rem;
    border-radius: 1rem;
    background: rgb(6 10 28 / 0.9);
    backdrop-filter: blur(14px);
    -webkit-backdrop-filter: blur(14px);
    box-shadow: inset 0 0 0 1px rgb(51 66 122 / 0.6);
  }

  .session-tabbar .session-tab-button {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.2rem;
    padding: 0.45rem 0.25rem;
    border: 0;
    border-radius: 0.7rem;
    background: none;
    font: inherit;
    font-size: 12px;
    font-weight: 600;
    color: var(--color-dim);
    cursor: pointer;
  }

  .session-content-shell,
  .session-content-shell.chat-collapsed {
    padding-bottom: 6rem;
  }

  .session-chat-dock {
    bottom: 4.9rem !important;
  }

  .session-header__title {
    margin-right: auto;
  }
}

@media (max-width: 720px) {
  .session-header__inner {
    flex-wrap: wrap;
    padding: 0.5rem 0.75rem;
  }

  .session-header__actions {
    width: 100%;
    margin-left: 0;
  }

  .session-switcher {
    flex: 1;
    min-width: 0;
  }

  .session-switcher__select {
    width: auto;
    flex: 1;
    min-width: 0;
  }

  .session-action-btn__label {
    display: none;
  }
}

@media (min-width: 640px) {
  .session-content-shell {
    padding-left: 1.5rem;
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
    padding-left: 1.5rem;
    padding-right: 27rem;
  }

  .session-content-shell.chat-collapsed {
    padding-bottom: 2rem;
    padding-right: 1.5rem;
  }

  .session-chat-dock {
    top: 4.75rem;
    bottom: 1.5rem;
    right: 1.5rem;
    left: auto;
    width: 24rem;
    height: auto;
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
    padding-right: 27rem;
  }


  .session-empty-bay--inventory {
    min-height: 38rem;
  }
}
</style>