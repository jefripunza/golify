<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { AppWindow, Database, Wrench, Box, GitBranch, Loader2 } from '@lucide/vue'

const props = defineProps<{ open: boolean; loading?: boolean }>()
const emit = defineEmits<{
  'update:open': [value: boolean]
  create: [input: { name: string; type: 'application' | 'database' | 'tool'; catalog?: string; image?: string }]
}>()

// ─── Catalog data ─────────────────────────────────────────────────────────
const DB_CATALOG = [
  { id: 'postgres', name: 'PostgreSQL', image: 'postgres:16', port: '5432', desc: 'Relational DB' },
  { id: 'mysql', name: 'MySQL', image: 'mysql:8', port: '3306', desc: 'Relational DB' },
  { id: 'mariadb', name: 'MariaDB', image: 'mariadb:11', port: '3306', desc: 'Relational DB' },
  { id: 'redis', name: 'Redis', image: 'redis:7', port: '6379', desc: 'Key-value / cache' },
  { id: 'mongo', name: 'MongoDB', image: 'mongo:7', port: '27017', desc: 'Document DB' },
  { id: 'sqlite', name: 'SQLite', image: 'nouchka/sqlite3', port: '', desc: 'Embedded DB' },
]

const TOOL_CATALOG = [
  { id: 'qdrant', name: 'Qdrant', image: 'qdrant/qdrant:latest', port: '6333', desc: 'Vector DB' },
  { id: 'weaviate', name: 'Weaviate (gowa)', image: 'semitechnologies/weaviate:latest', port: '8080', desc: 'Vector DB' },
  { id: 'meilisearch', name: 'Meilisearch', image: 'getmeili/meilisearch:latest', port: '7700', desc: 'Search engine' },
  { id: 'minio', name: 'MinIO', image: 'minio/minio:latest', port: '9000', desc: 'S3 object storage' },
  { id: 'n8n', name: 'n8n', image: 'n8nio/n8n:latest', port: '5678', desc: 'Workflow automation' },
  { id: 'pgadmin', name: 'pgAdmin', image: 'dpage/pgadmin4:latest', port: '80', desc: 'Postgres admin UI' },
]

type Step = 'root' | 'application-sub' | 'database' | 'tool' | 'app-docker' | 'app-vcs'
const step = ref<Step>('root')
const selectedType = ref<'application' | 'database' | 'tool' | null>(null)
const selectedCatalog = ref<{ id: string; name: string; image: string; port: string } | null>(null)
const form = ref({ name: '', image: '', repo: '' })

function reset() {
  step.value = 'root'
  selectedType.value = null
  selectedCatalog.value = null
  form.value = { name: '', image: '', repo: '' }
}

// Reset to the root picker every time the dialog opens, so a previous
// (completed or cancelled) flow never leaks into the next one.
watch(() => props.open, (open) => {
  if (open) reset()
})

const canSubmit = computed(() => {
  if (step.value === 'app-docker') return form.value.name && form.value.image
  if (step.value === 'app-vcs') return form.value.name && form.value.repo
  return false
})

function submit() {
  if (step.value === 'app-docker') {
    emit('create', { name: form.value.name, type: 'application', catalog: 'docker-image', image: form.value.image })
  } else if (step.value === 'app-vcs') {
    emit('create', { name: form.value.name, type: 'application', catalog: 'version-control', image: form.value.repo })
  }
}
</script>

<template>
  <Dialog :open="open" @update:open="(v: boolean) => { if (!v) reset(); emit('update:open', v) }">
    <DialogContent class="sm:max-w-md">
      <DialogHeader>
        <DialogTitle>Add Service</DialogTitle>
        <DialogDescription>Pick what to deploy into this environment.</DialogDescription>
      </DialogHeader>

      <!-- ROOT: pick category -->
      <div v-if="step === 'root'" class="grid gap-2">
        <button class="flex items-center gap-3 rounded-lg border p-3 text-left transition-colors hover:bg-accent" @click="step = 'application-sub'; selectedType = 'application'">
          <AppWindow class="size-5 text-primary" />
          <div>
            <p class="font-medium">Application</p>
            <p class="text-xs text-muted-foreground">Deploy your app from a Docker image or version control</p>
          </div>
        </button>
        <button class="flex items-center gap-3 rounded-lg border p-3 text-left transition-colors hover:bg-accent" @click="step = 'database'; selectedType = 'database'">
          <Database class="size-5 text-primary" />
          <div>
            <p class="font-medium">Database</p>
            <p class="text-xs text-muted-foreground">Deploy a database engine for storing your application data</p>
          </div>
        </button>
        <button class="flex items-center gap-3 rounded-lg border p-3 text-left transition-colors hover:bg-accent" @click="step = 'tool'; selectedType = 'tool'">
          <Wrench class="size-5 text-primary" />
          <div>
            <p class="font-medium">Tool</p>
            <p class="text-xs text-muted-foreground">Deploy a tool or add-on to extend your application</p>
          </div>
        </button>
      </div>

      <!-- APPLICATION SUB: docker image vs version control -->
      <div v-else-if="step === 'application-sub'" class="grid gap-2">
        <button class="flex items-center gap-3 rounded-lg border p-3 text-left transition-colors hover:bg-accent" @click="step = 'app-docker'">
          <Box class="size-5 text-primary" />
          <div>
            <p class="font-medium">Docker Image</p>
            <p class="text-xs text-muted-foreground">Deploy from a container image (nginx:latest, myapp:v1…)</p>
          </div>
        </button>
        <button class="flex items-center gap-3 rounded-lg border p-3 text-left transition-colors hover:bg-accent" @click="step = 'app-vcs'">
          <GitBranch class="size-5 text-primary" />
          <div>
            <p class="font-medium">Version Control</p>
            <p class="text-xs text-muted-foreground">Deploy from a git repository (auto-build)</p>
          </div>
        </button>
      </div>

      <!-- DATABASE: pick from catalog -->
      <div v-else-if="step === 'database'" class="grid gap-2">
        <button
          v-for="db in DB_CATALOG"
          :key="db.id"
          class="flex items-center gap-3 rounded-lg border p-3 text-left transition-colors hover:bg-accent"
          @click="selectedCatalog = db; emit('create', { name: db.name.toLowerCase(), type: 'database', catalog: db.id, image: db.image })"
        >
          <Database class="size-5 text-primary" />
          <div class="flex-1">
            <p class="font-medium">{{ db.name }}</p>
            <p class="text-xs text-muted-foreground">{{ db.desc }}</p>
          </div>
          <span class="font-mono text-xs text-muted-foreground">{{ db.image }}</span>
        </button>
      </div>

      <!-- TOOL: pick from catalog -->
      <div v-else-if="step === 'tool'" class="grid gap-2">
        <button
          v-for="tool in TOOL_CATALOG"
          :key="tool.id"
          class="flex items-center gap-3 rounded-lg border p-3 text-left transition-colors hover:bg-accent"
          @click="selectedCatalog = tool; emit('create', { name: tool.name.toLowerCase(), type: 'tool', catalog: tool.id, image: tool.image })"
        >
          <Wrench class="size-5 text-primary" />
          <div class="flex-1">
            <p class="font-medium">{{ tool.name }}</p>
            <p class="text-xs text-muted-foreground">{{ tool.desc }}</p>
          </div>
          <span class="font-mono text-xs text-muted-foreground">{{ tool.image }}</span>
        </button>
      </div>

      <!-- APP: docker image form -->
      <div v-else-if="step === 'app-docker'" class="grid gap-3">
        <div class="grid gap-1.5">
          <Label>Service name</Label>
          <Input v-model="form.name" placeholder="my-app" />
        </div>
        <div class="grid gap-1.5">
          <Label>Docker image</Label>
          <Input v-model="form.image" placeholder="nginx:latest" />
        </div>
      </div>

      <!-- APP: version control form -->
      <div v-else-if="step === 'app-vcs'" class="grid gap-3">
        <div class="grid gap-1.5">
          <Label>Service name</Label>
          <Input v-model="form.name" placeholder="my-app" />
        </div>
        <div class="grid gap-1.5">
          <Label>Repository URL</Label>
          <Input v-model="form.repo" placeholder="https://github.com/org/repo.git" />
        </div>
      </div>

      <DialogFooter v-if="step === 'app-docker' || step === 'app-vcs'">
        <Button class="w-full" :disabled="!canSubmit" @click="submit">
          <Loader2 v-if="loading" class="mr-1 size-4 animate-spin" />
          Add Service
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
