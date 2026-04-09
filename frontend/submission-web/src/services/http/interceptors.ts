import type { AxiosInstance, InternalAxiosRequestConfig } from 'axios'
import { clearTokenPair, getAccessToken, getRefreshToken, setTokenPair } from '@/services/auth/token'
import { refreshToken } from '@/services/repositories/authorAuthRepository'

let isRefreshing = false
let waitingQueue: Array<(token: string | null) => void> = []

function processQueue(token: string | null) {
  waitingQueue.forEach((callback) => callback(token))
  waitingQueue = []
}

function attachRequestInterceptor(client: AxiosInstance) {
  client.interceptors.request.use((config: InternalAxiosRequestConfig) => {
    const accessToken = getAccessToken()
    if (accessToken) {
      config.headers.Authorization = `Bearer ${accessToken}`
    }
    return config
  })
}

function attachResponseInterceptor(client: AxiosInstance) {
  client.interceptors.response.use(
    (response) => response,
    async (error) => {
      const originalRequest = error.config
      if (error.response?.status !== 401 || originalRequest._retry) {
        return Promise.reject(error)
      }

      const savedRefreshToken = getRefreshToken()
      if (!savedRefreshToken) {
        clearTokenPair()
        return Promise.reject(error)
      }

      if (isRefreshing) {
        return new Promise((resolve, reject) => {
          waitingQueue.push((token) => {
            if (!token) {
              reject(error)
              return
            }
            originalRequest.headers.Authorization = `Bearer ${token}`
            resolve(client(originalRequest))
          })
        })
      }

      originalRequest._retry = true
      isRefreshing = true

      try {
        const newTokenPair = await refreshToken(savedRefreshToken)
        setTokenPair(newTokenPair)
        processQueue(newTokenPair.access_token)
        originalRequest.headers.Authorization = `Bearer ${newTokenPair.access_token}`
        return client(originalRequest)
      } catch (refreshError) {
        processQueue(null)
        clearTokenPair()
        return Promise.reject(refreshError)
      } finally {
        isRefreshing = false
      }
    },
  )
}

export function setupInterceptors(clients: AxiosInstance[]) {
  clients.forEach((client) => {
    attachRequestInterceptor(client)
    attachResponseInterceptor(client)
  })
}
