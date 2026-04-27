<script setup>
import { API_URL } from '@/api'
import { computed } from 'vue'

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
  errorMessage: {
    type: String,
    default: '',
  },
})

const emit = defineEmits(['select', 'create'])

const remainingSlots = computed(() => {
  if (props.viewer?.character_limit < 0) return null
  return Math.max((props.viewer?.character_limit ?? 0) - (props.viewer?.owned_character_count ?? 0), 0)
})

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
      <div class="absolute inset-0 bg-[rgba(5,8,12,0.82)] backdrop-blur-md"></div>

      <div class="session-picker-shell relative w-full max-w-[960px] rounded-[2rem] border border-[rgba(126,200,227,0.14)] bg-[linear-gradient(180deg,rgba(9,18,34,0.98),rgba(5,10,22,0.98))] p-5 shadow-[0_40px_120px_rgba(0,0,0,0.52)] sm:p-7">
        <div class="flex flex-wrap items-center justify-between gap-4">
          <div>
            <p class="text-[11px] uppercase tracking-[0.24em] text-[#7ec8e3]/58">Choose Character</p>
            <h2 class="mt-2 font-[Cinzel] text-[28px] font-bold text-[#f6f7fb] sm:text-[34px]">Session Entry</h2>
          </div>

        </div>

        <div v-if="errorMessage" class="mt-5 rounded-[1rem] border border-[rgba(143,79,51,0.34)] bg-[rgba(52,25,20,0.76)] px-4 py-3 text-[13px] text-[#f4d2c3]">
          {{ errorMessage }}
        </div>

        <div v-if="characters.length" class="session-picker-scroll mt-6 max-h-[62vh] overflow-y-auto pr-1">
          <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
            <button
              v-for="character in characters"
              :key="character.id"
              @click="selectCharacter(character.id)"
              class="session-picker-card cursor-pointer rounded-[1.8rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(11,20,36,0.84)] p-4 text-left transition-all duration-200 hover:-translate-y-1 hover:border-[rgba(233,69,96,0.32)] hover:shadow-[0_20px_50px_rgba(233,69,96,0.12)]"
            >
              <div class="overflow-hidden rounded-[1rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(126,200,227,0.06)]">
                <div class="flex h-56 items-center justify-center bg-[linear-gradient(180deg,rgba(16,32,52,0.94),rgba(8,16,30,0.94))] sm:h-64">
                  <img
                    v-if="avatarUrl(character.portrait_id)"
                    :src="avatarUrl(character.portrait_id)"
                    :alt="character.name"
                    class="h-full w-full object-cover object-top"
                  />
                  <span v-else class="font-[Cinzel] text-[32px] font-bold text-[#d9b574]">{{ initials(character.name) }}</span>
                </div>
              </div>

              <div class="mt-4 flex items-start justify-between gap-3">
                <div class="min-w-0">
                  <p class="truncate font-[Cinzel] text-[24px] font-bold text-[#f6f7fb]">{{ character.name }}</p>
                  <p class="mt-1 text-[13px] text-[#b8c8cc]/58">{{ character.owner?.username || 'Unassigned' }}</p>
                </div>

                <span class="rounded-full border border-[rgba(126,200,227,0.16)] bg-[rgba(126,200,227,0.08)] px-2.5 py-1 text-[10px] font-semibold uppercase tracking-[0.16em] text-[#dff4fb]">
                  {{ character.inventory_width }}x{{ character.inventory_height }}
                </span>
              </div>
            </button>
          </div>
        </div>

        <div v-else class="session-picker-empty mt-6 rounded-[1.8rem] border border-dashed border-[rgba(126,200,227,0.18)] bg-[rgba(126,200,227,0.04)] px-5 py-10 text-center">
          <h3 class="font-[Cinzel] text-[26px] font-bold text-[#f6f7fb]">No Characters Yet</h3>
        </div>

        <div class="mt-6 flex flex-wrap items-center justify-between gap-4 border-t border-[rgba(126,200,227,0.12)] pt-5">
          <p class="text-[12px] text-[#d8dce7]/58">
            {{ viewer?.can_create_character
              ? (viewer?.is_gm ? 'Create a new blank character.' : `${remainingSlots ?? 0} slots available for a new character.`)
              : 'Character creation is locked right now.' }}
          </p>

          <button
            @click="createCharacter"
            :disabled="creating || !viewer?.can_create_character"
            class="session-picker-action cursor-pointer rounded-xl border border-[rgba(233,69,96,0.24)] bg-[rgba(233,69,96,0.12)] px-5 py-3 text-[14px] font-semibold text-[#ffe0e7] transition-all duration-200 hover:border-[rgba(233,69,96,0.42)] disabled:cursor-not-allowed disabled:opacity-60"
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
  border: 1px solid rgba(126, 200, 227, 0.08);
  pointer-events: none;
}

.session-picker-shell {
  overflow: hidden;
  border-radius: 0.85rem !important;
  border-color: rgba(126, 200, 227, 0.18) !important;
  background: linear-gradient(180deg, rgba(13, 18, 33, 0.98), rgba(8, 12, 24, 0.99)) !important;
  box-shadow: 0 40px 120px rgba(0, 0, 0, 0.52), inset 0 1px 0 rgba(255, 255, 255, 0.04) !important;
}

.session-picker-shell::before {
  content: '';
  position: absolute;
  inset: 10px;
  border: 1px solid rgba(126, 200, 227, 0.12);
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
  background: rgba(126, 200, 227, 0.24);
  border-radius: 999px;
}

.session-picker-scroll::-webkit-scrollbar-track {
  background: transparent;
}

.session-picker-card {
  position: relative;
  overflow: hidden;
  border-radius: 0.7rem !important;
  border-color: rgba(126, 200, 227, 0.16) !important;
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
  border-color: rgba(233, 69, 96, 0.28) !important;
  box-shadow: 0 26px 60px rgba(0, 0, 0, 0.28), inset 0 1px 0 rgba(255, 255, 255, 0.04) !important;
}

.session-picker-empty {
  border-color: rgba(126, 200, 227, 0.18) !important;
  background:
    repeating-linear-gradient(90deg, rgba(92, 121, 130, 0.1) 0 1px, transparent 1px 56px),
    repeating-linear-gradient(0deg, rgba(92, 121, 130, 0.1) 0 1px, transparent 1px 56px),
    linear-gradient(180deg, rgba(13, 18, 33, 0.88), rgba(8, 12, 24, 0.94)) !important;
}

.session-picker-action {
  border-color: rgba(233, 69, 96, 0.28) !important;
  background: linear-gradient(135deg, rgba(233, 69, 96, 0.9), rgba(194, 49, 82, 0.92)) !important;
  color: #ffffff !important;
  border-radius: 0.55rem !important;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.04), 0 12px 24px rgba(233, 69, 96, 0.18);
}

.session-picker-action:hover {
  border-color: rgba(233, 69, 96, 0.42) !important;
  color: #ffffff !important;
}
</style>
