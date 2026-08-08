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
import { useTeamsStore } from '@/stores'
import { Users, ArrowLeft, ShieldCheck, Plus } from '@lucide/vue'

const route = useRoute()
const store = useTeamsStore()
const teamId = computed(() => String(route.params.teamId))
const team = computed(() => store.get(teamId.value))

const scopes = ['projects', 'servers', 'sources', 's3', 'variables', 'keys', 'api-mcp', 'teams'] as const
</script>

<template>
  <div v-if="!team" class="text-sm text-muted-foreground">Team not found.</div>
  <div v-else class="grid gap-4">
    <RouterLink to="/teams" class="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground">
      <ArrowLeft class="size-3" />Back to teams
    </RouterLink>

    <header>
      <h1 class="flex items-center gap-2 text-2xl font-semibold tracking-tight">
        <Users class="size-5 text-primary" /> {{ team.name }}
      </h1>
      <p class="text-sm text-muted-foreground">{{ team.description }}</p>
    </header>

    <Card>
      <CardHeader>
        <div class="flex items-center justify-between">
          <CardTitle>Members</CardTitle>
          <Button size="sm" disabled>
            <Plus class="mr-1 size-3" />Invite
          </Button>
        </div>
      </CardHeader>
      <CardContent class="p-0">
        <table class="w-full text-sm">
          <thead class="border-b border-border bg-muted/40 text-left text-xs uppercase tracking-wider text-muted-foreground">
            <tr>
              <th class="px-4 py-2">Email</th>
              <th class="px-4 py-2">Role</th>
              <th class="px-4 py-2">Joined</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border">
            <tr v-for="m in team.members" :key="m.id">
              <td class="px-4 py-2 font-mono text-xs">{{ m.email }}</td>
              <td class="px-4 py-2">
                <Badge variant="outline">{{ m.role }}</Badge>
              </td>
              <td class="px-4 py-2 text-xs text-muted-foreground">
                {{ m.joinedAt.slice(0, 10) }}
              </td>
            </tr>
          </tbody>
        </table>
      </CardContent>
    </Card>

    <Card>
      <CardHeader>
        <CardTitle>Permissions matrix</CardTitle>
        <CardDescription>
          <code>*</code> = full · <code>read</code> = view-only · array = scoped to those IDs.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div class="grid gap-2 md:grid-cols-2">
          <div v-for="s in scopes" :key="s" class="flex items-center justify-between rounded-md border border-border p-3 text-sm">
            <span class="flex items-center gap-2">
              <ShieldCheck class="size-4 text-primary" /> {{ s }}
            </span>
            <Badge variant="outline" class="font-mono">
              {{ team.permissions[s] ?? '—' }}
            </Badge>
          </div>
        </div>
      </CardContent>
    </Card>
  </div>
</template>
