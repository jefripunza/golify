<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, RouterLink } from 'vue-router'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { useServersStore } from '@/stores'
import {
  Server as ServerIcon,
  Cpu,
  MemoryStick,
  HardDrive,
  Box,
  ArrowLeft,
} from '@lucide/vue'

const route = useRoute()
const store = useServersStore()
const serverId = computed(() => String(route.params.serverId))
const server = computed(() => store.get(serverId.value))

const containers = computed(() => {
  const list = server.value?.containers ?? 0
  return Array.from({ length: list }, (_, i) => ({
    name: `container-${i + 1}`,
    status: i % 4 === 0 ? 'stopped' : 'running',
  }))
})
</script>

<template>
  <div v-if="!server" class="text-sm text-muted-foreground">Server not found.</div>
  <div v-else class="grid gap-4">
    <RouterLink to="/servers" class="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground">
      <ArrowLeft class="size-3" />Back to servers
    </RouterLink>

    <header class="flex items-start justify-between">
      <div>
        <h1 class="flex items-center gap-2 text-2xl font-semibold tracking-tight">
          <ServerIcon class="size-5 text-primary" /> {{ server.name }}
        </h1>
        <p class="font-mono text-xs text-muted-foreground">{{ server.host }} · {{ server.ip }}</p>
      </div>
      <Badge :variant="server.status === 'online' ? 'default' : 'destructive'">{{ server.status }}</Badge>
    </header>

    <div class="grid gap-3 md:grid-cols-4">
      <Card>
        <CardHeader class="pb-2">
          <CardDescription class="flex items-center gap-1"><Cpu class="size-3" /> CPU</CardDescription>
          <CardTitle class="text-2xl">{{ server.cpu }}%</CardTitle>
        </CardHeader>
      </Card>
      <Card>
        <CardHeader class="pb-2">
          <CardDescription class="flex items-center gap-1"><MemoryStick class="size-3" /> Memory</CardDescription>
          <CardTitle class="text-2xl">{{ Math.round((server.memory / server.memoryTotal) * 100) }}%</CardTitle>
        </CardHeader>
        <CardContent class="font-mono text-xs text-muted-foreground">
          {{ server.memory }} / {{ server.memoryTotal }} MB
        </CardContent>
      </Card>
      <Card>
        <CardHeader class="pb-2">
          <CardDescription class="flex items-center gap-1"><HardDrive class="size-3" /> Disk</CardDescription>
          <CardTitle class="text-2xl">{{ server.disk }}%</CardTitle>
        </CardHeader>
      </Card>
      <Card>
        <CardHeader class="pb-2">
          <CardDescription class="flex items-center gap-1"><Box class="size-3" /> Containers</CardDescription>
          <CardTitle class="text-2xl">{{ server.containers }}</CardTitle>
        </CardHeader>
      </Card>
    </div>

    <Card>
      <CardHeader>
        <CardTitle>Containers</CardTitle>
        <CardDescription>Mock list — wire to Podman/Docker API later.</CardDescription>
      </CardHeader>
      <CardContent>
        <div class="grid gap-2 md:grid-cols-2 lg:grid-cols-3">
          <div
            v-for="c in containers"
            :key="c.name"
            class="flex items-center justify-between rounded-md border border-border p-3"
          >
            <span class="flex items-center gap-2 text-sm">
              <Box class="size-4 text-primary" /> {{ c.name }}
            </span>
            <Badge :variant="c.status === 'running' ? 'default' : 'secondary'">{{ c.status }}</Badge>
          </div>
        </div>
      </CardContent>
    </Card>

    <div class="flex gap-2">
      <Button variant="outline" disabled>Restart</Button>
      <Button variant="outline" disabled>Pull images</Button>
      <Button variant="destructive" disabled>Power off</Button>
    </div>
  </div>
</template>
