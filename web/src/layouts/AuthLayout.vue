<script setup lang="ts">
import { RouterView } from 'vue-router'
</script>

<template>
  <!--
    Auth layout — standalone page, NO sidebar, NO topbar, NO hamburger.
    Used for /login only.
  -->
  <div class="flex min-h-screen items-center justify-center bg-background p-4">
    <RouterView />
  </div>
</template>