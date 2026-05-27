<script setup>
import { X as XIcon } from '@lucide/vue'
import { computed, ref, watch } from 'vue'

const props = defineProps({
  visible: {
    type: Boolean,
    default: false,
  },
  saving: {
    type: Boolean,
    default: false,
  },
  error: {
    type: String,
    default: '',
  },
  members: {
    type: Array,
    default: () => [],
  },
  viewerUserId: {
    type: String,
    default: '',
  },
})

const emit = defineEmits(['close', 'save'])

const localError = ref('')
const form = ref(createFormState())

const displayError = computed(() => localError.value || props.error || '')

watch(
  [() => props.visible, () => props.viewerUserId, () => props.members],
  ([visible]) => {
    if (visible) {
      form.value = createFormState()
      localError.value = ''
      return
    }

    localError.value = ''
  },
  { immediate: true },
)

function createFormState() {
  return {
    name: '',
    backstory: '',
    owner_user_id: props.viewerUserId || props.members[0]?.user_id || '',
  }
}

function formatRole(role) {
  if (!role) return 'Player'
  return role
    .split('_')
    .filter(Boolean)
    .map(part => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ')
}

function close() {
  if (!props.saving) {
    emit('close')
  }
}

function emitSave() {
  const name = form.value.name.trim()
  if (!name) {
    localError.value = 'Character name cannot be empty'
    return
  }

  const backstory = form.value.backstory.trim()
  if (backstory.length > 5000) {
    localError.value = 'Backstory must be 5000 characters or less'
    return
  }

  if (!form.value.owner_user_id) {
    localError.value = 'Select who should own this character'
    return
  }

  localError.value = ''
  emit('save', {
    name,
    backstory,
    owner_user_id: form.value.owner_user_id,
  })
}
</script>

<template>
  <Teleport to="body">
    <div v-if="visible" class="fixed inset-0 z-[12600] flex items-center justify-center px-4 py-6 sm:px-6" @click.self="close">
      <div class="absolute inset-0 bg-[rgba(5,8,12,0.82)] backdrop-blur-md" @click="close"></div>

      <div class="relative w-full max-w-[760px] rounded-[2rem] border border-[rgba(126,200,227,0.16)] bg-[linear-gradient(180deg,rgba(9,18,34,0.98),rgba(5,10,22,0.98))] p-5 shadow-[0_40px_120px_rgba(0,0,0,0.52)] sm:p-7">
        <button
          @click="close"
          :disabled="saving"
          class="absolute right-5 top-5 flex h-10 w-10 cursor-pointer items-center justify-center rounded-xl border border-[rgba(126,200,227,0.14)] bg-[rgba(126,200,227,0.08)] text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(233,69,96,0.32)] disabled:cursor-not-allowed disabled:opacity-60"
          aria-label="Close GM character creation modal"
        >
          <XIcon class="h-5 w-5" :stroke-width="2" />
        </button>

        <div class="pr-12">
          <p class="text-[11px] uppercase tracking-[0.24em] text-[#7ec8e3]/58">GM Character Creation</p>
          <h2 class="mt-3 font-[Cinzel] text-[28px] font-bold text-[#f6f7fb] sm:text-[34px]">Create Named Character</h2>
          <p class="mt-3 max-w-[40rem] text-[14px] leading-relaxed text-[#d8dce7]/62">
            Create a GM-owned character for your own vault or assign it directly to another member.
          </p>
        </div>

        <p v-if="displayError" class="mt-6 rounded-[1.3rem] border border-[rgba(248,113,113,0.24)] bg-[rgba(127,29,29,0.18)] px-4 py-3 text-[13px] text-[#fecaca]">
          {{ displayError }}
        </p>

        <div class="mt-6 grid gap-5 lg:grid-cols-[minmax(0,1.1fr)_minmax(260px,0.9fr)]">
          <section class="space-y-5 rounded-[1.75rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(8,16,30,0.78)] p-5 sm:p-6">
            <label class="block">
              <span class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Name</span>
              <input
                v-model="form.name"
                type="text"
                maxlength="100"
                :disabled="saving"
                class="session-input mt-2 w-full rounded-[1.25rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.72)] px-4 py-3 text-[15px] text-[#f6f7fb] outline-none transition-all duration-200 placeholder:text-[#7ec8e3]/35 focus:border-[rgba(233,69,96,0.34)] disabled:cursor-not-allowed disabled:opacity-60"
                placeholder="Captain Elara Voss"
              />
            </label>

            <label class="block">
              <span class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Backstory</span>
              <textarea
                v-model="form.backstory"
                rows="8"
                :disabled="saving"
                class="session-input mt-2 w-full resize-none rounded-[1.25rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.72)] px-4 py-3 text-[15px] text-[#f6f7fb] outline-none transition-all duration-200 placeholder:text-[#7ec8e3]/35 focus:border-[rgba(233,69,96,0.34)] disabled:cursor-not-allowed disabled:opacity-60"
                placeholder="Outline the character role, tone, or campaign purpose."
              ></textarea>
            </label>
          </section>

          <aside class="rounded-[1.75rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(8,16,30,0.78)] p-5">
            <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Assigned To</p>
            <label class="mt-4 block">
              <span class="text-[12px] text-[#d8dce7]/52">Choose the member who will own the character immediately after creation.</span>
              <select
                v-model="form.owner_user_id"
                :disabled="saving"
                class="session-input mt-3 w-full rounded-[1.25rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.72)] px-4 py-3 text-[14px] text-[#f6f7fb] outline-none transition-all duration-200 focus:border-[rgba(233,69,96,0.34)] disabled:cursor-not-allowed disabled:opacity-60"
              >
                <option v-for="member in members" :key="member.user_id" :value="member.user_id">
                  {{ member.username }} · {{ formatRole(member.role) }}
                </option>
              </select>
            </label>

            <div class="mt-5 rounded-[1.4rem] border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] p-4">
              <p class="text-[11px] uppercase tracking-[0.2em] text-[#7ec8e3]/55">Flow</p>
              <p class="mt-3 text-[13px] leading-relaxed text-[#d8dce7]/58">
                Create into your own GM vault to keep it reusable, or assign it directly to a player, assistant GM, or another GM.
              </p>
            </div>
          </aside>
        </div>

        <div class="mt-6 flex flex-wrap items-center justify-between gap-4 border-t border-[rgba(126,200,227,0.12)] pt-5">
          <button
            type="button"
            @click="close"
            :disabled="saving"
            class="cursor-pointer rounded-xl border border-[rgba(126,200,227,0.16)] bg-[rgba(126,200,227,0.08)] px-4 py-2.5 text-[13px] font-semibold text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(126,200,227,0.3)] disabled:cursor-not-allowed disabled:opacity-60"
          >
            Cancel
          </button>
          <button
            type="button"
            @click="emitSave"
            :disabled="saving"
            class="cursor-pointer rounded-xl border border-[rgba(233,69,96,0.28)] bg-[linear-gradient(135deg,rgba(233,69,96,0.9),rgba(194,49,82,0.9))] px-4 py-2.5 text-[13px] font-semibold text-white transition-all duration-200 hover:-translate-y-0.5 hover:shadow-[0_12px_30px_rgba(233,69,96,0.24)] disabled:cursor-not-allowed disabled:opacity-60"
          >
            {{ saving ? 'Creating...' : 'Create Character' }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>