import { useEffect, useState } from 'react'
import { api } from '../api/client'

export const DEFAULT_SITE_NAME = 'SWM 软件版本管理平台'
const SITE_NAME_STORAGE_KEY = 'site_name'
export const SITE_NAME_UPDATED_EVENT = 'site-name-updated'
let siteNameRequest: Promise<string> | null = null

const normalizeSiteName = (value: unknown) => {
  const text = typeof value === 'string' ? value.trim() : ''
  return text || DEFAULT_SITE_NAME
}

export const readCachedSiteName = () => normalizeSiteName(localStorage.getItem(SITE_NAME_STORAGE_KEY))

export const cacheSiteName = (siteName: string) => {
  localStorage.setItem(SITE_NAME_STORAGE_KEY, normalizeSiteName(siteName))
}

export const emitSiteNameUpdated = (siteName: string) => {
  const normalized = normalizeSiteName(siteName)
  cacheSiteName(normalized)
  window.dispatchEvent(new CustomEvent<string>(SITE_NAME_UPDATED_EVENT, { detail: normalized }))
}

export const fetchSiteName = async () => {
  if (siteNameRequest) {
    return siteNameRequest
  }
  siteNameRequest = (async () => {
    try {
      const res = await api.get('/api/public/settings')
      const siteName = normalizeSiteName(res?.data?.site_name)
      cacheSiteName(siteName)
      return siteName
    } catch {
      return readCachedSiteName()
    } finally {
      siteNameRequest = null
    }
  })()
  return siteNameRequest
}

export const useSiteName = () => {
  const [siteName, setSiteName] = useState<string>(readCachedSiteName())

  useEffect(() => {
    let active = true
    const sync = async () => {
      const name = await fetchSiteName()
      if (active) {
        setSiteName(name)
      }
    }
    sync()
    const onUpdated = (event: Event) => {
      const custom = event as CustomEvent<string>
      const next = normalizeSiteName(custom.detail || readCachedSiteName())
      setSiteName(next)
    }
    window.addEventListener(SITE_NAME_UPDATED_EVENT, onUpdated)
    return () => {
      active = false
      window.removeEventListener(SITE_NAME_UPDATED_EVENT, onUpdated)
    }
  }, [])

  return siteName
}
