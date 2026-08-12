<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { onboardSchema } from '@/lib/validators'
import { setAuth } from '@/lib/api'
import type { OnboardValues } from '@/lib/validators'

const router = useRouter()

// ── wizard state ────────────────────────────────────────────────────────────
// Step 0: welcome / what is golify
// Step 1: create the first admin account (email + strong password + confirm)
const step = ref(0)
const values = ref<OnboardValues>({ email: '', password: '', confirm: '' })
const fieldErrors = ref<Partial<Record<keyof OnboardValues, string>>>({})
const submitError = ref('')
const loading = ref(false)

const steps = ['Welcome', 'Admin account'] as const
const stepIndex = computed(() => step.value)

function nextStep() {
  if (step.value === 0) {
    step.value = 1
    return
  }
  submit()
}

async function submit() {
  submitError.value = ''
  fieldErrors.value = {}
  const parsed = onboardSchema.safeParse(values.value)
  if (!parsed.success) {
    const fe: Partial<Record<keyof OnboardValues, string>> = {}
    for (const issue of parsed.error.issues) {
      const key = issue.path[0] as keyof OnboardValues
      if (!fe[key]) fe[key] = issue.message
    }
    fieldErrors.value = fe
    return
  }
  loading.value = true
  try {
    const res = await fetch('/api/v1/auth/onboard', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: parsed.data.email, password: parsed.data.password }),
    })
    const body = await res.json().catch(() => ({}))
    if (!res.ok) {
      submitError.value = (body as { error?: string }).error ?? `Onboarding failed (${res.status})`
      return
    }
    // auto-login: server returned a token
    setAuth(body)
    router.replace('/')
  } catch (e) {
    submitError.value = (e as Error).message
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <Card class="w-full max-w-md">
    <CardHeader>
      <CardTitle class="text-2xl">Welcome to Golify</CardTitle>
      <CardDescription>
        {{ step === 0 ? 'A self-hosted PaaS dashboard. Let us set up your workspace.' : 'Create the first admin account.' }}
      </CardDescription>
    </CardHeader>
    <CardContent>
      <!-- Step 0: welcome -->
      <div v-if="step === 0" class="space-y-4">
        <ul class="space-y-2 text-sm text-muted-foreground">
          <li class="flex gap-2"><span class="text-primary">✓</span> Deploy &amp; manage projects, environments, services</li>
          <li class="flex gap-2"><span class="text-primary">✓</span> Connect servers, sources and S3 storages</li>
          <li class="flex gap-2"><span class="text-primary">✓</span> Keys, shared variables, teams, API keys</li>
          <li class="flex gap-2"><span class="text-primary">✓</span> Real-time terminal &amp; logs over WebSocket</li>
        </ul>
        <p class="text-xs text-muted-foreground">
          There are no accounts yet, so this is a fresh install. You'll create the
          administrator account in the next step.
        </p>
        <Button type="button" class="w-full" @click="nextStep">Get started →</Button>
      </div>

      <!-- Step 1: admin account -->
      <form v-else class="space-y-4" @submit.prevent="submit">
        <div class="space-y-2">
          <Label for="onb-email">Email</Label>
          <Input
            id="onb-email"
            v-model="values.email"
            type="email"
            autocomplete="email"
            placeholder="admin@sawang.tech"
            required
          />
          <p v-if="fieldErrors.email" class="text-xs text-destructive">{{ fieldErrors.email }}</p>
        </div>
        <div class="space-y-2">
          <Label for="onb-pass">Password</Label>
          <Input
            id="onb-pass"
            v-model="values.password"
            type="password"
            autocomplete="new-password"
            placeholder="Min 8 chars, upper+lower+number+symbol"
            required
          />
          <p v-if="fieldErrors.password" class="text-xs text-destructive">{{ fieldErrors.password }}</p>
        </div>
        <div class="space-y-2">
          <Label for="onb-confirm">Repeat password</Label>
          <Input
            id="onb-confirm"
            v-model="values.confirm"
            type="password"
            autocomplete="new-password"
            placeholder="Repeat password"
            required
          />
          <p v-if="fieldErrors.confirm" class="text-xs text-destructive">{{ fieldErrors.confirm }}</p>
        </div>

        <p v-if="submitError" class="text-sm text-destructive">{{ submitError }}</p>

        <div class="flex gap-2">
          <Button type="button" variant="outline" class="flex-1" @click="step = 0">Back</Button>
          <Button type="submit" class="flex-1" :disabled="loading">
            {{ loading ? 'Creating…' : 'Create admin account' }}
          </Button>
        </div>
      </form>

      <!-- bottom step indicator -->
      <div class="mt-6 flex items-center justify-center gap-1.5">
        <span
          v-for="(s, i) in steps"
          :key="s"
          class="h-1.5 rounded-full transition-all"
          :class="i === stepIndex ? 'w-6 bg-primary' : 'w-1.5 bg-muted'"
        />
      </div>
    </CardContent>
  </Card>
</template>