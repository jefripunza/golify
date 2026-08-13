import { createApp } from 'vue'
import AppToaster from '@/components/AppToaster.vue'

export interface ToastInput {
  title: string
  description?: string
  variant?: 'default' | 'destructive'
}

// Global toast bus — AppToaster.vue registers itself on mount.
let push: ((t: ToastInput) => void) | null = null

export function registerToaster(fn: (t: ToastInput) => void) {
  push = fn
}

export function toast(t: ToastInput) {
  push?.(t)
}

export { AppToaster }
