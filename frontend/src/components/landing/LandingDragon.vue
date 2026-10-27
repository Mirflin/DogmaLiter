<script setup>
// A dragon silhouette built in parts so its wings can beat: three strokes, then a glide.
defineProps({
  mirror: { type: Boolean, default: false },
  // Seconds; lets two dragons in one sky beat out of step.
  offset: { type: Number, default: 0 },
})

const WING = 'M 131.5 80.2 C 139.0 57.8 149.0 35.2 164.0 7.8 L 180.2 -4.8 L 169.0 12.8 C 141.5 -9.8 114.0 -22.2 84.0 -29.8 C 91.5 -14.8 94.0 -4.8 91.5 5.2 C 79.0 -2.2 64.0 -4.8 49.0 -2.2 C 59.0 10.2 64.0 20.2 64.0 30.2 C 54.0 27.8 41.5 30.2 31.5 35.2 C 49.0 47.8 71.5 62.8 94.0 80.2Z'
const BONES = 'M 164.0 9.0 L 85.2 -27.2 M 161.5 14.0 L 50.2 -1.0 M 156.5 22.8 L 34.0 34.0'
</script>

<template>
  <svg class="dragon" :class="{ 'dragon--mirror': mirror }" viewBox="-6 -38 252 158" :style="{ '--offset': `${offset}s` }" aria-hidden="true">
    <g class="dragon__body">
      <g class="dragon__wing dragon__wing--far">
        <path :d="WING" transform="translate(14 -4) scale(.9)" />
      </g>
      <path class="dragon__flesh" d="M10 90C30 88 52 80 78 78C100 74 124 74 146 78C156 74 166 64 178 54C184 48 190 44 198 44L194 32L204 42C212 42 222 48 238 56L241 60L228 63L236 67L214 68C204 70 194 72 186 80C178 88 170 96 158 100C150 104 140 106 130 104L127 113L119 115L121 105C112 106 100 104 92 102L87 111L79 113L81 101C64 98 40 96 10 90Z" />
      <path class="dragon__flesh" d="M12 90L0 83L3 91L0 99Z" />
      <circle class="dragon__eye" cx="215" cy="52" r="1.8" />
      <g class="dragon__wing dragon__wing--near">
        <path class="dragon__flesh" :d="WING" />
        <path class="dragon__bones" :d="BONES" />
      </g>
    </g>
  </svg>
</template>

<style scoped>
.dragon {
  display: block;
  overflow: visible;
}

.dragon--mirror {
  transform: scaleX(-1);
}

.dragon__flesh {
  fill: var(--dragon, #050817);
}

.dragon__wing--far path {
  fill: var(--dragon-far, #0e1538);
}

.dragon__bones {
  fill: none;
  stroke: var(--dragon-bone, #18214d);
  stroke-width: 1.4;
  stroke-linecap: round;
}

.dragon__eye {
  fill: #d7ddff;
}

.dragon__wing {
  transform-box: view-box;
  transform-origin: 134px 79px;
  animation: flap 3.6s var(--offset) infinite;
}

.dragon__wing--far {
  animation-delay: calc(var(--offset) + 0.07s);
}

.dragon__body {
  animation: heave 3.6s var(--offset) infinite;
}

/* Three beats, then wings held out for a glide. Down-strokes are quicker than up-strokes. */
@keyframes flap {
  0% { transform: scaleY(1); animation-timing-function: cubic-bezier(0.55, 0, 0.6, 1); }
  9% { transform: scaleY(-0.85) skewX(12deg); animation-timing-function: cubic-bezier(0.3, 0, 0.4, 1); }
  21% { transform: scaleY(1); animation-timing-function: cubic-bezier(0.55, 0, 0.6, 1); }
  30% { transform: scaleY(-0.85) skewX(12deg); animation-timing-function: cubic-bezier(0.3, 0, 0.4, 1); }
  42% { transform: scaleY(1); animation-timing-function: cubic-bezier(0.55, 0, 0.6, 1); }
  51% { transform: scaleY(-0.85) skewX(12deg); animation-timing-function: cubic-bezier(0.3, 0, 0.4, 1); }
  64% { transform: scaleY(0.32); animation-timing-function: ease-in-out; }
  88% { transform: scaleY(0.24); animation-timing-function: ease-in-out; }
  100% { transform: scaleY(1); }
}

@keyframes heave {
  0%, 21%, 42% { transform: translateY(3px); }
  9%, 30%, 51% { transform: translateY(-5px); }
  64%, 88% { transform: translateY(0); }
  100% { transform: translateY(3px); }
}

@media (prefers-reduced-motion: reduce) {
  .dragon__wing,
  .dragon__body {
    animation: none;
  }

  .dragon__wing {
    transform: scaleY(0.3);
  }
}
</style>
