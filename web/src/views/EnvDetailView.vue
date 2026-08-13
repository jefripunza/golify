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
import { useProjectsStore } from '@/stores'
import { Box, ArrowRight, Globe } from '@lucide/vue'

const route = useRoute()
const store = useProjectsStore()
const projectId = computed(() => String(route.params.projectId))
const envId = computed(() => String(route.params.envId))
const project = computed(() => store.get(projectId.value))
const env = computed(() => store.getEnv(projectId.value, envId.value))

function statusColor(s: string) {
  switch (s) {
    case 'running': return 'default'
    case 'stopped': return 'secondary'
    case 'building':
    case 'deploying': return 'outline'
    case 'error': return 'destructive'
    default: return 'secondary'
  }
}
</script>

<template>
  <div v-if="!project || !env" class="text-sm text-muted-foreground">Environment not found.</div>
  <div v-else class="grid gap-4">
    <div class="flex items-center gap-2 text-xs text-muted-foreground">
      <RouterLink to="/projects" class="hover:text-foreground">Projects</RouterLink>
      <span>/</span>
      <RouterLink :to="`/projects/${project.id}`" class="hover:text-foreground">{{ project.name }}</RouterLink>
      <span>/</span>
      <RouterLink :to="`/projects/${project.id}?envs=1`" class="hover:text-foreground">{{ env.name }}</RouterLink>
    </div>

    <header>
      <h1 class="text-2xl font-semibold tracking-tight">{{ env.name }}</h1>
      <p class="flex items-center gap-2 text-sm text-muted-foreground">
        <Globe class="size-4" />
        <span class="truncate">{{ env.domains.join(', ') || 'no domain' }}</span>
        <Badge variant="secondary">{{ env.clusterStatus ?? 'Unknown' }}</Badge>
      </p>
    </header>

    <div class="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
      <RouterLink
        v-for="svc in env.services"
        :key="svc.id"
        :to="`/projects/${project.id}/${env.id}/${svc.id}`"
        class="block transition-transform hover:scale-[1.01]"
      >
        <Card>
          <CardHeader>
            <div class="flex items-center justify-between">
              <CardTitle class="flex items-center gap-2 text-base">
                <Box class="size-4 text-primary" />
                {{ svc.name }}
              </CardTitle>
              <Badge :variant="statusColor(svc.status)">{{ svc.status }}</Badge>
            </div>
            <CardDescription class="font-mono text-xs">
              {{ svc.kind }} · {{ svc.image || svc.composePath || '—' }}
            </CardDescription>
          </CardHeader>
          <CardContent class="flex items-center justify-between text-xs text-muted-foreground">
            <span>CPU {{ svc.cpu }}% · {{ svc.memory }} MB</span>
            <Button variant="ghost" size="sm">
              Open <ArrowRight class="ml-1 size-3" />
            </Button>
          </CardContent>
        </Card>
      </RouterLink>
    </div>
  </div>
</template>
