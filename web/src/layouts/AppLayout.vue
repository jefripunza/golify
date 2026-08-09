<script setup lang="ts">
import { RouterView } from 'vue-router'
import Sidebar from '@/components/Sidebar.vue'
import Topbar from '@/components/Topbar.vue'
import ErrorBoundary from '@/components/ErrorBoundary.vue'
</script>

<template>
  <!--
    App layout — authenticated dashboard shell:
    desktop: sticky sidebar (left) + content column (topbar + view)
    mobile:  content column + hamburger (topbar right) that opens the
             off-canvas sidebar panel
  -->
  <div class="min-h-screen bg-background text-foreground md:flex">
    <Sidebar />
    <div class="flex min-w-0 flex-1 flex-col">
      <Topbar />
      <main class="flex-1 px-4 py-6 md:px-6">
        <ErrorBoundary>
          <RouterView />
        </ErrorBoundary>
      </main>
    </div>
  </div>
</template>