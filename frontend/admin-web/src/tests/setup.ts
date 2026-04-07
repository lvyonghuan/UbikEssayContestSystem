import { afterAll, afterEach, beforeAll } from 'vitest'
import { setupServer } from 'msw/node'
import { handlers } from '@/mocks/handlers'

const server = setupServer(...handlers)

beforeAll(() => {
  server.listen({ onUnhandledRequest: 'bypass' })
})

afterEach(() => {
  server.resetHandlers()
  localStorage.clear()
})

afterAll(() => {
  server.close()
})
