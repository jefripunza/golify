<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter, RouterLink } from 'vue-router'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { useProjectsStore } from '@/stores'
import { Box, ArrowRight, Globe, Trash2 } from '@lucide/vue'
import type { Service } from '@/lib/types'
import ConfirmDeleteDialog from '@/components/ConfirmDeleteDialog.vue'

const route = useRoute()
const router = useRouter()
const store = useProjectsStore()
const projectId = computed(() => String(route.params.projectId))
const envId = computed(() => String(route.params.envId))
const project = computed(() => store.get(projectId.value))
const env = computed(() => store.getEnv(projectId.value, envId.value))

// Dummy services while the backend service CRUD is not wired up yet
// (per Pak Jefri: "service jangan dulu, dummy aja dulu service").
const dummyServices = ref<Service[]>([
  { id: 'svc-dummy-api', name: 'api', kind: 'container', image: 'example/api:latest', status: 'running', cpu: 2.4, memory: 128, ports: ['3000'] },
  { id: 'svc-dummy-web', name: 'web', kind: 'container', image: 'example/web:latest', status: 'running', cpu: 1.1, memory: 64, ports: ['8080'] },
  { id: 'svc-dummy-db', name: 'db', kind: 'container', image: 'postgres:16', status: 'stopped', cpu: 0, memory: 256, ports: ['5432'] },
])

const services = computed<Service[]>(() =>
  env.value?.services?.length ? env.value.services : dummyServices.value,
)

function statusColor(s: string) {
  switch (s) {
    case 'running': return 'default'
    case 'stopped': return 'secondary'
    case 'building':
    case 'deploying': return 'outline'
    case 'error': return 'destructive'
    default: return 'secondary'
  }
}

// ─── Delete cascade ──────────────────────────────────────────────────────
// env can only be deleted when it has 0 services; each service card has its
// own delete button so the user can empty the env first.
const deleteTarget = ref<Service | null>(null)
const deleting = ref(false)
const deleteError = ref('')

function requestDeleteService(s: Service) {
  deleteTarget.value = s
  deleteError.value = ''
}
async function confirmDeleteService() {
  if (!deleteTarget.value || deleting.value) return
  deleting.value = true
  deleteError.value = ''
  try {
    const t = deleteTarget.value
    if (t.id.startsWith('svc-dummy-')) {
      // FE-only dummy placeholder: remove from the local list.
      dummyServices.value = dummyServices.value.filter((s) => s.id !== t.id)
    } else {
      await store.removeService(projectId.value, envId.value, t.id)
    }
    deleteTarget.value = null
  } catch (e: any) {
    deleteError.value = e?.message || 'Failed to delete service'
  } finally {
    deleting.value = false
  }
}

const deleteEnvOpen = ref(false)
async function confirmDeleteEnv() {
  if (deleting.value) return
  deleting.value = true
  deleteError.value = ''
  try {
    await store.removeEnv(projectId.value, envId.value)
    // back to the env list of this project
    void router.push(`/project/${projectId.value}/environments?envs=1`)
  } catch (e: any) {
    deleteError.value = e?.message || 'Failed to delete environment'
    deleting.value = false
  }
}
</script>

<template>
  <div v-if="!project || !env" class="text-sm text-muted-foreground">Environment not found.</div>
  <div v-else class="grid gap-4">
    <div class="flex items-center gap-2 text-xs text-muted-foreground">
      <RouterLink to="/projects" class="hover:text-foreground">{{ project.name }}</RouterLink>
      <span>/</span>
      <RouterLink :to="`/project/${project.id}/environments?envs=1`" class="hover:text-foreground">{{ env.name }}</RouterLink>
    </div>

    <header>
      <div class="flex items-center justify-between gap-2">
        <h1 class="text-2xl font-semibold tracking-tight">{{ env.name }}</h1>
        <Button variant="ghost" size="sm" class="text-destructive hover:text-destructive" @click="deleteEnvOpen = true">
          <Trash2 class="mr-1 size-4" /> Delete env
        </Button>
      </div>
      <p class="flex items-center gap-2 text-sm text-muted-foreground">
        <Globe class="size-4" />
        <span class="truncate">{{ env.domains.join(', ') || 'no domain' }}</span>
        <Badge variant="secondary">{{ env.clusterStatus ?? 'Unknown' }}</Badge>
      </p>
    </header>

    <div class="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
      <div
        v-for="svc in services"
        :key="svc.id"
        class="cursor-pointer transition-transform hover:scale-[1.01]"
        @click="router.push(`/project/${project.id}/environment/${env.id}/service/${svc.id}`)"
      >
        <Card>
          <CardHeader>
            <div class="flex items-center justify-between">
              <CardTitle class="flex items-center gap-2 text-base">
                <Box class="size-4 text-primary" />
                {{ svc.name }}
              </CardTitle>
              <Badge :variant="statusColor(svc.status)">{{ svc.status }}</Badge>
            </div>
            <CardDescription class="font-mono text-xs">
              {{ svc.kind }} · {{ svc.image || svc.composePath || '—' }}
            </CardDescription>
          </CardHeader>
          <CardContent class="flex items-center justify-between text-xs text-muted-foreground">
            <span>CPU {{ svc.cpu }}% · {{ svc.memory }} MB</span>
            <div class="flex items-center gap-1">
              <Button variant="ghost" size="sm">
                Open <ArrowRight class="ml-1 size-3" />
              </Button>
              <Button variant="ghost" size="sm" class="text-destructive hover:text-destructive" @click.stop="requestDeleteService(svc)">
                <Trash2 class="size-3" />
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>

    <ConfirmDeleteDialog
      :open="deleteTarget !== null"
      :item-name="'service'"
      :confirm-text="deleteTarget?.name ?? ''"
      :loading="deleting"
      :error="deleteError"
      @update:open="(v: boolean) => { if (!v) deleteTarget = null }"
      @confirm="confirmDeleteService"
    />

    <ConfirmDeleteDialog
      :open="deleteEnvOpen"
      :item-name="'environment'"
      :confirm-text="env.name"
      :loading="deleting"
      :error="deleteError"
      @update:open="(v: boolean) => { if (!v) deleteEnvOpen = false }"
      @confirm="confirmDeleteEnv"
    />
  </div>
</template>
