<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import type { NavItem } from '@/lib/toc'

const props = defineProps<{ items: NavItem[] }>()

const activeId = ref<string | null>(null)
let observer: IntersectionObserver | null = null

// Scroll-aware highlight: the topmost section crossing the upper band of
// the viewport wins. Guarded — jsdom (and ancient browsers) just get no
// highlight rather than a crash.
onMounted(() => {
  if (typeof IntersectionObserver === 'undefined') return
  observer = new IntersectionObserver(
    (entries) => {
      for (const entry of entries) {
        if (entry.isIntersecting) activeId.value = entry.target.id
      }
    },
    { rootMargin: '-10% 0px -70% 0px' },
  )
  for (const item of props.items) {
    const el = document.getElementById(item.id)
    if (el) observer.observe(el)
  }
})

onBeforeUnmount(() => observer?.disconnect())
</script>

<template>
  <nav class="secnav">
    <h3 class="mn-label">On this page</h3>
    <a
      v-for="item in items"
      :key="item.id"
      :href="`#${item.id}`"
      class="link"
      :class="{ active: activeId === item.id }"
    >
      <span class="num mn-mono-sm">{{ item.num }}</span>
      <span class="title mn-body-sm">{{ item.title }}</span>
    </a>
  </nav>
</template>

<style scoped>
.secnav {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}
.secnav h3 {
  margin-bottom: var(--space-2);
}
.link {
  display: grid;
  grid-template-columns: 1.4em 1fr;
  gap: var(--space-2);
  align-items: baseline;
  padding: var(--space-1) var(--space-2);
  margin-left: calc(-1 * var(--space-2));
  border-left: 2px solid transparent;
  border-radius: var(--radius-xs);
  text-decoration: none;
  transition:
    color var(--duration-fast) var(--ease-out),
    background var(--duration-fast) var(--ease-out);
}
.num {
  color: var(--text-faint);
  text-align: right;
  font-variant-numeric: tabular-nums;
}
.title {
  color: var(--text-muted);
}
.link:hover {
  background: var(--bg-hover);
}
.link:hover .title {
  color: var(--text-secondary);
}
.link.active {
  background: var(--accent-dim);
  border-left-color: var(--accent);
}
.link.active .title {
  color: var(--accent);
  font-weight: 500;
}
.link.active .num {
  color: var(--eyebrow);
}
</style>
