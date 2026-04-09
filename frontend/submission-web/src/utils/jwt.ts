interface TokenPayload {
  [key: string]: unknown
}

function decodeBase64Url(input: string) {
  const normalized = input.replace(/-/g, '+').replace(/_/g, '/')
  const padded = normalized.padEnd(Math.ceil(normalized.length / 4) * 4, '=')
  return atob(padded)
}

function tryParsePayload(token: string | null): TokenPayload | null {
  if (!token) {
    return null
  }

  const parts = token.split('.')
  if (parts.length < 2) {
    return null
  }

  try {
    const payloadText = decodeBase64Url(parts[1])
    const payload = JSON.parse(payloadText)
    return payload && typeof payload === 'object' ? (payload as TokenPayload) : null
  } catch {
    return null
  }
}

function pickNumber(payload: TokenPayload, keys: string[]) {
  for (const key of keys) {
    const value = payload[key]
    if (typeof value === 'number' && Number.isFinite(value)) {
      return value
    }
    if (typeof value === 'string' && value.trim() && !Number.isNaN(Number(value))) {
      return Number(value)
    }
  }
  return undefined
}

function pickText(payload: TokenPayload, keys: string[]) {
  for (const key of keys) {
    const value = payload[key]
    if (typeof value === 'string' && value.trim()) {
      return value.trim()
    }
  }
  return undefined
}

export function extractAuthorIdentityFromToken(token: string | null) {
  const payload = tryParsePayload(token)
  if (!payload) {
    return { authorID: undefined, authorName: undefined }
  }

  const authorID = pickNumber(payload, ['authorID', 'authorId', 'author_id', 'userID', 'user_id', 'sub'])
  const authorName = pickText(payload, ['authorName', 'author_name', 'name', 'nickname', 'sub'])

  return { authorID, authorName }
}
