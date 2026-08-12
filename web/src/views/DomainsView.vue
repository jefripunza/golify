<script setup lang="ts">
import { onMounted, ref } from 'vue'
import {
  Card,
  CardContent,
} from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Globe, Plus, Pencil, Trash2, Loader2 } from '@lucide/vue'
import { domainSchema } from '@/lib/validators'
import { getAuth } from '@/lib/api'

interface DomainEntry {
  id: string
  host: string
  created_at: string
}

const auth = getAuth()
const items = ref<DomainEntry[]>([])
const loaded = ref(false)
const loading = ref(false)
const saving = ref(false)
const error = ref('')
const fieldError = ref('')

// dialog state
const dialogOpen = ref(false)
const editingId = ref<string | null>(null)
const host = ref('')

function apiHeaders() {
  return {
    'Content-Type': 'application/json',
    ...(auth?.token ? { Authorization: `Bearer ${auth.token}` } : {}),
  }
}

async function load() {
  try {
    const res = await fetch('/api/v1/domains', { headers: apiHeaders() })
    if (res.ok) items.value = await res.json()
  } catch {
    /* non-fatal */
  } finally {
    loaded.value = true
  }
}

function preview() {
  let d = host.value.trim()
  if (d.includes('://')) d = d.slice(d.indexOf('://') + 3)
  d = d.replace(/[/?#].*$/, '').replace(/\.+$/, '').toLowerCase()
  return d || ''
}

function openCreate() {
  editingId.value = null
  host.value = ''
  fieldError.value = ''
  error.value = ''
  dialogOpen.value = true
}

function openEdit(d: DomainEntry) {
  editingId.value = d.id
  host.value = d.host
  fieldError.value = ''
  error.value = ''
  dialogOpen.value = true
}

async function submit() {
  error.value = ''
  fieldError.value = ''
  const parsed = domainSchema.safeParse(host.value)
  if (!parsed.success) {
    fieldError.value = parsed.error.issues[0]?.message ?? 'Domain tidak valid'
    return
  }
  saving.value = true
  try {
    const isEdit = editingId.value !== null
    const res = await fetch(`/api/v1/domains${isEdit ? `/${editingId.value}` : ''}`, {
      method: isEdit ? 'PATCH' : 'POST',
      headers: apiHeaders(),
      body: JSON.stringify({ host: parsed.data }),
    })
    const data = await res.json().catch(() => ({}))
    if (!res.ok) {
      error.value = data.error ?? 'Gagal menyimpan domain'
      return
    }
    if (isEdit) {
      const i = items.value.findIndex((d) => d.id === editingId.value)
      if (i >= 0) items.value[i] = data
    } else {
      items.value.unshift(data)
    }
    dialogOpen.value = false
  } catch {
    error.value = 'Terjadi kesalahan koneksi'
  } finally {
    saving.value = false
  }
}

async function remove(id: string) {
  try {
    await fetch(`/api/v1/domains/${id}`, { method: 'DELETE', headers: apiHeaders() })
    items.value = items.value.filter((d) => d.id !== id)
  } catch {
    /* non-fatal */
  }
}

onMounted(load)
</script>

<template>
  <div class="grid gap-4">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight">Domains</h1>
        <p class="text-sm text-muted-foreground">
          Kelola daftar domain root. <code class="rounded bg-muted px-1.5 py-0.5">http://</code> /
          <code class="rounded bg-muted px-1.5 py-0.5">https://</code> otomatis dihilangkan saat disimpan.
        </p>
      </div>
      <div class="flex items-center gap-2">
        <Badge variant="outline" class="font-mono">{{ items.length }} total</Badge>
        <Button size="sm" @click="openCreate">
          <Plus class="mr-1 size-4" /> Tambah Domain
        </Button>
      </div>
    </div>

    <Card>
      <CardContent class="p-0">
        <template v-if="!loaded">
          <p class="p-4 text-sm text-muted-foreground">Memuat…</p>
        </template>
        <template v-else-if="items.length === 0">
          <p class="p-4 text-sm text-muted-foreground">Belum ada domain. Klik "Tambah Domain" untuk menambah.</p>
        </template>
        <table v-else class="w-full text-sm">
          <thead class="border-b border-border bg-muted/40 text-left text-xs uppercase tracking-wider text-muted-foreground">
            <tr>
              <th class="px-4 py-2">Domain</th>
              <th class="px-4 py-2">Ditambahkan</th>
              <th class="px-4 py-2 text-right">Aksi</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border">
            <tr v-for="d in items" :key="d.id" class="hover:bg-muted/30">
              <td class="px-4 py-2">
                <Globe class="mr-1 inline size-3 text-primary" />
                <span class="font-mono">{{ d.host }}</span>
              </td>
              <td class="px-4 py-2 text-xs text-muted-foreground">
                {{ d.created_at ? d.created_at.slice(0, 16).replace('T', ' ') : '—' }}
              </td>
              <td class="px-4 py-2 text-right">
                <Button variant="ghost" size="sm" type="button" @click="openEdit(d)">
                  <Pencil class="size-4" />
                </Button>
                <Button variant="ghost" size="sm" type="button" class="text-destructive hover:text-destructive" @click="remove(d.id)">
                  <Trash2 class="size-4" />
                </Button>
              </td>
            </tr>
          </tbody>
        </table>
      </CardContent>
    </Card>

    <Dialog v-model:open="dialogOpen">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{{ editingId ? 'Edit Domain' : 'Tambah Domain' }}</DialogTitle>
          <DialogDescription>
            Masukkan nama domain root (contoh: example.com). Subdomain (mis. sub.example.com) tidak diizinkan.
          </DialogDescription>
        </DialogHeader>
        <form class="grid gap-3" @submit.prevent="submit">
          <div class="grid gap-1.5">
            <Label for="d-host">Nama Domain</Label>
            <Input
              id="d-host"
              v-model="host"
              placeholder="example.com"
              :disabled="saving"
              autocomplete="off"
              spellcheck="false"
            />
            <p v-if="fieldError" class="text-xs text-destructive">{{ fieldError }}</p>
            <p v-if="!fieldError && preview()" class="text-xs text-muted-foreground">
              Akan disimpan sebagai: <span class="font-mono">{{ preview() }}</span>
            </p>
            <p v-if="error" class="text-xs text-destructive">{{ error }}</p>
          </div>
          <DialogFooter>
            <Button
              type="submit"
              :disabled="saving || !host.trim()"
            >
              <Loader2 v-if="saving" class="mr-1 size-4 animate-spin" />
              {{ saving ? 'Menyimpan…' : editingId ? 'Simpan Perubahan' : 'Simpan' }}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  </div>
</template>
