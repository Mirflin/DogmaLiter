<script setup>
import { ref, onMounted, onBeforeUnmount } from 'vue'
import LandingDragon from './LandingDragon.vue'

// The night over the logo's world, pinned behind the landing page while the content scrolls over it.
// Far to near: sky, a few stars, the moon, high cloud, floating islands, dragons, a low cloud bank.

const canvas = ref(null)
const reduced = ref(false)

// Deterministic so the sky does not reshuffle when the window resizes.
function mulberry32(seed) {
  return () => {
    seed |= 0
    seed = (seed + 0x6d2b79f5) | 0
    let t = Math.imul(seed ^ (seed >>> 15), 1 | seed)
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }
}

const rand = mulberry32(20)
const twinkles = Array.from({ length: 10 }, () => ({
  x: rand() * 100,
  y: rand() * 45,
  size: 1.6 + rand() * 1.4,
  duration: 3.5 + rand() * 4,
  delay: -rand() * 7,
}))

// Farther dragons are smaller, closer to the sky's colour and slower: all share one 110 s cycle,
// and the depth sets how much of it the crossing takes. Delays stagger them; at most two are up at once.
const flights = [
  { id: 1, depth: 'near', top: 30, size: 74, dir: 1, delay: 3, flap: 0, color: '#03061a', wing: '#070c26' },
  { id: 2, depth: 'far', top: 13, size: 32, dir: -1, delay: 20, flap: -1.1, color: '#070c26', wing: '#0a1030' },
  { id: 3, depth: 'mid', top: 22, size: 50, dir: 1, delay: 45, flap: -2.3, color: '#050920', wing: '#09102c' },
  { id: 4, depth: 'far', top: 38, size: 30, dir: -1, delay: 70, flap: -0.6, color: '#080d28', wing: '#0b1132' },
  { id: 5, depth: 'mid', top: 8, size: 44, dir: 1, delay: 90, flap: -1.7, color: '#060b22', wing: '#0a102e' },
]

// High cloud in a 1600 x 400 tile: broad nebula patches in three tints, long wisps and a few puffs.
const nebula = [
  { x: 300, y: 130, rx: 380, ry: 110, color: '#1f2c6e', core: 0, o: 0.85 },
  { x: 430, y: 175, rx: 240, ry: 75, color: '#24477a', core: 0, o: 0.5 },
  { x: 960, y: 90, rx: 470, ry: 100, color: '#233478', core: 0, o: 0.9 },
  { x: 1070, y: 120, rx: 220, ry: 65, color: '#2a3d7c', core: 0, o: 0.55 },
  { x: 1380, y: 250, rx: 260, ry: 90, color: '#203a74', core: 0, o: 0.75 },
  { x: 700, y: 300, rx: 320, ry: 80, color: '#22306c', core: 0, o: 0.65 },
  { x: 320, y: 96, rx: 330, ry: 22, color: '#34428a', core: 25, o: 0.9 },
  { x: 420, y: 122, rx: 220, ry: 14, color: '#34428a', core: 25, o: 0.8 },
  { x: 230, y: 168, rx: 170, ry: 12, color: '#34428a', core: 25, o: 0.7 },
  { x: 980, y: 58, rx: 340, ry: 24, color: '#34428a', core: 25, o: 0.9 },
  { x: 1120, y: 84, rx: 200, ry: 14, color: '#34428a', core: 25, o: 0.75 },
  { x: 760, y: 230, rx: 270, ry: 18, color: '#34428a', core: 25, o: 0.8 },
  { x: 1400, y: 200, rx: 190, ry: 20, color: '#34428a', core: 25, o: 0.85 },
  { x: 1330, y: 226, rx: 130, ry: 12, color: '#34428a', core: 25, o: 0.7 },
  { x: 560, y: 330, rx: 190, ry: 14, color: '#34428a', core: 25, o: 0.7 },
  { x: 190, y: 86, rx: 80, ry: 32, color: '#2e3b80', core: 20, o: 0.7 },
  { x: 250, y: 80, rx: 60, ry: 26, color: '#2e3b80', core: 20, o: 0.7 },
  { x: 890, y: 50, rx: 90, ry: 34, color: '#2e3b80', core: 20, o: 0.7 },
  { x: 1450, y: 192, rx: 70, ry: 26, color: '#2e3b80', core: 20, o: 0.7 },
  { x: 690, y: 222, rx: 76, ry: 28, color: '#2e3b80', core: 20, o: 0.7 },
]

// Puffs that cross a tile edge are repeated on the other side, so each tile wraps onto the next.
const nebulaTiled = nebula.flatMap((c) => [
  c,
  ...(c.x + c.rx > 1600 ? [{ ...c, x: c.x - 1600 }] : []),
  ...(c.x - c.rx < 0 ? [{ ...c, x: c.x + 1600 }] : []),
])

// Shooting stars: each streaks once per period, at its own spot, angle and length.
const meteors = [
  { top: 9, left: 16, angle: 18, length: 140, period: 11, delay: 3 },
  { top: 5, left: 58, angle: 28, length: 110, period: 14, delay: 7.5 },
  { top: 24, left: 34, angle: 14, length: 90, period: 17, delay: 12 },
  { top: 15, left: 78, angle: 34, length: 120, period: 13, delay: 1.5 },
  { top: 32, left: 6, angle: 22, length: 100, period: 19, delay: 16 },
]

const islands = [
  { id: 'far', fill: '#18225a', rim: '#2e3d86' },
  { id: 'mid', fill: '#0b1236', rim: '#5062b4' },
  { id: 'near', fill: '#0f173f', rim: '#4456a8' },
]

const sparkles = [
  { x: 13, y: 14, size: 12, delay: 0 },
  { x: 57, y: 8, size: 10, delay: 2.2 },
  { x: 83, y: 31, size: 11, delay: 3.6 },
]

let lastWidth = 0
let lastHeight = 0

function draw() {
  const el = canvas.value
  if (!el) return
  const width = window.innerWidth
  const height = Math.max(window.innerHeight, el.clientHeight)
  // Mobile toolbars change the height while scrolling; only redraw for real resizes.
  if (width === lastWidth && Math.abs(height - lastHeight) < 120) return
  lastWidth = width
  lastHeight = height

  const dpr = Math.min(window.devicePixelRatio || 1, 2)
  el.width = Math.round(width * dpr)
  el.height = Math.round(height * dpr)
  const ctx = el.getContext('2d')
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
  ctx.clearRect(0, 0, width, height)

  const random = mulberry32(7)
  const count = Math.round((width * height) / 11000)
  for (let i = 0; i < count; i += 1) {
    const x = random() * width
    // Bias towards the zenith; the horizon is haze.
    const y = Math.pow(random(), 1.6) * height * 0.8
    const fade = 1 - y / (height * 0.85)
    const r = random() > 0.93 ? 1.2 : 0.5 + random() * 0.4
    const color = random() > 0.85 ? '164, 174, 255' : '230, 234, 255'
    ctx.fillStyle = `rgba(${color}, ${(0.25 + random() * 0.55) * fade})`
    ctx.beginPath()
    ctx.arc(x, y, r, 0, Math.PI * 2)
    ctx.fill()
  }
}

onMounted(() => {
  reduced.value = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  draw()
  window.addEventListener('resize', draw, { passive: true })
})

onBeforeUnmount(() => window.removeEventListener('resize', draw))
</script>

<template>
  <div class="sky" :class="{ 'sky--still': reduced }" aria-hidden="true">
    <canvas ref="canvas" class="sky__stars" />

    <span
      v-for="(t, i) in twinkles"
      :key="`t${i}`"
      class="sky__twinkle"
      :style="{ left: `${t.x}%`, top: `${t.y}%`, width: `${t.size}px`, height: `${t.size}px`, animationDuration: `${t.duration}s`, animationDelay: `${t.delay}s` }"
    />

    <svg
      v-for="(s, i) in sparkles"
      :key="`s${i}`"
      class="sky__sparkle"
      :style="{ left: `${s.x}%`, top: `${s.y}%`, width: `${s.size}px`, height: `${s.size}px`, animationDelay: `${s.delay}s` }"
      viewBox="0 0 24 24"
    >
      <path d="M12 0c.6 5.4 1.9 8.4 4.2 9.8C18 11 20.6 11.6 24 12c-3.4.4-6 1-7.8 2.2C13.9 15.6 12.6 18.6 12 24c-.6-5.4-1.9-8.4-4.2-9.8C6 13 3.4 12.4 0 12c3.4-.4 6-1 7.8-2.2C10.1 8.4 11.4 5.4 12 0Z" />
    </svg>

    <span class="sky__moonlight" />
<!-- The moon sits in its own light: a warm near halo, a wide faint one, and thin cloud lit pink across it. -->
    <div class="sky__moon">
      <span class="sky__moon-halo sky__moon-halo--wide" />
      <span class="sky__moon-halo sky__moon-halo--near" />
      <svg class="sky__moon-disc" viewBox="0 0 200 200">
        <defs>
          <mask id="landing-crescent">
            <rect width="200" height="200" fill="white" />
            <circle cx="132" cy="72" r="78" fill="black" />
          </mask>
        </defs>
        <circle cx="100" cy="100" r="92" fill="currentColor" mask="url(#landing-crescent)" />
      </svg>
      <span class="sky__moon-veil sky__moon-veil--a" />
      <span class="sky__moon-veil sky__moon-veil--b" />
    </div>

    <!-- High cloud: soft nebula patches under long wisps. Each tile is drawn twice side by side so the
         drift loops seamlessly; the puffs are radial gradients, so their edges are soft without filters. -->
    <div class="sky__drift sky__drift--high">
      <div v-for="n in 2" :key="`h${n}`" class="sky__tile">
        <span
          v-for="(c, i) in nebulaTiled"
          :key="i"
          class="sky__puff"
          :style="{
            left: `${((c.x - c.rx) / 1600) * 100}%`,
            top: `${((c.y - c.ry) / 400) * 100}%`,
            width: `${((c.rx * 2) / 1600) * 100}%`,
            height: `${((c.ry * 2) / 400) * 100}%`,
            background: `radial-gradient(closest-side, ${c.color} ${c.core}%, transparent)`,
            opacity: c.o,
          }"
        />
      </div>
    </div>


    <!-- Distant dragons cross one at a time, behind the islands and the cloud bank. -->
    <div
      v-for="f in flights"
      :key="f.id"
      class="sky__flight"
      :class="`sky__flight--${f.dir > 0 ? 'east' : 'west'}-${f.depth}`"
      :style="{
        top: `${f.top}%`,
        '--size': f.size,
        '--dragon': f.color,
        '--dragon-far': f.wing,
        '--dragon-bone': f.wing,
        animationDelay: `${f.delay}s`,
      }"
    >
      <LandingDragon class="sky__dragon" :mirror="f.dir < 0" :offset="f.flap" />
    </div>

    <!-- Floating islands, far to near. -->
    <svg v-for="island in islands" :key="island.id" class="sky__island" :class="`sky__island--${island.id}`" viewBox="0 -30 200 190">
      <!-- The middle island is an older place: a two-tier rock with hanging vines, a great round tree,
           broken columns and an arch with a rune still glowing in it. -->
      <template v-if="island.id === 'mid'">
        <path :fill="island.fill" d="M0 40C30 26 80 22 120 26c30-4 60 0 80 12l-12 14-18 6-8 22-22 6-8 24-14 6-10 24-10-22-14-6-8-22-18-6-8-18-20-6-16-10Z" />
        <path fill="none" :stroke="island.rim" stroke-width="3" stroke-linecap="round" d="M2 39C30 26 80 23 120 27c30-4 58 0 78 11" />
        <g :fill="island.fill">
          <circle cx="40" cy="8" r="14" />
          <circle cx="57" cy="0" r="17" />
          <circle cx="74" cy="9" r="13" />
          <circle cx="48" cy="15.5" r="10" />
          <path d="M54 25V12h6v13Z" />
          <path d="M138 24.5V.5a15 15 0 0 1 30 0v24h-7V1.5a8 8 0 0 0-16 0v23Z" />
          <path d="M172 27l4-7 7 2-2 7.5Z" />
        </g>
        <!-- A roofless stone well: a round wall of blocks with a dark mouth and a glint of water. -->
        <g class="island__well">
          <path :fill="island.fill" d="M100 24V11h28v13Z" />
          <ellipse cx="114" cy="11" rx="14" ry="3.6" :fill="island.fill" />
          <ellipse cx="114" cy="11" rx="10.5" ry="2.3" fill="#050a22" />
          <ellipse cx="115.5" cy="11.4" rx="3" ry=".7" fill="#8fa2e8" opacity=".55" />
          <path fill="none" :stroke="island.rim" stroke-width=".9" opacity=".7" d="M100 17.5h28M107 11.5v6M121 11.5v6M114 17.5v6.5" />
          <path fill="none" :stroke="island.rim" stroke-width="1.2" d="M100 11a14 3.6 0 0 1 28 0" />
        </g>
        <circle class="island__rune" cx="153" cy="9.5" r="2.6" />
        <path fill="none" :stroke="island.fill" stroke-width="2.2" stroke-linecap="round" d="M26 60c-2 10 2 18-1 28M72 92c3 9-1 16 2 26M148 82c-2 8 1 14-1 22M178 58c2 8-1 14 1 20" />
      </template>
      <!-- The near island is a mountain top: twin peaks with moonlit snow, a tent, and a campfire on the edge. -->
      <template v-else-if="island.id === 'near'">
        <path :fill="island.fill" d="M0 36C30 28 70 26 100 28c40-2 75 0 100 10l-8 12-12 4-4 18-16 6-10 22-14 4-8 26-14 16-6-22-12-6-8-22-18-6-8-20-18-6-8-14-20-4Z" />
        <path fill="none" :stroke="island.rim" stroke-width="3" stroke-linecap="round" d="M2 36C30 28 70 27 100 29c40-2 74 0 97 9" />
        <g :fill="island.fill">
          <path d="M20 34L56-20l14 16 12-12 32 50Z" />
        </g>
        <path d="M56-20l-8 12 5-2 3 5 4-6 4 2Z" :fill="island.rim" />
        <path d="M82-16l-6 9 4-1 3 3 3-4 3 1Z" :fill="island.rim" opacity=".8" />
        <!-- A folding tent a shade lighter than the rock, a faint moonlit edge, its door a dark gap. -->
        <path d="M140 27.2L153 7.5 166 29Z" fill="#1b2559" />
        <path d="M152.4 8V3h1.2v5Z" fill="#1b2559" />
        <path d="M153 7.5L166 29" fill="none" :stroke="island.rim" stroke-width="1.1" stroke-linecap="round" opacity=".55" />
        <path d="M153 28.1V16l5.5 12.4Z" fill="#070c26" />
        <g class="island__fire">
          <path d="M171 29.4l12-3.4M171 26.4l12 3.6" stroke="#0f173f" stroke-width="2.4" stroke-linecap="round" />
          <path class="island__flame" d="M177 27.5c-4-3-4-7-1-11 0 3 2 4 3 4 0-3 1-5 3-6-1 4 2 6 1 9-1 2-3 4-6 4Z" fill="#f2a65a" />
          <path class="island__flame island__flame--core" d="M177.5 27.5c-2-2-1.5-4 0-6 .5 2 2 2 2.5 4 0 1-1 2-2.5 2Z" fill="#fde7a6" />
          <circle class="island__smoke" cx="178" cy="10.5" r="3" />
          <circle class="island__smoke island__smoke--late" cx="180" cy="4.5" r="4" />
        </g>
        <path fill="none" :stroke="island.fill" stroke-width="2.4" stroke-linecap="round" d="M54 72c-2 8 1 14-1 22M150 100c2 8-1 14 1 20" />
      </template>
      <template v-else>
        <path :fill="island.fill" d="M0 34C22 20 62 14 100 16c40-2 78 4 100 18l-10 10-10 20-16 6-10 26-18 8-12 34-14 12-8-22-12-16-14-12-12-22-18-12-16-18Z" />
        <!-- The far island spills a waterfall over its edge; it thins into mist as it falls. -->
        <g v-if="island.id === 'far'" class="island__falls">
          <defs>
            <linearGradient id="falls-fade" gradientUnits="userSpaceOnUse" x1="0" y1="24" x2="0" y2="270">
              <stop offset="0" stop-color="#fff" />
              <stop offset=".4" stop-color="#fff" stop-opacity=".85" />
              <stop offset="1" stop-color="#fff" stop-opacity="0" />
            </linearGradient>
            <mask id="falls-mask" maskUnits="userSpaceOnUse" x="150" y="10" width="80" height="270">
              <rect x="150" y="10" width="80" height="270" fill="url(#falls-fade)" />
            </mask>
            <filter id="falls-mist" x="-200%" y="-20%" width="500%" height="140%">
              <feGaussianBlur stdDeviation="5" />
            </filter>
          </defs>
          <g mask="url(#falls-mask)">
            <path d="M187 40C189 90 190 160 191 268" stroke="#4d66c4" stroke-width="16" opacity=".22" fill="none" filter="url(#falls-mist)" />
            <path d="M176 27C184 27 189 33 190 44L191 268" stroke="#3f58b4" stroke-width="7" stroke-linecap="round" opacity=".7" fill="none" />
            <path class="falls__flow" d="M176 27C184 27 189 33 190 44L191 268" stroke="#a9bbff" stroke-width="2" stroke-dasharray="12 18" stroke-linecap="round" fill="none" />
            <path class="falls__flow falls__flow--fine" d="M178 27C185 28 188 35 188.5 46L189 268" stroke="#dbe3ff" stroke-width="1" stroke-dasharray="5 25" stroke-linecap="round" fill="none" />
          </g>
          <ellipse cx="171" cy="24" rx="9" ry="2.2" fill="#3f58b4" opacity=".8" />
        </g>
        <g :fill="island.fill">
          <circle cx="34" cy="20" r="9" />
          <circle cx="48" cy="15" r="12" />
          <circle cx="62" cy="19" r="8" />
          <path d="M116 18V-6h6v-10l5-10 5 10v10h6V18Z" />
          <path d="M140 18V2h5l4-9 4 9h5v16Z" />
          <circle cx="170" cy="21" r="8" />
        </g>
        <path fill="none" :stroke="island.rim" stroke-width="3" stroke-linecap="round" d="M2 33C24 20 62 15 100 17c40-2 76 4 98 17" />
        <path fill="none" :stroke="island.fill" stroke-width="2.5" stroke-linecap="round" d="M92 136c-2 10 2 18-2 26M120 104c3 8 0 14 3 22M60 92c-3 8 0 14-4 20" />
        <circle cx="129" cy="2" r="1.8" fill="#f6d58a" opacity=".85" />
      </template>
    </svg>

    <!-- Low cloud bank in the logo's scalloped style. -->
    <div class="sky__drift sky__drift--bank">
      <svg v-for="n in 2" :key="`b${n}`" viewBox="0 0 1600 260" preserveAspectRatio="none">
        <path
          class="bank__back"
          d="M0 140C40 70 150 60 190 110c30-70 150-80 190-10 40-50 140-50 170 10 50-80 190-80 230-5 40-50 140-50 170 10 50-75 180-75 220-5 40-50 140-50 170 10 40-60 160-60 260 20V260H0Z"
        />
        <path
          class="bank__front"
          d="M0 190c30-50 120-60 160-20 30-50 130-58 160-2 40-38 120-40 150 7 40-57 150-59 180-3 40-37 120-37 150 8 40-55 150-58 185-4 35-36 115-38 145 6 40-54 150-56 185-4 35-36 115-38 145 6 40-34 110-34 140 4V260H0Z"
        />
        <path class="bank__rim" d="M0 190c30-50 120-60 160-20 30-50 130-58 160-2 40-38 120-40 150 7 40-57 150-59 180-3 40-37 120-37 150 8 40-55 150-58 185-4 35-36 115-38 145 6 40-54 150-56 185-4 35-36 115-38 145 6 40-34 110-34 140 4" />
      </svg>
    </div>

    <span
      v-for="(m, i) in meteors"
      :key="`m${i}`"
      class="sky__meteor"
      :style="{ top: `${m.top}%`, left: `${m.left}%`, width: `${m.length}px`, '--angle': `${m.angle}deg`, animationDuration: `${m.period}s`, animationDelay: `${m.delay}s` }"
    />
  </div>
</template>

<style scoped>
.sky {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100lvh;
  z-index: 0;
  overflow: hidden;
  pointer-events: none;
  background: linear-gradient(180deg, #050a20 0%, #0a1130 42%, #141c4f 78%, #1c2462 100%);
}

.sky__stars {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
}

.sky__twinkle {
  position: absolute;
  border-radius: 50%;
  background: #eef1ff;
  opacity: 0.2;
  animation: twinkle 5s ease-in-out infinite;
}

.sky__sparkle {
  position: absolute;
  fill: var(--color-orchid);
  opacity: 0.25;
  animation: sparkle 6s ease-in-out infinite;
}

/* Moon */
.sky__moon {
  /* Tucked partly past the right edge so it lives in the margin, clear of the text column. */
  --moon: clamp(120px, 11vw, 210px);
  position: absolute;
  top: clamp(4.75rem, 10vh, 7.5rem);
  right: calc(var(--moon) * -0.32);
  width: var(--moon);
  aspect-ratio: 1;
}

.sky__moon-disc {
  position: absolute;
  inset: 0;
  width: 100%;
  color: var(--color-orchid);
  opacity: 0.95;
  transform: rotate(-18deg);
}

.sky__moon-halo {
  position: absolute;
  border-radius: 50%;
  pointer-events: none;
}

/* Centred a little low and left, where the lit crescent is. */
.sky__moon-halo--near {
  left: -42%;
  top: -26%;
  width: 160%;
  height: 160%;
  background: radial-gradient(closest-side, rgb(255 214 240 / 0.3), rgb(232 144 210 / 0.14) 55%, transparent);
}

.sky__moon-halo--wide {
  left: -125%;
  top: -105%;
  width: 330%;
  height: 330%;
  background: radial-gradient(closest-side, rgb(232 144 210 / 0.16), rgb(232 144 210 / 0.06) 45%, transparent 75%);
}

.sky__moon-veil {
  position: absolute;
  border-radius: 50%;
  background: radial-gradient(closest-side, rgb(132 104 184 / 0.9), rgb(132 104 184 / 0.35) 55%, transparent);
}

.sky__moon-veil--a {
  left: -70%;
  top: 60%;
  width: 160%;
  height: 10%;
}

.sky__moon-veil--b {
  left: -25%;
  top: 78%;
  width: 105%;
  height: 6%;
  opacity: 0.8;
}

.sky__moonlight {
  position: absolute;
  top: -20vh;
  right: -20vw;
  width: 70vw;
  height: 80vh;
  background: radial-gradient(closest-side, rgb(232 144 210 / 0.1), transparent);
}

/* Drifting cloud layers */
.sky__drift {
  position: absolute;
  left: 0;
  display: flex;
  width: 200%;
  animation: drift 260s linear infinite;
}

.sky__drift svg {
  display: block;
  width: 50%;
  height: 100%;
}

.sky__tile {
  position: relative;
  width: 50%;
  height: 100%;
  overflow: hidden;
}

.sky__puff {
  position: absolute;
}

.sky__drift--high {
  top: 2%;
  height: 42%;
  opacity: 0.75;
  animation-duration: 320s;
}

.sky__drift--bank {
  bottom: -2px;
  height: clamp(110px, 22vh, 220px);
  animation-duration: 180s;
}

.bank__back {
  fill: #232e6c;
}

.bank__front {
  fill: #2b3777;
}

.bank__rim {
  fill: none;
  stroke: #45569f;
  stroke-width: 2;
  vector-effect: non-scaling-stroke;
}

/* Islands */
.sky__island {
  position: absolute;
  overflow: visible;
  animation: bob 9s ease-in-out infinite alternate;
}

.sky__island--far {
  top: 17%;
  left: 40%;
  width: clamp(70px, 7vw, 120px);
  animation-duration: 11s;
}

.sky__island--mid {
  /* Above the cloud bank, so the well never dips behind it as the island bobs. */
  z-index: 1;
  top: 75%;
  left: -3%;
  width: clamp(140px, 16vw, 280px);
  animation-delay: -4s;
}

.island__fire {
  filter: drop-shadow(0 0 3px rgb(242 166 90 / 0.7)) drop-shadow(0 0 6px rgb(242 140 70 / 0.25));
}

.island__flame {
  transform-box: fill-box;
  transform-origin: 50% 100%;
  animation: flicker 0.7s ease-in-out infinite alternate;
}

.island__flame--core {
  animation-duration: 0.45s;
}

.island__smoke {
  transform-box: fill-box;
  transform-origin: center;
  fill: #6f7db3;
  opacity: 0;
  animation: smoke 5s ease-out infinite;
}

.island__smoke--late {
  animation-delay: 2.5s;
}

@keyframes flicker {
  0% { transform: scale(0.9, 0.82) skewX(-4deg); }
  50% { transform: scale(1.04, 1.08) skewX(3deg); }
  100% { transform: scale(0.95, 1.14) skewX(-2deg); }
}

@keyframes smoke {
  0% { opacity: 0; transform: translate(0, 8px) scale(0.6); }
  20% { opacity: 0.3; }
  100% { opacity: 0; transform: translate(6px, -26px) scale(1.6); }
}

.island__rune {
  fill: #9fe6ff;
  filter: drop-shadow(0 0 3px rgb(159 230 255 / 0.95)) drop-shadow(0 0 8px rgb(120 200 255 / 0.5));
  animation: rune 4.5s ease-in-out infinite;
}

@keyframes rune {
  0%, 100% { opacity: 0.55; }
  50% { opacity: 1; }
}

.sky__island--near {
  top: 60%;
  right: 3%;
  width: clamp(150px, 19vw, 330px);
  animation-duration: 13s;
  animation-delay: -7s;
}

/* Waterfall: streaks run down the stream. */
.falls__flow {
  animation: falls 0.9s linear infinite;
}

.falls__flow--fine {
  animation-duration: 0.65s;
}

@keyframes falls {
  to { stroke-dashoffset: -60; }
}

/* Dragons */
.sky__flight {
  position: absolute;
  left: 0;
  /* As wide as its dragon, so -100% puts it fully past the left edge. */
  width: calc(var(--size) * 1px);
  animation: 110s linear infinite both;
}

.sky__flight--east-near { animation-name: east-near; }
.sky__flight--east-mid { animation-name: east-mid; }
.sky__flight--east-far { animation-name: east-far; }
.sky__flight--west-near { animation-name: west-near; }
.sky__flight--west-mid { animation-name: west-mid; }
.sky__flight--west-far { animation-name: west-far; }

.sky__dragon {
  width: 100%;
  animation: soar 6s ease-in-out infinite alternate;
}

/* Meteor */
.sky__meteor {
  position: absolute;
  height: 1.5px;
  border-radius: 999px;
  background: linear-gradient(90deg, rgb(238 241 255 / 0), rgb(238 241 255 / 0.85));
  opacity: 0;
  transform: rotate(var(--angle)) translateX(0);
  animation: meteor 14s cubic-bezier(0.4, 0, 0.6, 1) infinite both;
}

@keyframes twinkle {
  0%, 100% { opacity: 0.15; transform: scale(0.8); }
  50% { opacity: 0.9; transform: scale(1.1); }
}

@keyframes sparkle {
  0%, 100% { opacity: 0.2; transform: scale(0.7) rotate(0deg); }
  50% { opacity: 0.85; transform: scale(1) rotate(25deg); }
}

@keyframes drift {
  from { transform: translateX(0); }
  to { transform: translateX(-50%); }
}

@keyframes bob {
  from { transform: translateY(-6px); }
  to { transform: translateY(6px); }
}

@keyframes soar {
  from { translate: 0 -10px; rotate: -2deg; }
  to { translate: 0 10px; rotate: 2deg; }
}

/* Starts and ends past the edges, so loading the page or looping never shows a dragon popping in. */
@keyframes east-near { 0% { transform: translateX(-100%); } 20%, 100% { transform: translateX(100vw); } }
@keyframes east-mid { 0% { transform: translateX(-100%); } 30%, 100% { transform: translateX(100vw); } }
@keyframes east-far { 0% { transform: translateX(-100%); } 40%, 100% { transform: translateX(100vw); } }
@keyframes west-near { 0% { transform: translateX(100vw); } 20%, 100% { transform: translateX(-100%); } }
@keyframes west-mid { 0% { transform: translateX(100vw); } 30%, 100% { transform: translateX(-100%); } }
@keyframes west-far { 0% { transform: translateX(100vw); } 40%, 100% { transform: translateX(-100%); } }

@keyframes meteor {
  0% { opacity: 0; transform: rotate(var(--angle)) translateX(0); }
  1% { opacity: 1; }
  5%, 100% { opacity: 0; transform: rotate(var(--angle)) translateX(380px); }
}

.sky--still * {
  animation: none !important;
}

.sky--still .sky__flight,
.sky--still .sky__meteor {
  display: none;
}

@media (max-width: 960px) {
  .sky__moon {
    --moon: 64px;
    top: 4.6rem;
    right: -24px;
    opacity: 0.75;
  }

  .sky__island--far {
    left: 62%;
    top: 22%;
  }

  .sky__island--mid {
    top: 56%;
    left: -12%;
  }

  .sky__island--near {
    top: 70%;
    right: -10%;
  }

  .sky__flight {
    width: calc(var(--size) * 0.7px);
  }

  .sky__drift--bank {
    height: 72px;
  }
}
</style>
