<script setup lang="ts">
import { useToast } from '@/composables/useToast'

// Renders the app-wide toast queue as a stack of dismissible confirmations.
// Mounted once in AppShell so any component can raise a toast via useToast().
const { toasts, dismiss } = useToast()
</script>

<template>
  <div class="toast-host" role="status" aria-live="polite">
    <TransitionGroup name="toast">
      <button
        v-for="t in toasts"
        :key="t.id"
        class="toast mn-mono-sm"
        data-test="toast"
        :title="'Dismiss'"
        @click="dismiss(t.id)"
      >
        {{ t.message }}
      </button>
    </TransitionGroup>
  </div>
</template>

<style scoped>
.toast-host {
  position: fixed;
  right: var(--space-5);
  bottom: var(--space-5);
  z-index: 50;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: var(--space-2);
  pointer-events: none;
}
.toast {
  pointer-events: auto;
  cursor: pointer;
  padding: var(--space-2) var(--space-4);
  border: 1px solid var(--accent-border);
  border-radius: var(--radius-md);
  background: var(--bg-elevated);
  color: var(--text-primary);
  box-shadow: var(--shadow-pop, 0 6px 20px rgb(0 0 0 / 0.18));
}
.toast:focus-visible {
  outline: none;
  box-shadow: var(--shadow-focus);
}
.toast-enter-active,
.toast-leave-active {
  transition:
    opacity var(--duration-fast) var(--ease-out),
    transform var(--duration-fast) var(--ease-out);
}
.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translateY(6px);
}
@media (prefers-reduced-motion: reduce) {
  .toast-enter-active,
  .toast-leave-active {
    transition: none;
  }
}
</style>
