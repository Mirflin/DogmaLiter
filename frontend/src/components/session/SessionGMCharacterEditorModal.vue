<script setup>
import { X as XIcon } from '@lucide/vue'
import { API_URL } from '@/api'
import { computed, ref, watch } from 'vue'

const MAX_PORTRAIT_SIZE = 5 * 1024 * 1024
const ALLOWED_PORTRAIT_TYPES = ['image/jpeg', 'image/png', 'image/webp']

const baseAttributeFields = [
  { key: 'strength', label: 'Strength' },
  { key: 'dexterity', label: 'Dexterity' },
  { key: 'constitution', label: 'Constitution' },
  { key: 'intelligence', label: 'Intelligence' },
  { key: 'wisdom', label: 'Wisdom' },
  { key: 'charisma', label: 'Charisma' },
]

const currencyFields = [
  { key: 'currency_gold', label: 'Gold' },
  { key: 'currency_silver', label: 'Silver' },
  { key: 'currency_copper', label: 'Bronze' },
]

const props = defineProps({
  visible: {
    type: Boolean,
    default: false,
  },
  character: {
    type: Object,
    default: null,
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
})

const emit = defineEmits(['close', 'save'])

const fileInputRef = ref(null)
const localError = ref('')
const portraitFile = ref(null)
const portraitPreviewUrl = ref('')
const attributeKeyCounter = ref(0)
const form = ref(createFormState())

const displayError = computed(() => localError.value || props.error || '')
const portraitUrl = computed(() => portraitPreviewUrl.value || avatarUrl(props.character?.portrait_id))
const selectedOwner = computed(() => props.members.find(member => member.user_id === form.value.owner_user_id) ?? null)

watch(
  [() => props.visible, () => props.character],
  ([visible, character]) => {
    if (visible && character) {
      syncForm(character)
      return
    }

    if (!visible) {
      resetTransientState()
    }
  },
  { immediate: true },
)

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

function formatRole(role) {
  if (!role) return 'Player'
  return role
    .split('_')
    .filter(Boolean)
    .map(part => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ')
}

function nextAttributeKey() {
  attributeKeyCounter.value += 1
  return `gm-custom-${attributeKeyCounter.value}`
}

function mapCustomAttributes(attributes = []) {
  return attributes.map((attribute, index) => ({
    id: attribute.id ?? '',
    key: attribute.id || nextAttributeKey(),
    name: attribute.name ?? '',
    value: attribute.value ?? 0,
    sort_order: attribute.sort_order ?? index,
  }))
}

function createFormState(character = null) {
  return {
    name: character?.name ?? '',
    backstory: character?.backstory ?? '',
    owner_user_id: character?.user_id ?? props.members[0]?.user_id ?? '',
    inventory_width: character?.inventory_width ?? 10,
    inventory_height: character?.inventory_height ?? 6,
    currency_gold: character?.currency_gold ?? 0,
    currency_silver: character?.currency_silver ?? 0,
    currency_copper: character?.currency_copper ?? 0,
    base_attributes: {
      strength: character?.base_attributes?.strength ?? 10,
      dexterity: character?.base_attributes?.dexterity ?? 10,
      constitution: character?.base_attributes?.constitution ?? 10,
      intelligence: character?.base_attributes?.intelligence ?? 10,
      wisdom: character?.base_attributes?.wisdom ?? 10,
      charisma: character?.base_attributes?.charisma ?? 10,
    },
    custom_attributes: mapCustomAttributes(character?.custom_attributes ?? []),
  }
}

function clearPortraitSelection() {
  if (portraitPreviewUrl.value) {
    URL.revokeObjectURL(portraitPreviewUrl.value)
  }

  portraitPreviewUrl.value = ''
  portraitFile.value = null

  if (fileInputRef.value) {
    fileInputRef.value.value = ''
  }
}

function resetTransientState() {
  localError.value = ''
  clearPortraitSelection()
}

function syncForm(character) {
  form.value = createFormState(character)
  resetTransientState()
}

function close() {
  if (!props.saving) {
    emit('close')
  }
}

function selectPortrait() {
  fileInputRef.value?.click()
}

function handlePortraitSelected(event) {
  const file = event.target.files?.[0]
  if (!file) return

  if (!ALLOWED_PORTRAIT_TYPES.includes(file.type)) {
    localError.value = 'Only JPEG, PNG, and WebP images are allowed'
    return
  }

  if (file.size > MAX_PORTRAIT_SIZE) {
    localError.value = 'Portrait file must be under 5MB'
    return
  }

  clearPortraitSelection()
  portraitFile.value = file
  portraitPreviewUrl.value = URL.createObjectURL(file)
  localError.value = ''
}

function addCustomAttribute() {
  form.value.custom_attributes.push({
    id: '',
    key: nextAttributeKey(),
    name: '',
    value: 0,
    sort_order: form.value.custom_attributes.length,
  })
}

function removeCustomAttribute(index) {
  form.value.custom_attributes.splice(index, 1)
}

function moveCustomAttribute(index, direction) {
  const nextIndex = index + direction
  if (nextIndex < 0 || nextIndex >= form.value.custom_attributes.length) return

  const [attribute] = form.value.custom_attributes.splice(index, 1)
  form.value.custom_attributes.splice(nextIndex, 0, attribute)
}

function normalizeInteger(value, fallback = 0) {
  const parsed = Number.parseInt(value, 10)
  return Number.isNaN(parsed) ? fallback : parsed
}

function emitSave() {
  if (!props.character) return

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

  const customAttributes = []
  const seenNames = new Set()

  for (const [index, attribute] of form.value.custom_attributes.entries()) {
    const attributeName = attribute.name.trim()
    if (!attributeName) {
      localError.value = 'Custom attribute name cannot be empty'
      return
    }

    const normalizedName = attributeName.toLowerCase()
    if (seenNames.has(normalizedName)) {
      localError.value = 'Custom attribute names must be unique'
      return
    }

    seenNames.add(normalizedName)
    customAttributes.push({
      ...(attribute.id ? { id: attribute.id } : {}),
      name: attributeName,
      value: normalizeInteger(attribute.value),
      sort_order: index,
    })
  }

  localError.value = ''
  emit('save', {
    payload: {
      name,
      backstory,
      owner_user_id: form.value.owner_user_id,
      inventory_width: normalizeInteger(form.value.inventory_width, 10),
      inventory_height: normalizeInteger(form.value.inventory_height, 6),
      currency_gold: normalizeInteger(form.value.currency_gold),
      currency_silver: normalizeInteger(form.value.currency_silver),
      currency_copper: normalizeInteger(form.value.currency_copper),
      base_attributes: {
        strength: normalizeInteger(form.value.base_attributes.strength, 10),
        dexterity: normalizeInteger(form.value.base_attributes.dexterity, 10),
        constitution: normalizeInteger(form.value.base_attributes.constitution, 10),
        intelligence: normalizeInteger(form.value.base_attributes.intelligence, 10),
        wisdom: normalizeInteger(form.value.base_attributes.wisdom, 10),
        charisma: normalizeInteger(form.value.base_attributes.charisma, 10),
      },
      custom_attributes: customAttributes,
    },
    portraitFile: portraitFile.value,
  })
}
</script>

<template>
  <Teleport to="body">
    <div v-if="visible" class="fixed inset-0 z-[12500] flex items-center justify-center px-4 py-6 sm:px-6" @click.self="close">
      <div class="absolute inset-0 bg-[rgba(5,8,12,0.82)] backdrop-blur-md" @click="close"></div>

      <div class="relative max-h-[92vh] w-full max-w-[1080px] overflow-y-auto rounded-[2rem] border border-[rgba(126,200,227,0.16)] bg-[linear-gradient(180deg,rgba(9,18,34,0.98),rgba(5,10,22,0.98))] p-5 shadow-[0_40px_120px_rgba(0,0,0,0.52)] sm:p-7">
        <button
          @click="close"
          :disabled="saving"
          class="absolute right-5 top-5 flex h-10 w-10 cursor-pointer items-center justify-center rounded-xl border border-[rgba(126,200,227,0.14)] bg-[rgba(126,200,227,0.08)] text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(233,69,96,0.32)] disabled:cursor-not-allowed disabled:opacity-60"
          aria-label="Close GM character editor"
        >
          <XIcon class="h-5 w-5" :stroke-width="2" />
        </button>

        <div class="pr-12">
          <p class="text-[11px] uppercase tracking-[0.24em] text-[#7ec8e3]/58">GM Character Editor</p>
          <div class="mt-3 flex flex-wrap items-center gap-3">
            <h2 class="font-[Cinzel] text-[28px] font-bold text-[#f6f7fb] sm:text-[34px]">{{ character?.name || 'Character Setup' }}</h2>
            <span class="rounded-full border border-[rgba(126,200,227,0.16)] bg-[rgba(126,200,227,0.08)] px-3 py-1.5 text-[11px] uppercase tracking-[0.16em] text-[#8fd7ef]">
              {{ selectedOwner?.username || character?.owner?.username || 'Unassigned' }}
            </span>
          </div>
          <p class="mt-3 max-w-[48rem] text-[14px] leading-relaxed text-[#d8dce7]/62">
            Configure portrait, narrative profile, base attributes, inventory dimensions, custom stats, and character ownership.
          </p>
        </div>

        <p v-if="displayError" class="mt-6 rounded-[1.3rem] border border-[rgba(248,113,113,0.24)] bg-[rgba(127,29,29,0.18)] px-4 py-3 text-[13px] text-[#fecaca]">
          {{ displayError }}
        </p>

        <div v-if="character" class="mt-6 grid gap-6 xl:grid-cols-[320px_minmax(0,1fr)]">
          <aside class="space-y-5 rounded-[1.75rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(8,16,30,0.78)] p-5">
            <div>
              <p class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Portrait</p>
              <div class="mt-4 overflow-hidden rounded-[1.5rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.66)]">
                <div class="flex h-72 items-center justify-center bg-[linear-gradient(180deg,rgba(16,32,52,0.94),rgba(8,16,30,0.94))]">
                  <img v-if="portraitUrl" :src="portraitUrl" :alt="character.name" class="h-full w-full object-cover object-top" />
                  <span v-else class="font-[Cinzel] text-[38px] font-bold text-[#8fd7ef]">{{ initials(character.name) }}</span>
                </div>
              </div>
            </div>

            <input
              ref="fileInputRef"
              type="file"
              accept="image/jpeg,image/png,image/webp"
              class="hidden"
              @change="handlePortraitSelected"
            />

            <div class="flex flex-wrap gap-3">
              <button
                type="button"
                @click="selectPortrait"
                :disabled="saving"
                class="cursor-pointer rounded-xl border border-[rgba(126,200,227,0.16)] bg-[rgba(126,200,227,0.08)] px-4 py-2.5 text-[13px] font-semibold text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(126,200,227,0.3)] disabled:cursor-not-allowed disabled:opacity-60"
              >
                {{ portraitFile ? 'Replace Portrait' : 'Upload Portrait' }}
              </button>
              <button
                v-if="portraitFile"
                type="button"
                @click="clearPortraitSelection"
                :disabled="saving"
                class="cursor-pointer rounded-xl border border-[rgba(233,69,96,0.18)] bg-[rgba(233,69,96,0.08)] px-4 py-2.5 text-[13px] font-semibold text-[#ffe0e7] transition-all duration-200 hover:border-[rgba(233,69,96,0.35)] disabled:cursor-not-allowed disabled:opacity-60"
              >
                Clear Selection
              </button>
            </div>

            <div class="rounded-[1.4rem] border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] p-4">
              <p class="text-[11px] uppercase tracking-[0.2em] text-[#7ec8e3]/55">Inventory Space</p>
              <div class="mt-4 grid gap-3 sm:grid-cols-2 xl:grid-cols-1 2xl:grid-cols-2">
                <label class="rounded-[1.2rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.62)] px-4 py-3">
                  <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Width</span>
                  <input v-model.number="form.inventory_width" type="number" min="1" max="20" class="session-input mt-3 w-full border-0 bg-transparent p-0 text-[24px] font-semibold text-[#f6f7fb] outline-none" />
                </label>
                <label class="rounded-[1.2rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.62)] px-4 py-3">
                  <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Height</span>
                  <input v-model.number="form.inventory_height" type="number" min="1" max="20" class="session-input mt-3 w-full border-0 bg-transparent p-0 text-[24px] font-semibold text-[#f6f7fb] outline-none" />
                </label>
              </div>
              <p class="mt-3 text-[12px] leading-relaxed text-[#d8dce7]/54">Shrinking the grid is blocked if existing items would no longer fit.</p>
            </div>
          </aside>

          <section class="space-y-5 rounded-[1.75rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(8,16,30,0.78)] p-5 sm:p-6">
            <div class="grid gap-5 lg:grid-cols-[minmax(0,1.15fr)_minmax(300px,0.85fr)]">
              <div class="space-y-5">
                <label class="block">
                  <span class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Name</span>
                  <input
                    v-model="form.name"
                    type="text"
                    maxlength="100"
                    :disabled="saving"
                    class="session-input mt-2 w-full rounded-[1.25rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.72)] px-4 py-3 text-[15px] text-[#f6f7fb] outline-none transition-all duration-200 placeholder:text-[#7ec8e3]/35 focus:border-[rgba(233,69,96,0.34)] disabled:cursor-not-allowed disabled:opacity-60"
                  />
                </label>

                <label class="block">
                  <span class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Backstory</span>
                  <textarea
                    v-model="form.backstory"
                    rows="7"
                    :disabled="saving"
                    class="session-input mt-2 w-full resize-none rounded-[1.25rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.72)] px-4 py-3 text-[15px] text-[#f6f7fb] outline-none transition-all duration-200 placeholder:text-[#7ec8e3]/35 focus:border-[rgba(233,69,96,0.34)] disabled:cursor-not-allowed disabled:opacity-60"
                  ></textarea>
                </label>

                <label class="block">
                  <span class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Assigned To</span>
                  <select
                    v-model="form.owner_user_id"
                    :disabled="saving"
                    class="session-input mt-2 w-full rounded-[1.25rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.72)] px-4 py-3 text-[14px] text-[#f6f7fb] outline-none transition-all duration-200 focus:border-[rgba(233,69,96,0.34)] disabled:cursor-not-allowed disabled:opacity-60"
                  >
                    <option v-for="member in members" :key="member.user_id" :value="member.user_id">
                      {{ member.username }} · {{ formatRole(member.role) }}
                    </option>
                  </select>
                </label>
              </div>

              <div class="rounded-[1.5rem] border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] p-4">
                <div class="flex items-center justify-between gap-3">
                  <span class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Currency</span>
                </div>
                <div class="mt-4 grid gap-3">
                  <label
                    v-for="currency in currencyFields"
                    :key="currency.key"
                    class="rounded-[1.2rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.62)] px-4 py-3"
                  >
                    <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">{{ currency.label }}</span>
                    <input v-model.number="form[currency.key]" type="number" class="session-input mt-3 w-full border-0 bg-transparent p-0 text-[24px] font-semibold text-[#f6f7fb] outline-none" />
                  </label>
                </div>
              </div>
            </div>

            <div class="rounded-[1.5rem] border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] p-5">
              <div class="flex items-center justify-between gap-3">
                <span class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Base Attributes</span>
                <span class="text-[12px] text-[#d8dce7]/45">GM editable</span>
              </div>

              <div class="mt-4 grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
                <label
                  v-for="attribute in baseAttributeFields"
                  :key="attribute.key"
                  class="rounded-[1.2rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.62)] px-4 py-3"
                >
                  <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">{{ attribute.label }}</span>
                  <input v-model.number="form.base_attributes[attribute.key]" type="number" min="0" max="999" class="session-input mt-3 w-full border-0 bg-transparent p-0 text-[24px] font-semibold text-[#f6f7fb] outline-none" />
                </label>
              </div>
            </div>

            <div class="rounded-[1.5rem] border border-[rgba(126,200,227,0.1)] bg-[rgba(126,200,227,0.05)] p-5">
              <div class="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <span class="text-[11px] uppercase tracking-[0.22em] text-[#7ec8e3]/55">Custom Attributes</span>
                </div>
                <button
                  type="button"
                  @click="addCustomAttribute"
                  :disabled="saving"
                  class="cursor-pointer rounded-xl border border-[rgba(233,69,96,0.22)] bg-[rgba(233,69,96,0.12)] px-4 py-2.5 text-[13px] font-semibold text-[#ffe0e7] transition-all duration-200 hover:border-[rgba(233,69,96,0.4)] disabled:cursor-not-allowed disabled:opacity-60"
                >
                  Add Attribute
                </button>
              </div>

              <div v-if="form.custom_attributes.length" class="mt-5 space-y-3">
                <article
                  v-for="(attribute, index) in form.custom_attributes"
                  :key="attribute.key"
                  class="rounded-[1.2rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.62)] p-4"
                >
                  <div class="grid gap-3 lg:grid-cols-[minmax(0,1fr)_180px_auto]">
                    <label class="block">
                      <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Label</span>
                      <input v-model="attribute.name" type="text" maxlength="100" class="session-input mt-2 w-full rounded-xl border border-[rgba(126,200,227,0.12)] bg-[rgba(126,200,227,0.08)] px-3 py-2.5 text-[14px] text-[#f6f7fb] outline-none focus:border-[rgba(233,69,96,0.34)]" />
                    </label>
                    <label class="block">
                      <span class="text-[11px] uppercase tracking-[0.18em] text-[#7ec8e3]/45">Value</span>
                      <input v-model.number="attribute.value" type="number" class="session-input mt-2 w-full rounded-xl border border-[rgba(126,200,227,0.12)] bg-[rgba(126,200,227,0.08)] px-3 py-2.5 text-[14px] text-[#f6f7fb] outline-none focus:border-[rgba(233,69,96,0.34)]" />
                    </label>
                    <div class="flex flex-wrap items-end gap-2">
                      <button type="button" @click="moveCustomAttribute(index, -1)" :disabled="saving || index === 0" class="cursor-pointer rounded-xl border border-[rgba(126,200,227,0.14)] bg-[rgba(126,200,227,0.08)] px-3 py-2.5 text-[12px] font-semibold text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(126,200,227,0.3)] disabled:cursor-not-allowed disabled:opacity-60">Up</button>
                      <button type="button" @click="moveCustomAttribute(index, 1)" :disabled="saving || index === form.custom_attributes.length - 1" class="cursor-pointer rounded-xl border border-[rgba(126,200,227,0.14)] bg-[rgba(126,200,227,0.08)] px-3 py-2.5 text-[12px] font-semibold text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(126,200,227,0.3)] disabled:cursor-not-allowed disabled:opacity-60">Down</button>
                      <button type="button" @click="removeCustomAttribute(index)" :disabled="saving" class="cursor-pointer rounded-xl border border-[rgba(248,113,113,0.2)] bg-[rgba(248,113,113,0.12)] px-3 py-2.5 text-[12px] font-semibold text-[#fecaca] transition-all duration-200 hover:border-[rgba(248,113,113,0.35)] disabled:cursor-not-allowed disabled:opacity-60">Remove</button>
                    </div>
                  </div>
                </article>
              </div>

              <p v-else class="mt-5 text-[14px] text-[#d8dce7]/58">No custom attributes yet. Add rows here to create GM-managed stats.</p>
            </div>
          </section>
        </div>

        <div class="mt-6 flex flex-wrap items-center justify-between gap-4 border-t border-[rgba(126,200,227,0.12)] pt-5">
          <div class="flex flex-wrap gap-3">
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
              {{ saving ? 'Saving...' : 'Save GM Changes' }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>