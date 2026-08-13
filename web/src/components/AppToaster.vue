<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { registerToaster, type ToastInput } from '@/lib/toast'

export interface ToastItem extends ToastInput {
  id: number
}

const toasts = ref<ToastItem[]>([])
let nextId = 1

function pushToast(t: ToastInput) {
  const id = nextId++
  toasts.value.push({ ...t, id })
  setTimeout(() => {
    toasts.value = toasts.value.filter((x) => x.id !== id)
  }, 4000)
}

onMounted(() => registerToaster(pushToast))
</script>

<template>
  <div class="pointer-events-none fixed inset-x-0 top-4 z-[100] flex flex-col items-center gap-2 px-4">
    <TransitionGroup name="toast">
      <div
        v-for="t in toasts"
        :key="t.id"
        class="pointer-events-auto w-full max-w-sm rounded-lg border bg-background p-3 shadow-lg"
        :class="t.variant === 'destructive' ? 'border-destructive/40' : 'border-border'"
        role="status"
      >
        <p class="text-sm font-medium" :class="t.variant === 'destructive' ? 'text-destructive' : 'text-foreground'">
          {{ t.title }}
        </p>
        <p v-if="t.description" class="mt-0.5 text-xs text-muted-foreground">
          {{ t.description }}
        </p>
      </div>
    </TransitionGroup>
  </div>
</template>

<style scoped>
.toast-enter-active,
.toast-leave-active {
  transition: all 0.25s ease;
}
.toast-enter-from {
  opacity: 0;
  transform: translateY(-8px);
}
.toast-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
