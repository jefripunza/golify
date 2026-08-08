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
import { Layers, ArrowRight, Globe } from '@lucide/vue'

const route = useRoute()
const store = useProjectsStore()
const projectId = computed(() => String(route.params.projectId))
const project = computed(() => store.get(projectId.value))
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
              <Badge v-if="env.isProduction" variant="destructive">production</Badge>
              <Badge v-else variant="secondary">staging</Badge>
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
