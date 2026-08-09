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
import { loginSchema } from '@/lib/validators'

const auth = useAuthStore()
const router = useRouter()

const email = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')
const fieldErrors = ref<{ email?: string; password?: string }>({})

async function submit() {
  error.value = ''
  fieldErrors.value = {}
  const parsed = loginSchema.safeParse({ email: email.value, password: password.value })
  if (!parsed.success) {
    const fe: { email?: string; password?: string } = {}
    for (const issue of parsed.error.issues) {
      const key = issue.path[0] as 'email' | 'password'
      if (!fe[key]) fe[key] = issue.message
    }
    fieldErrors.value = fe
    return
  }
  loading.value = true
  try {
    await auth.login(parsed.data.email, parsed.data.password)
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
          Sign in to your dashboard.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form class="space-y-4" @submit.prevent="submit">
          <div class="space-y-2">
            <Label for="u">Email</Label>
            <Input
              id="u"
              v-model="email"
              type="email"
              autocomplete="username"
              placeholder="you@sawang.tech"
              required
            />
            <p v-if="fieldErrors.email" class="text-xs text-destructive">{{ fieldErrors.email }}</p>
          </div>
          <div class="space-y-2">
            <Label for="p">Password</Label>
            <Input
              id="p"
              v-model="password"
              type="password"
              autocomplete="current-password"
              required
            />
            <p v-if="fieldErrors.password" class="text-xs text-destructive">{{ fieldErrors.password }}</p>
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