<script setup>
import HomeLayout from '@/layouts/HomeLayout.vue'
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import api from '@/api'

const router = useRouter()
const title = ref('')
const sections = ref([{ subtitle: '', text: '' }])
const imageFile = ref(null)
const imagePreview = ref(null)
const submitting = ref(false)
const error = ref(null)

function addSection() {
  sections.value.push({ subtitle: '', text: '' })
}

function removeSection(index) {
  if (sections.value.length <= 1) return
  sections.value.splice(index, 1)
}

function onImageChange(e) {
  const file = e.target.files[0]
  if (!file) return
  imageFile.value = file
  imagePreview.value = URL.createObjectURL(file)
}

function removeImage() {
  imageFile.value = null
  imagePreview.value = null
}

async function submit() {
  if (!title.value.trim()) {
    error.value = 'Title is required'
    return
  }
  const hasContent = sections.value.some(s => s.text.trim())
  if (!hasContent) {
    error.value = 'At least one section must have content'
    return
  }

  submitting.value = true
  error.value = null

  try {
    const formData = new FormData()
    formData.append('title', title.value.trim())
    formData.append('content', JSON.stringify(sections.value.map(s => ({
      subtitle: s.subtitle.trim(),
      text: s.text.trim(),
    })).filter(s => s.text)))
    if (imageFile.value) {
      formData.append('image', imageFile.value)
    }

    const { data } = await api.post('/news', formData, {
      headers: { 'Content-Type': undefined },
    })

    router.push(`/news/${data.id}`)
  } catch (err) {
    error.value = err.response?.data?.error || 'Failed to create news post'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <HomeLayout>
    <div class="max-w-[800px] mx-auto px-6 py-8">
      <h1 class="font-[Cinzel] text-[28px] font-bold text-[#e8e8f0] tracking-wide mb-8">Create News Post</h1>

      <div v-if="error" class="mb-6 p-4 bg-[rgba(233,69,96,0.1)] border border-[rgba(233,69,96,0.3)] rounded-lg text-[#e94560] text-[13px]">
        {{ error }}
      </div>

      <div class="space-y-6">
        <div>
          <label class="block text-[#e8e8f0]/60 text-[13px] font-medium mb-2">Title</label>
          <input
            v-model="title"
            type="text"
            maxlength="300"
            placeholder="News title..."
            class="w-full px-4 py-3 bg-[rgba(15,15,35,0.6)] border border-[rgba(126,200,227,0.15)] rounded-lg text-[#e8e8f0] text-[14px] placeholder-[#7ec8e3]/25 focus:outline-none focus:border-[#e94560] transition-colors"
          />
        </div>

        <div>
          <label class="block text-[#e8e8f0]/60 text-[13px] font-medium mb-2">Cover Image</label>
          <div v-if="imagePreview" class="relative mb-3">
            <img :src="imagePreview" class="w-full h-48 object-cover rounded-lg" />
            <button
              @click="removeImage"
              class="absolute top-2 right-2 w-8 h-8 bg-[rgba(0,0,0,0.6)] rounded-full flex items-center justify-center text-white border-none cursor-pointer hover:bg-[rgba(233,69,96,0.8)] transition-colors"
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
            </button>
          </div>
          <label v-else class="flex items-center justify-center w-full h-36 bg-[rgba(15,15,35,0.6)] border-2 border-dashed border-[rgba(126,200,227,0.15)] rounded-lg cursor-pointer hover:border-[rgba(233,69,96,0.3)] transition-colors">
            <div class="text-center">
              <svg class="w-8 h-8 mx-auto text-[#7ec8e3]/30 mb-2" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M2.25 15.75l5.159-5.159a2.25 2.25 0 013.182 0l5.159 5.159m-1.5-1.5l1.409-1.409a2.25 2.25 0 013.182 0l2.909 2.909M3.75 21h16.5A2.25 2.25 0 0022.5 18.75V5.25A2.25 2.25 0 0020.25 3H3.75A2.25 2.25 0 001.5 5.25v13.5A2.25 2.25 0 003.75 21z" /></svg>
              <span class="text-[#7ec8e3]/30 text-[13px]">Click to upload image</span>
            </div>
            <input type="file" accept="image/*" class="hidden" @change="onImageChange" />
          </label>
        </div>

        <div>
          <div class="flex items-center justify-between mb-2">
            <label class="text-[#e8e8f0]/60 text-[13px] font-medium">Sections</label>
            <button
              @click="addSection"
              type="button"
              class="flex items-center gap-1.5 px-3 py-1.5 bg-[rgba(126,200,227,0.08)] text-[#7ec8e3]/70 text-[12px] font-medium rounded-lg border border-[rgba(126,200,227,0.15)] hover:border-[rgba(126,200,227,0.3)] hover:text-[#7ec8e3] transition-all duration-300 cursor-pointer"
            >
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" /></svg>
              Add Section
            </button>
          </div>

          <div class="space-y-4">
            <div
              v-for="(section, i) in sections"
              :key="i"
              class="bg-[rgba(15,15,35,0.4)] border border-[rgba(126,200,227,0.1)] rounded-lg p-4 relative"
            >
              <div class="flex items-center justify-between mb-3">
                <span class="text-[#7ec8e3]/30 text-[11px] font-medium tracking-wider uppercase">Section {{ i + 1 }}</span>
                <button
                  v-if="sections.length > 1"
                  @click="removeSection(i)"
                  type="button"
                  class="w-6 h-6 flex items-center justify-center rounded-full bg-transparent border-none text-[#7ec8e3]/30 hover:text-[#e94560] hover:bg-[rgba(233,69,96,0.1)] transition-all cursor-pointer"
                >
                  <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
                </button>
              </div>
              <input
                v-model="section.subtitle"
                type="text"
                maxlength="200"
                placeholder="Subtitle (optional)"
                class="w-full px-3 py-2 mb-3 bg-[rgba(15,15,35,0.6)] border border-[rgba(126,200,227,0.12)] rounded-lg text-[#e8e8f0] text-[13px] placeholder-[#7ec8e3]/25 focus:outline-none focus:border-[#e94560] transition-colors"
              />
              <textarea
                v-model="section.text"
                rows="6"
                placeholder="Section text..."
                class="w-full px-3 py-2 bg-[rgba(15,15,35,0.6)] border border-[rgba(126,200,227,0.12)] rounded-lg text-[#e8e8f0] text-[13px] placeholder-[#7ec8e3]/25 focus:outline-none focus:border-[#e94560] transition-colors resize-y"
              />
            </div>
          </div>
        </div>

        <div class="flex gap-3">
          <button
            @click="submit"
            :disabled="submitting"
            class="px-6 py-3 bg-linear-to-br from-[#e94560] to-[#c23152] text-white text-[14px] font-semibold rounded-lg border-none cursor-pointer hover:-translate-y-0.5 hover:shadow-[0_6px_20px_rgba(233,69,96,0.4)] transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed disabled:transform-none"
          >
            {{ submitting ? 'Publishing...' : 'Publish' }}
          </button>
          <button
            @click="router.push('/dashboard')"
            class="px-6 py-3 bg-transparent text-[#e8e8f0]/60 text-[14px] font-semibold rounded-lg border border-[rgba(126,200,227,0.15)] cursor-pointer hover:border-[rgba(126,200,227,0.3)] hover:text-[#e8e8f0] transition-all duration-300"
          >
            Cancel
          </button>
        </div>
      </div>
    </div>
  </HomeLayout>
</template>
