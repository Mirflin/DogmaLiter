<script setup>
import { computed, onMounted, ref } from 'vue'
import { Plus, Trash2 } from '@lucide/vue'
import { useAuthStore } from '@/stores/auth'
import { notify } from '@/notify'
import InventoryShapeEditor from './InventoryShapeEditor.vue'
import { countOpen, expandMask, fullMask, resizeMask } from './inventoryMask'

const props = defineProps({
  gameId: { type: String, required: true },
  characters: { type: Array, default: () => [] },
})

const emit = defineEmits(['applied'])

const auth = useAuthStore()
const presets = ref([])
const loading = ref(false)
const saving = ref(false)
const applying = ref(false)
const confirmDelete = ref(false)

// The preset being edited; id '' is a new one not saved yet.
const draft = ref(null)
const selectedCharacterIds = ref([])

const savedPreset = computed(() => presets.value.find(preset => preset.id === draft.value?.id) ?? null)
const isDirty = computed(() => {
  if (!draft.value) return false
  const saved = savedPreset.value
  if (!saved) return true
  return saved.name !== draft.value.name.trim() || saved.width !== draft.value.width || saved.height !== draft.value.height || saved.mask !== draft.value.mask
})
const canSave = computed(() => draft.value && draft.value.name.trim() && countOpen(draft.value.mask) > 0 && isDirty.value && !saving.value)
const canApply = computed(() => savedPreset.value && !isDirty.value && selectedCharacterIds.value.length > 0 && !applying.value)

const characterRows = computed(() => props.characters.map(character => ({
  id: character.id,
  name: character.name,
  owner: character.owner?.username || '',
  backpack: backpackLabel(character),
})))

function backpackLabel(character) {
  const size = `${character.inventory_width}×${character.inventory_height}`
  if (!character.inventory_mask) return `Grid ${size}`
  return `Shaped ${size} · ${countOpen(character.inventory_mask)} cells`
}

async function loadPresets() {
  loading.value = true
  try {
    const data = await auth.getInventoryPresets(props.gameId)
    presets.value = data.presets || []
    if (!draft.value && presets.value.length) selectPreset(presets.value[0])
  } catch (err) {
    notify.error(err, 'Failed to load backpack presets')
  } finally {
    loading.value = false
  }
}

function selectPreset(preset) {
  draft.value = { id: preset.id, name: preset.name, width: preset.width, height: preset.height, mask: preset.mask }
  confirmDelete.value = false
}

function startNewPreset() {
  draft.value = { id: '', name: 'New backpack', width: 10, height: 6, mask: fullMask(10, 6) }
  confirmDelete.value = false
}

function setSize(key, rawValue) {
  const value = Math.min(20, Math.max(1, Number.parseInt(rawValue, 10) || 1))
  const { width, height, mask } = draft.value
  const next = { width, height, [key]: value }
  draft.value = { ...draft.value, ...next, mask: resizeMask(expandMask(mask, width, height), width, height, next.width, next.height, '1') }
}

async function savePreset() {
  if (!canSave.value) return
  saving.value = true
  const payload = { name: draft.value.name.trim(), width: draft.value.width, height: draft.value.height, mask: draft.value.mask }
  try {
    const data = draft.value.id
      ? await auth.updateInventoryPreset(props.gameId, draft.value.id, payload)
      : await auth.createInventoryPreset(props.gameId, payload)
    const preset = data.preset
    presets.value = [...presets.value.filter(item => item.id !== preset.id), preset].sort((a, b) => a.name.localeCompare(b.name))
    selectPreset(preset)
    notify.success({ title: 'Preset saved', message: `${preset.name} is ready to hand out.` })
  } catch (err) {
    notify.error(err, 'Failed to save preset')
  } finally {
    saving.value = false
  }
}

async function deletePreset() {
  if (!draft.value) return
  if (!draft.value.id) {
    draft.value = null
    return
  }
  if (!confirmDelete.value) {
    confirmDelete.value = true
    return
  }
  try {
    await auth.deleteInventoryPreset(props.gameId, draft.value.id)
    presets.value = presets.value.filter(preset => preset.id !== draft.value.id)
    draft.value = null
    if (presets.value.length) selectPreset(presets.value[0])
  } catch (err) {
    notify.error(err, 'Failed to delete preset')
  } finally {
    confirmDelete.value = false
  }
}

function toggleCharacter(id) {
  selectedCharacterIds.value = selectedCharacterIds.value.includes(id)
    ? selectedCharacterIds.value.filter(value => value !== id)
    : [...selectedCharacterIds.value, id]
}

async function applyPreset() {
  if (!canApply.value) return
  applying.value = true
  try {
    const data = await auth.applyInventoryPreset(props.gameId, savedPreset.value.id, selectedCharacterIds.value)
    notify.success({ title: 'Backpack handed out', message: `${savedPreset.value.name} now shapes ${data.applied} character${data.applied === 1 ? '' : 's'}.` })
    selectedCharacterIds.value = []
    emit('applied')
  } catch (err) {
    notify.error(err, 'Failed to apply preset')
  } finally {
    applying.value = false
  }
}

function previewRects(preset) {
  const rects = []
  for (let index = 0; index < preset.mask.length; index += 1) {
    if (preset.mask[index] === '1') rects.push({ x: index % preset.width, y: Math.floor(index / preset.width) })
  }
  return rects
}

onMounted(loadPresets)
</script>

<template>
  <div class="packs">
    <aside class="packs__list">
      <div class="flex items-center justify-between gap-2">
        <h4 class="text-[15px] font-semibold text-ink">Backpack presets</h4>
        <button type="button" class="packs__new session-own" @click="startNewPreset">
          <Plus class="h-4 w-4" :stroke-width="2" /> New
        </button>
      </div>
      <p class="mt-1 text-[13px] text-dim">Draw a shape once, hand it to any character.</p>

      <p v-if="loading && !presets.length" class="mt-4 text-[13px] text-faint">Loading…</p>
      <p v-else-if="!presets.length && !draft" class="mt-4 text-[13px] text-faint">No presets yet. Start with New.</p>

      <ul class="mt-4 grid gap-1.5">
        <li v-for="preset in presets" :key="preset.id">
          <div
            role="button"
            tabindex="0"
            class="packs__item"
            :class="{ 'is-active': draft?.id === preset.id }"
            @click="selectPreset(preset)"
            @keydown.enter="selectPreset(preset)"
          >
            <svg class="packs__thumb" :viewBox="`0 0 ${preset.width} ${preset.height}`" aria-hidden="true">
              <rect v-for="rect in previewRects(preset)" :key="`${rect.x}-${rect.y}`" :x="rect.x + 0.08" :y="rect.y + 0.08" width="0.84" height="0.84" rx="0.12" />
            </svg>
            <span class="min-w-0">
              <span class="block truncate text-[14px] font-semibold text-ink">{{ preset.name }}</span>
              <span class="block text-[12px] text-faint">{{ preset.width }}×{{ preset.height }} · {{ countOpen(preset.mask) }} cells</span>
            </span>
          </div>
        </li>
      </ul>
    </aside>

    <section v-if="draft" class="packs__editor">
      <div class="grid gap-3 sm:grid-cols-[minmax(0,1fr)_5.5rem_5.5rem]">
        <label class="block">
          <span class="text-[13px] font-semibold text-dim">Name</span>
          <input v-model="draft.name" type="text" maxlength="60" class="session-input mt-1.5 w-full border border-line bg-n1/72 px-3 py-2 text-[14px] text-ink outline-none" />
        </label>
        <label class="block">
          <span class="text-[13px] font-semibold text-dim">Width</span>
          <input :value="draft.width" type="number" min="1" max="20" class="session-input mt-1.5 w-full border border-line bg-n1/72 px-3 py-2 text-[14px] text-ink outline-none" @change="setSize('width', $event.target.value)" />
        </label>
        <label class="block">
          <span class="text-[13px] font-semibold text-dim">Height</span>
          <input :value="draft.height" type="number" min="1" max="20" class="session-input mt-1.5 w-full border border-line bg-n1/72 px-3 py-2 text-[14px] text-ink outline-none" @change="setSize('height', $event.target.value)" />
        </label>
      </div>

      <InventoryShapeEditor v-model="draft.mask" :width="draft.width" :height="draft.height" class="mt-4" />

      <div class="mt-4 flex flex-wrap items-center justify-between gap-2">
        <button type="button" class="packs__danger session-own" @click="deletePreset">
          <Trash2 class="h-4 w-4" :stroke-width="2" />
          {{ !draft.id ? 'Discard' : confirmDelete ? 'Confirm delete' : 'Delete' }}
        </button>
        <button type="button" class="session-own packs__primary" :disabled="!canSave" @click="savePreset">
          {{ saving ? 'Saving…' : draft.id ? 'Save changes' : 'Create preset' }}
        </button>
      </div>
    </section>

    <section v-if="draft" class="packs__give">
      <h4 class="text-[15px] font-semibold text-ink">Give to characters</h4>
      <p class="mt-1 text-[13px] text-dim">Items stay where they are. If one would land on a closed cell, nothing changes and you'll see who to tidy up.</p>

      <ul class="packs__chars mt-3">
        <li v-for="character in characterRows" :key="character.id">
          <label class="packs__char">
            <input type="checkbox" :checked="selectedCharacterIds.includes(character.id)" @change="toggleCharacter(character.id)" />
            <span class="min-w-0 flex-1">
              <span class="block truncate text-[14px] font-semibold text-ink">{{ character.name }}</span>
              <span class="block truncate text-[12px] text-faint">{{ character.owner }}<template v-if="character.owner"> · </template>{{ character.backpack }}</span>
            </span>
          </label>
        </li>
      </ul>
      <p v-if="!characterRows.length" class="mt-3 text-[13px] text-faint">No characters in this game yet.</p>

      <button type="button" class="session-own packs__primary mt-4 w-full" :disabled="!canApply" @click="applyPreset">
        {{ applying ? 'Applying…' : selectedCharacterIds.length ? `Give to ${selectedCharacterIds.length}` : 'Pick characters' }}
      </button>
      <p v-if="isDirty" class="mt-2 text-[12px] text-faint">Save the preset before handing it out.</p>
    </section>
  </div>
</template>

<style scoped>
.packs {
  display: grid;
  gap: 1rem;
  align-items: start;
}

@media (min-width: 900px) {
  .packs {
    grid-template-columns: 16rem minmax(0, 1fr);
  }

  .packs__give {
    grid-column: 2;
  }
}

@media (min-width: 1400px) {
  .packs {
    grid-template-columns: 16rem minmax(0, 1fr) 19rem;
  }

  .packs__give {
    grid-column: auto;
  }
}

.packs__list,
.packs__editor,
.packs__give {
  border-radius: 1.25rem;
  background: rgb(8 13 36 / 0.82);
  box-shadow: inset 0 0 0 1px rgb(74 90 150 / 0.35);
  padding: 1.1rem;
}

.packs__item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  border-radius: 0.75rem;
  padding: 0.5rem 0.6rem;
  cursor: pointer;
  transition: background-color 150ms ease;
}

.packs__item:hover {
  background: var(--color-n3);
}

.packs__item.is-active {
  background: color-mix(in oklab, var(--color-primary) 22%, var(--color-n2));
  box-shadow: inset 0 0 0 1px color-mix(in oklab, var(--color-primary-ink) 40%, transparent);
}

.packs__thumb {
  width: 3rem;
  height: 2.25rem;
  flex-shrink: 0;
  border-radius: 0.35rem;
  background: var(--color-n0);
  padding: 3px;
  fill: var(--color-primary-ink);
}

.packs__chars {
  display: grid;
  gap: 0.25rem;
  max-height: 22rem;
  overflow-y: auto;
}

.packs__char {
  display: flex;
  align-items: center;
  gap: 0.7rem;
  border-radius: 0.7rem;
  padding: 0.45rem 0.55rem;
  cursor: pointer;
}

.packs__char:hover {
  background: var(--color-n3);
}

.packs__new,
.packs__danger {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.4rem 0.75rem;
  font-size: 13px;
  font-weight: 600;
}

.packs__new,
.packs__danger {
  border-radius: 0.6rem;
  background: var(--color-n3);
  color: var(--color-ink);
  cursor: pointer;
  transition: background-color 150ms ease;
}

.packs__new:hover,
.packs__danger:hover {
  background: var(--color-n4);
}

.packs__danger {
  color: var(--color-danger-soft);
}

.packs__primary {
  border-radius: 0.7rem;
  background: var(--color-primary);
  padding: 0.6rem 1.1rem;
  font-size: 13px;
  font-weight: 600;
  color: white;
  cursor: pointer;
  transition: background-color 150ms ease, opacity 150ms ease;
}

.packs__primary:hover:not(:disabled) {
  background: color-mix(in oklab, var(--color-primary) 85%, white);
}

.packs__primary:disabled {
  cursor: not-allowed;
  opacity: 0.45;
}
</style>
