<script setup lang="ts">
import { RouterView } from 'vue-router'
import AppToaster from '@/components/AppToaster.vue'
</script>

<template>
  <!--
    Root component — renders the active layout (AuthLayout for /login,
    AppLayout for the dashboard). Layouts own their chrome (sidebar/topbar).
  -->
  <RouterView />
  <AppToaster />
</template>
