const CHINA_TIMEZONE = 'Asia/Shanghai'
const CHINA_OFFSET = '+08:00'

function pad(value: number) {
  return String(value).padStart(2, '0')
}

function toChinaParts(date: Date) {
  const parts = new Intl.DateTimeFormat('zh-CN', {
    timeZone: CHINA_TIMEZONE,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  }).formatToParts(date)

  const pick = (type: Intl.DateTimeFormatPartTypes) => parts.find((item) => item.type === type)?.value || ''
  return {
    year: pick('year'),
    month: pick('month'),
    day: pick('day'),
    hour: pick('hour'),
    minute: pick('minute'),
    second: pick('second'),
  }
}

function parseChinaNaiveDate(raw: string) {
  const dateOnly = raw.match(/^(\d{4})-(\d{2})-(\d{2})$/)
  if (dateOnly) {
    const year = Number(dateOnly[1])
    const month = Number(dateOnly[2])
    const day = Number(dateOnly[3])
    return new Date(Date.UTC(year, month - 1, day, -8, 0, 0))
  }

  const dateTime = raw.match(/^(\d{4})-(\d{2})-(\d{2})[ T](\d{2}):(\d{2})(?::(\d{2}))?$/)
  if (dateTime) {
    const year = Number(dateTime[1])
    const month = Number(dateTime[2])
    const day = Number(dateTime[3])
    const hour = Number(dateTime[4])
    const minute = Number(dateTime[5])
    const second = Number(dateTime[6] || '0')
    return new Date(Date.UTC(year, month - 1, day, hour - 8, minute, second))
  }

  return null
}

function formatChinaDate(date: Date, withSeconds: boolean) {
  const parts = toChinaParts(date)
  if (withSeconds) {
    return `${parts.year}-${parts.month}-${parts.day} ${parts.hour}:${parts.minute}:${parts.second}`
  }
  return `${parts.year}-${parts.month}-${parts.day} ${parts.hour}:${parts.minute}`
}

export function formatChinaDateTime(input: string) {
  const raw = input.trim()
  if (!raw) {
    return '-'
  }

  if (/^\d{4}-\d{2}-\d{2}\s\d{2}:\d{2}(:\d{2})?$/.test(raw)) {
    return raw.length === 16 ? raw : raw.slice(0, 16)
  }

  const parsedNaive = parseChinaNaiveDate(raw)
  if (parsedNaive) {
    return formatChinaDate(parsedNaive, false)
  }

  const parsed = new Date(raw)
  if (Number.isNaN(parsed.getTime())) {
    return raw
  }
  return formatChinaDate(parsed, false)
}

export function toChinaPickerValue(input: string) {
  const formatted = formatChinaDateTime(input)
  return formatted === '-' ? '' : formatted
}

export function toChinaTimestamp(input: string) {
  const raw = input.trim()
  if (!raw) {
    return null
  }

  const parsedNaive = parseChinaNaiveDate(raw)
  if (parsedNaive) {
    return parsedNaive.getTime()
  }

  const parsed = new Date(raw)
  if (Number.isNaN(parsed.getTime())) {
    return null
  }
  return parsed.getTime()
}

export function toRfc3339(input: string) {
  const raw = input.trim()

  if (/^\d{4}-\d{2}-\d{2}$/.test(raw)) {
    return `${raw}T00:00:00${CHINA_OFFSET}`
  }

  if (/^\d{4}-\d{2}-\d{2}\s\d{2}:\d{2}$/.test(raw)) {
    return `${raw.replace(' ', 'T')}:00${CHINA_OFFSET}`
  }

  if (/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/.test(raw)) {
    return `${raw}:00${CHINA_OFFSET}`
  }

  if (/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[+-]\d{2}:\d{2}$/.test(raw)) {
    return raw
  }

  if (/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$/.test(raw)) {
    const parsed = new Date(raw)
    if (!Number.isNaN(parsed.getTime())) {
      return `${formatChinaDate(parsed, true).replace(' ', 'T')}${CHINA_OFFSET}`
    }
    return raw
  }

  const parsedNaive = parseChinaNaiveDate(raw)
  if (parsedNaive) {
    return `${formatChinaDate(parsedNaive, true).replace(' ', 'T')}${CHINA_OFFSET}`
  }

  const parsed = new Date(raw)
  if (!Number.isNaN(parsed.getTime())) {
    return `${formatChinaDate(parsed, true).replace(' ', 'T')}${CHINA_OFFSET}`
  }

  return raw
}
