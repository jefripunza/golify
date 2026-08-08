<script setup lang="ts">
import { ref, computed } from 'vue'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
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
import { useApiMcpStore } from '@/stores'
import { Plug, Plus, Copy, Check, Power, Trash2 } from '@lucide/vue'

const store = useApiMcpStore()
const showKey = ref(false)
const showMcp = ref(false)

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

const newMcp = ref({ name: '', url: '', transport: 'http' as 'http' | 'sse' | 'stdio', apiKeyId: '' })
function addMcp() {
  if (!newMcp.value.name || !newMcp.value.url || !newMcp.value.apiKeyId) return
  store.addMcp({ ...newMcp.value, enabled: true })
  newMcp.value = { name: '', url: '', transport: 'http', apiKeyId: '' }
  showMcp.value = false
}

const apiKeyOptions = computed(() => store.apiKeys)
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
                  <Button variant="ghost" size="sm" @click="store.revokeKey(k.id)">
                    <Trash2 class="size-3" />
                  </Button>
                </td>
              </tr>
            </tbody>
          </table>
        </CardContent>
      </Card>
    </section>

    <section class="grid gap-4">
      <div class="flex items-center justify-between">
        <div>
          <h2 class="flex items-center gap-2 text-xl font-semibold tracking-tight">
            <Plug class="size-5 text-primary" />MCP Endpoints
          </h2>
          <p class="text-sm text-muted-foreground">Expose this dashboard to AI agents via MCP.</p>
        </div>
        <Dialog v-model:open="showMcp">
          <DialogTrigger as-child>
            <Button>
              <Plus class="mr-1 size-4" />New MCP
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Add MCP endpoint</DialogTitle>
              <DialogDescription>Requires an existing API key for auth.</DialogDescription>
            </DialogHeader>
            <div class="grid gap-2">
              <input v-model="newMcp.name" placeholder="name" class="rounded-md border border-input bg-background px-3 py-2 text-sm" />
              <input v-model="newMcp.url" placeholder="https://..." class="rounded-md border border-input bg-background px-3 py-2 text-sm" />
              <select v-model="newMcp.transport" class="rounded-md border border-input bg-background px-3 py-2 text-sm">
                <option value="http">http</option>
                <option value="sse">sse</option>
                <option value="stdio">stdio</option>
              </select>
              <select v-model="newMcp.apiKeyId" class="rounded-md border border-input bg-background px-3 py-2 text-sm">
                <option value="">select API key…</option>
                <option v-for="k in apiKeyOptions" :key="k.id" :value="k.id">{{ k.name }} ({{ k.prefix }}…)</option>
              </select>
            </div>
            <DialogFooter>
              <Button variant="outline" @click="showMcp = false">Cancel</Button>
              <Button @click="addMcp">Add</Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>

      <div class="grid gap-3 md:grid-cols-2">
        <Card v-for="m in store.mcp" :key="m.id">
          <CardHeader>
            <div class="flex items-center justify-between">
              <CardTitle class="flex items-center gap-2 text-base">{{ m.name }}</CardTitle>
              <Badge :variant="m.enabled ? 'default' : 'secondary'">
                {{ m.enabled ? 'enabled' : 'disabled' }}
              </Badge>
            </div>
            <CardDescription class="font-mono text-xs break-all">{{ m.url }}</CardDescription>
          </CardHeader>
          <CardContent class="flex items-center justify-between">
            <Badge variant="outline" class="uppercase">{{ m.transport }}</Badge>
            <div class="flex gap-1">
              <Button variant="ghost" size="sm" @click="store.toggleMcp(m.id)">
                <Power class="size-3" />
              </Button>
              <Button variant="ghost" size="sm" @click="store.removeMcp(m.id)">
                <Trash2 class="size-3" />
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    </section>
  </div>
</template>
