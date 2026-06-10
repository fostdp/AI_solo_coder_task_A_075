# 智慧城市地下综合管廊燃气泄漏激光监测与联动处置系统

## 系统架构

```
┌─────────────────────────────────────────────────────────────┐
│                    Docker Compose 编排                       │
│                                                             │
│  ┌──────────┐   ┌──────────┐   ┌──────────┐               │
│  │ InfluxDB │   │PostgreSQL│   │ Mosquitto│               │
│  │  :8086   │   │  :5432   │   │  :1883   │               │
│  └────┬─────┘   └────┬─────┘   └────┬─────┘               │
│       │              │              │                       │
│  ┌────┴──────────────┴──────────────┴─────┐               │
│  │          Go Backend  (:8080)           │               │
│  │  ┌──────────────┐  ┌───────────────┐  │               │
│  │  │laser_receiver│→│ alarm_router   │  │               │
│  │  │(数据采集+校验)│  │(分级告警+推送) │  │               │
│  │  └──────┬───────┘  └───────┬───────┘  │               │
│  │         │                  │           │               │
│  │  ┌──────┴───────┐  ┌──────┴───────┐  │               │
│  │  │ leak_locator │  │emergency_ctrl │  │               │
│  │  │(泄漏定位+扩散)│  │(阀门+风机联动)│  │               │
│  │  └──────────────┘  └──────────────┘  │               │
│  │  pprof: :6060  metrics: /metrics      │               │
│  └───────────────────────────────────────┘               │
│       ↑                ↑                                   │
│  ┌────┴────┐     ┌─────┴────┐                             │
│  │Simulator│     │ Frontend │                             │
│  │(Python) │     │ :3000    │                             │
│  └─────────┘     │ Nginx+Gz │                             │
│                  └──────────┘                              │
└─────────────────────────────────────────────────────────────┘
```

### 模块通信架构

四个Go模块通过 `MessageBus` channel 通信：

| Channel | 生产者 | 消费者 | 说明 |
|---------|--------|--------|------|
| `ValidatedDetectorData` | laser_receiver | alarm_router | 校验后的检测器数据 |
| `ValidatedSensorData` | laser_receiver | (预留) | 校验后的传感器数据 |
| `ControlCommand` | alarm_router | emergency_controller | 自动/手动控制指令 |
| `LocateRequest` | HTTP handler | leak_locator | 泄漏定位请求 |
| `WSBroadcast` | 全模块 | WSForwarder→Hub | 前端实时推送 |

### 技术栈

- **后端**: Go 1.21 + Gin + Gorilla WebSocket + Eclipse Paho MQTT
- **前端**: React 18 + TypeScript + Vite + TailwindCSS + Leaflet + Canvas
- **时序数据库**: InfluxDB 2.7 (异步批量写入 + 三级降采样)
- **关系数据库**: PostgreSQL 15 (设备元数据、告警、控制日志)
- **消息队列**: Mosquitto 2 (QoS 2 + 持久会话)
- **监控**: Prometheus + pprof
- **部署**: Docker Compose

## 快速部署

### 前置条件

- Docker 20.10+
- Docker Compose v2+

### 一键启动

```bash
# 克隆项目
cd gas-leak-monitor

# 启动所有服务
docker compose up -d

# 查看服务状态
docker compose ps

# 查看日志
docker compose logs -f backend
```

### 服务端口

| 服务 | 端口 | 说明 |
|------|------|------|
| Frontend (Nginx) | 3000 | 前端Web界面 |
| Backend API | 8080 | Go HTTP/WS服务 |
| Backend Pprof | 6060 | 性能分析 |
| InfluxDB | 8086 | 时序数据库UI |
| PostgreSQL | 5432 | 关系数据库 |
| Mosquitto MQTT | 1883 | MQTT Broker |
| Mosquitto WS | 9001 | MQTT WebSocket |

### 环境变量覆盖

后端支持以下环境变量覆盖配置文件：

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `CONFIG_PATH` | 配置文件路径 | `/app/config.json` |
| `INFLUXDB_URL` | InfluxDB地址 | `http://influxdb:8086` |
| `POSTGRES_URL` | PostgreSQL连接串 | `postgres://postgres:postgres@postgres:5432/gas_monitor?sslmode=disable` |
| `MQTT_BROKER` | MQTT Broker地址 | `tcp://mosquitto:1883` |

### 停止服务

```bash
docker compose down
# 保留数据卷
docker compose down -v  # 删除数据卷
```

## 模拟器用法

### Docker模式（推荐）

模拟器随 `docker compose up` 自动启动，默认配置：
- 30公里管廊，300台检测器，1秒上报间隔
- 自动随机触发泄漏（每秒0.2%概率，最多3个同时）

### 自定义模拟器参数

编辑 `docker-compose.yml` 中 simulator 服务的 environment：

```yaml
simulator:
  environment:
    - API_HOST=backend
    - API_PORT=8080
    - TUNNEL_LENGTH=30000
    - NUM_DETECTORS=300
    - REPORT_INTERVAL=1.0
```

### 本地运行模拟器

```bash
cd scripts
python simulator.py --host localhost --port 8080 --interactive
```

### 命令行参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--host` | localhost | API主机 |
| `--port` | 8080 | API端口 |
| `--tunnel-length` | 30000 | 管廊长度(米) |
| `--detectors` | 300 | 检测器数量 |
| `--interval` | 1.0 | 上报间隔(秒) |
| `--no-auto-leak` | false | 禁用自动泄漏 |
| `--leak-distance` | - | 注入初始泄漏位置(米) |
| `--leak-rate` | - | 初始泄漏速率(%LEL) |
| `--leak-duration` | 0 | 泄漏持续时间(0=永久) |
| `--wind-speed` | - | 基础风速(m/s) |
| `--wind-direction` | - | 基础风向(度) |
| `--interactive` | false | 启用交互式CLI |

### 交互式CLI命令

启动时加 `--interactive` 进入交互模式：

```
> leak 5000 35 120          # 在5km处注入35%LEL泄漏，持续120秒
> leak 15000 60             # 在15km处注入60%LEL永久泄漏
> leaks                     # 列出所有活跃泄漏
> leak-remove leak-5000m-xxx  # 移除指定泄漏
> wind 3.0 45               # 设置全局风速3.0m/s，风向45°
> wind-partition P-010 2.0 180  # 设置P-010分区风速2.0m/s，风向180°
> auto-leak off             # 关闭自动泄漏
> quit                      # 退出
```

### 使用示例

```bash
# 在10km处注入25%LEL泄漏，模拟传感器故障场景
python simulator.py --no-auto-leak --leak-distance 10000 --leak-rate 25 --wind-speed 0

# 模拟强风条件下的泄漏
python simulator.py --leak-distance 5000 --leak-rate 40 --wind-speed 5.0 --wind-direction 270

# 交互式注入多个泄漏源
python simulator.py --interactive --no-auto-leak
> leak 3000 15
> leak 12000 55
> wind 2.5 90
```

## 监控与调试

### Prometheus Metrics

后端暴露 `/metrics` 端点，关键指标：

| 指标 | 说明 |
|------|------|
| `gas_monitor_http_requests_total` | HTTP请求计数 |
| `gas_monitor_http_request_duration_seconds` | HTTP请求延迟 |
| `gas_monitor_detector_data_received_total` | 检测器数据接收量 |
| `gas_monitor_detector_data_invalid_total` | 无效数据量 |
| `gas_monitor_influxdb_write_errors_total` | InfluxDB写入错误 |
| `gas_monitor_influxdb_pending_batch_size` | InfluxDB缓冲区大小 |
| `gas_monitor_alarms_triggered_total` | 告警触发量(按级别) |
| `gas_monitor_control_commands_sent_total` | 控制指令发送量 |
| `gas_monitor_mqtt_command_timeouts_total` | MQTT指令超时量 |
| `gas_monitor_mqtt_command_acked_total` | MQTT指令确认量 |
| `gas_monitor_leak_locate_duration_seconds` | 泄漏定位耗时 |
| `gas_monitor_leak_locate_confidence` | 定位置信度 |
| `gas_monitor_ws_clients_connected` | WebSocket连接数 |

### pprof性能分析

独立pprof服务运行在端口6060：

```bash
# CPU分析（30秒采样）
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# 堆内存分析
go tool pprof http://localhost:6060/debug/pprof/heap

# Goroutine分析
go tool pprof http://localhost:6060/debug/pprof/goroutine

# 通过主服务8080端口也可访问
curl http://localhost:8080/debug/pprof/
```

### InfluxDB降采样

系统配置了三级降采样任务：

| 任务 | 源Bucket | 目标Bucket | 粒度 | 数据保留 |
|------|---------|-----------|------|---------|
| downsample-1m | gas-data | gas-data-downsampled-1m | 1分钟 | 30天 |
| downsample-1h | gas-data | gas-data-downsampled-1h | 1小时 | 90天 |
| downsample-1d | gas-data | gas-data-downsampled-1d | 1天 | 永久 |

原始 `gas-data` Bucket保留7天，降采样后数据长期保存。

### MQTT Broker配置

Mosquitto已配置：
- **QoS 2**: 支持最高级别消息可靠性
- **持久会话**: `persist_client_subscription=true`，客户端重连后恢复订阅
- **消息排队**: `max_queued_messages=1000`，离线客户端最多缓存1000条消息
- **QoS 0排队**: `queue_qos0_messages=true`，QoS 0消息也参与排队
- **自动保存**: `autosave_interval=60`，每60秒持久化一次

### 前端Gzip

- **构建时**: `vite-plugin-compression` 生成 `.gz` 预压缩文件
- **运行时**: Nginx `gzip on` 动态压缩未预压缩的资源
- 压缩级别6，最小256字节，覆盖JS/CSS/JSON/SVG/字体等

## API接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/detectors` | 获取所有检测器 |
| GET | `/api/detectors/:id/history` | 检测器1小时浓度趋势 |
| GET | `/api/detectors/:id/health` | 检测器健康状态 |
| POST | `/api/detectors/data` | 上报检测器数据 |
| GET | `/api/sensors` | 获取所有传感器 |
| POST | `/api/sensors/data` | 上报传感器数据 |
| GET | `/api/alarms` | 查询告警列表 |
| PUT | `/api/alarms/:id/acknowledge` | 确认告警 |
| PUT | `/api/alarms/:id/resolve` | 解决告警 |
| POST | `/api/leak/locate` | 触发泄漏源定位 |
| GET | `/api/leak/latest` | 获取最近泄漏事件 |
| POST | `/api/control/valve` | 阀门控制 |
| POST | `/api/control/fan` | 排风机控制 |
| POST | `/api/control/notify` | 发送疏散通知 |
| GET | `/api/partitions` | 获取分区列表 |
| GET | `/metrics` | Prometheus指标 |
| GET | `/ws` | WebSocket连接 |

## 告警分级

| 级别 | 阈值 | 自动联动 |
|------|------|---------|
| L1 预警 | >10%LEL | MQTT+短信推送 |
| L2 报警 | >20%LEL | L1 + 自动关阀+开风机+疏散通知 |
| L3 紧急 | >50%LEL | L2 + (需扩展：相邻分区级联关断) |

## 项目结构

```
├── backend/
│   ├── cmd/main.go              # 主入口
│   ├── config.json              # 外置配置
│   ├── Dockerfile               # Go多阶段构建
│   ├── go.mod / go.sum
│   └── internal/
│       ├── alarm/               # alarm_router 分级告警
│       ├── config/              # 配置加载
│       ├── controller/          # emergency_controller 联动控制
│       ├── handler/             # HTTP处理器
│       ├── locator/             # leak_locator 定位算法
│       │   ├── locator.go       # PSO+贝叶斯+扩散
│       │   └── wind.go          # 风速降级策略
│       ├── metrics/             # Prometheus指标
│       ├── model/               # 数据模型
│       ├── module/              # MessageBus
│       ├── mqtt/                # MQTT客户端(ACK+重试)
│       ├── receiver/            # laser_receiver 数据采集+校验
│       ├── repository/          # InfluxDB/PostgreSQL
│       └── ws/                  # WebSocket Hub
├── src/                         # React前端
│   ├── components/
│   ├── pages/
│   ├── store/
│   ├── hooks/
│   ├── utils/
│   │   ├── corridor_map.js      # 管廊地图渲染类
│   │   ├── gas_panel.js         # 浓度面板渲染类
│   │   ├── heatmap.ts           # 热力图算法
│   │   └── api.ts               # API客户端
│   └── types/
├── scripts/
│   ├── simulator.py             # 激光检测器模拟器
│   ├── Dockerfile               # Python构建
│   └── requirements.txt
├── deploy/
│   ├── nginx/nginx.conf         # Nginx(Gzip+反向代理)
│   ├── mosquitto/mosquitto.conf # MQTT Broker(QoS2+持久)
│   └── influxdb/
│       ├── tasks/               # 降采样Flux任务
│       └── setup.sh             # InfluxDB初始化脚本
├── docker-compose.yml
├── Dockerfile                   # 前端多阶段构建
├── vite.config.ts               # Vite+Gzip插件
└── README.md
```
