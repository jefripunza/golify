<script setup lang="ts">
import { computed, ref, watch } from 'vue'
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
import { Loader2, TriangleAlert } from '@lucide/vue'

const props = defineProps<{
  open: boolean
  /** Display name of the item being deleted (shown in the warning). */
  itemName: string
  /** The exact text the user must type to enable the delete button. */
  confirmText: string
  loading?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:open', value: boolean): void
  (e: 'confirm'): void
}>()

const typed = ref('')

watch(
  () => props.open,
  (open) => {
    if (open) typed.value = ''
  },
)

const matches = computed(() => typed.value === props.confirmText)
const canDelete = computed(() => matches.value && !props.loading)

function onOpenChange(open: boolean) {
  if (!props.loading) emit('update:open', open)
}
</script>

<template>
  <Dialog :open="open" @update:open="onOpenChange">
    <DialogContent class="sm:max-w-md">
      <DialogHeader>
        <DialogTitle class="flex items-center gap-2">
          <TriangleAlert class="size-5 text-destructive" />
          Delete {{ itemName }}?
        </DialogTitle>
        <DialogDescription>
          This action <strong class="text-destructive">cannot be undone</strong>. To confirm,
          type <code class="rounded bg-muted px-1.5 py-0.5 font-mono">{{ confirmText }}</code> below.
        </DialogDescription>
      </DialogHeader>

      <div class="grid gap-1.5">
        <Label for="confirm-delete">Type the name to confirm</Label>
        <Input
          id="confirm-delete"
          v-model="typed"
          :disabled="loading"
          :placeholder="confirmText"
          autocomplete="off"
          spellcheck="false"
          @keyup.enter="canDelete && emit('confirm')"
        />
        <p v-if="!matches && typed" class="text-xs text-muted-foreground">
          Doesn't match yet — keep typing.
        </p>
      </div>

      <DialogFooter>
        <Button variant="outline" type="button" :disabled="loading" @click="onOpenChange(false)">
          Cancel
        </Button>
        <Button
          variant="destructive"
          type="button"
          :disabled="!canDelete"
          @click="emit('confirm')"
        >
          <Loader2 v-if="loading" class="mr-1 size-4 animate-spin" />
          {{ loading ? 'Deleting…' : 'Delete' }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
