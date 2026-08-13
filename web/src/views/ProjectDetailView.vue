<script setup lang="ts">
import { computed, watchEffect } from 'vue'
import { useRoute, useRouter, RouterLink } from 'vue-router'
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
import { Layers, ArrowRight, Globe, GitBranch } from '@lucide/vue'

const route = useRoute()
const router = useRouter()
const store = useProjectsStore()
const projectId = computed(() => String(route.params.projectId))
const project = computed(() => store.get(projectId.value))

// If the project has exactly ONE environment, skip the env list and go
// straight to that environment's services (user rule). Exception: when the
// user explicitly asked for the env list (?envs=1, e.g. clicking the env
// name in the breadcrumb), do NOT redirect.
watchEffect(() => {
  const p = project.value
  if (p && p.environments.length === 1 && route.query.envs === undefined) {
    const only = p.environments[0]
    if (route.path === `/projects/${p.id}`) {
      router.replace(`/projects/${p.id}/${only.id}`)
    }
  }
})

function statusColor(s: string) {
  switch (s) {
    case 'Running': return 'default'
    case 'Stopped': return 'secondary'
    default: return 'secondary'
  }
}
</script>

<template>
  <div v-if="!project" class="text-sm text-muted-foreground">Project not found.</div>
  <div v-else class="grid gap-4">
    <div class="flex items-center gap-2 text-xs text-muted-foreground">
      <RouterLink to="/projects" class="hover:text-foreground">Projects</RouterLink>
      <span>/</span>
      <span>{{ project.name }}</span>
    </div>

    <header>
      <h1 class="text-2xl font-semibold tracking-tight">{{ project.name }}</h1>
      <p class="text-sm text-muted-foreground">{{ project.description }}</p>
    </header>

    <div class="grid gap-3 md:grid-cols-2">
      <RouterLink
        v-for="env in project.environments"
        :key="env.id"
        :to="`/projects/${project.id}/${env.id}`"
        class="block transition-transform hover:scale-[1.01]"
      >
        <Card>
          <CardHeader>
            <div class="flex items-center justify-between">
              <CardTitle class="flex items-center gap-2 text-base">
                <Layers class="size-4 text-primary" />
                {{ env.name }}
              </CardTitle>
              <div class="flex items-center gap-1">
                <Badge v-if="env.isProduction" variant="destructive">production</Badge>
                <Badge v-else variant="secondary">staging</Badge>
                <Badge :variant="statusColor(env.clusterStatus ?? 'Unknown')">
                  {{ env.clusterStatus ?? 'Unknown' }}
                </Badge>
              </div>
            </div>
            <CardDescription class="flex items-center gap-2 text-xs">
              <Globe class="size-3" />
              <span class="truncate">{{ env.domains.join(', ') || 'no domain' }}</span>
            </CardDescription>
          </CardHeader>
          <CardContent class="flex items-center justify-between text-xs text-muted-foreground">
            <span>{{ env.services.length }} service(s)</span>
            <Button variant="ghost" size="sm">
              Open <ArrowRight class="ml-1 size-3" />
            </Button>
          </CardContent>
        </Card>
      </RouterLink>
    </div>

    <div v-if="project.sourceId" class="text-xs text-muted-foreground">
      <GitBranch class="mr-1 inline size-3" />Source: {{ project.sourceId }}
    </div>
  </div>
</template>
