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
import { useTeamsStore } from '@/stores'
import { Users, Plus, ShieldCheck } from '@lucide/vue'

const store = useTeamsStore()

const scopeLabels = [
  'projects',
  'servers',
  'sources',
  's3',
  'variables',
  'keys',
  'api-keys',
  'teams',
] as const
</script>

<template>
  <div class="grid gap-4">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight">Teams</h1>
        <p class="text-sm text-muted-foreground">
          Group members and scope their access to specific resources.
        </p>
      </div>
      <Button disabled>
        <Plus class="mr-1 size-4" />New team
      </Button>
    </div>

    <div class="grid gap-3 md:grid-cols-2">
      <RouterLink
        v-for="t in store.items"
        :key="t.id"
        :to="`/teams/${t.id}`"
        class="block transition-transform hover:scale-[1.01]"
      >
        <Card>
          <CardHeader>
            <div class="flex items-center justify-between">
              <CardTitle class="flex items-center gap-2 text-base">
                <Users class="size-4 text-primary" /> {{ t.name }}
              </CardTitle>
              <Badge variant="secondary">{{ t.members.length }} member(s)</Badge>
            </div>
            <CardDescription class="line-clamp-2 min-h-[2.5em]">{{ t.description }}</CardDescription>
          </CardHeader>
          <CardContent class="flex flex-wrap gap-1">
            <Badge v-for="s in scopeLabels" :key="s" variant="outline">
              <ShieldCheck class="mr-1 size-3" />{{ s }}: {{ t.permissions[s] ?? '—' }}
            </Badge>
          </CardContent>
        </Card>
      </RouterLink>
    </div>
  </div>
</template>
