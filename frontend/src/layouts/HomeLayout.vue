<script setup>
import NavBar from '@/components/NavBar.vue'
import FooterBar from '@/components/FooterBar.vue'
</script>

<template>
  <div class="home-layout-shell min-h-screen flex flex-col">
    <div class="home-layout-shell__ambient" aria-hidden="true">
      <span class="home-layout-shell__orb home-layout-shell__orb--primary"></span>
      <span class="home-layout-shell__orb home-layout-shell__orb--secondary"></span>
      <span class="home-layout-shell__orb home-layout-shell__orb--tertiary"></span>
    </div>
    <NavBar />
    <main class="home-layout-shell__main flex-1">
      <slot />
    </main>
    <FooterBar />
  </div>
</template>

<style scoped>
.home-layout-shell {
  position: relative;
  isolation: isolate;
  background: transparent;
}

.home-layout-shell::before {
  content: '';
  position: fixed;
  inset: 14px;
  z-index: -1;
  pointer-events: none;
  border: 1px solid rgba(126, 200, 227, 0.06);
}

.home-layout-shell__ambient {
  position: fixed;
  inset: 0;
  z-index: -2;
  pointer-events: none;
  overflow: hidden;
}

.home-layout-shell__ambient::before {
  content: '';
  position: absolute;
  inset: 0;
  background:
    linear-gradient(90deg, rgba(126, 200, 227, 0.025) 1px, transparent 1px),
    linear-gradient(0deg, rgba(126, 200, 227, 0.02) 1px, transparent 1px);
  background-size: 120px 120px;
  opacity: 0.7;
}

.home-layout-shell__orb {
  position: absolute;
  border-radius: 999px;
  filter: blur(110px);
  opacity: 0.5;
}

.home-layout-shell__orb--primary {
  top: -6rem;
  left: -3rem;
  width: 26rem;
  height: 26rem;
  background: rgba(233, 69, 96, 0.16);
}

.home-layout-shell__orb--secondary {
  top: 18rem;
  right: -8rem;
  width: 28rem;
  height: 28rem;
  background: rgba(126, 200, 227, 0.12);
}

.home-layout-shell__orb--tertiary {
  bottom: -8rem;
  left: 22%;
  width: 24rem;
  height: 24rem;
  background: rgba(233, 69, 96, 0.09);
}

.home-layout-shell__main {
  position: relative;
  z-index: 1;
}
</style>
