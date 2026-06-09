import { useState, useCallback } from 'react'
import { useStore } from '@/store'
import { api } from '@/utils/api'
import type { LeakSourceResult } from '@/types'
import { Shield, Wind, Bell, Crosshair, CheckCircle, XCircle, Loader2 } from 'lucide-react'

export default function ControlPage() {
  const partitions = useStore((s) => s.partitions)
  const updatePartitionState = useStore((s) => s.updatePartitionState)
  const leakSource = useStore((s) => s.leakSource)
  const detectors = useStore((s) => s.detectors)
  const [feedback, setFeedback] = useState<Record<string, { success: boolean; msg: string }>>({})
  const [locating, setLocating] = useState(false)

  const setFeedbackFor = useCallback((key: string, success: boolean, msg: string) => {
    setFeedback((prev) => ({ ...prev, [key]: { success, msg } }))
    setTimeout(() => {
      setFeedback((prev) => {
        const next = { ...prev }
        delete next[key]
        return next
      })
    }, 3000)
  }, [])

  const handleValveToggle = useCallback(
    async (partitionId: string, _valveId: string, currentState: string) => {
      const key = `valve-${partitionId}`
      const action = currentState === 'open' ? 'close' : 'open'
      try {
        const result = await api.controlValve({
          partition_id: partitionId,
          action: action as 'open' | 'close',
        })
        if (result.status === 'sent') {
          setFeedbackFor(key, true, `指令已发送(等待确认)`)
        } else {
          setFeedbackFor(key, false, '操作失败')
        }
      } catch {
        setFeedbackFor(key, false, '请求错误')
      }
    },
    [updatePartitionState, setFeedbackFor],
  )

  const handleFanToggle = useCallback(
    async (partitionId: string, _fanId: string, currentState: string) => {
      const key = `fan-${partitionId}`
      const action = currentState === 'running' ? 'stop' : 'start'
      try {
        const result = await api.controlFan({
          partition_id: partitionId,
          action: action as 'start' | 'stop',
        })
        if (result.status === 'sent') {
          setFeedbackFor(key, true, `指令已发送(等待确认)`)
        } else {
          setFeedbackFor(key, false, '操作失败')
        }
      } catch {
        setFeedbackFor(key, false, '请求错误')
      }
    },
    [updatePartitionState, setFeedbackFor],
  )

  const handleNotify = useCallback(
    async (partitionId: string) => {
      const key = `notify-${partitionId}`
      try {
        await api.sendNotification({
          partition_id: partitionId,
          message: '紧急疏散通知：检测到燃气泄漏，请立即撤离！',
        })
        setFeedbackFor(key, true, '通知已发送')
      } catch {
        setFeedbackFor(key, false, '请求错误')
      }
    },
    [setFeedbackFor],
  )

  const handleLocate = useCallback(async () => {
    setLocating(true)
    try {
      const activeDetectors = detectors.filter((d) => d.latest_concentration > 5)
      if (activeDetectors.length === 0) {
        setFeedbackFor('locate', false, '没有浓度超标的检测器')
        return
      }
      const result: LeakSourceResult = await api.locateLeak({
        detector_ids: activeDetectors.map((d) => d.id),
        concentrations: activeDetectors.map((d) => d.latest_concentration),
        wind_speed: 0,
        wind_direction: 0,
      })
      useStore.getState().setLeakSource(result)
      setFeedbackFor('locate', true, '定位成功')
    } catch {
      setFeedbackFor('locate', false, '定位失败')
    } finally {
      setLocating(false)
    }
  }, [detectors, setFeedbackFor])

  return (
    <div className="h-full flex flex-col bg-panel overflow-auto">
      <div className="p-4">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-base font-bold text-tech-blue">分区控制面板</h2>
          <button
            onClick={handleLocate}
            disabled={locating}
            className="flex items-center gap-2 px-4 py-2 bg-tech-red/15 text-tech-red text-sm rounded hover:bg-tech-red/25 transition-colors disabled:opacity-50"
          >
            {locating ? <Loader2 size={14} className="animate-spin" /> : <Crosshair size={14} />}
            泄漏源定位
          </button>
        </div>
        {feedback['locate'] && (
          <div className={`mb-3 flex items-center gap-2 text-xs px-3 py-2 rounded ${
            feedback['locate'].success ? 'bg-tech-green/15 text-tech-green' : 'bg-tech-red/15 text-tech-red'
          }`}>
            {feedback['locate'].success ? <CheckCircle size={12} /> : <XCircle size={12} />}
            {feedback['locate'].msg}
          </div>
        )}

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
          {partitions.map((p) => {
            const valveKey = `valve-${p.id}`
            const fanKey = `fan-${p.id}`
            const notifyKey = `notify-${p.id}`

            return (
              <div
                key={p.id}
                className="bg-panel-light border border-panel-border rounded-lg p-3 space-y-3"
              >
                <div className="flex items-center justify-between">
                  <span className="text-sm font-bold text-tech-blue">{p.name}</span>
                  <span className="text-[10px] text-gray-500 font-mono">
                    {p.start_distance.toFixed(0)}m - {p.end_distance.toFixed(0)}m
                  </span>
                </div>

                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <Shield size={14} className={p.valve_state === 'open' ? 'text-tech-green' : 'text-tech-red'} />
                    <span className="text-xs text-gray-400">阀门:</span>
                    <span className={`text-xs ${p.valve_state === 'open' ? 'text-tech-green' : 'text-tech-red'}`}>
                      {p.valve_state === 'open' ? '开启' : '关闭'}
                    </span>
                  </div>
                  <button
                    onClick={() => handleValveToggle(p.id, p.valve_id, p.valve_state)}
                    className={`text-[10px] px-2 py-0.5 rounded transition-colors ${
                      p.valve_state === 'open'
                        ? 'bg-tech-red/15 text-tech-red hover:bg-tech-red/25'
                        : 'bg-tech-green/15 text-tech-green hover:bg-tech-green/25'
                    }`}
                  >
                    {p.valve_state === 'open' ? '关闭' : '开启'}
                  </button>
                </div>
                {feedback[valveKey] && (
                  <div className={`text-[10px] ${feedback[valveKey].success ? 'text-tech-green' : 'text-tech-red'}`}>
                    {feedback[valveKey].msg}
                  </div>
                )}

                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <Wind size={14} className={p.fan_state === 'running' ? 'text-tech-green' : 'text-gray-500'} />
                    <span className="text-xs text-gray-400">风机:</span>
                    <span className={`text-xs ${p.fan_state === 'running' ? 'text-tech-green' : 'text-gray-500'}`}>
                      {p.fan_state === 'running' ? '运行' : '停止'}
                    </span>
                  </div>
                  <button
                    onClick={() => handleFanToggle(p.id, p.fan_id, p.fan_state)}
                    className={`text-[10px] px-2 py-0.5 rounded transition-colors ${
                      p.fan_state === 'running'
                        ? 'bg-tech-red/15 text-tech-red hover:bg-tech-red/25'
                        : 'bg-tech-green/15 text-tech-green hover:bg-tech-green/25'
                    }`}
                  >
                    {p.fan_state === 'running' ? '停止' : '启动'}
                  </button>
                </div>
                {feedback[fanKey] && (
                  <div className={`text-[10px] ${feedback[fanKey].success ? 'text-tech-green' : 'text-tech-red'}`}>
                    {feedback[fanKey].msg}
                  </div>
                )}

                <button
                  onClick={() => handleNotify(p.id)}
                  className="w-full flex items-center justify-center gap-2 px-3 py-1.5 bg-tech-orange/15 text-tech-orange text-xs rounded hover:bg-tech-orange/25 transition-colors"
                >
                  <Bell size={12} />
                  发送疏散通知
                </button>
                {feedback[notifyKey] && (
                  <div className={`text-[10px] ${feedback[notifyKey].success ? 'text-tech-green' : 'text-tech-red'}`}>
                    {feedback[notifyKey].msg}
                  </div>
                )}
              </div>
            )
          })}
        </div>

        {leakSource && (
          <div className="mt-6 bg-panel-light border border-tech-red/30 rounded-lg p-4">
            <h3 className="text-sm font-bold text-tech-red mb-3 flex items-center gap-2">
              <Crosshair size={14} />
              泄漏源定位结果
            </h3>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-3 text-xs">
              <div>
                <span className="text-gray-400">位置:</span>
                <span className="ml-1 font-mono">
                  {leakSource.source_position.lat.toFixed(6)}, {leakSource.source_position.lng.toFixed(6)}
                </span>
              </div>
              <div>
                <span className="text-gray-400">隧道距离:</span>
                <span className="ml-1 font-mono">{leakSource.source_position.distance.toFixed(0)} m</span>
              </div>
              <div>
                <span className="text-gray-400">泄漏率:</span>
                <span className="ml-1 font-mono text-tech-red">{leakSource.leak_rate.toFixed(2)} L/min</span>
              </div>
              <div>
                <span className="text-gray-400">置信度:</span>
                <span className="ml-1 font-mono">{(leakSource.confidence * 100).toFixed(1)}%</span>
              </div>
              <div>
                <span className="text-gray-400">扩散半径:</span>
                <span className="ml-1 font-mono">{leakSource.diffusion_radius.toFixed(0)} m</span>
              </div>
              <div>
                <span className="text-gray-400">定位方法:</span>
                <span className="ml-1 font-mono uppercase">{leakSource.method}</span>
              </div>
              <div>
                <span className="text-gray-400">定位时间:</span>
                <span className="ml-1 font-mono">
                  {new Date(leakSource.timestamp).toLocaleString('zh-CN')}
                </span>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
