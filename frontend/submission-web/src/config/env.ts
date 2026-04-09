export const appConfig = {
  useMock: import.meta.env.VITE_USE_MOCK === 'true',
  submissionBaseURL: import.meta.env.VITE_SUBMISSION_BASE_URL || '/api/submission',
  systemBaseURL: import.meta.env.VITE_SYSTEM_BASE_URL || '/api/system',
  requestTimeout: Number(import.meta.env.VITE_REQUEST_TIMEOUT || 12000),
}
