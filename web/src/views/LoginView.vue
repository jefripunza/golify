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
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const router = useRouter()

const username = ref('admin')
const password = ref('')
const loading = ref(false)
const error = ref('')

async function submit() {
  loading.value = true
  error.value = ''
  try {
    await auth.login(username.value, password.value)
    router.replace('/')
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-background p-4">
    <Card class="w-full max-w-sm">
      <CardHeader>
        <CardTitle class="text-2xl">Golify</CardTitle>
        <CardDescription>
          Sign in to the dashboard. Default: admin / bismillah.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form class="space-y-4" @submit.prevent="submit">
          <div class="space-y-2">
            <Label for="u">Username</Label>
            <Input id="u" v-model="username" autocomplete="username" required />
          </div>
          <div class="space-y-2">
            <Label for="p">Password</Label>
            <Input id="p" v-model="password" type="password" autocomplete="current-password" required />
          </div>
          <p v-if="error" class="text-sm text-destructive">{{ error }}</p>
          <Button type="submit" class="w-full" :disabled="loading">
            {{ loading ? 'Signing in…' : 'Sign in' }}
          </Button>
        </form>
      </CardContent>
    </Card>
  </div>
</template>