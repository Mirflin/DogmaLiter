<script setup>
import { Notification, Notivue, slateTheme } from 'notivue'

const notificationTheme = {
  ...slateTheme,
  '--nv-z': '14000',
  '--nv-width': '360px',
  '--nv-radius': '12px',
  '--nv-shadow': '0 12px 32px -8px rgba(2, 4, 14, 0.8)',
  '--nv-border-width': '1px',
  '--nv-global-bg': '#141e41',
  '--nv-global-border': '#24325f',
  '--nv-global-fg': '#e6eaff',
  '--nv-global-accent': '#a4aeff',
  '--nv-success-bg': '#141e41',
  '--nv-success-border': 'rgba(98, 214, 157, 0.4)',
  '--nv-success-accent': '#62d69d',
  '--nv-success-fg': '#e6eaff',
  '--nv-error-bg': '#141e41',
  '--nv-error-border': 'rgba(242, 122, 133, 0.45)',
  '--nv-error-accent': '#f27a85',
  '--nv-error-fg': '#e6eaff',
  '--nv-info-bg': '#141e41',
  '--nv-info-border': '#33427a',
  '--nv-info-accent': '#62b6ee',
  '--nv-info-fg': '#e6eaff',
  '--nv-warning-bg': '#141e41',
  '--nv-warning-border': 'rgba(239, 195, 94, 0.4)',
  '--nv-warning-accent': '#efc35e',
  '--nv-warning-fg': '#e6eaff',
}

const notivueStyles = {
  list: {
    zIndex: '20000',
  },
}
</script>

<template>
  <RouterView />

  <Notivue v-slot="item" :styles="notivueStyles">
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
  margin: 0;
  padding: 0;
}
</style>
