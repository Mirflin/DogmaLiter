<script setup>
import { useAuthStore } from '@/stores/auth'
import { API_URL } from '@/api'
import { ref, watch, computed, onUnmounted } from 'vue'
import { notify } from '@/notify'
import { X as XIcon } from '@lucide/vue'

const props = defineProps({
  visible: Boolean,
  gameId: String,
  // Rendered in place (the session's Table tab) instead of as a dialog.
  inline: Boolean,
})
const emit = defineEmits(['close', 'updated', 'deleted'])

const auth = useAuthStore()

const loading = ref(true)
const saving = ref(false)
const loadError = ref('')
const game = ref(null)

const STANDARD_ATTR_DEFS = [
  { key: 'strength', label: 'Strength' },
  { key: 'dexterity', label: 'Dexterity' },
  { key: 'constitution', label: 'Constitution' },
  { key: 'intelligence', label: 'Intelligence' },
  { key: 'wisdom', label: 'Wisdom' },
  { key: 'charisma', label: 'Charisma' },
]

const title = ref('')
const description = ref('')
const maxPlayers = ref(6)
const characterSlotsPerPlayer = ref(5)
const standardAttrs = ref(Object.fromEntries(STANDARD_ATTR_DEFS.map((attr) => [attr.key, true])))
const enableChat = ref(true)
const enableItemTrading = ref(true)
const enableHealth = ref(false)
const enableArmorClass = ref(false)

const inviteCode = ref('')
const inviteExpiresAt = ref(null)
const regenerating = ref(false)
const countdown = ref('')
const codeExpired = ref(false)
let countdownTimer = null

const inviteLinkText = computed(() => `${window.location.origin}/join/${inviteCode.value}`)

const coverFile = ref(null)
const coverPreview = ref(null)
const uploadingCover = ref(false)

const isOwner = computed(() => game.value?.owner_id === auth.user?.id || auth.user?.role === 'admin')

watch(() => props.visible || props.inline, async (val) => {
  if (val && props.gameId) {
    loading.value = true
    loadError.value = ''
    try {
      const data = await auth.getGame(props.gameId)
      game.value = data
      title.value = data.title
      description.value = data.description || ''
      maxPlayers.value = data.max_players
      characterSlotsPerPlayer.value = data.character_slots_per_player ?? 5
      const enabledList = Array.isArray(data.enabled_standard_attrs)
        ? new Set(data.enabled_standard_attrs)
        : null
      for (const attr of STANDARD_ATTR_DEFS) {
        standardAttrs.value[attr.key] = enabledList ? enabledList.has(attr.key) : data.show_standard_attrs !== false
      }
      enableChat.value = data.enable_chat
      enableItemTrading.value = data.enable_item_trading
      enableHealth.value = Boolean(data.enable_health)
      enableArmorClass.value = Boolean(data.enable_armor_class)
      inviteCode.value = data.invite_code || ''
      inviteExpiresAt.value = data.invite_code_expires_at || null
      coverPreview.value = data.cover_image_id ? `${API_URL}/api/uploads/${data.cover_image_id}` : null
      coverFile.value = null
      startCountdown()
    } catch {
      loadError.value = 'Failed to load game settings.'
      notify.error({
        title: 'Unable to open settings',
        message: loadError.value,
      })
    } finally {
      loading.value = false
    }
  } else {
    loadError.value = ''
    stopCountdown()
  }
}, { immediate: true })

onUnmounted(() => stopCountdown())

function startCountdown() {
  stopCountdown()
  updateCountdown()
  countdownTimer = setInterval(updateCountdown, 1000)
}

function stopCountdown() {
  if (countdownTimer) {
    clearInterval(countdownTimer)
    countdownTimer = null
  }
}

function updateCountdown() {
  if (!inviteExpiresAt.value) {
    countdown.value = ''
    codeExpired.value = true
    return
  }
  const now = Date.now()
  const exp = new Date(inviteExpiresAt.value).getTime()
  const diff = exp - now
  if (diff <= 0) {
    countdown.value = 'Expired'
    codeExpired.value = true
    stopCountdown()
    return
  }
  codeExpired.value = false
  const m = Math.floor(diff / 60000)
  const s = Math.floor((diff % 60000) / 1000)
  countdown.value = `${m}:${String(s).padStart(2, '0')}`
}

async function regenerate() {
  regenerating.value = true
  try {
    const data = await auth.regenerateInviteCode(props.gameId)
    inviteCode.value = data.invite_code
    inviteExpiresAt.value = data.invite_code_expires_at
    codeExpired.value = false
    startCountdown()
    notify.success({
      title: 'Invite code refreshed',
      message: 'A new invite code is ready to share.',
    })
  } catch (err) {
    notify.error(err, 'Failed to regenerate code')
  } finally {
    regenerating.value = false
  }
}

async function copyInviteLink() {
  const text = inviteLinkText.value
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text)
    } else {
      const el = document.createElement('textarea')
      el.value = text
      el.style.position = 'fixed'
      el.style.opacity = '0'
      document.body.appendChild(el)
      el.select()
      document.execCommand('copy')
      document.body.removeChild(el)
    }

    notify.success({
      title: 'Invite link copied',
      message: 'Share it with your players.',
    })
  } catch {
    notify.error('Failed to copy invite link')
  }
}

async function handleSave() {
  saving.value = true
  try {
    await auth.updateGame(props.gameId, {
      title: title.value,
      description: description.value,
      max_players: maxPlayers.value,
      character_slots_per_player: characterSlotsPerPlayer.value,
      enabled_standard_attrs: STANDARD_ATTR_DEFS.filter((attr) => standardAttrs.value[attr.key]).map((attr) => attr.key),
      enable_chat: enableChat.value,
      enable_item_trading: enableItemTrading.value,
      enable_health: enableHealth.value,
      enable_armor_class: enableArmorClass.value,
    })

    if (coverFile.value) {
      await auth.uploadCoverImage(props.gameId, coverFile.value)
    }

    notify.success({
      title: 'Settings saved',
      message: 'Game settings were updated.',
    })
    emit('updated')
    if (props.inline) notify.success({ title: 'Settings saved' })
    else emit('close')
  } catch (err) {
    notify.error(err, 'Failed to save')
  } finally {
    saving.value = false
  }
}

function handleCoverSelect(e) {
  const file = e.target.files?.[0]
  if (!file) return
  coverFile.value = file
  coverPreview.value = URL.createObjectURL(file)
}

function removeCover() {
  coverFile.value = null
  coverPreview.value = game.value?.cover_image_id ? `${API_URL}/api/uploads/${game.value.cover_image_id}` : null
}

async function handleDelete() {
  if (!confirm(`Delete "${title.value}"? This cannot be undone.`)) return
  try {
    await auth.deleteGame(props.gameId)
    notify.success({
      title: 'Game deleted',
      message: `"${title.value}" was removed.`,
    })
    emit('deleted', props.gameId)
    emit('close')
  } catch (err) {
    notify.error(err, 'Failed to delete')
  }
}

function close() {
  if (!saving.value) emit('close')
}
</script>

<template>
  <Teleport to="body" :disabled="inline">
    <div v-if="visible || inline" :class="inline ? 'gs gs--inline' : 'gs fixed inset-0 z-[10000] flex items-center justify-center'" @click.self="close">
      <div v-if="!inline" class="absolute inset-0 bg-black/60 backdrop-blur-sm" @click="close"></div>
      <div :class="inline ? 'gs__panel gs__panel--inline' : 'gs__panel relative w-full max-w-[640px] max-h-[90vh] overflow-y-auto mx-4'">
        <button v-if="!inline" @click="close" aria-label="Close settings" class="absolute top-4 right-4 flex h-9 w-9 items-center justify-center rounded-xl bg-n3 text-ink hover:bg-n4 cursor-pointer transition-colors duration-200"><XIcon class="h-5 w-5" :stroke-width="2" /></button>

        <h2 v-if="!inline" class="text-[24px] font-bold text-ink mb-6">Game settings</h2>

        <div v-if="loading" class="text-center py-12">
          <div class="w-8 h-8 border-2 border-primary border-t-transparent rounded-full animate-spin mx-auto"></div>
        </div>

        <div v-else-if="loadError" class="text-center py-8">
          <p class="text-danger text-[14px]">{{ loadError }}</p>
        </div>

        <div v-else-if="!isOwner" class="text-center py-8">
          <p class="text-faint text-[14px]">Only the game owner can change these settings.</p>
        </div>

        <div v-else class="gs__sections">
          <section class="gs__section">
            <div class="gs__head"><h3>The table</h3><p>How the game looks in lists and who can join.</p></div>
            <div class="gs__body">
              <div class="gs-cover">
                <div class="gs-cover__thumb">
                  <img v-if="coverPreview" :src="coverPreview" alt="" />
                  <span v-else>No cover</span>
                </div>
                <label class="gs-btn">
                  {{ coverPreview ? 'Change cover' : 'Add cover' }}
                  <input type="file" accept="image/jpeg,image/png,image/webp" class="hidden" @change="handleCoverSelect" />
                </label>
                <button v-if="coverFile" type="button" class="gs-link" @click="removeCover">Undo</button>
              </div>
              <label class="gs-field"><span>Title</span><input v-model="title" type="text" maxlength="200" /></label>
              <label class="gs-field"><span>Description</span><textarea v-model="description" rows="3"></textarea></label>
              <div class="gs-pair">
                <label class="gs-field">
                  <span>Max players</span>
                  <input v-model.number="maxPlayers" type="number" min="1" :max="auth.user?.max_players_per_game === -1 ? undefined : (auth.user?.max_players_per_game || 50)" />
                  <small>{{ auth.user?.max_players_per_game === -1 ? 'Unlimited on your plan' : `Up to ${auth.user?.max_players_per_game || '?'} on your plan` }}</small>
                </label>
                <label class="gs-field">
                  <span>Characters per player</span>
                  <input v-model.number="characterSlotsPerPlayer" type="number" min="1" max="50" />
                  <small>How many characters each player can own</small>
                </label>
              </div>
            </div>
          </section>

          <section class="gs__section">
            <div class="gs__head"><h3>Invite</h3><p>Players join with this code or link.</p></div>
            <div class="gs__body">
              <div class="gs-invite">
                <span class="gs-invite__code num" :class="{ 'is-expired': codeExpired }">{{ inviteCode || '—' }}</span>
                <span v-if="countdown" class="gs-invite__ttl" :class="{ 'text-danger': codeExpired }">{{ countdown }}</span>
              </div>
              <div class="flex flex-wrap gap-2">
                <button v-if="!codeExpired && inviteCode" type="button" class="gs-btn gs-btn--primary" @click="copyInviteLink">Copy invite link</button>
                <button type="button" class="gs-btn" :disabled="regenerating" @click="regenerate">{{ regenerating ? 'Generating…' : 'New code' }}</button>
              </div>
            </div>
          </section>

          <section class="gs__section">
            <div class="gs__head"><h3>Character sheet</h3><p>What every sheet, editor and item in this game tracks.</p></div>
            <div class="gs__body">
              <p class="gs-sub">Base attributes</p>
              <div class="gs-chips">
                <label v-for="attr in STANDARD_ATTR_DEFS" :key="attr.key" class="gs-chip" :class="{ 'is-on': standardAttrs[attr.key] }">
                  <input v-model="standardAttrs[attr.key]" type="checkbox" class="sr-only" />
                  {{ attr.label }}
                </label>
              </div>
              <label class="gs-toggle">
                <span class="gs-toggle__text"><strong>Health</strong><span>Current and maximum health on every sheet</span></span>
                <input v-model="enableHealth" type="checkbox" class="gs-switch peer" role="switch" />
              </label>
              <label class="gs-toggle">
                <span class="gs-toggle__text"><strong>Armor class</strong><span>A single armor class value per character</span></span>
                <input v-model="enableArmorClass" type="checkbox" class="gs-switch peer" role="switch" />
              </label>
            </div>
          </section>

          <section class="gs__section">
            <div class="gs__head"><h3>Features</h3><p>Switch parts of the session on or off.</p></div>
            <div class="gs__body">
              <label class="gs-toggle">
                <span class="gs-toggle__text"><strong>Party chat</strong><span>Messages and shared items beside the session</span></span>
                <input v-model="enableChat" type="checkbox" class="gs-switch peer" role="switch" />
              </label>
            </div>
          </section>

          <div class="gs__footer">
            <button type="button" class="gs-btn gs-btn--danger" @click="handleDelete">Delete game</button>
            <div class="flex gap-2">
              <button v-if="!inline" type="button" class="gs-btn" @click="close">Cancel</button>
              <button type="button" class="gs-btn gs-btn--primary" :disabled="saving" @click="handleSave">{{ saving ? 'Saving…' : 'Save changes' }}</button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.gs__panel {
  padding: 2rem;
  border-radius: 1.25rem;
  background: var(--color-n1);
  box-shadow: inset 0 0 0 1px rgb(74 90 150 / 0.35), 0 24px 60px -20px rgb(0 0 0 / 0.7);
}

.gs__panel--inline {
  padding: 1.5rem;
  background: rgb(8 13 36 / 0.82);
  box-shadow: inset 0 0 0 1px rgb(74 90 150 / 0.35);
}

.gs__sections {
  display: grid;
  gap: 0;
}

/* Each section: what it is on the left, its controls on the right. */
.gs__section {
  display: grid;
  grid-template-columns: minmax(0, 14rem) minmax(0, 1fr);
  gap: 1rem 2.5rem;
  padding: 1.5rem 0;
  border-top: 1px solid var(--color-n4);
}

.gs__section:first-child {
  padding-top: 0;
  border-top: 0;
}

.gs--inline .gs__section {
  grid-template-columns: minmax(0, 16rem) minmax(0, 36rem);
}

.gs:not(.gs--inline) .gs__section {
  grid-template-columns: minmax(0, 1fr);
}

.gs__head h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  color: var(--color-ink);
}

.gs__head p {
  margin: 0.3rem 0 0;
  font-size: 14px;
  line-height: 1.5;
  color: var(--color-dim);
}

.gs__body {
  display: grid;
  gap: 1rem;
}

.gs-field {
  display: grid;
  gap: 0.4rem;
}

.gs-field > span,
.gs-sub {
  margin: 0;
  font-size: 14px;
  font-weight: 500;
  color: var(--color-ink-soft);
}

.gs-field small {
  font-size: 13px;
  color: var(--color-faint);
}

.gs-field input,
.gs-field textarea {
  width: 100%;
  padding: 0.7rem 0.9rem;
  border: 1px solid var(--color-n5);
  border-radius: 0.7rem;
  background: var(--color-n0);
  color: var(--color-ink);
  font: inherit;
  font-size: 15px;
  outline: none;
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
}

.gs-field textarea {
  resize: vertical;
}

.gs-field input:focus,
.gs-field textarea:focus {
  border-color: var(--color-primary-ink);
  box-shadow: 0 0 0 3px rgb(164 174 255 / 0.18);
}

.gs-pair {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
}

.gs-cover {
  display: flex;
  align-items: center;
  gap: 0.9rem;
}

.gs-cover__thumb {
  display: grid;
  place-items: center;
  width: 6rem;
  height: 4rem;
  overflow: hidden;
  border-radius: 0.6rem;
  background: var(--color-n2);
  font-size: 12px;
  color: var(--color-faint);
}

.gs-cover__thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.gs-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 2.5rem;
  padding: 0 1rem;
  border: 0;
  border-radius: 0.7rem;
  background: var(--color-n3);
  font: inherit;
  font-size: 14px;
  font-weight: 600;
  color: var(--color-ink);
  cursor: pointer;
  transition: background-color 0.2s ease;
}

.gs-btn:hover {
  background: var(--color-n4);
}

.gs-btn:disabled {
  opacity: 0.6;
  cursor: progress;
}

.gs-btn--primary {
  background: var(--color-primary);
  color: #fff;
}

.gs-btn--primary:hover {
  background: var(--color-primary-hover);
}

.gs-btn--danger {
  background: transparent;
  color: var(--color-danger);
  box-shadow: inset 0 0 0 1px rgb(242 122 133 / 0.4);
}

.gs-btn--danger:hover {
  background: rgb(242 122 133 / 0.12);
}

.gs-link {
  border: 0;
  background: none;
  font: inherit;
  font-size: 14px;
  color: var(--color-dim);
  text-decoration: underline;
  text-underline-offset: 3px;
  cursor: pointer;
}

.gs-invite {
  display: flex;
  align-items: baseline;
  gap: 0.9rem;
}

.gs-invite__code {
  font-size: 28px;
  font-weight: 700;
  letter-spacing: 0.12em;
  color: var(--color-ink);
}

.gs-invite__code.is-expired {
  color: var(--color-faint);
  text-decoration: line-through;
}

.gs-invite__ttl {
  font-size: 14px;
  color: var(--color-dim);
}

.gs-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem;
}

.gs-chip {
  padding: 0.45rem 0.85rem;
  border-radius: 999px;
  font-size: 14px;
  font-weight: 500;
  color: var(--color-dim);
  background: var(--color-n2);
  box-shadow: inset 0 0 0 1px var(--color-n5);
  cursor: pointer;
  transition: background-color 0.15s ease, color 0.15s ease, box-shadow 0.15s ease;
}

.gs-chip.is-on {
  color: var(--color-ink);
  background: rgb(88 101 221 / 0.22);
  box-shadow: inset 0 0 0 1px rgb(164 174 255 / 0.45);
}

.gs-chip:has(input:focus-visible) {
  outline: 2px solid var(--color-primary-ink);
  outline-offset: 2px;
}

.gs-toggle {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.8rem 1rem;
  border-radius: 0.8rem;
  background: var(--color-n2);
  cursor: pointer;
}

.gs-toggle__text {
  display: grid;
  gap: 0.15rem;
}

.gs-toggle__text strong {
  font-size: 15px;
  font-weight: 600;
  color: var(--color-ink);
}

.gs-toggle__text span {
  font-size: 13px;
  color: var(--color-dim);
}

/* A real switch, built on the checkbox so keyboard and screen readers keep working. */
.gs .gs-switch {
  appearance: none;
  position: relative;
  width: 2.6rem;
  height: 1.5rem;
  flex-shrink: 0;
  border: 0 !important;
  border-radius: 999px;
  background: var(--color-n5) !important;
  cursor: pointer;
  transition: background-color 0.2s ease;
}

.gs .gs-switch::after {
  content: '';
  position: absolute;
  top: 0.2rem;
  left: 0.2rem;
  width: 1.1rem;
  height: 1.1rem;
  border: 0;
  border-radius: 50%;
  background: #fff;
  transform: none;
  transition: transform 0.2s cubic-bezier(0.16, 1, 0.3, 1);
}

.gs .gs-switch:checked {
  background: var(--color-primary) !important;
}

.gs .gs-switch:checked::after {
  transform: translateX(1.1rem);
}

.gs__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding-top: 1.5rem;
  border-top: 1px solid var(--color-n4);
}

@media (max-width: 760px) {
  .gs--inline .gs__section {
    grid-template-columns: minmax(0, 1fr);
  }

  .gs-pair {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
