<script setup lang="ts">
import { onMounted, ref } from 'vue'

const mounted = ref(false)
onMounted(() => {
  mounted.value = true
})
</script>

<template>
  <slot v-if="mounted" />
  <div v-else class="h-32 animate-pulse rounded-md bg-muted" />
</template>
