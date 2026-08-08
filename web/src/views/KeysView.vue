<script setup lang="ts">
import { ref } from 'vue'
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
import { useKeysStore } from '@/stores'
import { KeyRound, Plus, Copy, Check } from '@lucide/vue'

const store = useKeysStore()
const show = ref(false)
const name = ref('')
const pub = ref('')

function open() {
  name.value = ''
  pub.value = ''
  show.value = true
}
function submit() {
  if (!name.value || !pub.value) return
  const fp = `SHA256:${Math.random().toString(36).slice(2, 18)}${Math.random().toString(36).slice(2, 18)}`
  store.add({ name: name.value, publicKey: pub.value, fingerprint: fp })
  show.value = false
}

const copied = ref<string | null>(null)
async function copy(text: string, id: string) {
  await navigator.clipboard.writeText(text)
  copied.value = id
  setTimeout(() => (copied.value = null), 2000)
}
</script>

<template>
  <div class="grid gap-4">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight">Keys</h1>
        <p class="text-sm text-muted-foreground">
          SSH key pairs for connecting to remote servers. Private keys are encrypted at rest on BE.
        </p>
      </div>
      <Dialog v-model:open="show">
        <DialogTrigger as-child>
          <Button @click="open">
            <Plus class="mr-1 size-4" />Add key
          </Button>
        </DialogTrigger>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Add SSH key</DialogTitle>
            <DialogDescription>Paste your public key. Private key is not stored in the FE.</DialogDescription>
          </DialogHeader>
          <div class="grid gap-2">
            <input
              v-model="name"
              placeholder="name (e.g. deploy-key-vps2)"
              class="rounded-md border border-input bg-background px-3 py-2 text-sm"
            />
            <textarea
              v-model="pub"
              rows="6"
              placeholder="ssh-ed25519 AAAA..."
              class="rounded-md border border-input bg-background px-3 py-2 font-mono text-xs"
            />
          </div>
          <DialogFooter>
            <Button variant="outline" @click="show = false">Cancel</Button>
            <Button @click="submit">Save</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>

    <div class="grid gap-3 md:grid-cols-2">
      <Card v-for="k in store.items" :key="k.id">
        <CardHeader>
          <div class="flex items-center justify-between">
            <CardTitle class="flex items-center gap-2 text-base">
              <KeyRound class="size-4 text-primary" /> {{ k.name }}
            </CardTitle>
            <Badge variant="outline">SSH</Badge>
          </div>
          <CardDescription class="font-mono text-xs">
            {{ k.fingerprint }}
          </CardDescription>
        </CardHeader>
        <CardContent class="grid gap-1">
          <p class="break-all rounded-md bg-muted p-2 font-mono text-xs">{{ k.publicKey }}</p>
          <div class="flex justify-end">
            <Button variant="ghost" size="sm" @click="copy(k.publicKey, k.id)">
              <component :is="copied === k.id ? Check : Copy" class="mr-1 size-3" />
              {{ copied === k.id ? 'Copied!' : 'Copy' }}
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
