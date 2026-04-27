import { Navigate, Route, Routes } from 'react-router-dom'
import SystemDashboard from '../pages/SystemDashboard'
import SystemOrgs from '../pages/SystemOrgs'
import SystemApprovals from '../pages/SystemApprovals'
import SystemCreateEnterprise from '../pages/SystemCreateEnterprise'
import SystemApps from '../pages/SystemApps'
import SystemAuditLogs from '../pages/SystemAuditLogs'
import SystemTickets from '../pages/SystemTickets'
import SystemTicketSubmit from '../pages/SystemTicketSubmit'
import SystemTicketDetail from '../pages/SystemTicketDetail'
import SystemProfile from '../pages/SystemProfile'
import SystemUsers from '../pages/SystemUsers'
import SystemSettings from '../pages/SystemSettings'

export default function SystemAdminRoutes() {
  return (
    <Routes>
      <Route path="/system/dashboard" element={<SystemDashboard />} />
      <Route path="/system/orgs" element={<SystemOrgs />} />
      <Route path="/system/approvals/orgs" element={<SystemApprovals view="orgs" />} />
      <Route path="/system/approvals/apps" element={<SystemApprovals view="apps" />} />
      <Route path="/system/approvals" element={<Navigate to="/system/approvals/orgs" replace />} />
      <Route path="/system/create" element={<SystemCreateEnterprise />} />
      <Route path="/system/apps" element={<SystemApps />} />
      <Route path="/system/audit-logs" element={<SystemAuditLogs />} />
      <Route path="/system/tickets" element={<SystemTickets />} />
      <Route path="/system/tickets/new" element={<SystemTicketSubmit />} />
      <Route path="/system/tickets/:id" element={<SystemTicketDetail />} />
      <Route path="/system/profile" element={<SystemProfile />} />
      <Route path="/system/users" element={<SystemUsers />} />
      <Route path="/system/settings" element={<Navigate to="/system/settings/base" replace />} />
      <Route path="/system/settings/:tab" element={<SystemSettings />} />
      <Route path="/dashboard" element={<Navigate to="/system/dashboard" replace />} />
      <Route path="/" element={<Navigate to="/system/dashboard" replace />} />
      <Route path="*" element={<Navigate to="/system/dashboard" replace />} />
    </Routes>
  )
}
