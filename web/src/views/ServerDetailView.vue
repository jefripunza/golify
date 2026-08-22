<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
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
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { useServersStore } from '@/stores'
import {
  Server as ServerIcon,
  Cpu,
  MemoryStick,
  HardDrive,
  Box,
  ArrowLeft,
  Loader2,
  RefreshCw,
  KeyRound,
  CheckCircle2,
  XCircle,
  Pencil,
  Trash2,
} from '@lucide/vue'

const route = useRoute()
const store = useServersStore()
const serverId = computed(() => String(route.params.serverId))
const server = computed(() => store.get(serverId.value))

// ── Edit dialog ──────────────────────────────────────────────
const editOpen = ref(false)
const saving = ref(false)
const deleting = ref(false)
const testResult = ref<{ ok: boolean; message: string } | null>(null)
const form = reactive<any>({})

function openEdit() {
  const s = server.value
  if (!s) return
  Object.assign(form, {
    name: s.name,
    host: s.host,
    ip: s.ip,
    region: s.region,
    provider: s.provider,
    sshUser: s.sshUser,
    sshPort: s.sshPort,
    sshAuthType: s.sshAuthType,
    sshPassword: '', // jangan isi — biarkan kosong = pertahankan lama
    sshPrivateKey: '',
    sshPublicKey: s.sshPublicKey,
    kubeEnabled: s.kubeEnabled,
    kubeRole: s.kubeRole,
    kubeVersion: s.kubeVersion,
    kubeJoinCommand: s.kubeJoinCommand,
    kubeCluster: s.kubeCluster,
    labels: s.labels,
    notes: s.notes,
  })
  testResult.value = null
  editOpen.value = true
}

async function saveEdit() {
  if (!server.value) return
  saving.value = true
  try {
    await store.update(server.value.id, { ...form })
    editOpen.value = false
  } catch (e: any) {
    testResult.value = { ok: false, message: e?.message ?? 'Gagal menyimpan' }
  } finally {
    saving.value = false
  }
}

async function removeServer() {
  if (!server.value) return
  deleting.value = true
  try {
    await store.remove(server.value.id)
    location.href = '/servers'
  } catch (e: any) {
    testResult.value = { ok: false, message: e?.message ?? 'Gagal menghapus' }
  } finally {
    deleting.value = false
  }
}

// ── Test & stats real ────────────────────────────────────────
const testing = ref(false)
const loadingStats = ref(false)
const lastTest = ref<{ ok: boolean; message: string } | null>(null)
const stats = ref<any>(null)

async function testNow() {
  if (!server.value) return
  testing.value = true
  lastTest.value = null
  try {
    const res = await store.test(server.value.id)
    lastTest.value = res.ok
      ? { ok: true, message: `SSH OK · ${res.info?.hostname ?? ''} · ${res.info?.os ?? ''}` }
      : { ok: false, message: res.error ?? 'Gagal terhubung' }
    store.refresh()
  } catch (e: any) {
    lastTest.value = { ok: false, message: e?.message ?? 'Gagal koneksi' }
  } finally {
    testing.value = false
  }
}

async function loadStats() {
  if (!server.value) return
  loadingStats.value = true
  try {
    stats.value = await store.stats(server.value.id)
    store.refresh()
  } catch (e: any) {
    stats.value = { error: e?.message ?? 'Gagal ambil stats' }
  } finally {
    loadingStats.value = false
  }
}

onMounted(() => {
  loadStats()
})

const cpu = computed(() => stats.value?.cpu ?? server.value?.cpu ?? 0)
const memUsed = computed(() => stats.value?.memory_used_mb ?? server.value?.memory ?? 0)
const memTotal = computed(() => stats.value?.memory_total_mb ?? server.value?.memoryTotal ?? 1)
const disk = computed(() => stats.value?.disk_pct ?? server.value?.disk ?? 0)
const containers = computed(() => stats.value?.containers ?? server.value?.containers ?? 0)
const labelsObj = computed(() => {
  try { return JSON.parse(server.value?.labels ?? '{}') } catch { return {} }
})
</script>

<template>
  <div v-if="!server" class="text-sm text-muted-foreground">Server not found.</div>
  <div v-else class="grid gap-4">
    <RouterLink to="/servers" class="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground">
      <ArrowLeft class="size-3" />Back to servers
    </RouterLink>

    <header class="flex flex-wrap items-start justify-between gap-2">
      <div>
        <h1 class="flex items-center gap-2 text-2xl font-semibold tracking-tight">
          <ServerIcon class="size-5 text-primary" /> {{ server.name }}
        </h1>
        <p class="font-mono text-xs text-muted-foreground">
          {{ server.sshUser }}@{{ server.host }}:{{ server.sshPort }} · {{ server.ip }}
        </p>
      </div>
      <div class="flex items-center gap-2">
        <Badge v-if="server.kubeEnabled" variant="secondary" class="gap-1">
          <KeyRound class="size-3" /> K8s {{ server.kubeRole }}
        </Badge>
        <Badge :variant="server.status === 'online' ? 'default' : 'destructive'">{{ server.status }}</Badge>
        <Button variant="outline" size="sm" @click="openEdit"><Pencil class="mr-1 size-3.5" />Edit</Button>
        <Button variant="destructive" size="sm" :disabled="deleting" @click="removeServer">
          <Trash2 class="mr-1 size-3.5" />Delete
        </Button>
      </div>
    </header>

    <!-- Action bar: test + refresh stats -->
    <div class="flex flex-wrap gap-2">
      <Button variant="secondary" size="sm" :disabled="testing" @click="testNow">
        <Loader2 v-if="testing" class="mr-1 size-3.5 animate-spin" />
        <RefreshCw v-else class="mr-1 size-3.5" /> Test connection
      </Button>
      <Button variant="outline" size="sm" :disabled="loadingStats" @click="loadStats">
        <Loader2 v-if="loadingStats" class="mr-1 size-3.5 animate-spin" />
        <RefreshCw v-else class="mr-1 size-3.5" /> Refresh stats
      </Button>
      <div v-if="lastTest" :class="lastTest.ok ? 'text-emerald-600' : 'text-destructive'" class="flex items-center gap-1 text-sm">
        <CheckCircle2 v-if="lastTest.ok" class="size-4" />
        <XCircle v-else class="size-4" />
        {{ lastTest.message }}
      </div>
    </div>

    <!-- Stats cards (real via SSH bila tersedia) -->
    <div class="grid gap-3 md:grid-cols-4">
      <Card>
        <CardHeader class="pb-2">
          <CardDescription class="flex items-center gap-1"><Cpu class="size-3" /> CPU</CardDescription>
          <CardTitle class="text-2xl">{{ cpu }}%</CardTitle>
        </CardHeader>
      </Card>
      <Card>
        <CardHeader class="pb-2">
          <CardDescription class="flex items-center gap-1"><MemoryStick class="size-3" /> Memory</CardDescription>
          <CardTitle class="text-2xl">{{ Math.round((memUsed / Math.max(memTotal, 1)) * 100) }}%</CardTitle>
        </CardHeader>
        <CardContent class="font-mono text-xs text-muted-foreground">{{ memUsed }} / {{ memTotal }} MB</CardContent>
      </Card>
      <Card>
        <CardHeader class="pb-2">
          <CardDescription class="flex items-center gap-1"><HardDrive class="size-3" /> Disk</CardDescription>
          <CardTitle class="text-2xl">{{ disk }}%</CardTitle>
        </CardHeader>
      </Card>
      <Card>
        <CardHeader class="pb-2">
          <CardDescription class="flex items-center gap-1"><Box class="size-3" /> Containers</CardDescription>
          <CardTitle class="text-2xl">{{ containers }}</CardTitle>
        </CardHeader>
      </Card>
    </div>

    <!-- Stats detail (real) -->
    <Card v-if="stats && !stats.error">
      <CardHeader><CardTitle class="text-base">System stats (live)</CardTitle></CardHeader>
      <CardContent class="grid gap-2 text-sm sm:grid-cols-2">
        <div v-if="stats.os" class="flex justify-between"><span class="text-muted-foreground">OS</span><span class="font-mono">{{ stats.os }}</span></div>
        <div v-if="stats.kernel" class="flex justify-between"><span class="text-muted-foreground">Kernel</span><span class="font-mono">{{ stats.kernel }}</span></div>
        <div v-if="stats.uptime_sec" class="flex justify-between"><span class="text-muted-foreground">Uptime</span><span class="font-mono">{{ Math.floor(stats.uptime_sec / 3600) }}h {{ Math.floor((stats.uptime_sec % 3600) / 60) }}m</span></div>
        <div v-if="stats.kube_nodes !== undefined" class="flex justify-between"><span class="text-muted-foreground">K8s nodes</span><span class="font-mono">{{ stats.kube_nodes }}</span></div>
      </CardContent>
    </Card>
    <p v-if="stats && stats.error" class="text-sm text-destructive">Stats: {{ stats.error }}</p>

    <!-- Tabs: General / Kubernetes -->
    <Tabs default-value="general">
      <TabsList>
        <TabsTrigger value="general">⚙️ General</TabsTrigger>
        <TabsTrigger value="kubernetes">☸️ Kubernetes</TabsTrigger>
      </TabsList>

      <TabsContent value="general" class="grid gap-3">
        <Card>
          <CardHeader><CardTitle class="text-base">Connection</CardTitle></CardHeader>
          <CardContent class="grid gap-2 text-sm sm:grid-cols-2">
            <div class="flex justify-between"><span class="text-muted-foreground">User</span><span class="font-mono">{{ server.sshUser }}</span></div>
            <div class="flex justify-between"><span class="text-muted-foreground">Port</span><span class="font-mono">{{ server.sshPort }}</span></div>
            <div class="flex justify-between"><span class="text-muted-foreground">Auth</span><span class="font-mono">{{ server.sshAuthType }}</span></div>
            <div class="flex justify-between"><span class="text-muted-foreground">Region</span><span class="font-mono">{{ server.region }}</span></div>
            <div class="flex justify-between"><span class="text-muted-foreground">Provider</span><span class="font-mono">{{ server.provider }}</span></div>
          </CardContent>
        </Card>

        <Card v-if="server.notes">
          <CardHeader><CardTitle class="text-base">Notes</CardTitle></CardHeader>
          <CardContent class="text-sm whitespace-pre-wrap">{{ server.notes }}</CardContent>
        </Card>

        <Card>
          <CardHeader><CardTitle class="text-base">Labels</CardTitle></CardHeader>
          <CardContent class="flex flex-wrap gap-2">
            <Badge v-for="(v, k) in labelsObj" :key="k" variant="outline" class="font-mono">
              {{ k }}={{ v }}
            </Badge>
            <span v-if="Object.keys(labelsObj).length === 0" class="text-sm text-muted-foreground">Tidak ada label.</span>
          </CardContent>
        </Card>
      </TabsContent>

      <TabsContent value="kubernetes" class="grid gap-3">
        <Card>
          <CardHeader>
            <CardTitle class="flex items-center gap-2 text-base">
              <KeyRound class="size-4 text-primary" /> Kubernetes Cluster
            </CardTitle>
            <CardDescription>Status join ke cluster — dikelola dari sini.</CardDescription>
          </CardHeader>
          <CardContent class="grid gap-2 text-sm sm:grid-cols-2">
            <div class="flex justify-between"><span class="text-muted-foreground">Enabled</span><span>{{ server.kubeEnabled ? 'Ya' : 'Tidak' }}</span></div>
            <div class="flex justify-between"><span class="text-muted-foreground">Role</span><span class="font-mono">{{ server.kubeRole }}</span></div>
            <div class="flex justify-between"><span class="text-muted-foreground">Cluster</span><span class="font-mono">{{ server.kubeCluster || '—' }}</span></div>
            <div class="flex justify-between"><span class="text-muted-foreground">Version</span><span class="font-mono">{{ server.kubeVersion || '—' }}</span></div>
            <div v-if="server.kubeJoinCommand" class="sm:col-span-2">
              <p class="mb-1 text-muted-foreground">Join command</p>
              <pre class="overflow-x-auto rounded-md bg-muted p-2 font-mono text-xs">{{ server.kubeJoinCommand }}</pre>
            </div>
          </CardContent>
        </Card>
      </TabsContent>
    </Tabs>

    <!-- ── Edit dialog ──────────────────────────────────────── -->
    <Dialog v-model:open="editOpen">
      <DialogContent class="max-h-[85vh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Edit server</DialogTitle>
          <DialogDescription>Perbarui konfigurasi server.</DialogDescription>
        </DialogHeader>

        <div class="grid gap-4">
          <div class="grid gap-2 rounded-lg border p-3">
            <p class="text-sm font-medium">General</p>
            <div class="grid gap-2 sm:grid-cols-2">
              <div class="grid gap-1"><Label>Name *</Label><Input v-model="form.name" /></div>
              <div class="grid gap-1"><Label>Host / IP *</Label><Input v-model="form.host" /></div>
              <div class="grid gap-1"><Label>Region</Label><Input v-model="form.region" /></div>
              <div class="grid gap-1">
                <Label>Provider</Label>
                <Select v-model="form.provider">
                  <SelectTrigger><SelectValue /></SelectTrigger>
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

          <div class="grid gap-2 rounded-lg border p-3">
            <p class="flex items-center gap-1 text-sm font-medium"><KeyRound class="size-3.5 text-primary" /> SSH</p>
            <div class="grid gap-2 sm:grid-cols-3">
              <div class="grid gap-1"><Label>User</Label><Input v-model="form.sshUser" /></div>
              <div class="grid gap-1"><Label>Port</Label><Input v-model.number="form.sshPort" type="number" min="1" max="65535" /></div>
              <div class="grid gap-1">
                <Label>Auth</Label>
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
              <Label>Password (kosongkan = pertahankan)</Label>
              <Input v-model="form.sshPassword" type="password" placeholder="••••••••" />
            </div>
            <div v-else class="grid gap-1">
              <Label>Private key (kosongkan = pertahankan)</Label>
              <Textarea v-model="form.sshPrivateKey" class="min-h-[100px] font-mono text-xs" placeholder="-----BEGIN OPENSSH PRIVATE KEY-----" />
            </div>
          </div>

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
                <div class="grid gap-1"><Label>Version</Label><Input v-model="form.kubeVersion" placeholder="v1.33.0" /></div>
              </div>
              <div class="grid gap-1"><Label>Cluster</Label><Input v-model="form.kubeCluster" placeholder="golify" /></div>
              <div class="grid gap-1">
                <Label>Join command</Label>
                <Textarea v-model="form.kubeJoinCommand" class="min-h-[60px] font-mono text-xs" placeholder="kubeadm join ..." />
              </div>
            </div>
          </div>

          <div class="grid gap-2 rounded-lg border p-3">
            <p class="text-sm font-medium">Meta</p>
            <div class="grid gap-1"><Label>Labels (JSON)</Label><Input v-model="form.labels" class="font-mono text-xs" /></div>
            <div class="grid gap-1"><Label>Notes</Label><Textarea v-model="form.notes" class="min-h-[60px]" /></div>
          </div>

          <div v-if="testResult" :class="testResult.ok ? 'text-emerald-600' : 'text-destructive'" class="flex items-center gap-1 text-sm">
            <CheckCircle2 v-if="testResult.ok" class="size-4" /><XCircle v-else class="size-4" />
            {{ testResult.message }}
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" :disabled="saving" @click="editOpen = false">Cancel</Button>
          <Button :disabled="saving" @click="saveEdit">
            <Loader2 v-if="saving" class="mr-1 size-4 animate-spin" />Save
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
