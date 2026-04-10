function arrayBufferToHex(buffer: ArrayBuffer) {
  const bytes = new Uint8Array(buffer)
  let output = ''
  for (const byte of bytes) {
    output += byte.toString(16).padStart(2, '0')
  }
  return output
}

function getSubtleCrypto() {
  if (globalThis.crypto?.subtle) {
    return globalThis.crypto.subtle
  }
  throw new Error('当前环境不支持 Web Crypto，无法计算文件完整性哈希')
}

export async function calculateSHA256FromArrayBuffer(buffer: ArrayBuffer) {
  const subtle = getSubtleCrypto()
  const digest = await subtle.digest('SHA-256', buffer)
  return arrayBufferToHex(digest)
}

export async function calculateFileSHA256(file: File) {
  const fileBuffer = await file.arrayBuffer()
  return calculateSHA256FromArrayBuffer(fileBuffer)
}
