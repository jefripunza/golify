<script setup lang="ts">
import { RouterView } from 'vue-router'
import ErrorBoundary from '@/components/ErrorBoundary.vue'
</script>

<template>
  <!--
    Auth layout — standalone page, NO sidebar, NO topbar, NO hamburger.
    Used for /login (and /onboarding once added).
  -->
  <div class="flex min-h-screen items-center justify-center bg-background p-4">
    <ErrorBoundary>
      <RouterView />
    </ErrorBoundary>
  </div>
</template>