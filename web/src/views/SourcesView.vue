<script setup lang="ts">
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { useSourcesStore } from '@/stores'
import { GitBranch, Plus, Trash2 } from '@lucide/vue'
import type { SourceProvider } from '@/lib/types'

const store = useSourcesStore()

function iconFor(p: SourceProvider) {
  // lucide doesn't ship brand icons; use GitBranch for all providers.
  return GitBranch
}
</script>

<template>
  <div class="grid gap-4">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight">Sources</h1>
        <p class="text-sm text-muted-foreground">
          Version-control integrations. Supports GitHub, GitLab, Bitbucket, Gitea, Codeberg, and custom.
        </p>
      </div>
      <Button disabled>
        <Plus class="mr-1 size-4" />Add source
      </Button>
    </div>

    <div class="grid gap-3 md:grid-cols-2">
      <Card v-for="s in store.sources" :key="s.id">
        <CardHeader>
          <div class="flex items-center justify-between">
            <CardTitle class="flex items-center gap-2 text-base">
              <component :is="iconFor(s.provider)" class="size-4 text-primary" />
              {{ s.name }}
            </CardTitle>
            <Badge v-if="s.isGlobal" variant="secondary">global</Badge>
            <Badge v-else variant="outline">scoped</Badge>
          </div>
          <CardDescription class="font-mono text-xs break-all">{{ s.url }}</CardDescription>
        </CardHeader>
        <CardContent class="flex items-center justify-between">
          <Badge variant="outline" class="uppercase">{{ s.provider }}</Badge>
          <Button variant="ghost" size="sm" disabled>
            <Trash2 class="mr-1 size-3" />Remove
          </Button>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
