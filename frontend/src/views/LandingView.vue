<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import HomeLayout from '@/layouts/HomeLayout.vue'
import LandingDie from '@/components/landing/LandingDie.vue'
import LandingInventoryDemo from '@/components/landing/LandingInventoryDemo.vue'
import LandingSky from '@/components/landing/LandingSky.vue'
import api, { API_URL } from '@/api'

const news = ref([])
const freePlan = ref(null)
const page = ref(null)
const armed = ref(false)
const trade = ref('pending')

const playerTools = [
  { title: 'Your character, always up to date', text: 'Stats, health and gear on one sheet that keeps itself current.' },
  { title: 'A backpack with real space', text: 'Items take the room they should. Pack, stack and equip by dragging.' },
  { title: 'Trade with your party', text: 'Swap items with other players safely, right inside the session.' },
  { title: 'Chat that knows your loot', text: 'Talk, plan and show off a find without leaving the game.' },
]

const gmTools = [
  { title: 'Your own item library', text: 'Create the items of your world once and use them all campaign.' },
  { title: 'Hand out loot in bulk', text: 'Give many items at once, straight into a character’s backpack.' },
  { title: 'Know what happened', text: 'Trades, loot and messages stay on record for the whole game.' },
  { title: 'Your table, your rules', text: 'Invite players with a link, pick helpers and switch features on or off.' },
]

const steps = [
  { title: 'Create an account', text: 'Free, with no card needed.' },
  { title: 'Start a campaign', text: 'Name it, set the rules, share the invite code.' },
  { title: 'Play', text: 'Characters, loot and chat stay in one tab all night.' },
]

const freeLine = computed(() => {
  const plan = freePlan.value
  if (!plan) return ''
  const games = plan.max_games_owned === -1 ? 'unlimited campaigns' : `${plan.max_games_owned} campaigns`
  const players = plan.max_players_per_game === -1 ? 'unlimited players' : `up to ${plan.max_players_per_game} players each`
  return `Free plan: ${games}, ${players}.`
})

let observer = null

onMounted(async () => {
  if (!window.matchMedia('(prefers-reduced-motion: reduce)').matches && 'IntersectionObserver' in window) {
    armed.value = true
    observer = new IntersectionObserver((entries) => {
      for (const entry of entries) {
        if (entry.isIntersecting) {
          entry.target.classList.add('is-in')
          observer.unobserve(entry.target)
        }
      }
    }, { rootMargin: '0px 0px -12% 0px' })
    page.value?.querySelectorAll('[data-reveal]').forEach((el) => observer.observe(el))
  }

  const [newsResult, plansResult] = await Promise.allSettled([
    api.get('/news', { params: { limit: 3, offset: 0 } }),
    api.get('/plans'),
  ])
  if (newsResult.status === 'fulfilled') news.value = newsResult.value.data.posts || []
  if (plansResult.status === 'fulfilled') freePlan.value = (plansResult.value.data || []).find((plan) => plan.id === 'free') || null
})

onBeforeUnmount(() => observer?.disconnect())

function getPreviewText(content) {
  try {
    const sections = JSON.parse(content)
    return sections[0]?.text || ''
  } catch {
    return content || ''
  }
}

function formatDate(value) {
  return new Date(value).toLocaleDateString('en-US', { year: 'numeric', month: 'short', day: 'numeric' })
}
</script>

<template>
  <HomeLayout nav-overlay :backdrop="false">
    <div ref="page" class="landing" :class="{ 'landing--armed': armed }">
      <LandingSky />

      <!-- Hero -->
      <section class="hero" data-tilt-scope>
        <div class="hero__inner">
          <div class="hero__copy">
            <h1 class="hero__title">
              <span class="hero__line"><span>Your Tabletop,</span></span>
              <span class="hero__line"><span>Reimagined.</span></span>
            </h1>
            <p class="hero__lede">
              Characters, grid inventories, loot and trades for your whole party, in one browser tab. No downloads, no hosting.
            </p>
            <div class="hero__actions">
              <router-link to="/auth?mode=register" class="btn-primary group">
                Create free account
                <svg class="h-4 w-4 transition-transform duration-200 group-hover:translate-x-0.5" fill="none" stroke="currentColor" stroke-width="2.4" viewBox="0 0 24 24" aria-hidden="true"><path stroke-linecap="round" stroke-linejoin="round" d="M13.5 4.5 21 12m0 0-7.5 7.5M21 12H3" /></svg>
              </router-link>
              <router-link to="/auth" class="btn-quiet">I have an account</router-link>
            </div>
            <p class="hero__fine" :class="{ 'opacity-0': !freeLine }">{{ freeLine || 'Free plan available.' }}</p>
          </div>

          <div class="hero__tray">
            <div class="tray" aria-hidden="true" />
            <LandingDie class="hero__die" />
          </div>
        </div>
      </section>

      <!-- Inventory proof -->
      <section id="inventory" class="section scroll-mt-20">
        <div class="section__inner">
          <div class="section__head" data-reveal>
            <h2 class="section__title">Every item takes its real space</h2>
            <p class="section__lede">
              Items have a real size, so packing your bag is part of the game. Drag gear onto your character and the stats follow.
            </p>
          </div>
          <LandingInventoryDemo />
          <div class="demo-foot" data-reveal>
            <div>
              <ul class="rarity" aria-label="Rarity tiers">
                <li v-for="r in ['common', 'uncommon', 'rare', 'epic', 'masterwork', 'legendary', 'unique']" :key="r" :style="{ '--resin': `var(--color-r-${r})` }">
                  {{ r.charAt(0).toUpperCase() + r.slice(1) }}
                </li>
              </ul>
            </div>
            <p class="section__note">
              This is a simplified demo to try the idea. The inventory in a real game session looks different and does more, like rotating items, splitting stacks and trading.
            </p>
          </div>
        </div>
      </section>

      <!-- Players and GMs -->
      <section class="section">
        <div class="section__inner">
          <h2 class="section__title" data-reveal>Players and GMs, one table</h2>

          <div class="roles">
            <div class="roles__col" data-reveal="list">
              <h3 class="roles__who">For players</h3>
              <ul>
                <li v-for="(tool, i) in playerTools" :key="tool.title" :style="{ '--d': `${i * 80}ms` }">
                  <strong>{{ tool.title }}</strong>
                  <span>{{ tool.text }}</span>
                </li>
              </ul>
            </div>
            <div class="roles__col" data-reveal="list">
              <h3 class="roles__who">For game masters</h3>
              <ul>
                <li v-for="(tool, i) in gmTools" :key="tool.title" :style="{ '--d': `${i * 80}ms` }">
                  <strong>{{ tool.title }}</strong>
                  <span>{{ tool.text }}</span>
                </li>
              </ul>
            </div>
          </div>

          <div class="offer" data-reveal :data-state="trade">
            <span class="offer__swatch num" aria-hidden="true">CT</span>
            <p v-if="trade === 'pending'" class="offer__text">
              <span class="text-orchid font-semibold">Brannoc</span> offers you
              <span class="text-r-masterwork font-semibold">Cloak of the Tide</span>. It stays safe until you answer.
            </p>
            <p v-else-if="trade === 'accepted'" class="offer__text">
              <span class="text-r-masterwork font-semibold">Cloak of the Tide</span> is in Kaela’s backpack. Brannoc gets a heads-up.
            </p>
            <p v-else class="offer__text">Declined. The cloak goes back to Brannoc’s inventory.</p>
            <div class="offer__actions">
              <template v-if="trade === 'pending'">
                <button type="button" class="btn-quiet btn-sm" @click="trade = 'declined'">Decline</button>
                <button type="button" class="btn-primary btn-sm" @click="trade = 'accepted'">Accept</button>
              </template>
              <button v-else type="button" class="btn-quiet btn-sm" @click="trade = 'pending'">Show the offer again</button>
            </div>
          </div>
        </div>
      </section>

      <!-- Steps -->
      <section class="section">
        <div class="section__inner">
          <h2 class="section__title" data-reveal>From sign-up to first roll</h2>
          <ol class="steps" data-reveal="list">
            <li v-for="(step, i) in steps" :key="step.title" :style="{ '--d': `${i * 90}ms` }">
              <span class="steps__n num" aria-hidden="true">{{ i + 1 }}</span>
              <strong>{{ step.title }}</strong>
              <span>{{ step.text }}</span>
            </li>
          </ol>
        </div>
      </section>

      <!-- News -->
      <section v-if="news.length" class="section">
        <div class="section__inner">
          <div class="flex items-baseline justify-between gap-6" data-reveal>
            <h2 class="section__title">From the workshop</h2>
            <router-link to="/news" class="link">All news</router-link>
          </div>
          <div class="news">
            <router-link v-for="post in news" :key="post.id" :to="`/news/${post.id}`" class="news__item">
              <div class="news__img">
                <img v-if="post.image_id" :src="`${API_URL}/api/uploads/${post.image_id}`" :alt="post.title" />
              </div>
              <p class="text-[13px] text-faint">{{ formatDate(post.created_at) }}</p>
              <h3 class="mt-1 text-[17px] font-semibold text-ink">{{ post.title }}</h3>
              <p class="mt-1.5 text-[14px] leading-relaxed text-dim line-clamp-2">{{ getPreviewText(post.content) }}</p>
            </router-link>
          </div>
        </div>
      </section>

      <!-- Close -->
      <section class="close">
        <div class="close__inner" data-reveal>
          <h2 class="close__title">Roll for initiative.</h2>
          <p class="close__lede">Set up your campaign tonight. Your players only need the invite code.</p>
          <div class="hero__actions justify-center">
            <router-link to="/auth?mode=register" class="btn-primary">Create free account</router-link>
            <router-link to="/plans" class="btn-quiet">Compare plans</router-link>
          </div>
        </div>
      </section>
    </div>
  </HomeLayout>
</template>

<style scoped>
.landing {
  position: relative;
  /* Clips the fixed sky to the landing page, so the nav and footer stay solid. */
  clip-path: inset(0);
}

.landing > section {
  position: relative;
  z-index: 1;
}

/* ---------- Buttons ---------- */
.btn-primary,
.btn-quiet {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  min-height: 3.25rem;
  padding: 0 1.6rem;
  border-radius: 0.85rem;
  font: inherit;
  font-size: 16px;
  font-weight: 600;
  text-decoration: none;
  cursor: pointer;
  transition: background-color 0.2s ease, color 0.2s ease, transform 0.16s ease, box-shadow 0.2s ease;
}

.btn-primary {
  border: 0;
  color: #fff;
  background: var(--color-primary);
}

.btn-primary:hover {
  background: var(--color-primary-hover);
}

.btn-quiet {
  border: 0;
  color: var(--color-ink);
  background: var(--color-n3);
}

.btn-quiet:hover {
  background: var(--color-n4);
}

.btn-primary:active,
.btn-quiet:active {
  transform: scale(0.97);
}

.btn-sm {
  min-height: 2.5rem;
  padding: 0 1.1rem;
  font-size: 14px;
  border-radius: 0.65rem;
}

.link {
  color: var(--color-primary-ink);
  font-weight: 600;
  text-decoration: underline;
  text-decoration-color: rgb(164 174 255 / 0.35);
  text-underline-offset: 4px;
  transition: text-decoration-color 0.2s ease;
}

.link:hover {
  text-decoration-color: currentColor;
}

/* ---------- Hero ---------- */
.hero {
  position: relative;
  overflow: hidden;
}

.hero__inner {
  max-width: 1280px;
  margin: 0 auto;
  /* The nav floats over the hero, so the hero owns the full first screen. */
  padding: calc(4rem + clamp(2.5rem, 6vh, 5rem)) 1.5rem clamp(3.5rem, 8vh, 6rem);
  min-height: 100svh;
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1.05fr);
  align-items: center;
  gap: clamp(2rem, 5vw, 5rem);
}

.hero__copy {
  container-type: inline-size;
}

.hero__title {
  margin: 0;
  font-family: var(--font-num);
  font-weight: 700;
  font-size: clamp(2.4rem, 14.2cqi, 5.6rem);
  line-height: 0.92;
  letter-spacing: -0.01em;
  text-transform: uppercase;
  color: var(--color-ink);
}

.hero__line {
  display: block;
  overflow: hidden;
  padding-bottom: 0.06em;
}

.hero__line > span {
  display: block;
  white-space: nowrap;
}

.hero__line:last-child {
  color: var(--color-primary-ink);
}

.landing--armed .hero__line > span {
  animation: line-rise 0.9s cubic-bezier(0.16, 1, 0.3, 1) both;
}

.landing--armed .hero__line:nth-child(2) > span {
  animation-delay: 0.09s;
}

@keyframes line-rise {
  from { transform: translateY(105%); }
  to { transform: none; }
}

.hero__lede {
  margin: 1.75rem 0 0;
  max-width: 32rem;
  font-size: clamp(18px, 1.35vw, 20px);
  line-height: 1.55;
  color: var(--color-ink-soft);
}

.hero__actions {
  margin-top: 2.25rem;
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
}

.hero__fine {
  margin: 1.1rem 0 0;
  font-size: 14px;
  color: var(--color-faint);
  transition: opacity 0.3s ease;
}

.landing--armed .hero__lede,
.landing--armed .hero__actions,
.landing--armed .hero__fine {
  animation: fade-up 0.8s cubic-bezier(0.16, 1, 0.3, 1) both;
}

.landing--armed .hero__lede { animation-delay: 0.22s; }
.landing--armed .hero__actions { animation-delay: 0.32s; }
.landing--armed .hero__fine { animation-delay: 0.4s; }

@keyframes fade-up {
  from { opacity: 0; transform: translateY(14px); }
  to { opacity: 1; transform: none; }
}

.hero__tray {
  position: relative;
  display: grid;
  place-items: center;
  aspect-ratio: 1;
  width: 100%;
  max-width: 640px;
  justify-self: end;
}

/* A round dice tray pressed into the page. */
.tray {
  position: absolute;
  inset: 4%;
  border-radius: 50%;
  background: var(--color-n0);
  box-shadow:
    inset 0 18px 50px rgb(0 0 0 / 0.55),
    inset 0 0 0 1px var(--color-n3),
    0 0 0 14px var(--color-n2),
    0 0 0 15px var(--color-n3);
}

.hero__die {
  position: relative;
  width: 70%;
  margin-top: -4%;
}

/* ---------- Sections ---------- */
.section {
  padding: clamp(1.25rem, 3vw, 2.25rem) 1.5rem;
}

/* Each section sits in its own translucent panel over the night scene. */
.section__inner,
.close__inner {
  max-width: 1180px;
  margin: 0 auto;
  padding: clamp(1.75rem, 4.5vw, 4rem);
  border-radius: 2rem;
  background: rgb(8 13 36 / 0.82);
  box-shadow: inset 0 0 0 1px rgb(74 90 150 / 0.35);
}

@media (min-width: 961px) {
  .section__inner,
  .close__inner {
    backdrop-filter: blur(10px);
    -webkit-backdrop-filter: blur(10px);
  }
}

.section__head {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 1.5rem 4rem;
  align-items: end;
  margin-bottom: 2.5rem;
}

.section__title {
  margin: 0;
  font-size: clamp(2rem, 3.6vw, 3rem);
  line-height: 1.05;
  font-weight: 700;
  letter-spacing: -0.025em;
  color: var(--color-ink);
  text-wrap: balance;
}

.demo-foot {
  margin-top: 1.5rem;
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1rem 3rem;
}

.demo-foot .rarity {
  margin: 0;
}

.section__note {
  margin: 0;
  max-width: 21rem;
  font-size: 14px;
  line-height: 1.55;
  text-align: right;
  color: var(--color-faint);
}

@media (max-width: 960px) {
  .demo-foot {
    flex-direction: column;
  }

  .section__note {
    text-align: left;
  }
}

.section__lede {
  margin: 0;
  max-width: 36rem;
  font-size: 17px;
  line-height: 1.6;
  color: var(--color-dim);
}

.rarity {
  margin: 1.5rem 0 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem 1.25rem;
  font-size: 14px;
  color: var(--color-dim);
}

.credit {
  margin: 1rem 0 0;
  font-size: 13px;
  color: var(--color-faint);
}

.credit a {
  color: var(--color-dim);
  text-decoration: underline;
  text-decoration-color: rgb(154 165 207 / 0.35);
  text-underline-offset: 3px;
}

.rarity li {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
}

.rarity li::before {
  content: '';
  width: 0.8rem;
  height: 0.8rem;
  border-radius: 0.25rem;
  background: color-mix(in oklab, var(--resin) 40%, var(--color-n2));
  box-shadow: inset 0 0 0 1.5px var(--resin);
}

/* Roles */
.roles {
  margin-top: 3rem;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 3rem 4.5rem;
}

.roles__who {
  margin: 0 0 1.25rem;
  font-size: 15px;
  font-weight: 600;
  color: var(--color-primary-ink);
}

.roles ul {
  margin: 0;
  padding: 0;
  list-style: none;
  display: grid;
  gap: 1.5rem;
}

.roles li,
.steps li {
  display: grid;
  gap: 0.3rem;
}

.roles strong,
.steps strong {
  font-size: 18px;
  font-weight: 600;
  color: var(--color-ink);
}

.roles li span,
.steps li > span:last-child {
  font-size: 16px;
  line-height: 1.6;
  color: var(--color-dim);
  max-width: 34rem;
}

.offer {
  margin-top: 3.5rem;
  display: flex;
  align-items: center;
  gap: 1rem 1.25rem;
  flex-wrap: wrap;
  padding: 1.1rem 1.25rem;
  border-radius: 1rem;
  background: var(--color-n2);
  box-shadow: inset 0 0 0 1px rgb(232 144 210 / 0.35);
  transition: box-shadow 0.3s ease;
}

.offer[data-state='accepted'] {
  box-shadow: inset 0 0 0 1px rgb(98 214 157 / 0.4);
}

.offer[data-state='declined'] {
  box-shadow: inset 0 0 0 1px var(--color-n5);
}

.offer__swatch {
  display: grid;
  place-items: center;
  width: 2.75rem;
  height: 2.75rem;
  flex-shrink: 0;
  font-size: 17px;
  font-weight: 700;
  color: color-mix(in oklab, var(--color-r-masterwork) 40%, white);
  border-radius: 0.6rem;
  background: color-mix(in oklab, var(--color-r-masterwork) 26%, var(--color-n2));
  box-shadow: inset 0 0 0 1.5px var(--color-r-masterwork);
}

.offer__text {
  margin: 0;
  flex: 1 1 18rem;
  font-size: 16px;
  line-height: 1.5;
  color: var(--color-ink-soft);
}

.offer__actions {
  display: flex;
  gap: 0.5rem;
}

/* Steps */
.steps {
  margin: 3rem 0 0;
  padding: 0;
  list-style: none;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 2.5rem;
}

.steps__n {
  font-size: 4.5rem;
  line-height: 0.9;
  font-weight: 700;
  color: var(--color-n8);
  margin-bottom: 0.75rem;
}

/* News */
.news {
  margin-top: 2.5rem;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 2rem;
}

.news__item {
  text-decoration: none;
  color: inherit;
}

.news__img {
  aspect-ratio: 16 / 10;
  margin-bottom: 1rem;
  border-radius: 0.9rem;
  overflow: hidden;
  background: var(--color-n2);
}

.news__img img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform 0.5s cubic-bezier(0.16, 1, 0.3, 1);
}

.news__item:hover .news__img img {
  transform: scale(1.03);
}

/* Close */
.close {
  padding: clamp(1.25rem, 3vw, 2.25rem) 1.5rem clamp(4rem, 9vw, 7rem);
  text-align: center;
}

.close__inner {
  padding-block: clamp(3rem, 8vw, 6rem);
}

.close__title {
  margin: 0;
  font-family: var(--font-num);
  font-weight: 700;
  font-size: clamp(2.8rem, 6vw, 5rem);
  line-height: 0.95;
  text-transform: uppercase;
  color: var(--color-ink);
}

.close__lede {
  margin: 1.25rem auto 0;
  max-width: 30rem;
  font-size: 18px;
  line-height: 1.55;
  color: var(--color-dim);
}

/* ---------- Scroll reveals ---------- */
.landing--armed [data-reveal] {
  transition: opacity 0.8s cubic-bezier(0.16, 1, 0.3, 1), transform 0.8s cubic-bezier(0.16, 1, 0.3, 1);
}

.landing--armed [data-reveal]:not(.is-in) {
  opacity: 0;
  transform: translateY(24px);
}

.landing--armed [data-reveal='list'] {
  opacity: 1;
  transform: none;
}

.landing--armed [data-reveal='list'] li {
  transition: opacity 0.7s cubic-bezier(0.16, 1, 0.3, 1) var(--d, 0ms), transform 0.7s cubic-bezier(0.16, 1, 0.3, 1) var(--d, 0ms);
}

.landing--armed [data-reveal='list']:not(.is-in) li {
  opacity: 0;
  transform: translateY(16px);
}

/* ---------- Responsive ---------- */
@media (max-width: 960px) {
  .hero__inner {
    grid-template-columns: minmax(0, 1fr);
    min-height: 0;
    padding-top: 5.5rem;
  }

  .hero__tray {
    order: -1;
    max-width: 360px;
    justify-self: center;
  }

  .section__head,
  .roles,
  .news {
    grid-template-columns: minmax(0, 1fr);
  }
}

@media (max-width: 640px) {
  .steps {
    grid-template-columns: minmax(0, 1fr);
    gap: 1.75rem;
  }

  .steps li {
    grid-template-columns: 3.25rem minmax(0, 1fr);
    column-gap: 1rem;
  }

  .steps__n {
    grid-row: span 2;
    margin: 0;
    font-size: 3.25rem;
  }
}

@media (max-width: 520px) {
  .hero__inner {
    padding-left: 1rem;
    padding-right: 1rem;
  }

  .section,
  .close {
    padding-left: 0.75rem;
    padding-right: 0.75rem;
  }

  .section__inner,
  .close__inner {
    padding: 1.5rem 1.1rem;
    border-radius: 1.5rem;
  }

  .hero__tray {
    max-width: 280px;
  }

  .tray {
    box-shadow:
      inset 0 12px 36px rgb(0 0 0 / 0.55),
      inset 0 0 0 1px var(--color-n3),
      0 0 0 9px var(--color-n2),
      0 0 0 10px var(--color-n3);
  }

  .hero__actions > * {
    flex: 1 1 100%;
  }
}
</style>
