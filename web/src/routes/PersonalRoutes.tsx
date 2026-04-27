import { Navigate, Route, Routes } from 'react-router-dom'
import Dashboard from '../pages/Dashboard'
import Apps from '../pages/Apps'
import AppDetail from '../pages/AppDetail'
import ReleaseCreate from '../pages/ReleaseCreate'
import Devices from '../pages/Devices'
import Analytics from '../pages/Analytics'
import AdvancedOptions from '../pages/AdvancedOptions'
import Feedbacks from '../pages/Feedbacks'
import AuditLogs from '../pages/AuditLogs'
import JoinOrg from '../pages/JoinOrg'
import EnterpriseUpgrade from '../pages/EnterpriseUpgrade'
import Tickets from '../pages/Tickets'
import TicketSubmit from '../pages/TicketSubmit'
import TicketDetail from '../pages/TicketDetail'
import OrgProfile from '../pages/OrgProfile'
import Docs from '../pages/Docs'

export default function PersonalRoutes() {
  return (
    <Routes>
      <Route path="/dashboard" element={<Dashboard />} />
      <Route path="/apps" element={<Apps />} />
      <Route path="/apps/:id/:tab" element={<AppDetail />} />
      <Route path="/apps/:id" element={<AppDetail />} />
      <Route path="/apps/:id/releases/new" element={<ReleaseCreate />} />
      <Route path="/devices" element={<Devices />} />
      <Route path="/analytics" element={<Analytics />} />
      <Route path="/advanced" element={<AdvancedOptions />} />
      <Route path="/feedback" element={<Feedbacks />} />
      <Route path="/audit-logs" element={<AuditLogs />} />
      <Route path="/join-org" element={<JoinOrg />} />
      <Route path="/enterprise-upgrade" element={<EnterpriseUpgrade />} />
      <Route path="/tickets" element={<Tickets />} />
      <Route path="/tickets/new" element={<TicketSubmit />} />
      <Route path="/tickets/:id" element={<TicketDetail />} />
      <Route path="/profile" element={<OrgProfile />} />
      <Route path="/docs" element={<Docs />} />
      <Route path="/" element={<Navigate to="/dashboard" replace />} />
      <Route path="*" element={<Navigate to="/dashboard" replace />} />
    </Routes>
  )
}
