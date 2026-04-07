export const appConfig = {
  useMock: import.meta.env.VITE_USE_MOCK === 'true',
  adminBaseURL: import.meta.env.VITE_ADMIN_BASE_URL || '/api/admin',
  systemBaseURL: import.meta.env.VITE_SYSTEM_BASE_URL || '/api/system',
  requestTimeout: Number(import.meta.env.VITE_REQUEST_TIMEOUT || 12000),
}
