import { defineStore } from 'pinia'
import { ref } from 'vue'
import { useQuery } from '@pinia/colada'
import { http } from '@/lib/api'

export const useHealth = defineStore('health', () => {
  const q = useQuery<{ status: string; app: string }, Error>({
    key: ['health'],
    query: () => http.get('api/v1/health').json<{ status: string; app: string }>(),
  })
  return q
})
