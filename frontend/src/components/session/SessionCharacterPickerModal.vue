<script setup>
import { API_URL } from '@/api'
import { useImageFallback } from '@/composables/useImageFallback'
import { computed, ref, watch } from 'vue'

const props = defineProps({
  visible: {
    type: Boolean,
    default: false,
  },
  characters: {
    type: Array,
    default: () => [],
  },
  viewer: {
    type: Object,
    default: () => ({
      is_gm: false,
      can_create_character: false,
      character_limit: 0,
      owned_character_count: 0,
    }),
  },
  creating: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits(['select', 'create'])

const remainingSlots = computed(() => {
  if (props.viewer?.character_limit < 0) return null
  return Math.max((props.viewer?.character_limit ?? 0) - (props.viewer?.owned_character_count ?? 0), 0)
})

const searchQuery = ref('')
const filteredCharacters = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return props.characters
  return props.characters.filter((character) => {
    const name = String(character?.name || '').toLowerCase()
    const owner = String(character?.owner?.username || '').toLowerCase()
    return name.includes(q) || owner.includes(q)
  })
})

watch(() => props.visible, (visible) => {
  if (visible) searchQuery.value = ''
})

const { markFailed: markImageFailed, isFailed: isImageFailed } = useImageFallback()

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

function selectCharacter(characterId) {
  emit('select', characterId)
}

function createCharacter() {
  emit('create')
}
</script>

<template>
  <Teleport to="body">
    <div v-if="visible" class="session-picker fixed inset-0 z-[12000] flex items-center justify-center px-4 py-6 sm:py-8">
      <div class="absolute inset-0 bg-n0/82 backdrop-blur-md"></div>

      <div class="session-picker-shell relative w-full max-w-[960px] rounded-[2rem] border border-line bg-[linear-gradient(180deg,rgba(10,16,41,0.98),rgba(6,10,28,0.98))] p-5 shadow-[0_40px_120px_rgba(0,0,0,0.52)] sm:p-7">
        <div class="flex flex-wrap items-center justify-between gap-4">
          <div>
            <p class="text-[11px] uppercase tracking-[0.24em] text-dim">Choose Character</p>
            <h2 class="mt-2 font-heading text-[28px] font-bold text-ink sm:text-[34px]">Session Entry</h2>
          </div>

          <input
            v-if="characters.length"
            v-model="searchQuery"
            type="text"
            placeholder="Search by name or owner…"
            class="session-input w-full max-w-[280px] rounded-xl border border-line bg-n1/72 px-4 py-2.5 text-[14px] text-ink outline-none transition-colors focus:border-primary/40"
          />
        </div>

        <div v-if="filteredCharacters.length" class="session-picker-scroll mt-6 max-h-[62vh] overflow-y-auto pr-1">
          <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
            <button
              v-for="character in filteredCharacters"
              :key="character.id"
              @click="selectCharacter(character.id)"
              class="session-picker-card cursor-pointer rounded-[1.8rem] border border-line bg-n1/84 p-4 text-left transition-all duration-200 hover:-translate-y-1 hover:border-primary/32"
            >
              <div class="overflow-hidden rounded-[1rem] border border-line bg-n3">
                <div class="flex h-56 items-center justify-center bg-[linear-gradient(180deg,rgba(16,32,52,0.94),rgba(10,16,41,0.94))] sm:h-64">
                  <img
                    v-if="avatarUrl(character.portrait_id)"
                    :src="avatarUrl(character.portrait_id)"
                    :alt="character.name"
                    class="h-full w-full object-cover object-top"
                    @error="markImageFailed(avatarUrl(character.portrait_id))"
                  />
                  <div v-else class="flex h-24 w-24 items-center justify-center rounded-full border-2 border-[rgba(217,181,116,0.3)] bg-[linear-gradient(135deg,rgba(217,181,116,0.16),rgba(123,135,184,0.06))] font-heading text-[32px] font-bold text-warning">{{ initials(character.name) }}</div>
                </div>
              </div>

              <div class="mt-4 flex items-start justify-between gap-3">
                <div class="min-w-0">
                  <p class="truncate font-heading text-[24px] font-bold text-ink">{{ character.name }}</p>
                  <p class="mt-1 text-[13px] text-dim/58">{{ character.owner?.username || 'Unassigned' }}</p>
                </div>

                <span class="rounded-full border border-line bg-n3 px-2.5 py-1 text-[10px] font-semibold uppercase tracking-[0.16em] text-ink">
                  {{ character.inventory_width }}x{{ character.inventory_height }}
                </span>
              </div>
            </button>
          </div>
        </div>

        <div v-else-if="characters.length" class="mt-6 rounded-[1.8rem] border border-dashed border-line bg-n2 px-5 py-10 text-center">
          <p class="text-[15px] text-ink-soft/60">No characters match “{{ searchQuery }}”.</p>
        </div>

        <div v-else class="session-picker-empty mt-6 rounded-[1.8rem] border border-dashed border-line bg-n2 px-5 py-10 text-center">
          <h3 class="font-heading text-[26px] font-bold text-ink">No Characters Yet</h3>
        </div>

        <div class="mt-6 flex flex-wrap items-center justify-between gap-4 border-t border-line pt-5">
          <p class="text-[12px] text-ink-soft/58">
            {{ viewer?.can_create_character
              ? (viewer?.is_gm ? 'Create a new blank character.' : `${remainingSlots ?? 0} slots available for a new character.`)
              : 'Character creation is locked right now.' }}
          </p>

          <button
            @click="createCharacter"
            :disabled="creating || !viewer?.can_create_character"
            class="session-picker-action cursor-pointer rounded-xl border border-primary/24 bg-primary/12 px-5 py-3 text-[14px] font-semibold text-primary-soft transition-all duration-200 hover:border-primary/42 disabled:cursor-not-allowed disabled:opacity-60"
          >
            {{ creating ? 'Creating...' : 'Create Character' }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.session-picker {
  isolation: isolate;
}

.session-picker::before {
  content: '';
  position: absolute;
  inset: 18px;
  border: 1px solid rgba(123, 135, 184, 0.08);
  pointer-events: none;
}

.session-picker-shell {
  overflow: hidden;
  border-radius: 0.85rem !important;
  border-color: rgba(123, 135, 184, 0.18) !important;
  background: linear-gradient(180deg, rgba(13, 18, 33, 0.98), rgba(6, 10, 28, 0.99)) !important;
  box-shadow: 0 40px 120px rgba(0, 0, 0, 0.52), inset 0 1px 0 rgba(255, 255, 255, 0.04) !important;
}

.session-picker-shell::before {
  content: '';
  position: absolute;
  inset: 10px;
  border: 1px solid rgba(123, 135, 184, 0.12);
  pointer-events: none;
}

.session-picker-shell::after {
  content: '';
  position: absolute;
  inset: 0;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.035), transparent 68%),
    repeating-linear-gradient(90deg, rgba(255, 255, 255, 0.03) 0 1px, transparent 1px 72px);
  opacity: 0.36;
  pointer-events: none;
}

.session-picker-shell > * {
  position: relative;
  z-index: 1;
}

.session-picker-scroll::-webkit-scrollbar {
  width: 8px;
}

.session-picker-scroll::-webkit-scrollbar-thumb {
  background: rgba(123, 135, 184, 0.24);
  border-radius: 999px;
}

.session-picker-scroll::-webkit-scrollbar-track {
  background: transparent;
}

.session-picker-card {
  position: relative;
  overflow: hidden;
  border-radius: 0.7rem !important;
  border-color: rgba(123, 135, 184, 0.16) !important;
  background: linear-gradient(180deg, rgba(18, 27, 49, 0.88), rgba(9, 13, 24, 0.96)) !important;
  box-shadow: 0 20px 50px rgba(0, 0, 0, 0.24), inset 0 1px 0 rgba(255, 255, 255, 0.04) !important;
  clip-path: polygon(0 16px, 16px 0, 100% 0, 100% calc(100% - 16px), calc(100% - 16px) 100%, 0 100%);
}

.session-picker-card::before {
  content: '';
  position: absolute;
  inset: 0;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.03), transparent 24%);
  pointer-events: none;
}

.session-picker-card:hover {
  border-color: rgba(88, 101, 221, 0.28) !important;
  box-shadow: 0 26px 60px rgba(0, 0, 0, 0.28), inset 0 1px 0 rgba(255, 255, 255, 0.04) !important;
}

.session-picker-empty {
  border-color: rgba(123, 135, 184, 0.18) !important;
  background:
    repeating-linear-gradient(90deg, rgba(92, 121, 130, 0.1) 0 1px, transparent 1px 56px),
    repeating-linear-gradient(0deg, rgba(92, 121, 130, 0.1) 0 1px, transparent 1px 56px),
    linear-gradient(180deg, rgba(13, 18, 33, 0.88), rgba(6, 10, 28, 0.94)) !important;
}

.session-picker-action {
  border-color: rgba(88, 101, 221, 0.28) !important;
  background: linear-gradient(135deg, rgba(88, 101, 221, 0.9), rgba(74, 86, 200, 0.92)) !important;
  color: #ffffff !important;
  border-radius: 0.55rem !important;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.04), 0 12px 24px rgba(88, 101, 221, 0.18);
}

.session-picker-action:hover {
  border-color: rgba(88, 101, 221, 0.42) !important;
  color: #ffffff !important;
}
</style>
