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

const REMEMBER_KEY = 'golify.remembered'

const email = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')
const fieldErrors = ref<{ email?: string; password?: string }>({})
const remembered = ref(false)

// Prefill dari localStorage kalau pernah login sukses (biar logout tinggal klik Sign in).
try {
  const raw = localStorage.getItem(REMEMBER_KEY)
  if (raw) {
    const saved = JSON.parse(raw) as { email?: string; password?: string }
    if (saved.email && saved.password) {
      email.value = saved.email
      password.value = saved.password
      remembered.value = true
    }
  }
} catch {
  /* ignore corrupt storage */
}

function saveRemembered(em: string, pw: string) {
  try {
    localStorage.setItem(REMEMBER_KEY, JSON.stringify({ email: em, password: pw }))
    remembered.value = true
  } catch {
    /* storage full/blocked — ignore */
  }
}

function clearRemembered() {
  try {
    localStorage.removeItem(REMEMBER_KEY)
  } catch {
    /* ignore */
  }
  email.value = ''
  password.value = ''
  remembered.value = false
}

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
    saveRemembered(parsed.data.email, parsed.data.password)
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
          <div v-if="remembered" class="flex items-center justify-between text-xs text-muted-foreground">
            <span class="inline-flex items-center gap-1">
              <span class="size-1.5 rounded-full bg-emerald-500" />
              Saved — click Sign in to log back in
            </span>
            <button
              type="button"
              class="underline underline-offset-2 hover:text-foreground"
              @click="clearRemembered"
            >
              Clear saved login
            </button>
          </div>
        </form>
      </CardContent>
    </Card>
  </div>
</template>