<script setup>
import HomeLayout from '@/layouts/HomeLayout.vue'
import { useAuthStore } from '@/stores/auth'
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import api from '@/api'

const auth = useAuthStore()
const route = useRoute()
const plans = ref([])
const loading = ref(true)
const purchasing = ref(false)
const billingCycle = ref('monthly')
const successMessage = ref('')
const errorMessage = ref('')

const planAccents = {
  0: { border: 'rgba(126,200,227,0.15)', bg: 'rgba(126,200,227,0.03)', badge: null },
  1: { border: 'rgba(233,69,96,0.3)', bg: 'rgba(233,69,96,0.04)', badge: null },
  2: { border: 'rgba(160,120,255,0.35)', bg: 'rgba(160,120,255,0.05)', badge: 'Popular' },
}

function getAccent(index) {
  return planAccents[index] || planAccents[0]
}

function formatStorage(mb) {
  if (mb >= 1024) return (mb / 1024).toFixed(0) + ' GB'
  return mb + ' MB'
}

function formatPrice(price) {
  if (price === 0) return 'Free'
  return '$' + price.toFixed(2)
}

function yearlyPrice(monthly) {
  return (monthly * 10).toFixed(2)
}

const currentPlanId = computed(() => auth.user?.plan_id)
const currentPlan = computed(() => plans.value.find(p => p.id === currentPlanId.value))

function isLowerTier(plan) {
  if (!currentPlan.value) return false
  return plan.price_monthly < currentPlan.value.price_monthly
}

function planFeatures(plan) {
  return [
    { label: 'Games owned', value: plan.max_games_owned === -1 ? 'Unlimited' : String(plan.max_games_owned) },
    { label: 'Players per game', value: plan.max_players_per_game === -1 ? 'Unlimited' : String(plan.max_players_per_game) },
    { label: 'Maps per game', value: plan.max_maps_per_game === -1 ? 'Unlimited' : String(plan.max_maps_per_game) },
    { label: 'Items per game', value: plan.max_items_per_game === -1 ? 'Unlimited' : String(plan.max_items_per_game) },
    { label: 'Characters per game', value: plan.max_characters_per_game === -1 ? 'Unlimited' : String(plan.max_characters_per_game) },
    { label: 'Max file size', value: plan.max_upload_size_mb + ' MB' },
    { label: 'Upload storage', value: formatStorage(plan.storage_limit_mb) },
  ]
}

async function selectPlan(plan) {
  if (plan.price_monthly === 0 || currentPlanId.value === plan.id) return

  purchasing.value = true
  errorMessage.value = ''
  try {
    const { data } = await api.post('/payment/checkout', {
      plan_id: plan.id,
      billing_cycle: billingCycle.value,
    })
    window.location.href = data.url
  } catch (err) {
    errorMessage.value = err.response?.data?.error || 'Failed to start checkout'
  } finally {
    purchasing.value = false
  }
}

onMounted(async () => {
  if (route.query.success === 'true') {
    successMessage.value = 'Payment successful! Your plan has been upgraded.'
    await auth.fetchProfile()
  }
  if (route.query.canceled === 'true') {
    errorMessage.value = 'Payment was canceled.'
  }

  try {
    const { data } = await api.get('/plans')
    plans.value = data
  } catch {
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <HomeLayout>
    <div class="max-w-[1200px] mx-auto px-6 py-16">
      <h1 class="font-[Cinzel] text-[32px] md:text-[40px] font-bold text-center text-[#e8e8f0] tracking-wide mb-3">
        Choose Your Plan
      </h1>
      <p class="text-center text-[#7ec8e3]/50 text-[16px] mb-10 max-w-lg mx-auto">
        Flexible pricing for every play style.
      </p>

      <div v-if="successMessage" class="max-w-lg mx-auto mb-8 p-4 rounded-xl bg-green-500/10 border border-green-500/20 text-green-400 text-[14px] text-center">
        {{ successMessage }}
      </div>
      <div v-if="errorMessage" class="max-w-lg mx-auto mb-8 p-4 rounded-xl bg-[rgba(233,69,96,0.1)] border border-[rgba(233,69,96,0.2)] text-[#e94560] text-[14px] text-center">
        {{ errorMessage }}
      </div>
      <div class="flex items-center justify-center gap-0 mb-12">
        <button
          @click="billingCycle = 'monthly'"
          class="px-6 py-2.5 text-[14px] font-medium border border-[rgba(126,200,227,0.2)] rounded-l-xl cursor-pointer transition-all duration-200"
          :class="billingCycle === 'monthly'
            ? 'bg-[rgba(233,69,96,0.15)] text-[#e94560] border-[rgba(233,69,96,0.4)]'
            : 'bg-transparent text-[#7ec8e3]/50 hover:text-[#e8e8f0]'"
        >
          Monthly
        </button>
        <button
          @click="billingCycle = 'yearly'"
          class="relative px-6 py-2.5 text-[14px] font-medium border border-[rgba(126,200,227,0.2)] border-l-0 rounded-r-xl cursor-pointer transition-all duration-200"
          :class="billingCycle === 'yearly'
            ? 'bg-[rgba(233,69,96,0.15)] text-[#e94560] border-[rgba(233,69,96,0.4)]'
            : 'bg-transparent text-[#7ec8e3]/50 hover:text-[#e8e8f0]'"
        >
          Yearly
          <span class="absolute -top-3 -right-3 bg-[#e94560] text-white text-[10px] font-bold px-2 py-0.5 rounded-full whitespace-nowrap">
            2 months free
          </span>
        </button>
      </div>
      <div v-if="loading" class="text-center py-20">
        <div class="inline-block w-8 h-8 border-2 border-[#e94560] border-t-transparent rounded-full animate-spin" />
      </div>
      <div v-else class="grid gap-6" :class="plans.length <= 3 ? 'md:grid-cols-3' : 'md:grid-cols-4'">
        <div
          v-for="(plan, index) in plans"
          :key="plan.id"
          class="relative flex flex-col rounded-2xl border p-7 transition-all duration-300 hover:-translate-y-1 hover:shadow-[0_20px_60px_rgba(0,0,0,0.4)]"
          :style="{
            borderColor: getAccent(index).border,
            backgroundColor: getAccent(index).bg,
          }"
        >
          <span
            v-if="getAccent(index).badge"
            class="absolute -top-3 left-1/2 -translate-x-1/2 bg-[#a078ff] text-white text-[11px] font-bold px-3 py-1 rounded-full tracking-wide"
          >
            {{ getAccent(index).badge }}
          </span>
          <div class="mb-6">
            <p class="text-[#7ec8e3]/60 text-[13px] font-semibold uppercase tracking-widest mb-1">
              {{ plan.name }}
            </p>
            <div class="flex items-baseline gap-1">
              <span class="font-[Cinzel] text-[36px] font-bold text-[#e8e8f0]">
                {{ billingCycle === 'monthly' ? formatPrice(plan.price_monthly) : (plan.price_monthly === 0 ? 'Free' : '$' + yearlyPrice(plan.price_monthly)) }}
              </span>
              <span v-if="plan.price_monthly > 0" class="text-[#7ec8e3]/40 text-[13px]">
                /{{ billingCycle === 'monthly' ? 'mo' : 'yr' }}
              </span>
            </div>
            <p v-if="billingCycle === 'yearly' && plan.price_monthly > 0" class="text-[#7ec8e3]/30 text-[12px] mt-1">
              ${{ (plan.price_monthly * 10 / 12).toFixed(2) }}/mo billed yearly
            </p>
          </div>
          <button
            @click="selectPlan(plan)"
            class="w-full py-3 rounded-xl text-[14px] font-semibold cursor-pointer border transition-all duration-200 mb-7"
            :class="currentPlanId === plan.id || isLowerTier(plan)
              ? 'bg-transparent border-[rgba(126,200,227,0.2)] text-[#7ec8e3]/50 cursor-default'
              : index === 2
                ? 'bg-[#a078ff] border-[#a078ff] text-white hover:bg-[#8d62ff]'
                : index === 1
                  ? 'bg-[#e94560] border-[#e94560] text-white hover:bg-[#d63b55]'
                  : 'bg-transparent border-[rgba(126,200,227,0.25)] text-[#e8e8f0] hover:bg-[rgba(126,200,227,0.08)]'"
            :disabled="currentPlanId === plan.id || plan.price_monthly === 0 || purchasing || isLowerTier(plan)"
          >
            <template v-if="currentPlanId === plan.id">Current plan</template>
            <template v-else-if="isLowerTier(plan)">Not available</template>
            <template v-else-if="plan.price_monthly === 0">Free</template>
            <template v-else-if="purchasing">Processing...</template>
            <template v-else>Subscribe to {{ plan.name }}</template>
          </button>
          <ul class="space-y-3.5 list-none p-0 m-0 flex-1">
            <li
              v-for="feature in planFeatures(plan)"
              :key="feature.label"
              class="flex items-start gap-3"
            >
              <svg class="w-4 h-4 mt-0.5 flex-shrink-0 text-[#7ec8e3]/50" fill="none" stroke="currentColor" stroke-width="2.5" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
              </svg>
              <div>
                <span class="text-[#e8e8f0]/80 text-[14px]">{{ feature.label }}</span>
                <p class="text-[#7ec8e3]/40 text-[12px] m-0">{{ feature.value }}</p>
              </div>
            </li>
          </ul>
        </div>
      </div>
    </div>
  </HomeLayout>
</template>
