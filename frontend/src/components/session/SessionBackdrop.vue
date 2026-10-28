<script setup>
// Inside the d20: the session screen is seen from within the logo's die. Ten facets in the logo's own
// arrangement (a central face, lines from its corners out to a hexagon larger than the screen), each a
// neighbouring shade of navy as if lit from the upper right. Static: people sit on this screen for hours.

const T = [800, -540]
const UR = [1753, 10]
const LR = [1753, 1110]
const BO = [800, 1660]
const LL = [-153, 1110]
const UL = [-153, 10]
const A = [800, 250]
const B = [545, 700]
const C = [1055, 700]

const facets = [
  { points: [A, B, C], fill: '#0c1433' },
  { points: [A, UL, T], fill: '#0b1230' },
  { points: [A, T, UR], fill: '#111a40' },
  { points: [A, UR, C], fill: '#0f173a' },
  { points: [C, UR, LR], fill: '#0d1535' },
  { points: [C, LR, BO], fill: '#090f29' },
  { points: [B, C, BO], fill: '#0a1029' },
  { points: [B, BO, LL], fill: '#070c22' },
  { points: [B, LL, UL], fill: '#080d25' },
  { points: [A, B, UL], fill: '#0a112c' },
]

const edges = [[A, B], [B, C], [C, A], [A, T], [A, UL], [A, UR], [C, UR], [C, LR], [C, BO], [B, BO], [B, LL], [B, UL]]

const toPoints = (pts) => pts.map((p) => p.join(',')).join(' ')
</script>

<template>
  <div class="backdrop" aria-hidden="true">
    <svg viewBox="0 0 1600 1000" preserveAspectRatio="xMidYMid slice">
      <defs>
        <radialGradient id="session-light" cx="800" cy="560" r="620" gradientUnits="userSpaceOnUse">
          <stop offset="0" stop-color="#62b6ee" stop-opacity=".07" />
          <stop offset="1" stop-color="#62b6ee" stop-opacity="0" />
        </radialGradient>
        <radialGradient id="session-shade" cx="800" cy="500" r="1000" gradientUnits="userSpaceOnUse">
          <stop offset=".55" stop-color="#03061a" stop-opacity="0" />
          <stop offset="1" stop-color="#03061a" stop-opacity=".55" />
        </radialGradient>
      </defs>

      <polygon v-for="(facet, i) in facets" :key="i" :points="toPoints(facet.points)" :fill="facet.fill" />

      <!-- Light passing through the die, and the far faces sinking into shadow. -->
      <rect x="-200" y="-200" width="2000" height="1400" fill="url(#session-light)" />
      <rect x="-200" y="-200" width="2000" height="1400" fill="url(#session-shade)" />

      <g fill="none" stroke-linecap="round">
        <line
          v-for="(edge, i) in edges"
          :key="`e${i}`"
          :x1="edge[0][0]" :y1="edge[0][1]" :x2="edge[1][0]" :y2="edge[1][1]"
          stroke="#2c3a7c" stroke-width="1.5" stroke-opacity=".38" vector-effect="non-scaling-stroke"
        />
        <!-- Glass glints along a few edges, on the lit side. -->
        <path d="M812 268 L1040 682" stroke="#8fa2e8" stroke-width="1.2" stroke-opacity=".16" vector-effect="non-scaling-stroke" />
        <path d="M830 214 L1500 26" stroke="#8fa2e8" stroke-width="1.2" stroke-opacity=".12" vector-effect="non-scaling-stroke" />
        <path d="M1078 690 L1560 920" stroke="#8fa2e8" stroke-width="1" stroke-opacity=".08" vector-effect="non-scaling-stroke" />
      </g>
    </svg>
  </div>
</template>

<style scoped>
.backdrop {
  position: fixed;
  inset: 0;
  z-index: -2;
  pointer-events: none;
  background: var(--color-n1);
}

.backdrop svg {
  display: block;
  width: 100%;
  height: 100%;
}
</style>
