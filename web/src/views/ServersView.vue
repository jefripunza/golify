<script setup lang="ts">
import { RouterLink } from 'vue-router'
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
import { Server as ServerIcon, MapPin, Plus } from '@lucide/vue'

const store = useServersStore()

function statusVariant(s: string) {
  return s === 'online' ? 'default' : s === 'offline' ? 'destructive' : 'secondary'
}
</script>

<template>
  <div class="grid gap-4">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight">Servers</h1>
        <p class="text-sm text-muted-foreground">
          {{ store.onlineCount }} of {{ store.servers.length }} online.
        </p>
      </div>
      <Button disabled>
        <Plus class="mr-1 size-4" />Add server
      </Button>
    </div>

    <div class="grid gap-3 md:grid-cols-2">
      <RouterLink
        v-for="s in store.servers"
        :key="s.id"
        :to="`/servers/${s.id}`"
        class="block transition-transform hover:scale-[1.01]"
      >
        <Card>
          <CardHeader>
            <div class="flex items-center justify-between">
              <CardTitle class="flex items-center gap-2 text-base">
                <ServerIcon class="size-4 text-primary" />
                {{ s.name }}
              </CardTitle>
              <Badge :variant="statusVariant(s.status)">{{ s.status }}</Badge>
            </div>
            <CardDescription class="flex items-center gap-2 text-xs">
              <MapPin class="size-3" />
              <span>{{ s.region }} · {{ s.provider }} · {{ s.ip }}</span>
            </CardDescription>
          </CardHeader>
          <CardContent class="grid grid-cols-3 gap-2 text-xs text-muted-foreground">
            <div>
              <p class="uppercase">CPU</p>
              <p class="font-mono text-foreground">{{ s.cpu }}%</p>
            </div>
            <div>
              <p class="uppercase">MEM</p>
              <p class="font-mono text-foreground">
                {{ Math.round((s.memory / Math.max(s.memoryTotal, 1)) * 100) }}%
              </p>
            </div>
            <div>
              <p class="uppercase">CT</p>
              <p class="font-mono text-foreground">{{ s.containers }}</p>
            </div>
          </CardContent>
        </Card>
      </RouterLink>
    </div>
  </div>
</template>
