<script setup>
import HomeLayout from '@/layouts/HomeLayout.vue'
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import api, { API_URL } from '@/api'

const route = useRoute()
const router = useRouter()
const post = ref(null)
const loading = ref(true)
const error = ref(null)

onMounted(async () => {
  try {
    const { data } = await api.get(`/news/${route.params.id}`)
    post.value = data
  } catch {
    error.value = 'News post not found'
  } finally {
    loading.value = false
  }
})

function imageUrl(id) {
  return `${API_URL}/api/uploads/${id}`
}

function formatDate(d) {
  if (!d) return ''
  return new Date(d).toLocaleDateString('en-US', { month: 'long', day: 'numeric', year: 'numeric' })
}

function parseSections(content) {
  try {
    const sections = JSON.parse(content)
    if (Array.isArray(sections)) return sections
  } catch {}
  return [{ subtitle: '', text: content || '' }]
}
</script>

<template>
  <HomeLayout>
    <div class="max-w-[900px] mx-auto px-6 py-8">
      <button
        @click="router.push('/news')"
        class="flex items-center gap-2 text-[#7ec8e3]/50 text-[13px] mb-6 bg-transparent border-none cursor-pointer hover:text-[#7ec8e3] transition-colors"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M15 19l-7-7 7-7" /></svg>
        Back to News
      </button>

      <div v-if="loading" class="text-center py-16">
        <div class="w-8 h-8 border-2 border-[#e94560] border-t-transparent rounded-full animate-spin mx-auto"></div>
      </div>

      <div v-else-if="error" class="text-center py-16">
        <p class="text-[#e94560] text-[14px]">{{ error }}</p>
      </div>

      <article v-else-if="post">
        <div v-if="post.image_id" class="w-full h-72 md:h-96 rounded-xl overflow-hidden mb-8">
          <img :src="imageUrl(post.image_id)" :alt="post.title" class="w-full h-full object-cover" />
        </div>

        <div class="flex items-center gap-3 mb-4">
          <span class="text-[#7ec8e3]/40 text-[12px]">{{ formatDate(post.published_at) }}</span>
          <span v-if="post.author" class="text-[#7ec8e3]/30 text-[12px]">by {{ post.author.username }}</span>
        </div>

        <h1 class="font-[Cinzel] text-[28px] md:text-[36px] font-bold text-[#e8e8f0] tracking-wide mb-8 leading-tight">{{ post.title }}</h1>

        <div class="space-y-8">
          <div v-for="(section, i) in parseSections(post.content)" :key="i">
            <h2 v-if="section.subtitle" class="font-[Cinzel] text-[20px] md:text-[24px] font-semibold text-[#e8e8f0]/90 tracking-wide mb-3">{{ section.subtitle }}</h2>
            <p class="text-[#e8e8f0]/70 text-[15px] leading-relaxed whitespace-pre-line">{{ section.text }}</p>
          </div>
        </div>
      </article>
    </div>
  </HomeLayout>
</template>
