<script setup>
import { computed, ref, watch } from 'vue'

const props = defineProps({
  // [{ key, label, sortable?, filterable?, align?: 'left'|'right', headerClass?, cellClass?, filterOptions?: [{value,label}] }]
  columns: { type: Array, default: () => [] },
  rows: { type: Array, default: () => [] },
  rowKey: { type: String, default: 'id' },
  searchable: { type: Boolean, default: true },
  searchPlaceholder: { type: String, default: 'Search...' },
  searchKeys: { type: Array, default: null },
  paginate: { type: Boolean, default: true },
  pageSize: { type: Number, default: 10 },
  minWidth: { type: String, default: '640px' },
  minHeight: { type: String, default: '' },
  emptyText: { type: String, default: 'No data' },
})

const search = ref('')
const sortKey = ref('')
const sortDir = ref('asc')
const columnFilters = ref({})
const page = ref(1)

const filterableColumns = computed(() => props.columns.filter(column => column.filterable))
const hasToolbar = computed(() => props.searchable || filterableColumns.value.length > 0)
const effectiveSearchKeys = computed(() => props.searchKeys || props.columns.map(column => column.key))

watch(
  filterableColumns,
  (columns) => {
    for (const column of columns) {
      if (!(column.key in columnFilters.value)) columnFilters.value[column.key] = ''
    }
  },
  { immediate: true },
)

function distinctValues(column) {
  if (Array.isArray(column.filterOptions)) return column.filterOptions
  const set = new Set()
  for (const row of props.rows) {
    const value = row?.[column.key]
    if (value !== undefined && value !== null && value !== '') set.add(String(value))
  }
  return [...set].sort((a, b) => a.localeCompare(b)).map(value => ({ value, label: value }))
}

const filteredRows = computed(() => {
  let result = props.rows

  for (const [key, value] of Object.entries(columnFilters.value)) {
    if (value === '' || value == null) continue
    result = result.filter(row => String(row?.[key] ?? '') === String(value))
  }

  const query = search.value.trim().toLowerCase()
  if (query) {
    result = result.filter(row => effectiveSearchKeys.value.some(key => String(row?.[key] ?? '').toLowerCase().includes(query)))
  }

  if (sortKey.value) {
    const direction = sortDir.value === 'asc' ? 1 : -1
    result = [...result].sort((a, b) => {
      const av = a?.[sortKey.value]
      const bv = b?.[sortKey.value]
      if (typeof av === 'number' && typeof bv === 'number') return (av - bv) * direction
      return String(av ?? '').localeCompare(String(bv ?? '')) * direction
    })
  }

  return result
})

const totalPages = computed(() => Math.max(1, Math.ceil(filteredRows.value.length / props.pageSize)))
const pagedRows = computed(() => {
  if (!props.paginate) return filteredRows.value
  const start = (page.value - 1) * props.pageSize
  return filteredRows.value.slice(start, start + props.pageSize)
})

watch(search, () => { page.value = 1 })
watch(columnFilters, () => { page.value = 1 }, { deep: true })
watch(totalPages, (total) => { if (page.value > total) page.value = total })

function toggleSort(column) {
  if (!column.sortable) return
  if (sortKey.value === column.key) {
    sortDir.value = sortDir.value === 'asc' ? 'desc' : 'asc'
  } else {
    sortKey.value = column.key
    sortDir.value = 'asc'
  }
}

function rowId(row, index) {
  return row?.[props.rowKey] ?? index
}
</script>

<template>
  <div class="overflow-hidden rounded-[1.5rem] border border-[rgba(126,200,227,0.12)] bg-[rgba(8,16,30,0.78)]">
    <div v-if="hasToolbar || $slots.toolbar" class="flex flex-wrap items-center gap-3 border-b border-[rgba(126,200,227,0.1)] px-5 py-4">
      <div v-if="searchable" class="flex min-w-[13rem] flex-1 items-center gap-2 rounded-xl border border-[rgba(126,200,227,0.12)] bg-[rgba(7,17,31,0.72)] px-3 py-2">
        <svg class="h-4 w-4 shrink-0 text-[#7ec8e3]/45" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-4.35-4.35M17 11a6 6 0 11-12 0 6 6 0 0112 0z" /></svg>
        <input v-model="search" type="text" :placeholder="searchPlaceholder" class="w-full bg-transparent text-[13px] text-[#f6f7fb] outline-none placeholder:text-[#7ec8e3]/30" />
      </div>

      <label v-for="column in filterableColumns" :key="column.key" class="flex items-center gap-2">
        <span class="text-[11px] uppercase tracking-[0.14em] text-[#7ec8e3]/45">{{ column.label }}</span>
        <select v-model="columnFilters[column.key]" class="rounded-lg border border-[rgba(126,200,227,0.16)] bg-[rgba(7,17,31,0.72)] px-2.5 py-1.5 text-[12px] text-[#f6f7fb] outline-none">
          <option value="">All</option>
          <option v-for="option in distinctValues(column)" :key="option.value" :value="option.value">{{ option.label }}</option>
        </select>
      </label>

      <slot name="toolbar" />
    </div>

    <div class="overflow-x-auto" :style="minHeight ? { minHeight } : {}">
      <table class="w-full text-left text-[13px]" :style="{ minWidth }">
        <thead>
          <tr class="text-[11px] uppercase tracking-[0.16em] text-[#7ec8e3]/45">
            <th
              v-for="column in columns"
              :key="column.key"
              class="px-5 py-3 font-medium"
              :class="[column.align === 'right' ? 'text-right' : '', column.headerClass]"
            >
              <button v-if="column.sortable" type="button" @click="toggleSort(column)" class="inline-flex cursor-pointer items-center gap-1 transition-colors hover:text-[#8fd7ef]">
                {{ column.label }}
                <span v-if="sortKey === column.key" class="text-[9px]">{{ sortDir === 'asc' ? '▲' : '▼' }}</span>
              </button>
              <span v-else>{{ column.label }}</span>
            </th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="(row, index) in pagedRows"
            :key="rowId(row, index)"
            class="border-t border-[rgba(126,200,227,0.06)] align-top text-[#e8e8f0]/80 hover:bg-[rgba(126,200,227,0.04)]"
          >
            <td
              v-for="column in columns"
              :key="column.key"
              class="px-5 py-3"
              :class="[column.align === 'right' ? 'text-right' : '', column.cellClass]"
            >
              <slot :name="`cell-${column.key}`" :row="row" :value="row?.[column.key]">{{ row?.[column.key] }}</slot>
            </td>
          </tr>
          <tr v-if="!pagedRows.length">
            <td :colspan="columns.length" class="px-5 py-10 text-center text-[#7ec8e3]/40">{{ emptyText }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="paginate && totalPages > 1" class="flex items-center justify-between gap-3 border-t border-[rgba(126,200,227,0.1)] px-5 py-3">
      <span class="text-[12px] text-[#7ec8e3]/45">Page {{ page }} / {{ totalPages }} · {{ filteredRows.length }} rows</span>
      <div class="flex gap-2">
        <button type="button" @click="page = Math.max(1, page - 1)" :disabled="page <= 1" class="cursor-pointer rounded-lg border border-[rgba(126,200,227,0.16)] bg-[rgba(126,200,227,0.08)] px-3 py-1.5 text-[12px] font-semibold text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(126,200,227,0.3)] disabled:cursor-not-allowed disabled:opacity-40">Prev</button>
        <button type="button" @click="page = Math.min(totalPages, page + 1)" :disabled="page >= totalPages" class="cursor-pointer rounded-lg border border-[rgba(126,200,227,0.16)] bg-[rgba(126,200,227,0.08)] px-3 py-1.5 text-[12px] font-semibold text-[#f6f7fb] transition-all duration-200 hover:border-[rgba(126,200,227,0.3)] disabled:cursor-not-allowed disabled:opacity-40">Next</button>
      </div>
    </div>
  </div>
</template>
