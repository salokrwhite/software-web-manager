export const ACTION_LABELS: Record<string, string> = {
  'system.org.create': '系统创建企业',
  'system.org.approve': '系统审核通过企业',
  'system.org.disable': '系统禁用企业',
  'system.app.approve': '系统审核通过应用',
  'system.app.reject': '系统驳回应用',
  'org.create': '创建企业',
  'org.update': '更新企业信息',
  'org.delete': '删除企业',
  'org.owner_transfer': '转移企业所有者',
  'org_member.add': '添加企业成员',
  'org_member.update': '更新企业成员',
  'org_member.remove': '移除企业成员',
  'org_invite.create': '发送企业邀请',
  'app.create': '创建应用',
  'app.update': '更新应用信息',
  'app.delete': '删除应用',
  'app_member.add': '添加应用成员',
  'channel.create': '创建渠道',
  'release.create': '创建版本',
  'release.publish': '发布版本到渠道',
  'release.revoke': '撤销发布',
  'release.submit': '提交版本审核',
  'release.approve': '审核通过版本',
  'release.reject': '审核拒绝版本',
  'release.rollback': '回滚渠道版本',
  'release_channel.update': '更新发布渠道',
  'artifact.upload': '上传安装包',
  'app_secret.create': '创建应用密钥',
  'app_secret.revoke': '撤销应用密钥',
  'app_secret.policy_update': '更新应用密钥策略',
  'ticket.create': '创建工单',
  'ticket.status.update': '更新工单状态',
  'ticket.message.create': '新增工单对话'
}

export const TARGET_TYPE_LABELS: Record<string, string> = {
  org: '企业',
  org_member: '企业成员',
  org_invite: '企业邀请',
  app: '应用',
  app_member: '应用成员',
  channel: '渠道',
  release: '版本',
  release_channel: '发布渠道',
  app_secret: '应用密钥',
  artifact: '安装包',
  ticket: '工单',
  ticket_message: '工单对话'
}

const findCodeByLabel = (input: string, labels: Record<string, string>): string => {
  const normalized = input.trim().toLowerCase()
  if (!normalized) {
    return ''
  }
  for (const [code, label] of Object.entries(labels)) {
    if (label.trim().toLowerCase() === normalized) {
      return code
    }
  }
  return input.trim()
}

export const formatAction = (action?: string): string => {
  const key = (action || '').trim()
  if (!key) return '其他操作'
  return ACTION_LABELS[key] || '其他操作'
}

export const formatTargetType = (targetType?: string): string => {
  const key = (targetType || '').trim()
  if (!key) return '其他'
  return TARGET_TYPE_LABELS[key] || '其他'
}

export const formatTargetId = (id?: string | number | null): string => {
  if (id === null || id === undefined || id === '') return '-'
  const raw = String(id)
  if (raw.length <= 16) return raw
  return `${raw.slice(0, 8)}...${raw.slice(-8)}`
}

export const normalizeAuditActionQuery = (action?: string): string | undefined => {
  const value = (action || '').trim()
  if (!value) {
    return undefined
  }
  return findCodeByLabel(value, ACTION_LABELS)
}

export const normalizeAuditTargetTypeQuery = (targetType?: string): string | undefined => {
  const value = (targetType || '').trim()
  if (!value) {
    return undefined
  }
  return findCodeByLabel(value, TARGET_TYPE_LABELS)
}
