import { defineStore } from 'pinia'
import { clearTokenPair, getAccessToken, hasTokenPair, setTokenPair } from '@/services/auth/token'
import { login, register } from '@/services/repositories/authorAuthRepository'
import type { Author } from '@/types/api'
import { extractAuthorIdentityFromToken } from '@/utils/jwt'

function buildInitialState() {
  const token = getAccessToken()
  const identity = extractAuthorIdentityFromToken(token)

  return {
    isAuthenticated: hasTokenPair(),
    loading: false,
    errorMessage: '',
    authorName: identity.authorName || '',
    authorID: identity.authorID as number | undefined,
  }
}

export const useAuthStore = defineStore('author-auth', {
  state: () => buildInitialState(),
  actions: {
    hydrateIdentity() {
      const token = getAccessToken()
      const identity = extractAuthorIdentityFromToken(token)
      this.authorName = identity.authorName || this.authorName
      this.authorID = identity.authorID
      this.isAuthenticated = hasTokenPair()
    },
    async signIn(identifier: string, password: string) {
      this.loading = true
      this.errorMessage = ''

      try {
        const trimmedIdentifier = identifier.trim()
        const tokenPair = await login({
          authorName: trimmedIdentifier,
          authorEmail: trimmedIdentifier.includes('@') ? trimmedIdentifier : undefined,
          password,
        })

        setTokenPair(tokenPair)
        this.isAuthenticated = true
        this.authorName = trimmedIdentifier
        this.hydrateIdentity()
      } catch (error) {
        this.errorMessage = error instanceof Error ? error.message : '登录失败'
        this.isAuthenticated = false
        throw error
      } finally {
        this.loading = false
      }
    },
    async signUpThenSignIn(payload: Author) {
      this.loading = true
      this.errorMessage = ''

      try {
        await register(payload)
        const identifier = payload.authorName || payload.authorEmail
        if (!identifier || !payload.password) {
          throw new Error('注册信息不完整，无法自动登录')
        }

        await this.signIn(identifier, payload.password)
      } catch (error) {
        this.errorMessage = error instanceof Error ? error.message : '注册失败'
        throw error
      } finally {
        this.loading = false
      }
    },
    signOut() {
      clearTokenPair()
      this.isAuthenticated = false
      this.authorName = ''
      this.authorID = undefined
      this.errorMessage = ''
    },
  },
})
