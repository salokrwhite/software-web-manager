import { Navigate, useLocation } from 'react-router-dom'
import { buildAbsoluteRedirect } from '../utils/redirect'

export function RequireAuth({ children }: { children: JSX.Element }) {
  const token = sessionStorage.getItem('access_token')
  const location = useLocation()
  if (!token) {
    const redirect = buildAbsoluteRedirect(location.pathname, location.search || '', location.hash || '')
    return <Navigate to={`/login?redirect=${encodeURIComponent(redirect)}`} replace />
  }
  return children
}
