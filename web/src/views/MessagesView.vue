<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useMessagesStore } from '@/stores/messages'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'

const messages = useMessagesStore()
onMounted(() => {
  if (!messages.list.data.value) void messages.list.refresh()
})

const items = computed(() => messages.list.data.value ?? [])
const count = computed(() => items.value.length)
const isLoading = computed(() => messages.list.asyncStatus.value === 'loading')
const errorMsg = computed(() => messages.list.error.value)

function timeAgo(iso: string) {
  return new Date(iso).toLocaleString()
}
</script>

<template>
  <div class="grid gap-4">
    <div class="flex items-end justify-between">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight">Messages</h1>
        <p class="text-sm text-muted-foreground">{{ count }} stored.</p>
      </div>
      <Button variant="outline" @click="messages.list.refresh()">Refresh</Button>
    </div>

    <p v-if="isLoading" class="text-sm text-muted-foreground">Loading…</p>
    <p v-else-if="errorMsg" class="text-sm text-destructive">
      Error: {{ errorMsg.message }}
    </p>
    <p v-else-if="!count" class="text-sm text-muted-foreground">
      No messages yet — send one from the composer.
    </p>

    <div v-else class="grid gap-3">
      <Card v-for="m in items" :key="m.id">
        <CardHeader>
          <div class="flex items-center justify-between gap-2">
            <CardTitle class="text-base">{{ m.title || '(no title)' }}</CardTitle>
            <Badge v-if="m.priority >= 5" variant="destructive">priority {{ m.priority }}</Badge>
            <Badge v-else variant="secondary">priority {{ m.priority }}</Badge>
          </div>
          <CardDescription>{{ timeAgo(m.created_at) }}</CardDescription>
        </CardHeader>
        <CardContent>
          <p class="whitespace-pre-wrap text-sm">{{ m.message }}</p>
          <Button
            size="sm"
            variant="ghost"
            class="mt-2"
            @click="messages.remove.mutate(m.id)"
          >Delete</Button>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
