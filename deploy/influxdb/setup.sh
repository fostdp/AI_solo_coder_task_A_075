#!/bin/sh
set -e

INFLUX_URL="http://influxdb:8086"
TOKEN="my-super-secret-token"
ORG="gas-monitor"

echo "Waiting for InfluxDB to be ready..."
until curl -s "${INFLUX_URL}/health" | grep -q "pass"; do
    sleep 2
done
echo "InfluxDB is ready."

echo "Creating downsampled buckets..."

curl -s -X POST "${INFLUX_URL}/api/v2/buckets" \
    -H "Authorization: Token ${TOKEN}" \
    -H "Content-Type: application/json" \
    -d '{
        "orgID": "'"${ORG}"'",
        "name": "gas-data-downsampled-1m",
        "retentionRules": [{"type": "expire", "everySeconds": 2592000}]
    }' || echo "Bucket gas-data-downsampled-1m may already exist"

curl -s -X POST "${INFLUX_URL}/api/v2/buckets" \
    -H "Authorization: Token ${TOKEN}" \
    -H "Content-Type: application/json" \
    -d '{
        "orgID": "'"${ORG}"'",
        "name": "gas-data-downsampled-1h",
        "retentionRules": [{"type": "expire", "everySeconds": 7776000}]
    }' || echo "Bucket gas-data-downsampled-1h may already exist"

curl -s -X POST "${INFLUX_URL}/api/v2/buckets" \
    -H "Authorization: Token ${TOKEN}" \
    -H "Content-Type: application/json" \
    -d '{
        "orgID": "'"${ORG}"'",
        "name": "gas-data-downsampled-1d",
        "retentionRules": [{"type": "expire", "everySeconds": 0}]
    }' || echo "Bucket gas-data-downsampled-1d may already exist"

echo "Creating downsample tasks..."

ORG_ID=$(curl -s "${INFLUX_URL}/api/v2/orgs?org=${ORG}" \
    -H "Authorization: Token ${TOKEN}" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

TASK_1M=$(cat /scripts/downsample_1m.flux)
curl -s -X POST "${INFLUX_URL}/api/v2/tasks" \
    -H "Authorization: Token ${TOKEN}" \
    -H "Content-Type: application/json" \
    -d '{
        "orgID": "'"${ORG_ID}"'",
        "name": "downsample-1m",
        "every": "1m",
        "flux": '"$(echo "$TASK_1M" | python3 -c 'import sys,json; print(json.dumps(sys.stdin.read()))')"'
    }' || echo "Task downsample-1m may already exist"

TASK_1H=$(cat /scripts/downsample_1h.flux)
curl -s -X POST "${INFLUX_URL}/api/v2/tasks" \
    -H "Authorization: Token ${TOKEN}" \
    -H "Content-Type: application/json" \
    -d '{
        "orgID": "'"${ORG_ID}"'",
        "name": "downsample-1h",
        "every": "1h",
        "flux": '"$(echo "$TASK_1H" | python3 -c 'import sys,json; print(json.dumps(sys.stdin.read()))')"'
    }' || echo "Task downsample-1h may already exist"

TASK_1D=$(cat /scripts/downsample_1d.flux)
curl -s -X POST "${INFLUX_URL}/api/v2/tasks" \
    -H "Authorization: Token ${TOKEN}" \
    -H "Content-Type: application/json" \
    -d '{
        "orgID": "'"${ORG_ID}"'",
        "name": "downsample-1d",
        "every": "1d",
        "flux": '"$(echo "$TASK_1D" | python3 -c 'import sys,json; print(json.dumps(sys.stdin.read()))')"'
    }' || echo "Task downsample-1d may already exist"

echo "InfluxDB setup complete."
