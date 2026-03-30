<script setup></script>

<template>
  <RouterView />
</template>

<style>
@import url('https://fonts.googleapis.com/css2?family=Raleway:wght@300;400;500;600;700&family=Cinzel:wght@400;600;700&display=swap');

html, body {
  @apply h-full bg-[#0a0a1a] text-[#e8e8f0] antialiased;
  margin: 0;
  padding: 0;
  font-family: 'Raleway', sans-serif;
}

#app {
  @apply h-full;
}
</style>
