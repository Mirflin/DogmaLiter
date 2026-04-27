<script setup>
import { API_URL } from '@/api'
import SessionCharacterPickerModal from '@/components/session/SessionCharacterPickerModal.vue'
import SessionItemManager from '@/components/session/SessionItemManager.vue'
import { useAuthStore } from '@/stores/auth'
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const CHAT_POLL_INTERVAL = 15000
const CHAT_MESSAGE_LIMIT = 40
const MAX_PORTRAIT_SIZE = 5 * 1024 * 1024
const ALLOWED_PORTRAIT_TYPES = ['image/jpeg', 'image/png', 'image/webp']

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
const error = ref(null)
const characterError = ref(null)
const characterSaveError = ref('')
const characterSaveNotice = ref('')
const pickerError = ref('')
const chatError = ref(null)
const chatDraft = ref('')
const chatMessagesRef = ref(null)
const chatInputRef = ref(null)
const portraitInputRef = ref(null)
const characterSavePending = ref(false)
const portraitFile = ref(null)
const portraitPreviewUrl = ref('')
const characterForm = ref(createCharacterFormState())

let chatPollHandle = null
let characterRequestId = 0

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
const characters = computed(() => session.value?.characters ?? [])
const items = computed(() => session.value?.items ?? [])
const chatMessages = computed(() => session.value?.messages ?? [])
const selectedCharacterSummary = computed(() => characters.value.find(character => character.id === activeCharacterId.value) ?? null)
const characterSnapshot = computed(() => activeCharacter.value ?? selectedCharacterSummary.value ?? null)
const canEditCharacter = computed(() => {
  if (!characterSnapshot.value) return false
  return isGM.value || characterSnapshot.value.user_id === viewer.value?.user_id
})
const attributeCards = computed(() => {
  const stats = characterSnapshot.value?.base_attributes
  if (!stats) return []

  return [
    { key: 'strength', label: 'Strength', value: stats.strength ?? 0 },
    { key: 'dexterity', label: 'Dexterity', value: stats.dexterity ?? 0 },
    { key: 'constitution', label: 'Constitution', value: stats.constitution ?? 0 },
    { key: 'intelligence', label: 'Intelligence', value: stats.intelligence ?? 0 },
    { key: 'wisdom', label: 'Wisdom', value: stats.wisdom ?? 0 },
    { key: 'charisma', label: 'Charisma', value: stats.charisma ?? 0 },
  ]
})
const inventoryItems = computed(() => activeCharacter.value?.inventory ?? [])
const equipment = computed(() => activeCharacter.value?.equipment ?? [])
const customAttributes = computed(() => characterSnapshot.value?.custom_attributes ?? [])
const inventoryWidth = computed(() => characterSnapshot.value?.inventory_width ?? 0)
const inventoryHeight = computed(() => characterSnapshot.value?.inventory_height ?? 0)
const inventoryCapacity = computed(() => inventoryWidth.value * inventoryHeight.value)
const occupiedInventoryCells = computed(() => inventoryItems.value.reduce((total, entry) => {
  const width = entry.item?.grid_width ?? 1
  const height = entry.item?.grid_height ?? 1
  return total + (width * height)
}, 0))
const memberCharacterCount = computed(() => characters.value.reduce((counts, character) => {
  counts[character.user_id] = (counts[character.user_id] ?? 0) + 1
  return counts
}, {}))
const currencyCards = computed(() => {
  const snapshot = characterSnapshot.value

  return [
    { key: 'gold', label: 'Gold', value: snapshot?.currency_gold ?? 0 },
    { key: 'silver', label: 'Silver', value: snapshot?.currency_silver ?? 0 },
    { key: 'copper', label: 'Bronze', value: snapshot?.currency_copper ?? 0 },
  ]
})
const currentPortraitUrl = computed(() => portraitPreviewUrl.value || avatarUrl(characterSnapshot.value?.portrait_id))
const tabs = computed(() => {
  const baseTabs = [
    { id: 'sheet', label: 'Character', icon: 'sheet', description: 'Portrait, identity, profile editing, and read-only character stats.' },
    { id: 'inventory', label: 'Inventory', icon: 'inventory', description: 'Reserved inventory workspace with currency and storage stats.' },
    { id: 'characters', label: 'Roster', icon: 'characters', description: 'Quick switching between characters available to this viewer.' },
  ]

  if (isGM.value) {
    baseTabs.push(
      { id: 'players', label: 'Players', icon: 'players', description: 'GM roster view for members and their accessible characters.' },
      { id: 'items', label: 'Items', icon: 'items', description: 'File-manager style item workspace for GM operations.' },
    )
  }

  return baseTabs
})
const activeTabMeta = computed(() => tabs.value.find(tab => tab.id === activeTab.value) ?? tabs.value[0] ?? null)

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
    common: 'border-[rgba(148,163,184,0.35)] bg-[rgba(148,163,184,0.12)] text-[#e2e8f0]',
    uncommon: 'border-[rgba(110,231,183,0.35)] bg-[rgba(22,163,74,0.14)] text-[#86efac]',
    rare: 'border-[rgba(96,165,250,0.35)] bg-[rgba(37,99,235,0.16)] text-[#93c5fd]',
    epic: 'border-[rgba(244,114,182,0.35)] bg-[rgba(190,24,93,0.16)] text-[#f9a8d4]',
    legendary: 'border-[rgba(251,191,36,0.35)] bg-[rgba(217,119,6,0.18)] text-[#fde68a]',
    artifact: 'border-[rgba(248,113,113,0.4)] bg-[rgba(153,27,27,0.18)] text-[#fca5a5]',
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
  }
}

function normalizeCurrencyValue(value) {
  const parsed = Number.parseInt(value, 10)
  if (Number.isNaN(parsed)) return 0
  return Math.min(999999999, Math.max(0, parsed))
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
  syncCharacterForm(null)
}

function goBack() {
  router.push(`/games/${gameId.value}`)
}

function charactersForUser(userId) {
  return characters.value.filter(character => character.user_id === userId)
}

function openCharacterPicker() {
  if (!characters.value.length && !viewer.value?.can_create_character) return
  pickerError.value = ''
  pickerVisible.value = true
}

function openProfileEditor() {
  if (!canEditCharacter.value) return
  characterSaveError.value = ''
  characterSaveNotice.value = ''
  profileEditMode.value = true
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
  activeTab.value = nextTab
  pickerVisible.value = false
  pickerError.value = ''

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
  const nextOwnedCount = currentOwnedCount + 1
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
  pickerError.value = ''

  if (session.value) {
    refreshing.value = true
  } else {
    loading.value = true
  }

  try {
    const data = await auth.getGameSession(gameId.value)
    session.value = data

    if (!tabs.value.some(tab => tab.id === activeTab.value)) {
      activeTab.value = 'sheet'
    }

    const availableCharacters = data.characters ?? []
    const keepCurrentCharacter = Boolean(previousCharacterId && availableCharacters.some(character => character.id === previousCharacterId))

    if (promptSelection) {
      clearActiveCharacter()
      pickerVisible.value = Boolean(availableCharacters.length || data.viewer?.can_create_character)
    } else if (keepCurrentCharacter) {
      pickerVisible.value = false
      activeCharacterId.value = previousCharacterId
      if (activeCharacter.value?.id !== previousCharacterId) {
        await loadCharacter(previousCharacterId)
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

async function createCharacter() {
  if (creatingCharacter.value || !viewer.value?.can_create_character) return

  creatingCharacter.value = true
  pickerError.value = ''

  try {
    const data = await auth.createGameCharacter(gameId.value)
    if (data?.character) {
      mergeCreatedCharacterIntoSession(data.character)
      await switchCharacter(data.character.id, {
        nextTab: 'sheet',
        prefetchedCharacter: data.character,
      })
    }

    await loadSession({ preserveCharacter: true, promptSelection: false })
  } catch (err) {
    pickerError.value = err.response?.data?.error || 'Failed to create a new character'
  } finally {
    creatingCharacter.value = false
  }
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
    let latestCharacter = snapshot

    if (hasProfileChanges) {
      const data = await auth.updateGameCharacter(gameId.value, snapshot.id, payload)
      if (data?.character) {
        latestCharacter = data.character
        mergeUpdatedCharacterIntoSession(latestCharacter)
      }
    }

    if (portraitFile.value) {
      const portraitResponse = await auth.uploadCharacterPortrait(gameId.value, snapshot.id, portraitFile.value)
      if (portraitResponse?.character) {
        latestCharacter = portraitResponse.character
        mergeUpdatedCharacterIntoSession(latestCharacter)
      }
    }

    syncCharacterForm(latestCharacter)
    characterSaveNotice.value = 'Profile saved'
  } catch (err) {
    characterSaveError.value = err.response?.data?.error || 'Failed to save character profile'
  } finally {
    characterSavePending.value = false
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
  if (!activeCharacterId.value) {
    syncCharacterForm(null)
  }
})

onMounted(() => {
  loadSession({ preserveCharacter: false, promptSelection: true })
})

onBeforeUnmount(() => {
  stopChatPolling()
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
      <svg class="mb-4 h-16 w-16 text-[#e94560]" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
      </svg>
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
        :error-message="pickerError"
        @select="switchCharacter"
        @create="createCharacter"
      />

      <header class="session-header sticky top-0 z-30 border-b border-[rgba(126,200,227,0.08)] bg-[rgba(7,17,31,0.82)] backdrop-blur-xl">
        <div class="mx-auto flex max-w-[1920px] flex-col gap-4 px-4 py-4 pl-[4.75rem] sm:px-6 sm:pl-[5.75rem] lg:flex-row lg:items-center lg:justify-between xl:pl-[7rem]">
          <div class="flex min-w-0 items-center gap-3">
            <button
              @click="goBack"
              title="Exit session"
              class="flex h-10 w-10 shrink-0 cursor-pointer items-center justify-center rounded-xl border border-[rgba(126,200,227,0.14)] bg-[rgba(126,200,227,0.06)] text-[#e8e8f0]/70 transition-all duration-200 hover:border-[#e94560] hover:text-[#e94560]"
            >
              <svg class="h-5 w-5" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" d="M10.5 19.5L3 12m0 0l7.5-7.5M3 12h18" />
              </svg>
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
            <button
              @click="loadSession({ preserveCharacter: true, promptSelection: false })"
              :disabled="refreshing"
              class="flex cursor-pointer items-center gap-2 rounded-xl border border-[rgba(126,200,227,0.16)] bg-[rgba(126,200,227,0.08)] px-4 py-2.5 text-[13px] font-semibold text-[#e8e8f0] transition-all duration-200 hover:-translate-y-0.5 hover:border-[rgba(126,200,227,0.35)] disabled:cursor-not-allowed disabled:opacity-60"
            >
              <svg class="h-4 w-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24" :class="refreshing ? 'animate-spin' : ''">
                <path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992V4.356M2.977 14.652H7.97v4.992m12.056-4.992a8.25 8.25 0 00-13.435-5.728L2.977 14.652m18.046-5.304a8.25 8.25 0 01-13.435 5.728L7.97 19.644" />
              </svg>
              {{ refreshing ? 'Refreshing' : 'Refresh Workspace' }}
            </button>
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
          <svg v-if="tab.icon === 'sheet'" class="h-5 w-5" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 6.75a3.75 3.75 0 11-7.5 0 3.75 3.75 0 017.5 0zM4.5 20.25a7.5 7.5 0 0115 0" />
          </svg>
          <svg v-else-if="tab.icon === 'inventory'" class="h-5 w-5" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" d="M20.25 7.5l-8.25-4.5L3.75 7.5m16.5 0v9l-8.25 4.5-8.25-4.5v-9m16.5 0L12 12m0 0L3.75 7.5m8.25 4.5v9" />
          </svg>
          <svg v-else-if="tab.icon === 'characters'" class="h-5 w-5" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" d="M15 19.128A9.38 9.38 0 0012 18.75a9.38 9.38 0 00-3 .378M15 7.5a3 3 0 11-6 0 3 3 0 016 0zm6 10.5a6 6 0 00-7.743-5.743m7.743 5.743A6 6 0 0112.75 21m0-8.743A6 6 0 003 18m9.75-5.743A6 6 0 0011.25 21" />
          </svg>
          <svg v-else-if="tab.icon === 'players'" class="h-5 w-5" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" d="M18 7.5a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0zM9.75 8.25a3 3 0 11-6 0 3 3 0 016 0zm8.25 10.5v-.75A3.75 3.75 0 0014.25 14.25h-1.5A3.75 3.75 0 009 18v.75m9 0h3m-3 0H8.25m0 0H3m5.25 0v-.75A5.25 5.25 0 0113.5 12.75h.75" />
          </svg>
          <svg v-else class="h-5 w-5" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" d="M3.75 5.25h6.879a1.5 1.5 0 011.06.44l1.122 1.12a1.5 1.5 0 001.06.44h6.379m-16.5 0v10.5A2.25 2.25 0 006 20.25h12a2.25 2.25 0 002.25-2.25V9A1.5 1.5 0 0018.75 7.5h-3.879a1.5 1.5 0 01-1.06-.44l-1.122-1.12a1.5 1.5 0 00-1.06-.44H5.25A1.5 1.5 0 003.75 7.5v0" />
          </svg>

          <span v-if="activeTab === tab.id" class="absolute -right-1 h-2.5 w-2.5 rounded-full bg-[#c58a38]"></span>
          <span class="session-tooltip">{{ tab.label }}</span>
        </button>
      </nav>

      <div class="mx-auto max-w-[1920px] session-content-shell" :class="{ 'chat-collapsed': chatCollapsed }">
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
                <p class="mt-2 max-w-[44rem] text-[13px] leading-relaxed text-[#d8dce7]/62">{{ activeTabMeta?.description }}</p>
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
                        <p class="mt-2 max-w-[32rem] text-[14px] leading-relaxed text-[#d8dce7]/62">Basic profile info, portrait, and currency are editable only in profile mode. Base parameters stay read-only.</p>
                      </div>

                      <div class="flex flex-wrap gap-3">
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
                        <button
                          @click="openCharacterPicker"
                          class="cursor-pointer rounded-xl border border-[rgba(126,200,227,0.14)] bg-[rgba(126,200,227,0.08)] px-4 py-2.5 text-[13px] font-semibold text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(126,200,227,0.32)]"
                        >
                          Change Character
                        </button>
                      </div>
                    </div>

                    <p v-if="characterSaveError" class="mt-5 rounded-[1.3rem] border border-[rgba(248,113,113,0.24)] bg-[rgba(127,29,29,0.18)] px-4 py-3 text-[13px] text-[#fecaca]">
                      {{ characterSaveError }}
                    </p>
                    <p v-else-if="characterSaveNotice" class="mt-5 rounded-[1.3rem] border border-[rgba(110,231,183,0.22)] bg-[rgba(21,128,61,0.16)] px-4 py-3 text-[13px] text-[#bbf7d0]">
                      {{ characterSaveNotice }}
                    </p>
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
                              <span class="text-[12px] text-[#d8dce7]/45">Editable in profile mode</span>
                            </div>
                            <div class="mt-4 grid gap-3 sm:grid-cols-3">
                              <label class="rounded-[1.3rem] border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] px-4 py-3">
                                <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Gold</span>
                                <input v-model.number="characterForm.gold" type="number" min="0" class="session-input mt-3 w-full border-0 bg-transparent p-0 text-[24px] font-semibold text-[#f6f7fb] outline-none" />
                              </label>
                              <label class="rounded-[1.3rem] border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] px-4 py-3">
                                <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Silver</span>
                                <input v-model.number="characterForm.silver" type="number" min="0" class="session-input mt-3 w-full border-0 bg-transparent p-0 text-[24px] font-semibold text-[#f6f7fb] outline-none" />
                              </label>
                              <label class="rounded-[1.3rem] border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] px-4 py-3">
                                <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Bronze</span>
                                <input v-model.number="characterForm.copper" type="number" min="0" class="session-input mt-3 w-full border-0 bg-transparent p-0 text-[24px] font-semibold text-[#f6f7fb] outline-none" />
                              </label>
                            </div>
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
                        <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Backstory</p>
                        <p class="mt-4 whitespace-pre-wrap text-[15px] leading-relaxed text-[#e8e8f0]/78">
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

                      <div class="rounded-[1.5rem] border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] p-5">
                        <div class="flex items-center justify-between gap-4">
                          <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Base Attributes</p>
                          <span class="text-[12px] text-[#d8dce7]/45">Read-only</span>
                        </div>
                        <div class="mt-4 grid gap-3 sm:grid-cols-2 2xl:grid-cols-3">
                          <div
                            v-for="attribute in attributeCards"
                            :key="attribute.key"
                            class="rounded-[1.3rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.62)] px-4 py-4"
                          >
                            <p class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">{{ attribute.label }}</p>
                            <p class="mt-2 text-[24px] font-bold text-[#f6f7fb]">{{ attribute.value }}</p>
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

                    <div v-if="customAttributes.length" class="mt-4 grid gap-3 md:grid-cols-2">
                      <div
                        v-for="attribute in customAttributes"
                        :key="attribute.id"
                        class="rounded-2xl border border-[rgba(233,69,96,0.12)] bg-[rgba(233,69,96,0.08)] px-4 py-3"
                      >
                        <p class="text-[11px] uppercase tracking-[0.18em] text-[#ff8fa3]/60">{{ attribute.name }}</p>
                        <p class="mt-2 text-[20px] font-semibold text-[#f6f7fb]">{{ attribute.value }}</p>
                      </div>
                    </div>

                    <p v-else class="mt-4 text-[14px] text-[#d8dce7]/56">No custom attributes were configured for this character.</p>
                  </article>

                  <article class="session-equipment-panel rounded-[1.75rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(8,16,30,0.78)] p-5">
                    <div class="flex items-center justify-between gap-4">
                      <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Equipment</p>
                      <span class="text-[12px] text-[#e8e8f0]/45">{{ equipment.length }} slots filled</span>
                    </div>

                    <div v-if="equipment.length" class="mt-4 space-y-3">
                      <div
                        v-for="slot in equipment"
                        :key="`${slot.slot}-${slot.inventory_item_id}`"
                        class="rounded-2xl border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] px-4 py-3"
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

          <section v-else-if="activeTab === 'inventory'" class="session-inventory-layout grid gap-6 xl:grid-cols-[minmax(0,1fr)_360px]">
            <article class="session-inventory-board rounded-[2rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(11,20,36,0.88)] p-6 shadow-[0_24px_60px_rgba(0,0,0,0.22)] sm:p-8">
              <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Inventory</p>
              <h2 class="mt-3 font-[Cinzel] text-[34px] font-bold text-[#f6f7fb]">Reserved Empty Workspace</h2>
              <p class="mt-4 max-w-[38rem] text-[15px] leading-relaxed text-[#d8dce7]/60">
                This tab stays intentionally empty for the actual inventory canvas. Currency and capacity remain visible here, but the storage surface itself is left blank for the next iteration.
              </p>

              <div class="session-empty-bay session-empty-bay--inventory mt-8 rounded-[1.8rem] border border-dashed border-[rgba(126,200,227,0.18)] bg-[rgba(126,200,227,0.04)] px-6 py-16 text-center">
                <div class="session-inventory-grid-phantom" aria-hidden="true">
                  <span v-for="index in 40" :key="`inventory-cell-${index}`" class="session-inventory-grid-phantom__cell"></span>
                </div>
                <div class="session-inventory-grid-copy">
                  <p class="text-[12px] uppercase tracking-[0.2em] text-[#7ec8e3]/48">Inventory Canvas</p>
                  <p class="mt-4 text-[15px] text-[#d8dce7]/58">The surface stays empty for now, but the layout now reads like a storage board instead of a generic placeholder.</p>
                </div>
              </div>
            </article>

            <aside class="session-inventory-sidebar space-y-5">
              <article class="session-inventory-summary rounded-[2rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(11,20,36,0.88)] p-5 shadow-[0_24px_60px_rgba(0,0,0,0.22)]">
                <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Currency</p>
                <div class="mt-4 grid gap-3">
                  <div
                    v-for="currency in currencyCards"
                    :key="currency.key"
                    class="flex items-center justify-between rounded-2xl border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] px-4 py-3"
                  >
                    <span class="text-[12px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">{{ currency.label }}</span>
                    <span class="text-[16px] font-semibold text-[#f6f7fb]">{{ currency.value }}</span>
                  </div>
                </div>
              </article>

              <article class="session-inventory-meta rounded-[2rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(11,20,36,0.88)] p-5 shadow-[0_24px_60px_rgba(0,0,0,0.22)]">
                <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Storage Stats</p>
                <div class="mt-4 space-y-3 text-[13px] text-[#d8dce7]/62">
                  <div class="rounded-2xl border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] px-4 py-3">Grid: {{ inventoryWidth }} x {{ inventoryHeight }}</div>
                  <div class="rounded-2xl border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] px-4 py-3">Capacity: {{ inventoryCapacity }} cells</div>
                  <div class="rounded-2xl border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] px-4 py-3">Occupied: {{ occupiedInventoryCells }} cells</div>
                </div>
              </article>
            </aside>
          </section>

          <section v-else-if="activeTab === 'characters'" class="space-y-5">
            <article class="rounded-[2rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(11,20,36,0.88)] p-5 shadow-[0_24px_60px_rgba(0,0,0,0.22)] sm:p-6">
              <div class="flex flex-wrap items-start justify-between gap-4">
                <div>
                  <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Character Switcher</p>
                  <h3 class="mt-2 font-[Cinzel] text-[30px] font-bold text-[#f6f7fb]">Available Characters</h3>
                  <p class="mt-2 max-w-[40rem] text-[14px] leading-relaxed text-[#d8dce7]/62">
                    Use this roster for fast switching. The entry modal uses the same list and creation flow.
                  </p>
                </div>

                <div class="flex flex-wrap gap-3">
                  <button
                    @click="openCharacterPicker"
                    class="cursor-pointer rounded-xl border border-[rgba(126,200,227,0.16)] bg-[rgba(126,200,227,0.08)] px-4 py-2.5 text-[13px] font-semibold text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(126,200,227,0.3)]"
                  >
                    Open Picker Modal
                  </button>
                  <button
                    v-if="viewer.can_create_character"
                    @click="createCharacter"
                    :disabled="creatingCharacter"
                    class="cursor-pointer rounded-xl border border-[rgba(233,69,96,0.24)] bg-[rgba(233,69,96,0.12)] px-4 py-2.5 text-[13px] font-semibold text-[#ffe0e7] transition-all duration-200 hover:border-[rgba(233,69,96,0.45)] disabled:cursor-not-allowed disabled:opacity-60"
                  >
                    {{ creatingCharacter ? 'Creating...' : 'Create New Character' }}
                  </button>
                </div>
              </div>
            </article>

            <div class="grid gap-4 md:grid-cols-2 2xl:grid-cols-3">
              <article
                v-for="character in characters"
                :key="character.id"
                class="rounded-[1.8rem] border p-5 transition-all duration-200"
                :class="character.id === activeCharacterId
                  ? 'border-[rgba(233,69,96,0.36)] bg-[rgba(233,69,96,0.11)] shadow-[0_20px_50px_rgba(233,69,96,0.12)]'
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

                  <span class="rounded-full border border-[rgba(126,200,227,0.14)] bg-[rgba(126,200,227,0.08)] px-2.5 py-1 text-[10px] font-semibold uppercase tracking-[0.16em] text-[#8fd7ef]">
                    {{ character.inventory_width }}x{{ character.inventory_height }}
                  </span>
                </div>

                <p class="mt-4 line-clamp-3 text-[14px] leading-relaxed text-[#d8dce7]/68">
                  {{ character.backstory || 'Blank dossier. Select this character to continue editing and session play.' }}
                </p>

                <div class="mt-5 flex flex-wrap gap-2 text-[11px] text-[#d8dce7]/60">
                  <span
                    v-for="attribute in character.custom_attributes"
                    :key="attribute.id"
                    class="rounded-full border border-[rgba(233,69,96,0.16)] bg-[rgba(233,69,96,0.08)] px-2.5 py-1"
                  >
                    {{ attribute.name }}: {{ attribute.value }}
                  </span>
                  <span v-if="!character.custom_attributes.length" class="rounded-full border border-[rgba(126,200,227,0.14)] px-2.5 py-1">No custom stats</span>
                </div>

                <div class="mt-6 flex flex-wrap gap-3">
                  <button
                    @click="switchCharacter(character.id, { nextTab: activeTab })"
                    class="cursor-pointer rounded-xl border border-[rgba(233,69,96,0.24)] bg-[rgba(233,69,96,0.12)] px-4 py-2.5 text-[13px] font-semibold text-[#ffe0e7] transition-all duration-200 hover:border-[rgba(233,69,96,0.45)]"
                  >
                    Make Active
                  </button>
                  <button
                    @click="switchCharacter(character.id, { nextTab: 'sheet' })"
                    class="cursor-pointer rounded-xl border border-[rgba(126,200,227,0.16)] bg-[rgba(126,200,227,0.08)] px-4 py-2.5 text-[13px] font-semibold text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(126,200,227,0.3)]"
                  >
                    Open Character Tab
                  </button>
                </div>
              </article>

              <article v-if="!characters.length" class="md:col-span-2 2xl:col-span-3 rounded-[2rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(11,20,36,0.88)] p-8 text-center">
                <h3 class="font-[Cinzel] text-[24px] font-bold text-[#f6f7fb]">No characters available</h3>
                <p class="mt-3 text-[14px] text-[#d8dce7]/58">Create a new empty character with a random name to enter the session.</p>
              </article>
            </div>
          </section>

          <section v-else-if="activeTab === 'players' && isGM" class="grid gap-6 xl:grid-cols-[minmax(0,1.1fr)_340px]">
            <article class="rounded-[2rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(11,20,36,0.88)] p-5 shadow-[0_24px_60px_rgba(0,0,0,0.22)] sm:p-6">
              <div class="flex items-center justify-between gap-4">
                <div>
                  <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Player Manager</p>
                  <h3 class="mt-2 font-[Cinzel] text-[30px] font-bold text-[#f6f7fb]">Campaign Roster</h3>
                </div>
                <span class="rounded-full border border-[rgba(126,200,227,0.14)] bg-[rgba(126,200,227,0.08)] px-3 py-1.5 text-[11px] uppercase tracking-[0.16em] text-[#8fd7ef]">
                  {{ game.members?.length ?? 0 }} members
                </span>
              </div>

              <div class="mt-6 space-y-3">
                <div
                  v-for="member in game.members"
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
                      <span class="rounded-full border border-[rgba(126,200,227,0.14)] px-2.5 py-1">{{ memberCharacterCount[member.user_id] ?? 0 }} characters</span>
                      <span class="rounded-full border border-[rgba(233,69,96,0.16)] px-2.5 py-1">{{ charactersForUser(member.user_id).length ? 'Ready' : 'No characters' }}</span>
                    </div>
                  </div>

                  <div class="mt-4 flex flex-wrap gap-2">
                    <button
                      v-for="character in charactersForUser(member.user_id)"
                      :key="character.id"
                      @click="switchCharacter(character.id, { nextTab: 'sheet' })"
                      class="cursor-pointer rounded-full border border-[rgba(126,200,227,0.14)] bg-[rgba(7,17,31,0.66)] px-3 py-1.5 text-[12px] text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(233,69,96,0.35)]"
                    >
                      Inspect {{ character.name }}
                    </button>
                  </div>
                </div>
              </div>
            </article>

            <aside class="space-y-5">
              <article class="rounded-[2rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(11,20,36,0.88)] p-5 shadow-[0_24px_60px_rgba(0,0,0,0.22)]">
                <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">GM Workflow</p>
                <p class="mt-4 text-[14px] leading-relaxed text-[#d8dce7]/62">Jump into any accessible character, then move directly to items or the chat dock without changing route.</p>
              </article>

              <article class="rounded-[2rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(11,20,36,0.88)] p-5 shadow-[0_24px_60px_rgba(0,0,0,0.22)]">
                <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Roster Scope</p>
                <p class="mt-4 text-[14px] leading-relaxed text-[#d8dce7]/62">GM views include all session members and their currently accessible characters.</p>
              </article>
            </aside>
          </section>

          <SessionItemManager v-else-if="activeTab === 'items' && isGM" :items="items" :game-id="gameId" />
        </main>
      </div>

      <button
        v-if="chatCollapsed"
        @click="focusChatInput"
        class="session-chat-launcher"
        :class="game.enable_chat
          ? 'border-[rgba(197,138,56,0.24)] bg-[rgba(16,19,23,0.96)] text-[#f3ead9]'
          : 'border-[rgba(143,79,51,0.34)] bg-[rgba(42,22,18,0.92)] text-[#f5d2c8]'"
      >
        <svg class="h-5 w-5" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" d="M8.625 9.75h6.75m-6.75 3h4.5m-8.006 6.375h13.761A2.625 2.625 0 0021.5 16.5v-9A2.625 2.625 0 0018.875 4.875H5.125A2.625 2.625 0 002.5 7.5v9a2.625 2.625 0 002.625 2.625z" />
        </svg>
        <span class="text-[11px] font-semibold uppercase tracking-[0.18em]">Chat</span>
        <span class="rounded-full border border-[rgba(126,200,227,0.14)] px-2 py-0.5 text-[10px] font-semibold">{{ chatMessages.length }}</span>
      </button>

      <aside v-else class="session-chat-dock">
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
                <svg class="h-5 w-5" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M9.75 5.25L16.5 12l-6.75 6.75" />
                </svg>
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
                    <p class="mt-2 whitespace-pre-wrap break-words text-[14px] leading-relaxed text-[#e8e8f0]/78">{{ message.content }}</p>
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
    position: sticky;
    top: 7.35rem;
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