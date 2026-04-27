import { Navigate, Route, Routes } from 'react-router-dom'
import UserDashboard from '../pages/UserDashboard'
import JoinOrg from '../pages/JoinOrg'
import EnterpriseUpgrade from '../pages/EnterpriseUpgrade'

export default function OrglessRoutes() {
  return (
    <Routes>
      <Route path="/dashboard" element={<UserDashboard />} />
      <Route path="/join-org" element={<JoinOrg />} />
      <Route path="/enterprise-upgrade" element={<EnterpriseUpgrade />} />
      <Route path="/" element={<Navigate to="/dashboard" replace />} />
      <Route path="*" element={<Navigate to="/dashboard" replace />} />
    </Routes>
  )
}
