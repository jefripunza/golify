<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
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
import AppDialog from '@/components/AppDialog.vue'
import { useProjectsStore } from '@/stores'
import { toast } from '@/lib/toast'
import type { Project } from '@/lib/types'
import { FolderTree, Plus, Pencil, Loader2, Trash2 } from '@lucide/vue'
import ConfirmDeleteDialog from '@/components/ConfirmDeleteDialog.vue'

const projects = useProjectsStore()
const router = useRouter()

// NOTE: no onMounted refresh here — the store's watchEffect fetches once on
// mount (token guard), and realtime WS (/api/ws/realtime) refetches whenever
// data actually changes. Adding another refresh here caused double hits.

// dialog state (create + edit)
const dialogOpen = ref(false)
const editing = ref<Project | null>(null)
const name = ref('')
const description = ref('')
const creating = ref(false)
const deleting = ref<string | null>(null)
const error = ref('')

function openProject(p: Project) {
  // navigate to the project's environments (list env page)
  void router.push(`/project/${p.id}/environments`)
}

function openCreate() {
  editing.value = null
  name.value = ''
  description.value = ''
  error.value = ''
  dialogOpen.value = true
}

function openEdit(p: Project) {
  editing.value = p
  name.value = p.name
  description.value = p.description ?? ''
  error.value = ''
  dialogOpen.value = true
}

async function submit() {
  if (!name.value.trim() || creating.value) return
  creating.value = true
  error.value = ''
  try {
    if (editing.value) {
      await projects.update(editing.value.id, name.value.trim(), description.value.trim())
    } else {
      await projects.create(name.value.trim(), description.value.trim())
    }
    dialogOpen.value = false
  } catch (e: any) {
    error.value = e?.message || 'Failed to save project'
  } finally {
    creating.value = false
  }
}

async function removeProject(id: string) {
  if (deleting.value) return
  deleting.value = id
  try {
    await projects.remove(id)
  } catch (e: any) {
    error.value = e?.message || 'Failed to delete project'
  } finally {
    deleting.value = null
  }
}

// delete confirmation dialog state
const deleteTarget = ref<Project | null>(null)
const deleteError = ref('')
function requestDelete(p: Project) {
  // Cascade rule: project with environments can't be deleted. Tell the
  // user right away with a toast instead of opening the confirm dialog.
  const envCount = p.envCount ?? p.environments?.length ?? 0
  if (envCount > 0) {
    toast({
      title: 'Cannot delete project',
      description: `Project "${p.name}" still has ${envCount} environment(s). Delete all environments (clusters) first.`,
      variant: 'destructive',
    })
    return
  }
  deleteTarget.value = p
  deleteError.value = ''
}
async function confirmDelete() {
  if (!deleteTarget.value || deleting.value) return
  deleting.value = deleteTarget.value.id
  deleteError.value = ''
  try {
    await projects.remove(deleteTarget.value.id)
    deleteTarget.value = null
  } catch (e: any) {
    deleteError.value = e?.message || 'Failed to delete project'
    toast({
      title: 'Cannot delete project',
      description: deleteError.value,
      variant: 'destructive',
    })
  } finally {
    deleting.value = null
  }
}
</script>

<template>
  <div class="grid gap-4">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight">Projects</h1>
        <p class="text-sm text-muted-foreground">
          Each project is a folder of environments. Every environment is a Kubernetes cluster (kind).
        </p>
      </div>
      <Button size="sm" @click="openCreate">
        <Plus class="mr-1 size-4" /> New Project
      </Button>
    </div>

    <div class="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
      <div
        v-for="p in projects.projects"
        :key="p.id"
        class="cursor-pointer transition-transform hover:scale-[1.01]"
        @click="openProject(p)"
      >
        <Card>
          <CardHeader>
            <div class="flex items-center justify-between gap-2">
              <CardTitle class="flex items-center gap-2 text-base">
                <FolderTree class="size-4 text-primary" />
                {{ p.name }}
              </CardTitle>
              <Badge variant="secondary">{{ p.envCount ?? p.environments.length }} env</Badge>
            </div>
            <CardDescription class="line-clamp-2 min-h-[2.5em]">{{ p.description || '—' }}</CardDescription>
          </CardHeader>
          <CardContent class="flex items-center justify-between text-xs text-muted-foreground">
            <div class="flex items-center gap-1">
              <Button variant="ghost" size="sm" type="button" @click.stop="openEdit(p)">
                <Pencil class="size-3" />
              </Button>
              <Button
                variant="ghost"
                size="sm"
                type="button"
                class="text-destructive hover:text-destructive"
                :disabled="deleting === p.id"
                @click.stop="requestDelete(p)"
              >
                <Loader2 v-if="deleting === p.id" class="size-3 animate-spin" />
                <Trash2 v-else class="size-3" />
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>

    <AppDialog v-model:open="dialogOpen" :title="editing ? 'Edit Project' : 'New Project'" :description="editing
      ? 'Edit project name or description. Environments (and their clusters) are untouched.'
      : 'Create a project folder. A default production environment with its own Kubernetes cluster is created automatically.'">
      <form id="p-form" class="grid gap-3" @submit.prevent="submit">
        <div class="grid gap-1.5">
          <Label for="p-name">Name</Label>
          <Input id="p-name" v-model="name" placeholder="mis. my-app" required />
        </div>
        <div class="grid gap-1.5">
          <Label for="p-desc">Description</Label>
          <Textarea id="p-desc" v-model="description" rows="2" placeholder="Optional — project description" />
        </div>
        <p v-if="error" class="text-sm text-destructive">{{ error }}</p>
      </form>
      <template #footer>
        <Button type="submit" form="p-form" :disabled="creating || !name.trim()">
          <Loader2 v-if="creating" class="mr-1 size-4 animate-spin" />
          {{ creating
            ? (editing ? 'Saving…' : 'Creating project…')
            : (editing ? 'Save Changes' : 'Create Project') }}
        </Button>
      </template>
    </AppDialog>

    <ConfirmDeleteDialog
      :open="deleteTarget !== null"
      :item-name="'project'"
      :confirm-text="deleteTarget?.name ?? ''"
      :loading="deleting !== null"
      :error="deleteError"
      @update:open="(v: boolean) => { if (!v) deleteTarget = null }"
      @confirm="confirmDelete"
    />
  </div>
</template>
