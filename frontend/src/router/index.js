import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import AuthView from '@/views/AuthView.vue'
import VerifyView from '@/views/VerifyView.vue'
import LandingView from '@/views/LandingView.vue'
import DashboardView from '@/views/DashboardView.vue'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'home',
      component: LandingView,
    },
    {
      path: '/dashboard',
      name: 'dashboard',
      component: DashboardView,
      meta: { requiresAuth: true },
    },
    {
      path: '/auth',
      name: 'auth',
      component: AuthView,
    },
    {
      path: '/verify',
      name: 'verify',
      component: VerifyView,
    },
  ],
})

router.beforeEach(async (to, from, next) => {
  const auth = useAuthStore()
  await auth.waitForInit()

  if (to.name === 'home' && auth.isAuthenticated) {
    return next({ name: 'dashboard' })
  }

  if (to.name === 'auth' && auth.isAuthenticated) {
    return next({ name: 'dashboard' })
  }

  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    return next({ name: 'auth' })
  }

  next()
})

export default router
