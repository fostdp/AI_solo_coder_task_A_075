-- 智慧城市地下综合管廊燃气泄漏激光监测系统 - PostgreSQL 初始化脚本

-- 防火分区表
CREATE TABLE IF NOT EXISTS partitions (
    id VARCHAR(32) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    start_distance FLOAT NOT NULL,
    end_distance FLOAT NOT NULL,
    valve_id VARCHAR(32),
    fan_id VARCHAR(32)
);

-- 激光甲烷检测器表
CREATE TABLE IF NOT EXISTS detectors (
    id VARCHAR(32) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    latitude FLOAT NOT NULL,
    longitude FLOAT NOT NULL,
    distance_along_tunnel FLOAT NOT NULL,
    partition_id VARCHAR(32) REFERENCES partitions(id),
    status VARCHAR(20) DEFAULT 'online',
    installed_at TIMESTAMP DEFAULT NOW()
);

-- 氧气/温湿度传感器表
CREATE TABLE IF NOT EXISTS sensors (
    id VARCHAR(32) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(20) NOT NULL,
    latitude FLOAT NOT NULL,
    longitude FLOAT NOT NULL,
    partition_id VARCHAR(32) REFERENCES partitions(id),
    status VARCHAR(20) DEFAULT 'online',
    installed_at TIMESTAMP DEFAULT NOW()
);

-- 告警记录表
CREATE TABLE IF NOT EXISTS alarms (
    id VARCHAR(32) PRIMARY KEY,
    level INT NOT NULL,
    detector_id VARCHAR(32) REFERENCES detectors(id),
    concentration FLOAT NOT NULL,
    threshold FLOAT NOT NULL,
    message TEXT,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- 泄漏事件表
CREATE TABLE IF NOT EXISTS leak_events (
    id VARCHAR(32) PRIMARY KEY,
    source_lat FLOAT NOT NULL,
    source_lng FLOAT NOT NULL,
    source_distance FLOAT NOT NULL,
    leak_rate FLOAT NOT NULL,
    confidence FLOAT NOT NULL,
    diffusion_radius FLOAT NOT NULL,
    method VARCHAR(20) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    status VARCHAR(20) DEFAULT 'active'
);

-- 联动控制日志表
CREATE TABLE IF NOT EXISTS control_logs (
    id VARCHAR(32) PRIMARY KEY,
    device_id VARCHAR(32) NOT NULL,
    device_type VARCHAR(20) NOT NULL,
    action VARCHAR(20) NOT NULL,
    success BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW()
);

-- 风速风向数据表
CREATE TABLE IF NOT EXISTS wind_data (
    id VARCHAR(32) PRIMARY KEY,
    speed FLOAT NOT NULL,
    direction FLOAT NOT NULL,
    partition_id VARCHAR(32) REFERENCES partitions(id),
    timestamp TIMESTAMP DEFAULT NOW()
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_alarms_detector ON alarms(detector_id);
CREATE INDEX IF NOT EXISTS idx_alarms_status ON alarms(status);
CREATE INDEX IF NOT EXISTS idx_alarms_created ON alarms(created_at);
CREATE INDEX IF NOT EXISTS idx_detectors_partition ON detectors(partition_id);
CREATE INDEX IF NOT EXISTS idx_leak_events_status ON leak_events(status);
CREATE INDEX IF NOT EXISTS idx_control_logs_device ON control_logs(device_id);
CREATE INDEX IF NOT EXISTS idx_wind_partition ON wind_data(partition_id);

-- 插入30个防火分区（每区1公里）
INSERT INTO partitions (id, name, start_distance, end_distance, valve_id, fan_id) VALUES
('P-001', '1号防火分区', 0, 1000, 'V-001', 'F-001'),
('P-002', '2号防火分区', 1000, 2000, 'V-002', 'F-002'),
('P-003', '3号防火分区', 2000, 3000, 'V-003', 'F-003'),
('P-004', '4号防火分区', 3000, 4000, 'V-004', 'F-004'),
('P-005', '5号防火分区', 4000, 5000, 'V-005', 'F-005'),
('P-006', '6号防火分区', 5000, 6000, 'V-006', 'F-006'),
('P-007', '7号防火分区', 6000, 7000, 'V-007', 'F-007'),
('P-008', '8号防火分区', 7000, 8000, 'V-008', 'F-008'),
('P-009', '9号防火分区', 8000, 9000, 'V-009', 'F-009'),
('P-010', '10号防火分区', 9000, 10000, 'V-010', 'F-010'),
('P-011', '11号防火分区', 10000, 11000, 'V-011', 'F-011'),
('P-012', '12号防火分区', 11000, 12000, 'V-012', 'F-012'),
('P-013', '13号防火分区', 12000, 13000, 'V-013', 'F-013'),
('P-014', '14号防火分区', 13000, 14000, 'V-014', 'F-014'),
('P-015', '15号防火分区', 14000, 15000, 'V-015', 'F-015'),
('P-016', '16号防火分区', 15000, 16000, 'V-016', 'F-016'),
('P-017', '17号防火分区', 16000, 17000, 'V-017', 'F-017'),
('P-018', '18号防火分区', 17000, 18000, 'V-018', 'F-018'),
('P-019', '19号防火分区', 18000, 19000, 'V-019', 'F-019'),
('P-020', '20号防火分区', 19000, 20000, 'V-020', 'F-020'),
('P-021', '21号防火分区', 20000, 21000, 'V-021', 'F-021'),
('P-022', '22号防火分区', 21000, 22000, 'V-022', 'F-022'),
('P-023', '23号防火分区', 22000, 23000, 'V-023', 'F-023'),
('P-024', '24号防火分区', 23000, 24000, 'V-024', 'F-024'),
('P-025', '25号防火分区', 24000, 25000, 'V-025', 'F-025'),
('P-026', '26号防火分区', 25000, 26000, 'V-026', 'F-026'),
('P-027', '27号防火分区', 26000, 27000, 'V-027', 'F-027'),
('P-028', '28号防火分区', 27000, 28000, 'V-028', 'F-028'),
('P-029', '29号防火分区', 28000, 29000, 'V-029', 'F-029'),
('P-030', '30号防火分区', 29000, 30000, 'V-030', 'F-030');

-- 插入300台激光甲烷检测器（每100米一台，沿30公里管廊）
-- 坐标沿弯曲路径：起点(31.2304, 121.4737)，终点约(31.2400, 121.6900)
-- 使用正弦曲线模拟管廊弯曲
INSERT INTO detectors (id, name, latitude, longitude, distance_along_tunnel, partition_id, status) SELECT
    d.id, d.name, d.latitude, d.longitude, d.distance, d.partition_id, d.status
FROM (
    SELECT
        'D-' || lpad(i::text, 3, '0') as id,
        '激光甲烷检测器-' || lpad(i::text, 3, '0') as name,
        31.2304 + (i * 100.0 / 30000.0) * 0.0096 + 0.002 * sin(2.0 * pi() * i * 100.0 / 10000.0) as latitude,
        121.4737 + (i * 100.0 / 30000.0) * 0.2163 + 0.003 * sin(2.0 * pi() * i * 100.0 / 8000.0) as longitude,
        i * 100.0 as distance,
        'P-' || lpad(ceil(i * 100.0 / 1000.0)::int::text, 3, '0') as partition_id,
        'online' as status
    FROM generate_series(1, 300) as i
) d;

-- 插入50台环境传感器（20台O2 + 15台温度 + 15台湿度，分布在通风口和阀门井）
INSERT INTO sensors (id, name, type, latitude, longitude, partition_id, status) SELECT
    s.id, s.name, s.type, s.latitude, s.longitude, s.partition_id, s.status
FROM (
    SELECT
        'S-O2-' || lpad(i::text, 3, '0') as id,
        '氧气传感器-' || lpad(i::text, 3, '0') as name,
        'o2' as type,
        31.2304 + ((i * 1500.0) / 30000.0) * 0.0096 + 0.002 * sin(2.0 * pi() * (i * 1500.0) / 10000.0) as latitude,
        121.4737 + ((i * 1500.0) / 30000.0) * 0.2163 + 0.003 * sin(2.0 * pi() * (i * 1500.0) / 8000.0) as longitude,
        'P-' || lpad(ceil(i * 1500.0 / 1000.0)::int::text, 3, '0') as partition_id,
        'online' as status
    FROM generate_series(1, 20) as i
    UNION ALL
    SELECT
        'S-T-' || lpad(i::text, 3, '0') as id,
        '温度传感器-' || lpad(i::text, 3, '0') as name,
        'temperature' as type,
        31.2304 + ((i * 2000.0) / 30000.0) * 0.0096 + 0.002 * sin(2.0 * pi() * (i * 2000.0) / 10000.0) as latitude,
        121.4737 + ((i * 2000.0) / 30000.0) * 0.2163 + 0.003 * sin(2.0 * pi() * (i * 2000.0) / 8000.0) as longitude,
        'P-' || lpad(ceil(i * 2000.0 / 1000.0)::int::text, 3, '0') as partition_id,
        'online' as status
    FROM generate_series(1, 15) as i
    UNION ALL
    SELECT
        'S-H-' || lpad(i::text, 3, '0') as id,
        '湿度传感器-' || lpad(i::text, 3, '0') as name,
        'humidity' as type,
        31.2304 + ((i * 2000.0 + 1000.0) / 30000.0) * 0.0096 + 0.002 * sin(2.0 * pi() * (i * 2000.0 + 1000.0) / 10000.0) as latitude,
        121.4737 + ((i * 2000.0 + 1000.0) / 30000.0) * 0.2163 + 0.003 * sin(2.0 * pi() * (i * 2000.0 + 1000.0) / 8000.0) as longitude,
        'P-' || lpad(ceil((i * 2000.0 + 1000.0) / 1000.0)::int::text, 3, '0') as partition_id,
        'online' as status
    FROM generate_series(1, 15) as i
) s;
