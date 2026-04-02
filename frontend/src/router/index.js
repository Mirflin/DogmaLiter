import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import AuthView from '@/views/AuthView.vue'
import VerifyView from '@/views/VerifyView.vue'
import LandingView from '@/views/LandingView.vue'
import DashboardView from '@/views/DashboardView.vue'
import SettingsView from '@/views/SettingsView.vue'
import ProfileView from '@/views/ProfileView.vue'
import PlansView from '@/views/PlansView.vue'
import GamesView from '@/views/GamesView.vue'
import CreateGameView from '@/views/CreateGameView.vue'
import GameDetailView from '@/views/GameDetailView.vue'
import GameSessionView from '@/views/GameSessionView.vue'
import NewsView from '@/views/NewsView.vue'
import NewsDetailView from '@/views/NewsDetailView.vue'
import CreateNewsView from '@/views/CreateNewsView.vue'
import TermsView from '@/views/TermsView.vue'
import PrivacyView from '@/views/PrivacyView.vue'

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
      path: '/settings',
      name: 'settings',
      component: SettingsView,
      meta: { requiresAuth: true },
    },
    {
      path: '/profile',
      name: 'profile',
      component: ProfileView,
      meta: { requiresAuth: true },
    },
    {
      path: '/auth',
      name: 'auth',
      component: AuthView,
    },
    {
      path: '/plans',
      name: 'plans',
      component: PlansView,
    },
    {
      path: '/games',
      name: 'games',
      component: GamesView,
      meta: { requiresAuth: true },
    },
    {
      path: '/games/create',
      name: 'create-game',
      component: CreateGameView,
      meta: { requiresAuth: true },
    },
    {
      path: '/games/:id',
      name: 'game-detail',
      component: GameDetailView,
      meta: { requiresAuth: true },
    },
    {
      path: '/games/:id/play',
      name: 'game-session',
      component: GameSessionView,
      meta: { requiresAuth: true },
    },
    {
      path: '/news',
      name: 'news',
      component: NewsView,
    },
    {
      path: '/news/create',
      name: 'create-news',
      component: CreateNewsView,
      meta: { requiresAuth: true },
    },
    {
      path: '/news/:id',
      name: 'news-detail',
      component: NewsDetailView,
    },
    {
      path: '/join/:code',
      name: 'join-invite',
      redirect: to => ({ path: '/dashboard', query: { join: to.params.code } }),
    },
    {
      path: '/verify',
      name: 'verify',
      component: VerifyView,
    },
    {
      path: '/terms',
      name: 'terms',
      component: TermsView,
    },
    {
      path: '/privacy',
      name: 'privacy',
      component: PrivacyView,
    },
  ],
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  await auth.waitForInit()

  if (to.name === 'home' && auth.isAuthenticated) {
    return { name: 'dashboard' }
  }

  if (to.name === 'auth' && auth.isAuthenticated) {
    return { name: 'dashboard' }
  }

  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    return { name: 'auth' }
  }
})

export default router
