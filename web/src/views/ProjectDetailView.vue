<script setup lang="ts">
import { computed, ref, watchEffect } from 'vue'
import { useRoute, useRouter, RouterLink } from 'vue-router'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { useProjectsStore } from '@/stores'
import { Layers, Plus, Pencil, Trash2, Loader2, GitBranch, Server } from '@lucide/vue'
import AppDialog from '@/components/AppDialog.vue'
import ConfirmDeleteDialog from '@/components/ConfirmDeleteDialog.vue'
import type { Environment } from '@/lib/types'

const route = useRoute()
const router = useRouter()
const store = useProjectsStore()
const projectId = computed(() => String(route.params.projectId))
const project = computed(() => store.get(projectId.value))

// If the project has exactly ONE environment, skip the env list and go
// straight to that environment's services (user rule). Exception: when the
// user explicitly asked for the env list (?envs=1, e.g. clicking the env
// name in the breadcrumb), do NOT redirect.
watchEffect(() => {
  const p = project.value
  if (p && p.environments.length === 1 && route.query.envs === undefined) {
    const only = p.environments[0]
    if (route.path === `/project/${p.id}/environments`) {
      router.replace(`/project/${p.id}/environment/${only.id}/services`)
    }
  }
})

function openEnv(env: Environment) {
  void router.push(`/project/${projectId.value}/environment/${env.id}/services`)
}

// ─── Add / Edit env dialog (name + description, same fields as project) ──
const dialogOpen = ref(false)
const editingEnv = ref<Environment | null>(null)
const name = ref('')
const description = ref('')
const saving = ref(false)
const error = ref('')

function openCreate() {
  editingEnv.value = null
  name.value = ''
  description.value = ''
  error.value = ''
  dialogOpen.value = true
}

function openEdit(env: Environment) {
  editingEnv.value = env
  name.value = env.name
  description.value = env.description ?? ''
  error.value = ''
  dialogOpen.value = true
}

async function submit() {
  if (saving.value || !name.value.trim()) return
  saving.value = true
  error.value = ''
  try {
    const input = { name: name.value.trim(), description: description.value.trim() }
    if (editingEnv.value) {
      await store.updateEnv(projectId.value, editingEnv.value.id, input)
    } else {
      await store.createEnv(projectId.value, input)
    }
    dialogOpen.value = false
  } catch (e: any) {
    error.value = e?.message || 'Failed to save environment'
  } finally {
    saving.value = false
  }
}

// ─── Delete env (cascade: only when env has 0 services) ─────────────────
const deleteTarget = ref<Environment | null>(null)
const deleteError = ref('')
const deleting = ref(false)

function requestDelete(env: Environment) {
  deleteTarget.value = env
  deleteError.value = ''
}

async function confirmDelete() {
  if (!deleteTarget.value || deleting.value) return
  deleting.value = true
  deleteError.value = ''
  try {
    await store.removeEnv(projectId.value, deleteTarget.value.id)
    deleteTarget.value = null
  } catch (e: any) {
    deleteError.value = e?.message || 'Failed to delete environment'
  } finally {
    deleting.value = false
  }
}

function statusColor(s: string) {
  switch (s) {
    case 'Running': return 'default'
    case 'Stopped': return 'secondary'
    default: return 'secondary'
  }
}
</script>

<template>
  <div v-if="!project" class="text-sm text-muted-foreground">Project not found.</div>
  <div v-else class="grid gap-4">
    <div class="flex items-center gap-2 text-xs text-muted-foreground">
      <RouterLink to="/projects" class="hover:text-foreground">{{ project.name }}</RouterLink>
      <span>/</span>
      <span>environments</span>
    </div>

    <div class="flex items-center justify-between gap-2">
      <header>
        <h1 class="text-2xl font-semibold tracking-tight">{{ project.name }}</h1>
        <p class="text-sm text-muted-foreground">{{ project.description }}</p>
      </header>
      <Button size="sm" @click="openCreate">
        <Plus class="mr-1 size-4" /> Add env
      </Button>
    </div>

    <div class="grid gap-3 md:grid-cols-2">
      <div
        v-for="env in project.environments"
        :key="env.id"
        class="cursor-pointer transition-transform hover:scale-[1.01]"
        @click="openEnv(env)"
      >
        <Card>
          <CardHeader>
            <div class="flex items-center justify-between gap-2">
              <CardTitle class="flex items-center gap-2 text-base">
                <Layers class="size-4 text-primary" />
                {{ env.name }}
              </CardTitle>
            </div>
            <CardDescription v-if="env.description" class="line-clamp-2">{{ env.description }}</CardDescription>
            <div v-if="env.ipInternal" class="mt-1 flex items-center gap-1.5 text-xs text-muted-foreground">
              <Server class="size-3.5" />
              <span class="font-mono">{{ env.ipInternal }}</span>
            </div>
          </CardHeader>
          <CardContent class="flex items-center justify-between text-xs text-muted-foreground">
            <span>{{ env.services.length }} service(s)</span>
            <div class="flex items-center gap-1">
              <Button variant="ghost" size="sm" type="button" @click.stop="openEdit(env)">
                <Pencil class="size-3" />
              </Button>
              <Button
                variant="ghost"
                size="sm"
                type="button"
                class="text-destructive hover:text-destructive"
                :disabled="deleting"
                @click.stop="requestDelete(env)"
              >
                <Loader2 v-if="deleting" class="size-3 animate-spin" />
                <Trash2 v-else class="size-3" />
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>

    <div v-if="project.sourceId" class="text-xs text-muted-foreground">
      <GitBranch class="mr-1 inline size-3" />Source: {{ project.sourceId }}
    </div>

    <AppDialog
      v-model:open="dialogOpen"
      :title="editingEnv ? 'Edit Environment' : 'Add Environment'"
      :description="editingEnv
        ? 'Edit environment name or description.'
        : 'Create a new environment. Each environment is a Kubernetes cluster (kind).'"
    >
      <form id="env-form" class="grid gap-3" @submit.prevent="submit">
        <div class="grid gap-1.5">
          <Label for="env-name">Name</Label>
          <Input id="env-name" v-model="name" placeholder="mis. staging" required />
        </div>
        <div class="grid gap-1.5">
          <Label for="env-desc">Description</Label>
          <Textarea id="env-desc" v-model="description" rows="2" placeholder="Optional — environment description" />
        </div>
        <p v-if="error" class="text-sm text-destructive">{{ error }}</p>
      </form>
      <template #footer>
        <Button type="submit" form="env-form" :disabled="saving || !name.trim()">
          <Loader2 v-if="saving" class="mr-1 size-4 animate-spin" />
          {{ saving
            ? (editingEnv ? 'Saving…' : 'Creating environment…')
            : (editingEnv ? 'Save Changes' : 'Create Environment') }}
        </Button>
      </template>
    </AppDialog>

    <ConfirmDeleteDialog
      :open="deleteTarget !== null"
      :item-name="'environment'"
      :confirm-text="deleteTarget?.name ?? ''"
      :loading="deleting"
      :error="deleteError"
      @update:open="(v: boolean) => { if (!v) deleteTarget = null }"
      @confirm="confirmDelete"
    />
  </div>
</template>
