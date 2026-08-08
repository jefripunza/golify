import { defineStore } from 'pinia'
import { useQuery, useMutation, useQueryCache } from '@pinia/colada'
import { http, type Message } from '@/lib/api'

export const useMessagesStore = defineStore('messages', () => {
  const cache = useQueryCache()

  const list = useQuery({
    key: ['messages'],
    query: () => http.get('api/v1/message').json<Message[]>(),
  })

  const send = useMutation({
    mutation: async (input: { title: string; message: string; priority: number }) => {
      const r = await http.post('api/v1/message', { json: input }).json<{ id: number }>()
      return r.id
    },
    onSettled: () => cache.invalidateQueries({ key: ['messages'] }),
  })

  const remove = useMutation({
    mutation: async (id: number) => {
      await http.delete(`api/v1/message/${id}`)
    },
    onSettled: () => cache.invalidateQueries({ key: ['messages'] }),
  })

  return { list, send, remove }
})

export const useHealth = defineStore('health', () => {
  return useQuery({
    key: ['health'],
    query: () => http.get('api/v1/health').json<{ status: string; app: string }>(),
  })
})
