<script setup lang="ts">
import { ref, onErrorCaptured } from 'vue'
import { Button } from '@/components/ui/button'

// ErrorBoundary — catches render/lifecycle errors from any descendant and
// shows a readable panel with the error title + stack, plus a "Copy" button
// that copies `title\n\nstack` to the clipboard.

const error = ref<Error | null>(null)
const copied = ref(false)

onErrorCaptured((err, _instance, info) => {
  error.value = err instanceof Error ? err : new Error(String(err))
  error.value.message = `${error.value.message}\n\n[info] ${info}`
  return false // stop propagation; we render our own fallback
})

async function copyDetails() {
  if (!error.value) return
  const title = error.value.name || 'Error'
  const stack = error.value.stack || error.value.message || ''
  const text = `${title}\n\n${stack}`
  try {
    await navigator.clipboard.writeText(text)
    copied.value = true
    setTimeout(() => (copied.value = false), 2000)
  } catch {
    // fallback for older browsers
    const ta = document.createElement('textarea')
    ta.value = text
    document.body.appendChild(ta)
    ta.select()
    document.execCommand('copy')
    document.body.removeChild(ta)
    copied.value = true
    setTimeout(() => (copied.value = false), 2000)
  }
}

function reset() {
  error.value = null
}

function reload() {
  window.location.reload()
}
</script>

<template>
  <div v-if="error" class="flex min-h-[60vh] items-center justify-center p-4">
    <div class="w-full max-w-lg rounded-xl border border-destructive/30 bg-card p-6 shadow-sm">
      <p class="text-xs font-semibold uppercase tracking-wider text-destructive">Application error</p>
      <h1 class="mt-2 text-lg font-semibold">{{ error.name || 'Error' }}</h1>
      <pre
        class="mt-3 max-h-64 overflow-auto whitespace-pre-wrap rounded-lg bg-muted p-3 font-mono text-xs text-muted-foreground"
      >{{ error.message }}{{ error.stack ? '\n\n' + error.stack : '' }}</pre>
      <div class="mt-4 flex flex-wrap gap-2">
        <Button type="button" variant="outline" size="sm" @click="copyDetails">
          {{ copied ? 'Copied!' : 'Copy error' }}
        </Button>
        <Button type="button" variant="outline" size="sm" @click="reset">Try again</Button>
        <Button type="button" size="sm" @click="reload">Reload page</Button>
      </div>
    </div>
  </div>
  <slot v-else />
</template>