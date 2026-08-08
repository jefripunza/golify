<script setup lang="ts">
import { computed, ref } from 'vue'
import { useMessagesStore } from '@/stores/messages'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'

const messages = useMessagesStore()
const title = ref('')
const body = ref('')
const priority = ref(0)
const status = ref<{ ok: boolean; text: string } | null>(null)

const submitting = computed(() => messages.send.asyncStatus.value === 'loading')

async function submit() {
  status.value = null
  if (!body.value.trim()) {
    status.value = { ok: false, text: 'message body required' }
    return
  }
  try {
    await messages.send.mutate({ title: title.value, message: body.value, priority: priority.value })
    status.value = { ok: true, text: 'sent ✓' }
    title.value = ''
    body.value = ''
    priority.value = 0
  } catch (e) {
    status.value = { ok: false, text: (e as Error).message }
  }
}
</script>

<template>
  <Card class="max-w-xl">
    <CardHeader>
      <CardTitle>Send a message</CardTitle>
      <CardDescription>POST /api/v1/message</CardDescription>
    </CardHeader>
    <CardContent>
      <form class="grid gap-4" @submit.prevent="submit">
        <div class="grid gap-1.5">
          <Label for="title">Title</Label>
          <Input id="title" v-model="title" placeholder="Optional" />
        </div>
        <div class="grid gap-1.5">
          <Label for="body">Message</Label>
          <Textarea id="body" v-model="body" :rows="5" placeholder="Required" />
        </div>
        <div class="grid gap-1.5">
          <Label for="prio">Priority (0–10)</Label>
          <Input id="prio" v-model.number="priority" type="number" min="0" max="10" />
        </div>
        <div class="flex items-center gap-3">
          <Button type="submit" :disabled="submitting">
            {{ submitting ? 'Sending…' : 'Send' }}
          </Button>
          <span v-if="status" :class="status.ok ? 'text-sm text-green-500' : 'text-sm text-destructive'">
            {{ status.text }}
          </span>
        </div>
      </form>
    </CardContent>
  </Card>
</template>
