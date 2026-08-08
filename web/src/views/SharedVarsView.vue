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
import { useVarsStore } from '@/stores'
import { Variable, Plus, Eye, EyeOff } from '@lucide/vue'
import { ref } from 'vue'

const store = useVarsStore()
const showSecrets = ref(false)
</script>

<template>
  <div class="grid gap-4">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight">Shared Variables</h1>
        <p class="text-sm text-muted-foreground">
          Define once, apply globally or scope to project/environment/service.
        </p>
      </div>
      <div class="flex gap-2">
        <Button variant="outline" @click="showSecrets = !showSecrets">
          <component :is="showSecrets ? EyeOff : Eye" class="mr-1 size-4" />
          {{ showSecrets ? 'Hide' : 'Show' }} secrets
        </Button>
        <Button disabled>
          <Plus class="mr-1 size-4" />New variable
        </Button>
      </div>
    </div>

    <Card>
      <CardContent class="p-0">
        <table class="w-full text-sm">
          <thead class="border-b border-border bg-muted/40 text-left text-xs uppercase tracking-wider text-muted-foreground">
            <tr>
              <th class="px-4 py-2">Key</th>
              <th class="px-4 py-2">Value</th>
              <th class="px-4 py-2">Scope</th>
              <th class="px-4 py-2">Updated</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border">
            <tr v-for="v in store.items" :key="v.id" class="hover:bg-muted/30">
              <td class="px-4 py-2 font-mono">
                <Variable class="mr-1 inline size-3 text-primary" />
                {{ v.key }}
              </td>
              <td class="px-4 py-2 font-mono">
                {{ v.isSecret && !showSecrets ? '••••••••' : v.value }}
              </td>
              <td class="px-4 py-2">
                <Badge variant="outline">{{ v.scope }}</Badge>
                <span v-if="v.scopeRef" class="ml-1 text-xs text-muted-foreground">
                  ({{ v.scopeRef }})
                </span>
              </td>
              <td class="px-4 py-2 text-xs text-muted-foreground">
                {{ v.updatedAt.slice(0, 16).replace('T', ' ') }}
              </td>
            </tr>
          </tbody>
        </table>
      </CardContent>
    </Card>
  </div>
</template>
