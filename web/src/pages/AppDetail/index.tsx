import { Alert, Breadcrumb, Card, Grid, Tabs } from 'antd'
import { Link, useNavigate, useParams } from 'react-router-dom'
import AppHeader from './components/AppHeader'
import AppStats from './components/AppStats'
import useAppDetailData from './hooks/useAppDetailData'
import { AdvancedTab } from './tabs/AdvancedTab'
import AppSecretsTab from './tabs/AppSecretsTab'
import AttributesTab from './tabs/AttributesTab'
import ChannelsTab from './tabs/ChannelsTab'
import GrayControlTab from './tabs/GrayControlTab'
import MaintenanceTab from './tabs/MaintenanceTab'
import MembersTab from './tabs/MembersTab'
import RegionRulesTab from './tabs/RegionRulesTab'
import ReleasePlanTab from './tabs/ReleasePlanTab'
import ReleasesTab from './tabs/ReleasesTab'

export default function AppDetail() {
  const { id, tab } = useParams()
  const navigate = useNavigate()
  const screens = Grid.useBreakpoint()
  const isMobile = !screens.lg
  const {
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
    reload
  } = useAppDetailData(id)

  const orgType = (sessionStorage.getItem('org_type') || '').toLowerCase()
  const systemRole = (sessionStorage.getItem('system_role') || '').toLowerCase()
  const isPersonal = orgType === 'personal'
  const canSystemReview = systemRole === 'system_admin'
  const canReviewRelease = !isPersonal || canSystemReview
  const appStatus = (app?.status || '').toLowerCase()
  const isPending = isPersonal && appStatus === 'pending'
  const isRejected = isPersonal && appStatus === 'rejected'
  const isLocked = isPending || isRejected
  const lockMessage = isPending
    ? '该应用正在审核中，审核通过后才能进行操作。'
    : ''

  if (!app) return null

  const hasPendingReview = releases.some((r) => (r.status || '').toLowerCase() === 'in_review')
  const showReviewHint = !canReviewRelease && hasPendingReview

  const items: any[] = [
    {
      key: 'releases',
      label: '版本管理',
      children: (
        <ReleasesTab
          appId={app.id}
          channels={channels}
          releases={releases}
          releaseChannels={releaseChannels}
          regionTemplates={regionTemplates}
          activeRegionTemplateId={activeRegionTemplateId}
          regionEnabled={regionEnabled}
          isLocked={isLocked}
          isPersonal={isPersonal}
          canReviewRelease={canReviewRelease}
          reload={reload}
          setReleases={setReleases}
          loading={loading}
        />
      )
    },
    {
      key: 'channels',
      label: `渠道 (${channels.length})`,
      children: (
        <ChannelsTab
          appId={app.id}
          channels={channels}
          isLocked={isLocked}
          reload={reload}
        />
      )
    },
    {
      key: 'release-channels',
      label: '灰度策略',
      children: (
        <GrayControlTab
          appId={app.id}
          releaseChannels={releaseChannels}
          releases={releases}
          channels={channels}
          isLocked={isLocked}
          reload={reload}
        />
      )
    },
    {
      key: 'region-rules',
      label: '地区策略',
      forceRender: true,
      children: (
        <RegionRulesTab
          appId={app.id}
          releaseChannels={releaseChannels}
          regionTemplates={regionTemplates}
          activeRegionTemplateId={activeRegionTemplateId}
          regionEnabled={regionEnabled}
          regionOptions={regionOptions}
          isLocked={isLocked}
          reload={reload}
          setRegionTemplates={setRegionTemplates}
          setActiveRegionTemplateId={setActiveRegionTemplateId}
          setRegionEnabled={setRegionEnabled}
        />
      )
    },
    {
      key: 'release-plan',
      label: '发布模板',
      children: (
        <ReleasePlanTab
          releases={releases}
          releaseTemplates={releaseTemplates}
          releaseChannels={releaseChannels}
          isLocked={isLocked}
          reload={reload}
        />
      )
    },
    {
      key: 'app-secrets',
      label: `应用密钥 (${appSecrets.length})`,
      children: (
        <AppSecretsTab
          appId={app.id}
          appSecrets={appSecrets}
          isLocked={isLocked}
          reload={reload}
        />
      )
    },
    {
      key: 'attributes',
      label: '应用属性',
      children: <AttributesTab app={app} />
    },
    {
      key: 'advanced',
      label: '高级选项',
      children: <AdvancedTab appId={app.id} app={app} isLocked={isLocked} onReload={reload} />
    },
    {
      key: 'maintenance',
      label: '维护模式',
      children: <MaintenanceTab appId={app.id} app={app} isLocked={isLocked} />
    }
  ]

  if (!isPersonal) {
    items.push({
      key: 'members',
      label: `成员 (${appMembers.length})`,
      children: (
        <MembersTab
          appId={app.id}
          appMembers={appMembers}
          isLocked={isLocked}
          reload={reload}
        />
      )
    })
  }

  const normalizedTab = (tab || '').toLowerCase()
  const availableKeys = new Set(items.map((item) => item.key))
  const activeKey = availableKeys.has(normalizedTab) ? normalizedTab : 'releases'

  return (
    <div>
      <Breadcrumb
        style={{ marginBottom: isMobile ? 12 : 16 }}
        items={[
          { title: <Link to="/apps">应用管理</Link> },
          { title: app.name }
        ]}
      />

      {isPending && (
        <Alert
          style={{ marginBottom: 16 }}
          type="warning"
          showIcon
          message={lockMessage}
        />
      )}
      {isRejected && (
        <Alert
          style={{ marginBottom: 16 }}
          type="error"
          showIcon
          message="该应用已被驳回，请修改后重新提交审核。"
          description={app.rejection_reason || '未提供驳回理由'}
        />
      )}
      {!isLocked && showReviewHint && (
        <Alert
          style={{ marginBottom: 16 }}
          type="warning"
          showIcon
          message="该应用正在审核中，审核通过后才能进行操作。"
        />
      )}

      <AppHeader app={app} isLocked={isLocked} canEdit={!isPending} lockReason={isPending ? 'pending' : isRejected ? 'rejected' : ''} onReload={reload} />

      <AppStats
        releasesCount={releases.length}
        channelsCount={channels.length}
        appSecretsCount={appSecrets.length}
        activeReleaseChannelsCount={releaseChannels.filter((rc) => rc.status === 'active').length}
      />

      <Card
        style={{ borderRadius: isMobile ? 10 : 12, border: 'none', boxShadow: '0 2px 8px rgba(0,0,0,0.04)' }}
        styles={{ body: { padding: isMobile ? '12px 0' : '24px 0' } }}
      >
        <Tabs
          activeKey={activeKey}
          onChange={(key) => {
            if (!id) return
            navigate(`/apps/${id}/${key}`)
          }}
          size={isMobile ? 'small' : 'middle'}
          tabBarGutter={isMobile ? 12 : 24}
          style={{
            padding: isMobile ? '0 12px' : '0 24px'
          }}
          tabBarStyle={{
            marginBottom: isMobile ? 12 : 16,
            overflowX: 'auto',
            overflowY: 'hidden'
          }}
          items={items}
        />
      </Card>
    </div>
  )
}
