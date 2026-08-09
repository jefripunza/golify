<script setup lang="ts">
import { RouterView } from 'vue-router'
</script>

<template>
  <!--
    Root component — renders the active layout (AuthLayout for /login,
    AppLayout for the dashboard). Layouts own their chrome (sidebar/topbar).
  -->
  <RouterView />
</template>