import { Navigate, Route, Routes } from 'react-router-dom'
import Dashboard from '../pages/Dashboard'
import Apps from '../pages/Apps'
import AppDetail from '../pages/AppDetail'
import ReleaseCreate from '../pages/ReleaseCreate'
import Analytics from '../pages/Analytics'
import AdvancedOptions from '../pages/AdvancedOptions'
import Feedbacks from '../pages/Feedbacks'
import OrgMembers from '../pages/OrgMembers'
import SubUsers from '../pages/SubUsers'
import MemberInvites from '../pages/MemberInvites'
import RoleManagement from '../pages/RoleManagement'
import OrgAttributes from '../pages/OrgAttributes'
import Devices from '../pages/Devices'
import AuditLogs from '../pages/AuditLogs'
import Tickets from '../pages/Tickets'
import TicketSubmit from '../pages/TicketSubmit'
import TicketDetail from '../pages/TicketDetail'
import Docs from '../pages/Docs'
import OrgProfile from '../pages/OrgProfile'

type AdminRoutesProps = {
  systemRole: string
}

export default function AdminRoutes({ systemRole }: AdminRoutesProps) {
  return (
    <Routes>
      <Route path="/dashboard" element={<Dashboard />} />
      <Route path="/apps" element={<Apps />} />
      <Route path="/apps/:id/:tab" element={<AppDetail />} />
      <Route path="/apps/:id" element={<AppDetail />} />
      <Route path="/apps/:id/releases/new" element={<ReleaseCreate />} />
      <Route path="/analytics" element={<Analytics />} />
      <Route path="/advanced" element={<AdvancedOptions />} />
      <Route path="/feedback" element={<Feedbacks />} />
      <Route
        path="/orgs"
        element={systemRole === 'org_admin' ? <OrgMembers /> : <Navigate to="/dashboard" replace />}
      />
      <Route
        path="/org-attributes"
        element={systemRole === 'org_admin' ? <OrgAttributes /> : <Navigate to="/dashboard" replace />}
      />
      <Route path="/sub-users" element={<SubUsers />} />
      <Route path="/role-management" element={<RoleManagement />} />
      <Route path="/member-invites" element={<MemberInvites />} />
      <Route path="/devices" element={<Devices />} />
      <Route path="/audit-logs" element={<AuditLogs />} />
      <Route path="/tickets" element={<Tickets />} />
      <Route path="/tickets/new" element={<TicketSubmit />} />
      <Route path="/tickets/:id" element={<TicketDetail />} />
      <Route path="/docs" element={<Docs />} />
      <Route path="/profile" element={<OrgProfile />} />
      <Route path="/" element={<Navigate to="/dashboard" replace />} />
      <Route path="*" element={<Navigate to="/dashboard" replace />} />
    </Routes>
  )
}
