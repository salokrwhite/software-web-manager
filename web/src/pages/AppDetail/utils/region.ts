export const hasRegionRules = (rules: any) => {
  if (!rules) return false
  if (Array.isArray(rules.templates) && rules.templates.length > 0) {
    return true
  }
  const allow = rules.allow || {}
  const deny = rules.deny || {}
  return (allow.countries?.length || 0) > 0
    || (allow.provinces?.length || 0) > 0
    || (allow.cities?.length || 0) > 0
    || (deny.countries?.length || 0) > 0
    || (deny.provinces?.length || 0) > 0
    || (deny.cities?.length || 0) > 0
}

export const buildRegionTemplate = (values: any, base?: any) => ({
  id: base?.id || `tpl_${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 8)}`,
  name: values.name || base?.name || '未命名模板',
  allow: {
    countries: values.allow_countries || [],
    provinces: values.allow_provinces || [],
    cities: values.allow_cities || []
  },
  deny: {
    countries: values.deny_countries || [],
    provinces: values.deny_provinces || [],
    cities: values.deny_cities || []
  }
})

export const buildRegionRulesFromTemplates = (templates: any[], activeId: string) => ({
  mode: 'allow_deny',
  active_template_id: activeId || '',
  templates
})

export const filterRegionOptions = (countries: string[] | undefined, options: string[]) => {
  if (!Array.isArray(countries) || countries.length === 0) {
    return []
  }
  const set = new Set(countries.map((c) => c.trim()).filter(Boolean))
  return options.filter((item) => {
    const prefix = item.split('|')[0]?.trim()
    return prefix && set.has(prefix)
  })
}

export const filterCityOptions = (countries: string[] | undefined, provinces: string[] | undefined, options: string[]) => {
  if (Array.isArray(provinces) && provinces.length > 0) {
    const set = new Set(provinces.map((p) => p.trim()).filter(Boolean))
    return options.filter((item) => {
      const parts = item.split('|').map((p) => p.trim())
      if (parts.length < 3) return false
      const prefix = `${parts[0]}|${parts[1]}`
      return set.has(prefix)
    })
  }
  return filterRegionOptions(countries, options)
}

export const formatRegionLabel = (value: string) => {
  const parts = value.split('|').map((p) => p.trim()).filter(Boolean)
  if (parts.length <= 1) return value
  return parts.slice(1).join('|')
}

export const formatCityLabel = (value: string) => {
  const parts = value.split('|').map((p) => p.trim()).filter(Boolean)
  if (parts.length <= 2) return value
  return parts.slice(2).join('|')
}

export const applyRegionTemplateToForm = (form: any, tpl: any) => {
  const normalized = tpl || { allow: {}, deny: {} }
  form.setFieldsValue({
    name: normalized.name || '',
    allow_countries: normalized.allow?.countries || [],
    allow_provinces: normalized.allow?.provinces || [],
    allow_cities: normalized.allow?.cities || [],
    deny_countries: normalized.deny?.countries || [],
    deny_provinces: normalized.deny?.provinces || [],
    deny_cities: normalized.deny?.cities || []
  })
}
