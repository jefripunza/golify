<script setup lang="ts">
import { RouterView } from 'vue-router'
import Sidebar from '@/components/Sidebar.vue'
import Topbar from '@/components/Topbar.vue'
</script>

<template>
  <div class="min-h-screen bg-background text-foreground md:flex">
    <Sidebar />
    <div class="flex min-w-0 flex-1 flex-col">
      <Topbar />
      <main class="flex-1 px-4 py-6 md:px-6">
        <RouterView />
      </main>
    </div>
  </div>
</template>