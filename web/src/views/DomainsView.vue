<script setup lang="ts">
import { onMounted, ref } from 'vue'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Globe, Plus, Trash2, Loader2 } from '@lucide/vue'
import { domainSchema } from '@/lib/validators'
import { getAuth } from '@/lib/api'

interface DomainEntry {
  id: number
  host: string
  created_at: string
}

const auth = getAuth()
const items = ref<DomainEntry[]>([])
const host = ref('')
const loading = ref(false)
const saving = ref(false)
const error = ref('')
const fieldError = ref('')
const loaded = ref(false)

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
  // live-strip scheme for display-only hint (validation happens on submit)
  let d = host.value.trim()
  if (d.includes('://')) d = d.slice(d.indexOf('://') + 3)
  d = d.replace(/[/?#].*$/, '').replace(/\.+$/, '').toLowerCase()
  return d || ''
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
    const res = await fetch('/api/v1/domains', {
      method: 'POST',
      headers: apiHeaders(),
      body: JSON.stringify({ host: parsed.data }),
    })
    const data = await res.json().catch(() => ({}))
    if (!res.ok) {
      error.value = data.error ?? 'Gagal menyimpan domain'
      return
    }
    items.value.unshift(data)
    host.value = ''
  } catch {
    error.value = 'Terjadi kesalahan koneksi'
  } finally {
    saving.value = false
  }
}

async function remove(id: number) {
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
          Subdomain tidak diizinkan.
        </p>
      </div>
      <Badge variant="outline" class="font-mono">{{ items.length }} total</Badge>
    </div>

    <Card>
      <CardHeader>
        <CardTitle class="flex items-center gap-2 text-base">
          <Globe class="size-4 text-primary" /> Tambah Domain
        </CardTitle>
        <CardDescription>Masukkan nama domain root (contoh: example.com). Subdomain (mis. sub.example.com) tidak diizinkan.</CardDescription>
      </CardHeader>
      <CardContent>
        <form class="flex flex-col gap-3 sm:flex-row sm:items-start" @submit.prevent="submit">
          <div class="flex-1 space-y-2">
            <Label for="d">Nama Domain</Label>
            <Input
              id="d"
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
          <Button type="submit" class="mt-6" :disabled="saving || !host.trim()">
            <Loader2 v-if="saving" class="mr-1 size-4 animate-spin" />
            <Plus v-else class="mr-1 size-4" />
            Simpan
          </Button>
        </form>
      </CardContent>
    </Card>

    <Card>
      <CardContent class="p-0">
        <template v-if="!loaded">
          <p class="p-4 text-sm text-muted-foreground">Memuat…</p>
        </template>
        <template v-else-if="items.length === 0">
          <p class="p-4 text-sm text-muted-foreground">Belum ada domain. Tambahkan yang pertama di atas.</p>
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
                <Button variant="ghost" size="sm" class="text-destructive hover:text-destructive" @click="remove(d.id)">
                  <Trash2 class="size-4" />
                </Button>
              </td>
            </tr>
          </tbody>
        </table>
      </CardContent>
    </Card>
  </div>
</template>
