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

const islands = [
  { id: 'far', fill: '#18225a', rim: '#2e3d86' },
  { id: 'mid', fill: '#131c4e', rim: '#3a4b9a' },
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
    <svg class="sky__moon" viewBox="0 0 200 200">
      <defs>
        <mask id="landing-crescent">
          <rect width="200" height="200" fill="white" />
          <circle cx="132" cy="72" r="78" fill="black" />
        </mask>
      </defs>
      <circle cx="100" cy="100" r="92" fill="currentColor" mask="url(#landing-crescent)" />
    </svg>

    <!-- High cloud: long thin streaks, drawn twice side by side so the drift loops seamlessly. -->
    <div class="sky__drift sky__drift--high">
      <svg v-for="n in 2" :key="`h${n}`" viewBox="0 0 1600 300" preserveAspectRatio="none">
        <defs>
          <filter :id="`wisp${n}`" x="-10%" y="-200%" width="120%" height="500%">
            <feGaussianBlur stdDeviation="6" />
          </filter>
        </defs>
        <g fill="#2a3778" :filter="`url(#wisp${n})`">
          <ellipse cx="300" cy="70" rx="280" ry="9" />
          <ellipse cx="380" cy="88" rx="170" ry="6" />
          <ellipse cx="1000" cy="40" rx="240" ry="8" />
          <ellipse cx="1420" cy="140" rx="160" ry="7" />
          <ellipse cx="760" cy="200" rx="200" ry="7" />
        </g>
      </svg>
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
      <path :fill="island.fill" d="M0 34C22 20 62 14 100 16c40-2 78 4 100 18l-10 10-10 20-16 6-10 26-18 8-12 34-14 12-8-22-12-16-14-12-12-22-18-12-16-18Z" />
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

    <span class="sky__meteor" />
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
  color: var(--color-orchid);
  opacity: 0.92;
  transform: rotate(-18deg);
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

.sky__drift--high {
  top: 7%;
  height: 26%;
  opacity: 0.55;
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
  top: 74%;
  left: -8%;
  width: clamp(130px, 15vw, 260px);
  animation-delay: -4s;
}

.sky__island--near {
  top: 60%;
  right: 3%;
  width: clamp(150px, 19vw, 330px);
  animation-duration: 13s;
  animation-delay: -7s;
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
  top: 10%;
  left: 18%;
  width: 140px;
  height: 1.5px;
  border-radius: 999px;
  background: linear-gradient(90deg, rgb(238 241 255 / 0), rgb(238 241 255 / 0.85));
  opacity: 0;
  transform: rotate(18deg) translateX(0);
  animation: meteor 22s cubic-bezier(0.4, 0, 0.6, 1) 8s infinite;
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
  0% { opacity: 0; transform: rotate(18deg) translateX(0); }
  1% { opacity: 1; }
  5% { opacity: 0; transform: rotate(18deg) translateX(420px); }
  100% { opacity: 0; transform: rotate(18deg) translateX(420px); }
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
    opacity: 0.7;
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
