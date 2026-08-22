<script setup lang="ts">
import { reactive, ref } from 'vue'
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
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useServersStore } from '@/stores'
import {
  Server as ServerIcon,
  MapPin,
  Plus,
  Loader2,
  CheckCircle2,
  XCircle,
  RefreshCw,
  KeyRound,
} from '@lucide/vue'

const store = useServersStore()

const dialogOpen = ref(false)
const saving = ref(false)
const testing = ref(false)
const testResult = ref<{ ok: boolean; message: string } | null>(null)

const empty = () => ({
  name: '',
  host: '',
  ip: '',
  region: '',
  provider: 'self-hosted' as const,
  sshUser: 'root',
  sshPort: 22,
  sshAuthType: 'password' as const,
  sshPassword: '',
  sshPrivateKey: '',
  sshPublicKey: '',
  kubeEnabled: false,
  kubeRole: 'worker' as const,
  kubeVersion: '',
  kubeJoinCommand: '',
  kubeCluster: '',
  labels: '{}',
  notes: '',
})
const form = reactive(empty())

function openDialog() {
  Object.assign(form, empty())
  testResult.value = null
  dialogOpen.value = true
}

async function testConnection() {
  testing.value = true
  testResult.value = null
  try {
    // Test via API langsung: buat dulu (tanpa menyimpan permanen?).
    // Simpler: test setelah save. Tapi user minta Test sebelum save.
    // Solusi: buat server draft via POST, test, kalau user batal → hapus.
    const created = await store.create({ ...form })
    const res = await store.test(created.id)
    testResult.value = res.ok
      ? { ok: true, message: `Koneksi OK · ${res.info?.hostname ?? ''} · ${res.info?.os ?? ''}` }
      : { ok: false, message: res.error ?? 'Gagal terhubung' }
    // Jangan simpan draft — hapus setelah test agar tidak polusi.
    await store.remove(created.id)
    dialogOpen.value = false
  } catch (e: any) {
    testResult.value = { ok: false, message: e?.message ?? 'Gagal koneksi' }
  } finally {
    testing.value = false
  }
}

async function save() {
  saving.value = true
  try {
    await store.create({ ...form })
    dialogOpen.value = false
  } catch (e: any) {
    testResult.value = { ok: false, message: e?.message ?? 'Gagal menyimpan' }
  } finally {
    saving.value = false
  }
}

function statusVariant(s: string) {
  return s === 'online' ? 'default' : s === 'offline' ? 'destructive' : 'secondary'
}
</script>

<template>
  <div class="grid gap-4">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight">Servers</h1>
        <p class="text-sm text-muted-foreground">
          {{ store.onlineCount }} of {{ store.servers.length }} online · dikelola dari satu dashboard.
        </p>
      </div>
      <Button @click="openDialog">
        <Plus class="mr-1 size-4" />Add server
      </Button>
    </div>

    <div class="grid gap-3 md:grid-cols-2">
      <RouterLink
        v-for="s in store.servers"
        :key="s.id"
        :to="`/servers/${s.id}`"
        class="block transition-transform hover:scale-[1.01]"
      >
        <Card>
          <CardHeader>
            <div class="flex items-center justify-between">
              <CardTitle class="flex items-center gap-2 text-base">
                <ServerIcon class="size-4 text-primary" />
                {{ s.name }}
              </CardTitle>
              <div class="flex items-center gap-2">
                <Badge v-if="s.kubeEnabled" variant="secondary" class="gap-1">
                  <KeyRound class="size-3" /> K8s
                </Badge>
                <Badge :variant="statusVariant(s.status)">{{ s.status }}</Badge>
              </div>
            </div>
            <CardDescription class="flex items-center gap-2 text-xs">
              <MapPin class="size-3" />
              <span>{{ s.region }} · {{ s.provider }} · {{ s.ip }}</span>
            </CardDescription>
          </CardHeader>
          <CardContent class="grid grid-cols-3 gap-2 text-xs text-muted-foreground">
            <div>
              <p class="uppercase">CPU</p>
              <p class="font-mono text-foreground">{{ s.cpu }}%</p>
            </div>
            <div>
              <p class="uppercase">MEM</p>
              <p class="font-mono text-foreground">
                {{ Math.round((s.memory / Math.max(s.memoryTotal, 1)) * 100) }}%
              </p>
            </div>
            <div>
              <p class="uppercase">CT</p>
              <p class="font-mono text-foreground">{{ s.containers }}</p>
            </div>
          </CardContent>
        </Card>
      </RouterLink>
    </div>

    <!-- ── Add server dialog ───────────────────────────────────── -->
    <Dialog v-model:open="dialogOpen">
      <DialogContent class="max-h-[85vh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Add server</DialogTitle>
          <DialogDescription>
            Hubungkan server remote via SSH. Semua server diakses dari satu dashboard ini.
          </DialogDescription>
        </DialogHeader>

        <div class="grid gap-4">
          <!-- General -->
          <div class="grid gap-2 rounded-lg border p-3">
            <p class="text-sm font-medium">General</p>
            <div class="grid gap-2 sm:grid-cols-2">
              <div class="grid gap-1">
                <Label>Name *</Label>
                <Input v-model="form.name" placeholder="vps-jakarta-1" />
              </div>
              <div class="grid gap-1">
                <Label>Host / IP *</Label>
                <Input v-model="form.host" placeholder="203.0.113.10" />
              </div>
              <div class="grid gap-1">
                <Label>Region</Label>
                <Input v-model="form.region" placeholder="ap-southeast-3" />
              </div>
              <div class="grid gap-1">
                <Label>Provider</Label>
                <Select v-model="form.provider">
                  <SelectTrigger><SelectValue placeholder="Provider" /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="self-hosted">Self-hosted</SelectItem>
                    <SelectItem value="aws">AWS</SelectItem>
                    <SelectItem value="gcp">GCP</SelectItem>
                    <SelectItem value="azure">Azure</SelectItem>
                    <SelectItem value="digitalocean">DigitalOcean</SelectItem>
                    <SelectItem value="hetzner">Hetzner</SelectItem>
                    <SelectItem value="other">Other</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
          </div>

          <!-- SSH -->
          <div class="grid gap-2 rounded-lg border p-3">
            <p class="flex items-center gap-1 text-sm font-medium">
              <KeyRound class="size-3.5 text-primary" /> SSH Connection
            </p>
            <div class="grid gap-2 sm:grid-cols-3">
              <div class="grid gap-1">
                <Label>User</Label>
                <Input v-model="form.sshUser" placeholder="root" />
              </div>
              <div class="grid gap-1">
                <Label>Port</Label>
                <Input v-model.number="form.sshPort" type="number" min="1" max="65535" />
              </div>
              <div class="grid gap-1">
                <Label>Auth method</Label>
                <Select v-model="form.sshAuthType">
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="password">Password</SelectItem>
                    <SelectItem value="private_key">Private key</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
            <div v-if="form.sshAuthType === 'password'" class="grid gap-1">
              <Label>Password</Label>
              <Input v-model="form.sshPassword" type="password" placeholder="••••••••" />
            </div>
            <div v-else class="grid gap-1">
              <Label>Private key (PEM)</Label>
              <Textarea
                v-model="form.sshPrivateKey"
                placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"
                class="min-h-[120px] font-mono text-xs"
              />
            </div>
            <div class="grid gap-1">
              <Label>Public key (opsional)</Label>
              <Input v-model="form.sshPublicKey" placeholder="ssh-ed25519 AAAA... user@host" />
            </div>
          </div>

          <!-- Kubernetes -->
          <div class="grid gap-2 rounded-lg border p-3">
            <label class="flex items-center gap-2 text-sm font-medium">
              <input v-model="form.kubeEnabled" type="checkbox" class="size-4" />
              Join Kubernetes cluster
            </label>
            <div v-if="form.kubeEnabled" class="grid gap-2">
              <div class="grid gap-2 sm:grid-cols-2">
                <div class="grid gap-1">
                  <Label>Role</Label>
                  <Select v-model="form.kubeRole">
                    <SelectTrigger><SelectValue /></SelectTrigger>
                    <SelectContent>
                      <SelectItem value="control-plane">Control-plane</SelectItem>
                      <SelectItem value="worker">Worker</SelectItem>
                      <SelectItem value="etcd">Etcd</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div class="grid gap-1">
                  <Label>Kubernetes version</Label>
                  <Input v-model="form.kubeVersion" placeholder="v1.33.0" />
                </div>
              </div>
              <div class="grid gap-1">
                <Label>Cluster name</Label>
                <Input v-model="form.kubeCluster" placeholder="golify" />
              </div>
              <div class="grid gap-1">
                <Label>Join command (kubeadm)</Label>
                <Textarea
                  v-model="form.kubeJoinCommand"
                  placeholder="kubeadm join 203.0.113.1:6443 --token ..."
                  class="min-h-[80px] font-mono text-xs"
                />
              </div>
            </div>
          </div>

          <!-- Meta -->
          <div class="grid gap-2 rounded-lg border p-3">
            <p class="text-sm font-medium">Meta</p>
            <div class="grid gap-2">
              <div class="grid gap-1">
                <Label>Labels (JSON)</Label>
                <Input v-model="form.labels" placeholder='{"role":"worker","region":"jakarta"}' class="font-mono text-xs" />
              </div>
              <div class="grid gap-1">
                <Label>Notes</Label>
                <Textarea v-model="form.notes" placeholder="Catatan..." class="min-h-[60px]" />
              </div>
            </div>
          </div>

          <div v-if="testResult" :class="testResult.ok ? 'text-emerald-600' : 'text-destructive'" class="flex items-center gap-1 text-sm">
            <CheckCircle2 v-if="testResult.ok" class="size-4" />
            <XCircle v-else class="size-4" />
            {{ testResult.message }}
          </div>
        </div>

        <DialogFooter class="flex gap-2">
          <Button variant="outline" :disabled="saving || testing" @click="dialogOpen = false">Cancel</Button>
          <Button variant="secondary" :disabled="saving" @click="testConnection">
            <Loader2 v-if="testing" class="mr-1 size-4 animate-spin" />
            <RefreshCw v-else class="mr-1 size-4" /> Test connection
          </Button>
          <Button :disabled="saving || testing" @click="save">
            <Loader2 v-if="saving" class="mr-1 size-4 animate-spin" />
            <Plus v-else class="mr-1 size-4" /> Save
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
