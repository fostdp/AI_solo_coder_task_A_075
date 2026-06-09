const API_BASE = '/api'

async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${url}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  })
  if (!res.ok) throw new Error(`API Error: ${res.status} ${res.statusText}`)
  return res.json()
}

export const api = {
  getDetectors: () => request<import('@/types').DetectorInfo[]>('/detectors'),
  getDetectorHistory: (id: string, start: string, end: string, interval = '1m') =>
    request<import('@/types').DetectorHistory>(`/detectors/${id}/history?start=${start}&end=${end}&interval=${interval}`),
  getDetectorHealth: (id: string) =>
    request<import('@/types').DetectorHealth>(`/detectors/${id}/health`),
  getSensors: () => request<import('@/types').SensorInfo[]>('/sensors'),
  getAlarms: (params?: { level?: number; status?: string; page?: number }) => {
    const qs = params ? '?' + new URLSearchParams(params as Record<string, string>).toString() : ''
    return request<{ total: number; items: import('@/types').Alarm[] }>(`/alarms${qs}`)
  },
  acknowledgeAlarm: (id: string) =>
    request<import('@/types').Alarm>(`/alarms/${id}/acknowledge`, { method: 'PUT' }),
  resolveAlarm: (id: string) =>
    request<import('@/types').Alarm>(`/alarms/${id}/resolve`, { method: 'PUT' }),
  locateLeak: (data: { detector_ids: string[]; concentrations: number[]; wind_speed: number; wind_direction: number }) =>
    request<import('@/types').LeakSourceResult>('/leak/locate', { method: 'POST', body: JSON.stringify(data) }),
  getLatestLeak: () => request<import('@/types').LeakSourceResult | null>('/leak/latest'),
  controlValve: (data: { device_id: string; action: 'open' | 'closed' }) =>
    request<import('@/types').ControlResult>('/control/valve', { method: 'POST', body: JSON.stringify(data) }),
  controlFan: (data: { device_id: string; action: 'start' | 'stop' }) =>
    request<import('@/types').ControlResult>('/control/fan', { method: 'POST', body: JSON.stringify(data) }),
  sendNotification: (data: { partition_id: string; message: string }) =>
    request<import('@/types').ControlResult>('/control/notify', { method: 'POST', body: JSON.stringify(data) }),
  getPartitions: () => request<import('@/types').Partition[]>('/partitions'),
}
