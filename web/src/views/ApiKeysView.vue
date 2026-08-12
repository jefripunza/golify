<script setup lang="ts">
import { ref } from 'vue'
import {
  Card,
  CardContent,
} from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { useApiKeysStore } from '@/stores'
import { Copy, Check, Plus, Trash2 } from '@lucide/vue'
import ConfirmDeleteDialog from '@/components/ConfirmDeleteDialog.vue'

const store = useApiKeysStore()
const showKey = ref(false)

const copied = ref<string | null>(null)
async function copy(text: string, id: string) {
  await navigator.clipboard.writeText(text)
  copied.value = id
  setTimeout(() => (copied.value = null), 2000)
}

const newKey = ref({ name: '', scopes: 'projects:read' })
function addKey() {
  if (!newKey.value.name) return
  store.addKey({
    name: newKey.value.name,
    prefix: 'ghk_' + Math.random().toString(36).slice(2, 10),
    scopes: newKey.value.scopes.split(/[,\s]+/).filter(Boolean),
    lastUsedAt: null,
    expiresAt: null,
  })
  newKey.value = { name: '', scopes: 'projects:read' }
  showKey.value = false
}

// delete confirmation state
const deleteTarget = ref<{ kind: 'key'; id: string; name: string } | null>(null)
const deleting = ref(false)
function requestDelete(id: string, name: string) {
  deleteTarget.value = { kind: 'key', id, name }
}
async function confirmDelete() {
  if (!deleteTarget.value || deleting.value) return
  deleting.value = true
  try {
    store.revokeKey(deleteTarget.value.id)
    deleteTarget.value = null
  } finally {
    deleting.value = false
  }
}
</script>

<template>
  <div class="grid gap-6">
    <section class="grid gap-4">
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-semibold tracking-tight">API Keys</h1>
          <p class="text-sm text-muted-foreground">For CLI and external integrations.</p>
        </div>
        <Dialog v-model:open="showKey">
          <DialogTrigger as-child>
            <Button>
              <Plus class="mr-1 size-4" />New key
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Create API key</DialogTitle>
              <DialogDescription>Shown once on creation. Store it safely.</DialogDescription>
            </DialogHeader>
            <div class="grid gap-2">
              <input
                v-model="newKey.name"
                placeholder="name (e.g. ci-deploy)"
                class="rounded-md border border-input bg-background px-3 py-2 text-sm"
              />
              <input
                v-model="newKey.scopes"
                placeholder="scopes (comma-separated)"
                class="rounded-md border border-input bg-background px-3 py-2 text-sm"
              />
            </div>
            <DialogFooter>
              <Button variant="outline" @click="showKey = false">Cancel</Button>
              <Button @click="addKey">Create</Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>

      <Card>
        <CardContent class="p-0">
          <table class="w-full text-sm">
            <thead class="border-b border-border bg-muted/40 text-left text-xs uppercase tracking-wider text-muted-foreground">
              <tr>
                <th class="px-4 py-2">Name</th>
                <th class="px-4 py-2">Prefix</th>
                <th class="px-4 py-2">Scopes</th>
                <th class="px-4 py-2">Last used</th>
                <th class="px-4 py-2">Expires</th>
                <th class="px-4 py-2"></th>
              </tr>
            </thead>
            <tbody class="divide-y divide-border">
              <tr v-for="k in store.apiKeys" :key="k.id" class="hover:bg-muted/30">
                <td class="px-4 py-2 font-medium">{{ k.name }}</td>
                <td class="px-4 py-2 font-mono text-xs">
                  <button @click="copy(k.prefix, k.id)" class="flex items-center gap-1 hover:underline">
                    <component :is="copied === k.id ? Check : Copy" class="size-3" />
                    {{ k.prefix }}…
                  </button>
                </td>
                <td class="px-4 py-2">
                  <div class="flex flex-wrap gap-1">
                    <Badge v-for="s in k.scopes" :key="s" variant="outline">{{ s }}</Badge>
                  </div>
                </td>
                <td class="px-4 py-2 text-xs text-muted-foreground">
                  {{ k.lastUsedAt ? k.lastUsedAt.slice(0, 16).replace('T', ' ') : 'never' }}
                </td>
                <td class="px-4 py-2 text-xs text-muted-foreground">
                  {{ k.expiresAt ? k.expiresAt.slice(0, 10) : 'never' }}
                </td>
                <td class="px-4 py-2">
                  <Button variant="ghost" size="sm" type="button" @click="requestDelete(k.id, k.name)">
                    <Trash2 class="size-3" />
                  </Button>
                </td>
              </tr>
            </tbody>
          </table>
        </CardContent>
      </Card>
    </section>

    <ConfirmDeleteDialog
      :open="deleteTarget !== null"
      :item-name="'API key'"
      :confirm-text="deleteTarget?.name ?? ''"
      :loading="deleting"
      @update:open="(v: boolean) => { if (!v) deleteTarget = null }"
      @confirm="confirmDelete"
    />
  </div>
</template>
