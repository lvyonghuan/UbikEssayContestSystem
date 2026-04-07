import type { TokenPair } from '@/types/api'

const ACCESS_KEY = 'ubik_admin_access_token'
const REFRESH_KEY = 'ubik_admin_refresh_token'

export function getAccessToken() {
  return localStorage.getItem(ACCESS_KEY)
}

export function getRefreshToken() {
  return localStorage.getItem(REFRESH_KEY)
}

export function setTokenPair(tokenPair: TokenPair) {
  localStorage.setItem(ACCESS_KEY, tokenPair.access_token)
  localStorage.setItem(REFRESH_KEY, tokenPair.refresh_token)
}

export function clearTokenPair() {
  localStorage.removeItem(ACCESS_KEY)
  localStorage.removeItem(REFRESH_KEY)
}

export function hasTokenPair() {
  return Boolean(getAccessToken() && getRefreshToken())
}
