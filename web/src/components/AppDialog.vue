<script setup lang="ts">
import { computed } from 'vue'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'

/**
 * AppDialog — the single reusable modal dialog for the whole app.
 *
 * Rules (app-wide, enforced here so every dialog behaves the same):
 *  - Clicking OUTSIDE the dialog does NOT close it (user rule).
 *  - Closing only happens via: the ✕ close button, a Cancel action, or
 *    Escape. No accidental dismissals while typing in a form.
 *
 * Usage:
 *  <AppDialog v-model:open="open" title="New Project" description="...">
 *    <template #trigger><Button>New Project</Button></template>   <!-- optional -->
 *    <form ...> ... body fields ... </form>
 *    <template #footer>
 *      <Button variant="outline" @click="open = false">Cancel</Button>
 *      <Button type="submit">Save</Button>
 *    </template>
 *  </AppDialog>
 */
const props = withDefaults(
  defineProps<{
    open: boolean
    title?: string
    description?: string
    /** max-width class for the panel (default sm:max-w-md) */
    widthClass?: string
    /** show the ✕ close button (default true) */
    showCloseButton?: boolean
  }>(),
  {
    title: '',
    description: '',
    widthClass: 'sm:max-w-md',
    showCloseButton: true,
  },
)

const emit = defineEmits<{
  'update:open': [value: boolean]
}>()

const openModel = computed({
  get: () => props.open,
  set: (v: boolean) => emit('update:open', v),
})
</script>

<template>
  <Dialog v-model:open="openModel">
    <DialogTrigger v-if="$slots.trigger" as-child>
      <slot name="trigger" />
    </DialogTrigger>

    <DialogContent :class="widthClass" :show-close-button="showCloseButton">
      <DialogHeader v-if="title || description || $slots.header">
        <slot name="header">
          <DialogTitle v-if="title">{{ title }}</DialogTitle>
          <DialogDescription v-if="description">{{ description }}</DialogDescription>
        </slot>
      </DialogHeader>

      <slot />

      <DialogFooter v-if="$slots.footer">
        <slot name="footer" />
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
