import { defineStore } from 'pinia'
import { computed } from 'vue'
import { useQuery } from '@pinia/colada'
import { http } from '@/lib/api'

export const useHealth = defineStore('health', () => {
  const q = useQuery<{ status: string; app: string }, Error>({
    key: ['health'],
    query: () => http.get('api/v1/health').json<{ status: string; app: string }>(),
  })
  // Colada v1: `state` is a ComputedRef<{data,error,status}> — `q.state.value.data`.
  const data = computed(() => q.state.value.data ?? null)
  const isPending = computed(() => q.asyncStatus.value === 'loading')
  const error = computed(() => q.state.value.error ?? null)

  return { data, isPending, error }
})