export const parseRegionRules = (raw: any) => {
  if (!raw) return null
  if (typeof raw === 'string') {
    try {
      return JSON.parse(raw)
    } catch {
      return null
    }
  }
  return raw
}

export const parseJSONValue = (raw: any) => {
  if (!raw) return null
  if (typeof raw === 'string') {
    try {
      return JSON.parse(raw)
    } catch {
      return null
    }
  }
  return raw
}

export const parseWhitelist = (raw: any) => {
  const parsed = parseJSONValue(raw)
  return Array.isArray(parsed) ? parsed : []
}
