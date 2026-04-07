import { defineStore } from 'pinia'
import { login } from '@/services/repositories/authRepository'
import { clearTokenPair, hasTokenPair, setTokenPair } from '@/services/auth/token'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    isAuthenticated: hasTokenPair(),
    loading: false,
    errorMessage: '',
    adminName: '',
  }),
  actions: {
    async signIn(adminName: string, password: string) {
      this.loading = true
      this.errorMessage = ''
      try {
        const tokenPair = await login({ adminName, password })
        setTokenPair(tokenPair)
        this.adminName = adminName
        this.isAuthenticated = true
      } catch (error) {
        this.errorMessage = error instanceof Error ? error.message : '登录失败'
        this.isAuthenticated = false
        throw error
      } finally {
        this.loading = false
      }
    },
    signOut() {
      clearTokenPair()
      this.isAuthenticated = false
      this.adminName = ''
    },
  },
})
