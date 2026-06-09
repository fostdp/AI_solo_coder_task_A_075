import { NavLink } from 'react-router-dom'
import { useStore } from '@/store'
import { Wifi, WifiOff } from 'lucide-react'
import { useEffect, useState } from 'react'

export default function NavBar() {
  const wsConnected = useStore((s) => s.wsConnected)
  const [time, setTime] = useState(new Date())

  useEffect(() => {
    const timer = setInterval(() => setTime(new Date()), 1000)
    return () => clearInterval(timer)
  }, [])

  const links = [
    { to: '/', label: '实时监测' },
    { to: '/alarms', label: '告警管理' },
    { to: '/control', label: '应急联动' },
  ]

  return (
    <div className="h-12 bg-panel-light border-b border-panel-border flex items-center px-4 shrink-0">
      <div className="text-tech-blue font-bold text-base tracking-wider mr-8">
        管廊燃气泄漏激光监测系统
      </div>
      <nav className="flex gap-1">
        {links.map((link) => (
          <NavLink
            key={link.to}
            to={link.to}
            className={({ isActive }) =>
              `px-4 py-1.5 rounded text-sm transition-colors ${
                isActive
                  ? 'bg-tech-blue/15 text-tech-blue'
                  : 'text-gray-400 hover:text-gray-200 hover:bg-panel-border/50'
              }`
            }
          >
            {link.label}
          </NavLink>
        ))}
      </nav>
      <div className="ml-auto flex items-center gap-4">
        <div className="flex items-center gap-1.5">
          {wsConnected ? (
            <>
              <Wifi size={14} className="text-tech-green" />
              <span className="w-2 h-2 rounded-full bg-tech-green" />
            </>
          ) : (
            <>
              <WifiOff size={14} className="text-tech-red" />
              <span className="w-2 h-2 rounded-full bg-tech-red" />
            </>
          )}
          <span className="text-xs text-gray-400 ml-1">
            {wsConnected ? '已连接' : '未连接'}
          </span>
        </div>
        <div className="text-xs text-gray-400 font-mono">
          {time.toLocaleString('zh-CN', {
            year: 'numeric',
            month: '2-digit',
            day: '2-digit',
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit',
            hour12: false,
          })}
        </div>
      </div>
    </div>
  )
}
