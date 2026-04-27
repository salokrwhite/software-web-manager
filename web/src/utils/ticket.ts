export const TICKET_STATUS_LABELS: Record<string, string> = {
  submitted: '已提交',
  in_progress: '处理中',
  resolved: '已完成'
}

export const TICKET_STEPS = [
  { key: 'submitted', title: '已提交' },
  { key: 'in_progress', title: '处理中' },
  { key: 'resolved', title: '已完成' }
]

export const formatTicketStatus = (status?: string): string => {
  const key = (status || '').trim().toLowerCase()
  if (!key) return '未知'
  return TICKET_STATUS_LABELS[key] || key
}

export const getTicketStepIndex = (status?: string): number => {
  const key = (status || '').trim().toLowerCase()
  if (key === 'in_progress') return 1
  if (key === 'resolved') return 2
  return 0
}

export const getTicketStepsStatus = (status?: string): 'wait' | 'process' | 'finish' => {
  const key = (status || '').trim().toLowerCase()
  if (key === 'resolved') return 'finish'
  if (key === 'submitted' || key === 'in_progress') return 'process'
  return 'wait'
}
