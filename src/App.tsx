import { Routes, Route } from 'react-router-dom'
import MonitorPage from '@/pages/MonitorPage'
import AlarmPage from '@/pages/AlarmPage'
import ControlPage from '@/pages/ControlPage'
import NavBar from '@/components/NavBar'

export default function App() {
  return (
    <div className="h-screen w-screen flex flex-col bg-panel">
      <NavBar />
      <div className="flex-1 overflow-hidden">
        <Routes>
          <Route path="/" element={<MonitorPage />} />
          <Route path="/alarms" element={<AlarmPage />} />
          <Route path="/control" element={<ControlPage />} />
        </Routes>
      </div>
    </div>
  )
}
