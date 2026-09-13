<template>
  <div class="toast-host" aria-live="polite">
    <TransitionGroup name="toast">
      <div v-for="t in toasts" :key="t.id" class="toast" :class="`toast--${t.type}`">
        {{ t.message }}
      </div>
    </TransitionGroup>
  </div>
</template>

<script setup>
import { toasts } from '../stores/toast'
</script>
