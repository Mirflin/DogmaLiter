<script setup>
import { computed, ref, watch } from 'vue'
import SearchableSelect from '@/components/session/SearchableSelect.vue'

const props = defineProps({
  columns: { type: Array, default: () => [] },
  rows: { type: Array, default: () => [] },
  rowKey: { type: String, default: 'id' },
  searchable: { type: Boolean, default: true },
  searchPlaceholder: { type: String, default: 'Search...' },
  searchKeys: { type: Array, default: null },
  paginate: { type: Boolean, default: true },
  pageSize: { type: Number, default: 10 },
  pageSizeOptions: { type: Array, default: () => [5, 10, 20, 50] },
  minWidth: { type: String, default: '640px' },
  minHeight: { type: String, default: '' },
  emptyText: { type: String, default: 'No data' },
  fillHeight: { type: Boolean, default: false },
})

const search = ref('')
const sortKey = ref('')
const sortDir = ref('asc')
const columnFilters = ref({})
const page = ref(1)
const pageSize = ref(props.pageSize)
watch(() => props.pageSize, (value) => { pageSize.value = value })
watch(pageSize, () => { page.value = 1 })

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

function filterOptions(column) {
  return [{ value: '', label: 'All' }, ...distinctValues(column)]
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

const totalPages = computed(() => Math.max(1, Math.ceil(filteredRows.value.length / pageSize.value)))
const pageNumbers = computed(() => {
  const total = totalPages.value
  const current = page.value
  const delta = 1
  const range = []
  const withDots = []
  let last = 0

  for (let i = 1; i <= total; i++) {
    if (i === 1 || i === total || (i >= current - delta && i <= current + delta)) {
      range.push(i)
    }
  }

  for (const i of range) {
    if (last) {
      if (i - last === 2) withDots.push(last + 1)
      else if (i - last > 2) withDots.push('…')
    }
    withDots.push(i)
    last = i
  }

  return withDots
})
const pagedRows = computed(() => {
  if (!props.paginate) return filteredRows.value
  const start = (page.value - 1) * pageSize.value
  return filteredRows.value.slice(start, start + pageSize.value)
})
const fillerCount = computed(() => {
  if (!props.paginate || props.fillHeight) return 0
  const occupied = pagedRows.value.length || 1
  return Math.max(0, pageSize.value - occupied)
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
  <div class="overflow-hidden rounded-[1.5rem] border border-line bg-n1/78" :class="fillHeight ? 'flex h-full flex-col' : ''">
    <div v-if="hasToolbar || $slots.toolbar" class="flex shrink-0 flex-wrap items-center gap-3 border-b border-line-soft px-5 py-4">
      <div v-if="searchable" class="flex min-w-[13rem] flex-1 items-center gap-2 rounded-xl border border-line bg-n1/72 px-3 py-2">
        <svg class="h-4 w-4 shrink-0 text-faint" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-4.35-4.35M17 11a6 6 0 11-12 0 6 6 0 0112 0z" /></svg>
        <input v-model="search" type="text" :placeholder="searchPlaceholder" class="w-full bg-transparent text-[13px] text-ink outline-none placeholder:text-faint" />
      </div>

      <label v-for="column in filterableColumns" :key="column.key" class="flex items-center gap-2">
        <span class="text-[13px] font-semibold text-faint">{{ column.label }}</span>
        <SearchableSelect
          v-model="columnFilters[column.key]"
          :options="filterOptions(column)"
          :placeholder="`Search ${column.label.toLowerCase()}…`"
          class="min-w-[10rem]"
        />
      </label>

      <slot name="toolbar" />
    </div>

    <div class="overflow-x-auto" :class="fillHeight ? 'flex-1 min-h-0 overflow-y-auto' : ''" :style="minHeight ? { minHeight } : {}">
      <table class="w-full text-left text-[13px]" :style="{ minWidth }">
        <thead>
          <tr class="text-[13px] font-semibold text-faint" :class="fillHeight ? 'sticky top-0 z-10 bg-n1/98' : ''">
            <th
              v-for="column in columns"
              :key="column.key"
              class="px-5 py-3 font-medium"
              :class="[column.align === 'right' ? 'text-right' : '', column.headerClass]"
            >
              <button v-if="column.sortable" type="button" @click="toggleSort(column)" class="inline-flex cursor-pointer items-center gap-1 transition-colors hover:text-sky">
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
            class="h-[3.25rem] border-t border-line-soft align-middle text-ink/80 hover:bg-n2"
          >
            <td
              v-for="column in columns"
              :key="column.key"
              class="whitespace-nowrap px-5 py-2"
              :class="[column.align === 'right' ? 'text-right' : '', column.cellClass]"
            >
              <slot :name="`cell-${column.key}`" :row="row" :value="row?.[column.key]">{{ row?.[column.key] }}</slot>
            </td>
          </tr>
          <tr v-if="!pagedRows.length" class="h-[3.25rem]">
            <td :colspan="columns.length" class="px-5 py-2 text-center text-faint">{{ emptyText }}</td>
          </tr>
          <tr v-for="n in fillerCount" :key="`filler-${n}`" aria-hidden="true" class="h-[3.25rem] border-t border-line-soft">
            <td :colspan="columns.length" class="px-5 py-2">&nbsp;</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="paginate" class="flex shrink-0 flex-wrap items-center justify-between gap-3 border-t border-line-soft px-5 py-3">
      <div class="flex flex-wrap items-center gap-3">
        <span class="text-[12px] text-faint">{{ filteredRows.length }} rows</span>
        <label class="flex items-center gap-1.5 text-[12px] text-faint">
          Rows:
          <select
            v-model.number="pageSize"
            class="rounded-lg border border-line bg-n1/72 px-2 py-1 text-[12px] text-ink outline-none"
          >
            <option v-for="size in pageSizeOptions" :key="size" :value="size">{{ size }}</option>
          </select>
        </label>
      </div>
      <div class="flex flex-wrap items-center gap-1.5">
        <button type="button" @click="page = Math.max(1, page - 1)" :disabled="page <= 1" class="cursor-pointer rounded-lg border border-line bg-n3 px-3 py-1.5 text-[12px] font-semibold text-ink transition-all duration-200 hover:border-line-strong disabled:cursor-not-allowed disabled:opacity-40">Prev</button>
        <template v-for="(item, index) in pageNumbers" :key="`page-${index}`">
          <span v-if="item === '…'" class="px-1.5 text-[12px] text-faint">…</span>
          <button
            v-else
            type="button"
            @click="page = item"
            class="min-w-[2.15rem] cursor-pointer rounded-lg border px-2.5 py-1.5 text-[12px] font-semibold transition-all duration-200"
            :class="item === page
              ? 'border-primary/40 bg-primary/18 text-primary-soft'
              : 'border-line bg-n3 text-ink hover:border-line-strong'"
          >
            {{ item }}
          </button>
        </template>
        <button type="button" @click="page = Math.min(totalPages, page + 1)" :disabled="page >= totalPages" class="cursor-pointer rounded-lg border border-line bg-n3 px-3 py-1.5 text-[12px] font-semibold text-ink transition-all duration-200 hover:border-line-strong disabled:cursor-not-allowed disabled:opacity-40">Next</button>
      </div>
    </div>
  </div>
</template>
