<script setup lang="ts">
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { useS3Store } from '@/stores'
import { Database, Plus, Star } from '@lucide/vue'

const store = useS3Store()
</script>

<template>
  <div class="grid gap-4">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight">S3 Storages</h1>
        <p class="text-sm text-muted-foreground">
          Backup destinations. Will be used to snapshot <code class="rounded bg-muted px-1.5 py-0.5">/app/data</code>.
        </p>
      </div>
      <Button disabled>
        <Plus class="mr-1 size-4" />Add storage
      </Button>
    </div>

    <div class="grid gap-3 md:grid-cols-2">
      <Card v-for="s in store.items" :key="s.id">
        <CardHeader>
          <div class="flex items-center justify-between">
            <CardTitle class="flex items-center gap-2 text-base">
              <Database class="size-4 text-primary" /> {{ s.name }}
            </CardTitle>
            <Badge v-if="s.isDefault" variant="default">
              <Star class="mr-1 size-3" />default
            </Badge>
          </div>
          <CardDescription class="font-mono text-xs break-all">{{ s.endpoint }}</CardDescription>
        </CardHeader>
        <CardContent class="grid gap-1 text-xs text-muted-foreground">
          <div>region: <span class="font-mono text-foreground">{{ s.region }}</span></div>
          <div>bucket: <span class="font-mono text-foreground">{{ s.bucket }}</span></div>
          <div>access key: <span class="font-mono text-foreground">{{ s.accessKeyId }}</span></div>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
