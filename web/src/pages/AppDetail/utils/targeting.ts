const normalizeList = (value: any): string[] => {
  if (!Array.isArray(value)) return []
  return value
    .map((item) => String(item || '').trim())
    .filter(Boolean)
}

export const buildTargetingRules = (values: any) => {
  const out: any = {}
  const userIDs = normalizeList(values?.user_ids)
  const deviceIDs = normalizeList(values?.device_ids)
  const platforms = normalizeList(values?.platforms)
  const archs = normalizeList(values?.archs)
  const minVersion = String(values?.min_version || '').trim()
  const maxVersion = String(values?.max_version || '').trim()

  if (userIDs.length > 0) out.user_ids = userIDs
  if (deviceIDs.length > 0) out.device_ids = deviceIDs
  if (platforms.length > 0) out.platforms = platforms
  if (archs.length > 0) out.archs = archs
  if (minVersion) out.min_version = minVersion
  if (maxVersion) out.max_version = maxVersion
  return out
}

export const hasTargetingRules = (rules: any) => {
  if (!rules || typeof rules !== 'object') return false
  return Object.keys(buildTargetingRules(rules)).length > 0
}

export const summarizeTargetingRules = (rules: any) => {
  const normalized = buildTargetingRules(rules)
  const parts: string[] = []

  if (Array.isArray(normalized.platforms) && normalized.platforms.length > 0) {
    parts.push(`平台 ${normalized.platforms.join(', ')}`)
  }
  if (Array.isArray(normalized.archs) && normalized.archs.length > 0) {
    parts.push(`架构 ${normalized.archs.join(', ')}`)
  }
  if (Array.isArray(normalized.user_ids) && normalized.user_ids.length > 0) {
    parts.push(`用户 ${normalized.user_ids.length}`)
  }
  if (Array.isArray(normalized.device_ids) && normalized.device_ids.length > 0) {
    parts.push(`设备 ${normalized.device_ids.length}`)
  }
  if (normalized.min_version || normalized.max_version) {
    parts.push(`版本 ${normalized.min_version || '*'} ~ ${normalized.max_version || '*'}`)
  }

  return parts
}
