const ALLOWED_EXTENSIONS = ['doc', 'docx'] as const

export function getFileExtension(fileName: string) {
  const lastDot = fileName.lastIndexOf('.')
  if (lastDot < 0 || lastDot === fileName.length - 1) {
    return ''
  }
  return fileName.slice(lastDot + 1).toLowerCase()
}

export function isAllowedDocFile(file: File) {
  const extension = getFileExtension(file.name)
  return ALLOWED_EXTENSIONS.includes(extension as (typeof ALLOWED_EXTENSIONS)[number])
}

export function validateDocFile(file: File) {
  if (!isAllowedDocFile(file)) {
    return '仅支持上传 .doc 或 .docx 文件'
  }
  return ''
}
