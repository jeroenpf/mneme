import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'registry',
    component: () => import('@/pages/RegistryView.vue'),
  },
  {
    path: '/doc/:id',
    name: 'document',
    component: () => import('@/pages/DocumentView.vue'),
    props: true,
  },
  {
    path: '/memory',
    name: 'memory',
    component: () => import('@/pages/MemoryView.vue'),
  },
]

export const router = createRouter({
  history: createWebHistory(),
  routes,
})
