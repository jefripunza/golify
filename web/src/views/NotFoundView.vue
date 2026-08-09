<script setup lang="ts">
import { useRouter } from 'vue-router'
import { Button } from '@/components/ui/button'

const router = useRouter()
</script>

<template>
  <div class="flex min-h-[60vh] flex-col items-center justify-center gap-4 text-center">
    <p class="text-6xl font-bold tracking-tight text-muted-foreground">404</p>
    <h1 class="text-2xl font-semibold">Page not found</h1>
    <p class="max-w-sm text-sm text-muted-foreground">
      The page you're looking for doesn't exist or has been moved.
    </p>
    <Button type="button" @click="router.replace('/')">Back to Dashboard</Button>
  </div>
</template>