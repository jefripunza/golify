<script setup lang="ts">
import { computed } from 'vue'
import { useMessagesStore } from '@/stores/messages'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Code2, Database, ShieldCheck } from '@lucide/vue'

const messages = useMessagesStore()
const count = computed(() => messages.list.data.value?.length ?? 0)
</script>

<template>
  <div class="grid gap-4">
    <Card>
      <CardHeader>
        <CardTitle class="text-2xl">Gotify</CardTitle>
        <CardDescription>A self-hosted push notification server.</CardDescription>
      </CardHeader>
      <CardContent class="grid gap-3 text-sm">
        <p class="text-muted-foreground">
          Send and receive push notifications via a simple REST API.
          The Vue dashboard and Go backend are packaged as a single binary
          (<code class="rounded bg-muted px-1.5 py-0.5">run</code>) — the FE is embedded via
          <code class="rounded bg-muted px-1.5 py-0.5">go:embed</code>.
        </p>
        <div class="flex flex-wrap gap-2">
          <Badge variant="secondary"><Code2 class="mr-1 size-3" />Vue 3</Badge>
          <Badge variant="secondary"><Code2 class="mr-1 size-3" />Go Fiber v3</Badge>
          <Badge variant="secondary"><Database class="mr-1 size-3" />SQLite</Badge>
          <Badge variant="secondary"><ShieldCheck class="mr-1 size-3" />JWT</Badge>
        </div>
      </CardContent>
    </Card>

    <Card>
      <CardHeader>
        <CardTitle>Quick start</CardTitle>
      </CardHeader>
      <CardContent class="grid gap-2 text-sm">
        <div>
          <span class="text-muted-foreground">Send a message:</span>
          <pre class="mt-1 rounded-md bg-muted p-3 text-xs"><code>curl -X POST http://localhost/api/v1/message \
  -H 'Content-Type: application/json' \
  -d '{"title":"hi","message":"world","priority":3}'</code></pre>
        </div>
        <RouterLink to="/send" class="self-start">
          <Button>Open composer</Button>
        </RouterLink>
      </CardContent>
    </Card>

    <p class="text-center text-sm text-muted-foreground">
      {{ count }} message{{ count === 1 ? '' : 's' }} stored.
    </p>
  </div>
</template>
