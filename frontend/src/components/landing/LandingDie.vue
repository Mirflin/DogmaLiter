<script setup>
import { ref, onMounted, onBeforeUnmount } from 'vue'
import logo from '@/assets/DLlogo.png'

const root = ref(null)
const landed = ref(false)
const shine = ref(0)
const tilt = ref({ x: 0, y: 0 })

// Glints sit on the painted stars of the logo (percent of the image box).
const glints = [
  { x: 29.6, y: 15.4, s: 1, d: 0 },
  { x: 53.4, y: 5.6, s: 0.8, d: 1.3 },
  { x: 71.8, y: 15.9, s: 1.1, d: 2.1 },
  { x: 77.9, y: 20.8, s: 0.7, d: 3.4 },
]

let target = null
let frame = 0
let canTilt = false

function onPointerMove(event) {
  if (!canTilt || !root.value) return
  const rect = root.value.getBoundingClientRect()
  const x = (event.clientX - (rect.left + rect.width / 2)) / rect.width
  const y = (event.clientY - (rect.top + rect.height / 2)) / rect.height
  cancelAnimationFrame(frame)
  frame = requestAnimationFrame(() => {
    tilt.value = {
      x: Math.max(-1, Math.min(1, x)) * 12,
      y: Math.max(-1, Math.min(1, y)) * -9,
    }
  })
}

function onPointerLeave() {
  tilt.value = { x: 0, y: 0 }
}

function sweep() {
  shine.value += 1
}

onMounted(() => {
  const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  canTilt = !reduced && window.matchMedia('(pointer: fine)').matches
  // The closest [data-tilt-scope] ancestor drives the tilt, so the whole hero reacts.
  target = root.value?.closest('[data-tilt-scope]') || root.value
  target?.addEventListener('pointermove', onPointerMove)
  target?.addEventListener('pointerleave', onPointerLeave)
  if (reduced) {
    landed.value = true
    return
  }
  requestAnimationFrame(() => { landed.value = true })
})

onBeforeUnmount(() => {
  cancelAnimationFrame(frame)
  target?.removeEventListener('pointermove', onPointerMove)
  target?.removeEventListener('pointerleave', onPointerLeave)
})
</script>

<template>
  <div ref="root" class="die" :class="{ 'die--landed': landed }" @pointerenter="sweep">
    <div class="die__shadow" aria-hidden="true" />
    <div class="die__roll">
      <div class="die__tilt" :style="{ '--tx': `${tilt.x}deg`, '--ty': `${tilt.y}deg` }">
        <img :src="logo" alt="DogmaLiter — a d20 die framing a painted fantasy landscape" class="die__art" draggable="false" />
        <span :key="shine" class="die__shine" :data-replay="shine > 0 || undefined" :style="{ maskImage: `url(${logo})`, WebkitMaskImage: `url(${logo})` }" aria-hidden="true" />
        <svg
          v-for="(g, i) in glints"
          :key="i"
          class="die__glint"
          :style="{ left: `${g.x}%`, top: `${g.y}%`, '--s': g.s, animationDelay: `${g.d}s` }"
          viewBox="0 0 24 24"
          aria-hidden="true"
        >
          <path d="M12 0c.6 5.4 1.9 8.4 4.2 9.8C18 11 20.6 11.6 24 12c-3.4.4-6 1-7.8 2.2C13.9 15.6 12.6 18.6 12 24c-.6-5.4-1.9-8.4-4.2-9.8C6 13 3.4 12.4 0 12c3.4-.4 6-1 7.8-2.2C10.1 8.4 11.4 5.4 12 0Z" />
        </svg>
      </div>
    </div>
  </div>
</template>

<style scoped>
.die {
  position: relative;
  aspect-ratio: 784 / 887;
  width: 100%;
  perspective: 1100px;
  user-select: none;
}

.die__roll {
  position: absolute;
  inset: 0;
  opacity: 0;
}

.die--landed .die__roll {
  animation: die-roll 1.25s cubic-bezier(0.16, 1, 0.3, 1) both;
}

.die__tilt {
  position: absolute;
  inset: 0;
  transform-style: preserve-3d;
  transform: rotateY(var(--tx, 0deg)) rotateX(var(--ty, 0deg));
  transition: transform 0.7s cubic-bezier(0.16, 1, 0.3, 1);
}

.die__art {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: contain;
  filter: drop-shadow(0 28px 36px rgb(2 4 14 / 0.55));
  pointer-events: none;
}

.die__shine {
  position: absolute;
  inset: 0;
  mask-size: contain;
  mask-repeat: no-repeat;
  mask-position: center;
  -webkit-mask-size: contain;
  -webkit-mask-repeat: no-repeat;
  -webkit-mask-position: center;
  background: linear-gradient(105deg, transparent 38%, rgb(255 255 255 / 0.32) 48%, rgb(255 255 255 / 0.08) 54%, transparent 62%);
  background-size: 260% 100%;
  background-position: 120% 0;
  mix-blend-mode: soft-light;
  pointer-events: none;
}

.die--landed .die__shine {
  animation: die-shine 1.4s cubic-bezier(0.65, 0, 0.35, 1) 0.9s both;
}

.die--landed .die__shine[data-replay] {
  animation-delay: 0s;
}

.die__glint {
  position: absolute;
  width: 4.2%;
  aspect-ratio: 1;
  translate: -50% -50%;
  transform: translateZ(40px) scale(var(--s));
  fill: #fde7f6;
  opacity: 0;
  filter: drop-shadow(0 0 6px rgb(232 144 210 / 0.8));
  animation: die-glint 4.8s ease-in-out infinite;
  pointer-events: none;
}

.die__shadow {
  position: absolute;
  left: 18%;
  right: 18%;
  bottom: -3%;
  height: 7%;
  border-radius: 50%;
  background: radial-gradient(closest-side, rgb(2 4 14 / 0.75), transparent);
  opacity: 0;
  transform: scaleX(0.3);
}

.die--landed .die__shadow {
  animation: die-shadow 1.25s cubic-bezier(0.16, 1, 0.3, 1) both;
}

@keyframes die-roll {
  0% {
    opacity: 0;
    transform: translate3d(42%, -30%, 0) rotate(-260deg) scale(0.5);
  }
  35% {
    opacity: 1;
  }
  82% {
    transform: translate3d(0, 0, 0) rotate(4deg) scale(1);
  }
  100% {
    opacity: 1;
    transform: none;
  }
}

@keyframes die-shadow {
  0% { opacity: 0; transform: scaleX(0.3); }
  100% { opacity: 1; transform: scaleX(1); }
}

@keyframes die-shine {
  0% { background-position: 120% 0; }
  100% { background-position: -40% 0; }
}

@keyframes die-glint {
  0%, 62%, 100% { opacity: 0; transform: translateZ(40px) scale(calc(var(--s) * 0.4)) rotate(0deg); }
  72% { opacity: 1; transform: translateZ(40px) scale(var(--s)) rotate(20deg); }
  84% { opacity: 0; transform: translateZ(40px) scale(calc(var(--s) * 0.6)) rotate(40deg); }
}

@media (prefers-reduced-motion: reduce) {
  .die__roll,
  .die__shadow {
    opacity: 1;
    transform: none;
    animation: none !important;
  }

  .die__glint,
  .die__shine {
    display: none;
  }
}
</style>
