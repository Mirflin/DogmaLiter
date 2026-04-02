<script setup>
import HomeLayout from '@/layouts/HomeLayout.vue'
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import api, { API_URL } from '@/api'

const auth = useAuthStore()

const router = useRouter()
const posts = ref([])
const loading = ref(true)
const total = ref(0)
const page = ref(1)
const limit = 12

const totalPages = computed(() => Math.ceil(total.value / limit))

onMounted(() => loadNews())

async function loadNews() {
  loading.value = true
  try {
    const { data } = await api.get('/news', { params: { limit, offset: (page.value - 1) * limit } })
    posts.value = data.posts || []
    total.value = data.total || 0
  } catch {} finally {
    loading.value = false
  }
}

function goToPage(p) {
  if (p < 1 || p > totalPages.value) return
  page.value = p
  loadNews()
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

function imageUrl(post) {
  if (post.image_id) return `${API_URL}/api/uploads/${post.image_id}`
  return null
}

function truncate(text, len = 120) {
  if (!text) return ''
  return text.length > len ? text.substring(0, len) + '…' : text
}

function formatDate(d) {
  if (!d) return ''
  return new Date(d).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
}

function getPreviewText(content) {
  try {
    const sections = JSON.parse(content)
    if (Array.isArray(sections) && sections.length > 0) return sections[0].text || ''
  } catch {}
  return content || ''
}
</script>

<template>
  <HomeLayout>
    <div class="max-w-[1400px] mx-auto px-6 py-8">
      <div class="flex items-center justify-between mb-8">
        <h1 class="font-[Cinzel] text-[28px] font-bold text-[#e8e8f0] tracking-wide">News</h1>
        <router-link
          v-if="auth.user?.role === 'admin'"
          to="/news/create"
          class="px-5 py-2.5 bg-linear-to-br from-[#e94560] to-[#c23152] text-white text-[13px] font-semibold no-underline rounded-lg border border-[#e94560] hover:-translate-y-0.5 hover:shadow-[0_6px_20px_rgba(233,69,96,0.4)] transition-all duration-300"
        >
          Create News
        </router-link>
      </div>

      <div v-if="loading" class="text-center py-16">
        <div class="w-8 h-8 border-2 border-[#e94560] border-t-transparent rounded-full animate-spin mx-auto"></div>
      </div>

      <div v-else-if="posts.length === 0" class="text-center py-16">
        <p class="text-[#7ec8e3]/40 text-[14px]">No news yet</p>
      </div>

      <div v-else class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
        <div
          v-for="post in posts"
          :key="post.id"
          @click="router.push(`/news/${post.id}`)"
          class="bg-[rgba(15,15,35,0.6)] border border-[rgba(126,200,227,0.1)] rounded-xl overflow-hidden hover:border-[rgba(233,69,96,0.3)] hover:-translate-y-1 transition-all duration-300 cursor-pointer group"
        >
          <div class="h-44 bg-[rgba(126,200,227,0.05)] flex items-center justify-center relative overflow-hidden">
            <img v-if="imageUrl(post)" :src="imageUrl(post)" :alt="post.title" class="w-full h-full object-cover" />
            <span v-else class="text-[#7ec8e3]/20 text-sm">No image</span>
            <div class="absolute inset-0 bg-linear-to-t from-[rgba(15,15,35,0.8)] to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-300" />
          </div>
          <div class="p-4">
            <p class="text-[#7ec8e3]/40 text-[11px] mb-2">{{ formatDate(post.published_at) }}</p>
            <h3 class="text-[#e8e8f0] text-[15px] font-semibold mb-2 line-clamp-2">{{ post.title }}</h3>
            <p class="text-[#e8e8f0]/50 text-[13px] leading-relaxed line-clamp-3">{{ truncate(getPreviewText(post.content), 150) }}</p>
          </div>
        </div>
      </div>
      <div v-if="!loading && totalPages > 1" class="flex items-center justify-center gap-2 mt-10">
        <button
          @click="goToPage(page - 1)"
          :disabled="page === 1"
          class="px-3 py-2 text-[13px] rounded-lg border border-[rgba(126,200,227,0.15)] bg-[rgba(15,15,35,0.6)] text-[#e8e8f0]/70 cursor-pointer hover:border-[rgba(233,69,96,0.3)] hover:text-[#e8e8f0] transition-all duration-200 disabled:opacity-30 disabled:cursor-not-allowed"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M15 19l-7-7 7-7" /></svg>
        </button>

        <button
          v-for="p in totalPages"
          :key="p"
          @click="goToPage(p)"
          :class="p === page ? 'bg-[#e94560] text-white border-[#e94560]' : 'bg-[rgba(15,15,35,0.6)] text-[#e8e8f0]/70 border-[rgba(126,200,227,0.15)] hover:border-[rgba(233,69,96,0.3)] hover:text-[#e8e8f0]'"
          class="w-9 h-9 text-[13px] font-medium rounded-lg border cursor-pointer transition-all duration-200"
        >
          {{ p }}
        </button>

        <button
          @click="goToPage(page + 1)"
          :disabled="page === totalPages"
          class="px-3 py-2 text-[13px] rounded-lg border border-[rgba(126,200,227,0.15)] bg-[rgba(15,15,35,0.6)] text-[#e8e8f0]/70 cursor-pointer hover:border-[rgba(233,69,96,0.3)] hover:text-[#e8e8f0] transition-all duration-200 disabled:opacity-30 disabled:cursor-not-allowed"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" /></svg>
        </button>
      </div>
    </div>
  </HomeLayout>
</template>
