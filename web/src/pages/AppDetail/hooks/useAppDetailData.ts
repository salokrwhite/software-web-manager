import { message } from 'antd'
import { useEffect, useState } from 'react'
import api from '../../../api/client'
import { parseJSONValue, parseRegionRules, parseWhitelist } from '../utils/parse'

const useAppDetailData = (appId?: string) => {
  const [app, setApp] = useState<any>(null)
  const [channels, setChannels] = useState<any[]>([])
  const [releases, setReleases] = useState<any[]>([])
  const [appSecrets, setAppSecrets] = useState<any[]>([])
  const [releaseChannels, setReleaseChannels] = useState<any[]>([])
  const [appMembers, setAppMembers] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [releaseTemplates, setReleaseTemplates] = useState<any[]>([])
  const [regionTemplates, setRegionTemplates] = useState<any[]>([])
  const [activeRegionTemplateId, setActiveRegionTemplateId] = useState<string>('')
  const [regionEnabled, setRegionEnabled] = useState(false)
  const [regionOptions, setRegionOptions] = useState<{ countries: string[]; provinces: string[]; cities: string[] }>({
    countries: [],
    provinces: [],
    cities: []
  })

  const load = async () => {
    if (!appId) return
    setLoading(true)
    try {
      const [appRes, channelRes, releaseRes, secretRes, relChannelRes, memberRes, templateRes, regionRes, regionOptionRes] = await Promise.all([
        api.get(`/api/apps/${appId}`),
        api.get(`/api/apps/${appId}/channels`),
        api.get(`/api/apps/${appId}/releases`),
        api.get(`/api/apps/${appId}/app-secrets`),
        api.get(`/api/apps/${appId}/release-channels`),
        api.get(`/api/apps/${appId}/members`),
        api.get('/api/release-templates'),
        api.get(`/api/apps/${appId}/region-rules`),
        api.get('/api/geo/regions')
      ])
      const rawApp = appRes.data.app || {}
      setApp({
        id: rawApp.ID || rawApp.id,
        name: rawApp.Name || rawApp.name,
        slug: rawApp.Slug || rawApp.slug,
        description: rawApp.Description || rawApp.description || '',
        public_key: rawApp.PublicKey || rawApp.public_key || '',
        created_at: rawApp.CreatedAt || rawApp.created_at,
        feedback_enabled: rawApp.FeedbackEnabled ?? rawApp.feedback_enabled ?? true,
        heartbeat_interval_seconds: rawApp.HeartbeatIntervalSeconds ?? rawApp.heartbeat_interval_seconds ?? 60,
        online_enabled: rawApp.OnlineEnabled ?? rawApp.online_enabled ?? true,
        status: (rawApp.Status || rawApp.status || 'active').toLowerCase(),
        submitted_at: rawApp.SubmittedAt || rawApp.submitted_at,
        rejection_reason: rawApp.RejectionReason || rawApp.rejection_reason
      })
      setChannels((channelRes.data.items || []).map((c: any) => ({
        id: c.ID || c.id,
        name: c.Name || c.name,
        code: c.Code || c.code,
        is_default: c.IsDefault ?? c.is_default,
        min_supported_version: c.MinSupportedVersion || c.min_supported_version
      })))
      setReleases((prev) => (releaseRes.data.items || []).map((r: any) => {
        const id = r.ID || r.id
        const prevItem = prev.find((item) => item.id === id)
        return {
          id,
          version: r.Version || r.version,
          version_code: r.VersionCode ?? r.version_code ?? null,
          status: r.Status || r.status,
          created_at: r.CreatedAt || r.created_at,
          notes: r.Notes || r.notes,
          external_download_url: r.ExternalDownloadURL || r.external_download_url || '',
          release_template_id: r.ReleaseTemplateID || r.release_template_id || null,
          approved_at: r.ApprovedAt || r.approved_at,
          submitted_at: r.SubmittedAt || r.submitted_at,
          artifact_count: r.ArtifactCount ?? r.artifact_count ?? prevItem?.artifact_count ?? 0
        }
      }))
      setAppSecrets((secretRes.data.items || []).map((k: any) => ({
        id: k.ID || k.id || k.app_id || rawApp.ID || rawApp.id,
        app_id: k.AppID || k.app_id || rawApp.ID || rawApp.id,
        name: k.Name || k.name || k.app_secret_name || 'app_secret',
        type: k.Type || k.type || 'app_secret',
        key_hash: k.KeyHash || k.key_hash,
        scopes: k.Scopes || k.scopes,
        expires_at: k.ExpiresAt || k.expires_at,
        last_used_at: k.LastUsedAt || k.last_used_at,
        created_at: k.CreatedAt || k.created_at,
        updated_at: k.UpdatedAt || k.updated_at
      })))
      setReleaseChannels((relChannelRes.data.items || []).map((item: any) => ({
        ...item,
        targeting_rules: parseJSONValue(item.targeting_rules),
        region_rules: parseRegionRules(item.region_rules),
        whitelist: parseWhitelist(item.whitelist)
      })))
      setAppMembers((memberRes.data.items || []).map((m: any) => ({
        app_id: m.AppID || m.app_id,
        user_id: m.UserID || m.user_id,
        role: m.Role || m.role,
        created_at: m.CreatedAt || m.created_at
      })))
      setReleaseTemplates((templateRes.data.items || []).map((t: any) => ({
        id: t.ID || t.id,
        name: t.Name || t.name,
        schedule_at: t.ScheduleAt || t.schedule_at,
        window_start: t.WindowStart || t.window_start,
        window_end: t.WindowEnd || t.window_end,
        emergency: t.Emergency ?? t.emergency
      })))
      const regionRules = parseRegionRules(regionRes?.data?.region_rules)
      const templates = Array.isArray(regionRules?.templates) ? regionRules.templates : []
      const activeTemplateId = regionRules?.active_template_id || (templates[0]?.id || '')
      setRegionTemplates(templates)
      setActiveRegionTemplateId(activeTemplateId)
      setRegionEnabled(templates.length > 0)
      const optionData = regionOptionRes?.data || {}
      setRegionOptions({
        countries: Array.isArray(optionData.countries) ? optionData.countries : [],
        provinces: Array.isArray(optionData.provinces) ? optionData.provinces : [],
        cities: Array.isArray(optionData.cities) ? optionData.cities : []
      })
    } catch (err: any) {
      message.error(err?.response?.data?.error || '加载失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (appId) load()
  }, [appId])

  return {
    loading,
    app,
    channels,
    releases,
    appSecrets,
    releaseChannels,
    appMembers,
    releaseTemplates,
    regionTemplates,
    activeRegionTemplateId,
    regionEnabled,
    regionOptions,
    setReleases,
    setRegionTemplates,
    setActiveRegionTemplateId,
    setRegionEnabled,
    reload: load
  }
}

export { useAppDetailData }
export default useAppDetailData
