-- 智慧城市地下综合管廊燃气泄漏激光监测系统 - InfluxDB 初始化脚本
-- 使用 InfluxDB 2.x Flux 语法

-- 创建组织
CREATE ORG gas-monitor

-- 创建Bucket (保留30天数据)
CREATE BUCKET sensor_data WITH RETENTION 30d IN gas-monitor

-- 以下是 InfluxDB 2.x CLI 命令，用于初始化
-- influx bucket create -n sensor_data -o gas-monitor -r 30d

-- Measurement 定义说明:
-- methane_concentration: 甲烷浓度时序数据
--   tags: detector_id, partition_id
--   fields: concentration (float), status (string)
--
-- sensor_o2: 氧气浓度时序数据
--   tags: sensor_id, partition_id
--   fields: value (float), status (string)
--
-- sensor_temperature: 温度时序数据
--   tags: sensor_id, partition_id
--   fields: value (float), status (string)
--
-- sensor_humidity: 湿度时序数据
--   tags: sensor_id, partition_id
--   fields: value (float), status (string)
--
-- wind: 风速风向时序数据
--   tags: partition_id
--   fields: speed (float), direction (float)

-- 示例数据写入 (使用 Line Protocol):
-- methane_concentration,detector_id=D-001,partition_id=P-001 concentration=2.5,status="online" 1717891200000000000
-- sensor_o2,sensor_id=S-O2-001,partition_id=P-001 value=20.9,status="online" 1717891200000000000
-- sensor_temperature,sensor_id=S-T-001,partition_id=P-001 value=22.5,status="online" 1717891200000000000
-- sensor_humidity,sensor_id=S-H-001,partition_id=P-001 value=65.0,status="online" 1717891200000000000
-- wind,partition_id=P-001 speed=1.5,direction=45.0 1717891200000000000

-- 查询近1小时甲烷浓度趋势 (Flux):
-- from(bucket: "sensor_data")
--   |> range(start: -1h)
--   |> filter(fn: (r) => r._measurement == "methane_concentration")
--   |> filter(fn: (r) => r.detector_id == "D-001")
--   |> aggregateWindow(every: 1m, fn: mean, createEmpty: false)
--   |> yield(name: "mean")

-- 查询所有检测器最新浓度:
-- from(bucket: "sensor_data")
--   |> range(start: -5m)
--   |> filter(fn: (r) => r._measurement == "methane_concentration")
--   |> filter(fn: (r) => r._field == "concentration")
--   |> last()
--   |> yield(name: "latest")
