function pad(value: number) {
  return String(value).padStart(2, '0')
}

function localOffset() {
  const minutes = -new Date().getTimezoneOffset()
  const sign = minutes >= 0 ? '+' : '-'
  const abs = Math.abs(minutes)
  const h = pad(Math.floor(abs / 60))
  const m = pad(abs % 60)
  return `${sign}${h}:${m}`
}

export function toRfc3339(input: string) {
  const raw = input.trim()

  if (/^\d{4}-\d{2}-\d{2}$/.test(raw)) {
    return `${raw}T00:00:00${localOffset()}`
  }

  if (/^\d{4}-\d{2}-\d{2}\s\d{2}:\d{2}$/.test(raw)) {
    return `${raw.replace(' ', 'T')}:00${localOffset()}`
  }

  if (/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}(:\d{2})?(Z|[+-]\d{2}:\d{2})$/.test(raw)) {
    return raw.length === 16 ? `${raw}:00${localOffset()}` : raw
  }

  const parsed = new Date(raw)
  if (!Number.isNaN(parsed.getTime())) {
    return parsed.toISOString()
  }

  return raw
}
