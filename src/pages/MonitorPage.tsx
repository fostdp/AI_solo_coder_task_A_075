import { useEffect, useCallback } from 'react'
import { useStore } from '@/store'
import { api } from '@/utils/api'
import { useWebSocket } from '@/hooks/useWebSocket'
import SensorList from '@/components/SensorList'
import MapView from '@/components/MapView'
import AlarmPanel from '@/components/AlarmPanel'
import StatusBar from '@/components/StatusBar'

export default function MonitorPage() {
  const setDetectors = useStore((s) => s.setDetectors)
  const setSensors = useStore((s) => s.setSensors)
  const setAlarms = useStore((s) => s.setAlarms)
  const setPartitions = useStore((s) => s.setPartitions)
  const setWsConnected = useStore((s) => s.setWsConnected)
  const handleWSMessage = useStore((s) => s.handleWSMessage)

  useEffect(() => {
    api.getDetectors().then(setDetectors).catch(() => {})
    api.getSensors().then(setSensors).catch(() => {})
    api.getAlarms().then((res) => setAlarms(res.items)).catch(() => {})
    api.getPartitions().then(setPartitions).catch(() => {})
  }, [setDetectors, setSensors, setAlarms, setPartitions])

  const onWSMessage = useCallback(
    (msg: Parameters<typeof handleWSMessage>[0]) => {
      setWsConnected(true)
      handleWSMessage(msg)
    },
    [handleWSMessage, setWsConnected],
  )

  useWebSocket(onWSMessage)

  return (
    <div className="h-full flex flex-col">
      <div className="flex-1 flex overflow-hidden">
        <SensorList />
        <div className="flex-1 relative">
          <MapView />
        </div>
        <AlarmPanel />
      </div>
      <StatusBar />
    </div>
  )
}
