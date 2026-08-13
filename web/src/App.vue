<script setup lang="ts">
import { onBeforeUnmount, onMounted } from 'vue'
import { RouterView } from 'vue-router'
import AppToaster from '@/components/AppToaster.vue'
import { startRealtime, stopRealtime } from '@/lib/realtime'

onMounted(startRealtime)
onBeforeUnmount(stopRealtime)
</script>

<template>
  <!--
    Root component — renders the active layout (AuthLayout for /login,
    AppLayout for the dashboard). Layouts own their chrome (sidebar/topbar).
  -->
  <RouterView />
  <AppToaster />
</template>
