<script setup>
import { Notification, Notivue, slateTheme } from 'notivue'

const notificationTheme = {
  ...slateTheme,
  '--nv-width': '360px',
  '--nv-radius': '18px',
  '--nv-shadow': '0 18px 48px rgba(0, 0, 0, 0.35)',
  '--nv-border-width': '1px',
  '--nv-global-bg': 'rgba(12, 18, 32, 0.96)',
  '--nv-global-border': 'rgba(126, 200, 227, 0.16)',
  '--nv-global-fg': '#e8e8f0',
  '--nv-global-accent': '#7ec8e3',
  '--nv-success-bg': 'rgba(10, 32, 24, 0.96)',
  '--nv-success-border': 'rgba(109, 236, 169, 0.28)',
  '--nv-success-accent': '#6deca9',
  '--nv-success-fg': '#e8e8f0',
  '--nv-error-bg': 'rgba(40, 12, 20, 0.96)',
  '--nv-error-border': 'rgba(233, 69, 96, 0.28)',
  '--nv-error-accent': '#ff8fa3',
  '--nv-error-fg': '#ffe3e8',
  '--nv-info-bg': 'rgba(12, 24, 40, 0.96)',
  '--nv-info-border': 'rgba(126, 200, 227, 0.22)',
  '--nv-info-accent': '#7ec8e3',
  '--nv-info-fg': '#e8f5fb',
  '--nv-warning-bg': 'rgba(40, 26, 8, 0.96)',
  '--nv-warning-border': 'rgba(245, 158, 11, 0.24)',
  '--nv-warning-accent': '#f59e0b',
  '--nv-warning-fg': '#fff3d6',
}
</script>

<template>
  <RouterView />

  <Notivue v-slot="item">
    <Notification :item="item" :theme="notificationTheme" />
  </Notivue>
</template>

<style>
html,
body,
#app {
  min-height: 100%;
}

body {
  @apply text-[#e8e8f0] antialiased;
  margin: 0;
  padding: 0;
  font-family: 'Raleway', sans-serif;
}
</style>
