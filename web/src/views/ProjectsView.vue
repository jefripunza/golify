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
import { useProjectsStore } from '@/stores'
import { FolderTree, Plus } from '@lucide/vue'

const projects = useProjectsStore()
</script>

<template>
  <div class="grid gap-4">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight">Projects</h1>
        <p class="text-sm text-muted-foreground">
          {{ projects.projects.length }} project(s).
          Click into one to see environments and services.
        </p>
      </div>
      <Button disabled>
        <Plus class="mr-1 size-4" />New project
      </Button>
    </div>

    <div class="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
      <RouterLink
        v-for="p in projects.projects"
        :key="p.id"
        :to="`/projects/${p.id}`"
        class="block transition-transform hover:scale-[1.01]"
      >
        <Card>
          <CardHeader>
            <div class="flex items-center justify-between">
              <CardTitle class="flex items-center gap-2 text-base">
                <FolderTree class="size-4 text-primary" />
                {{ p.name }}
              </CardTitle>
              <Badge variant="secondary">{{ p.environments.length }} env</Badge>
            </div>
            <CardDescription class="line-clamp-2 min-h-[2.5em]">{{ p.description }}</CardDescription>
          </CardHeader>
          <CardContent class="text-xs text-muted-foreground">
            <p>{{ p.environments.reduce((a, e) => a + e.services.length, 0) }} services total</p>
            <p class="mt-1">{{ p.createdAt.slice(0, 10) }}</p>
          </CardContent>
        </Card>
      </RouterLink>
    </div>
  </div>
</template>
