import { createRouter, createWebHashHistory } from 'vue-router'
import Index from '../pages/Index.vue'

const routes = [
  {
    path: '/',
    redirect: '/dns'
  },
  {
    path: '/dns',
    component: Index,
    meta: { tab: 'dns' }
  },
  {
    path: '/domains',
    component: Index,
    meta: { tab: 'domains' }
  },
  {
    path: '/routes',
    component: Index,
    meta: { tab: 'routes' }
  },
  {
    path: '/users',
    component: Index,
    meta: { tab: 'users' }
  },
  {
    path: '/certificates',
    component: Index,
    meta: { tab: 'certificates' }
  }
]

const router = createRouter({
  history: createWebHashHistory(),
  routes
})

export default router