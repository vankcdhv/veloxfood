#!/bin/sh
set -eu

ES_URL="${ES_URL:-http://elasticsearch:9200}"
KIBANA_URL="${KIBANA_URL:-http://kibana:5601}"
INDEX_PATTERN="service-logs-*"
ILM_POLICY="service-logs-policy"
INDEX_TEMPLATE="service-logs-template"
RETENTION_DAYS="${RETENTION_DAYS:-7}"
ROLLOVER_MAX_SIZE="${ROLLOVER_MAX_SIZE:-1gb}"
ROLLOVER_MAX_AGE="${ROLLOVER_MAX_AGE:-1d}"

log() { echo "[logging-init] $*"; }

wait_for() {
  url="$1"; name="$2"; max="${3:-60}"
  i=0
  until curl -sf -o /dev/null "$url"; do
    i=$((i+1))
    if [ "$i" -ge "$max" ]; then
      log "ERROR: $name did not become ready at $url after ${max}s"
      exit 1
    fi
    sleep 2
  done
  log "$name ready at $url"
}

log "waiting for Elasticsearch..."
wait_for "$ES_URL/_cluster/health?wait_for_status=yellow&timeout=30s" "Elasticsearch" 60

log "creating ILM policy '$ILM_POLICY' (delete after ${RETENTION_DAYS}d)..."
# Daily index pattern from fluentd logstash_format — no rollover alias is set,
# so rollover action would error. Keep only the delete phase.
curl -sf -X PUT "$ES_URL/_ilm/policy/$ILM_POLICY" \
  -H 'Content-Type: application/json' \
  -d "{
    \"policy\": {
      \"phases\": {
        \"hot\": { \"actions\": {} },
        \"delete\": {
          \"min_age\": \"${RETENTION_DAYS}d\",
          \"actions\": { \"delete\": {} }
        }
      }
    }
  }" > /dev/null
log "ILM policy applied"

log "creating index template '$INDEX_TEMPLATE' for pattern '$INDEX_PATTERN'..."
curl -sf -X PUT "$ES_URL/_index_template/$INDEX_TEMPLATE" \
  -H 'Content-Type: application/json' \
  -d "{
    \"index_patterns\": [\"$INDEX_PATTERN\"],
    \"priority\": 100,
    \"template\": {
      \"settings\": {
        \"number_of_shards\": 1,
        \"number_of_replicas\": 0,
        \"index.lifecycle.name\": \"$ILM_POLICY\"
      },
      \"mappings\": {
        \"properties\": {
          \"@timestamp\":  { \"type\": \"date\" },
          \"level\":       { \"type\": \"keyword\" },
          \"msg\":         { \"type\": \"text\" },
          \"service\":     { \"type\": \"keyword\" },
          \"trace_id\":    { \"type\": \"keyword\" },
          \"method\":      { \"type\": \"keyword\" },
          \"path\":        { \"type\": \"keyword\" },
          \"status\":      { \"type\": \"integer\" },
          \"latency_ms\":  { \"type\": \"long\" },
          \"client_ip\":   { \"type\": \"ip\" },
          \"error\":       { \"type\": \"text\" },
          \"caller_file\": { \"type\": \"keyword\" },
          \"caller_line\": { \"type\": \"integer\" },
          \"caller_func\": { \"type\": \"keyword\" },
          \"stream\":      { \"type\": \"keyword\" },
          \"container_name\": { \"type\": \"keyword\" },
          \"container_id\":   { \"type\": \"keyword\" }
        }
      }
    }
  }" > /dev/null
log "index template applied"

log "waiting for Kibana..."
wait_for "$KIBANA_URL/api/status" "Kibana" 90

log "creating Kibana data view '$INDEX_PATTERN'..."
http_code=$(curl -s -o /tmp/dv.out -w '%{http_code}' -X POST "$KIBANA_URL/api/data_views/data_view" \
  -H 'Content-Type: application/json' \
  -H 'kbn-xsrf: true' \
  -d "{
    \"data_view\": {
      \"title\": \"$INDEX_PATTERN\",
      \"name\": \"service-logs\",
      \"timeFieldName\": \"@timestamp\"
    },
    \"override\": false
  }")
case "$http_code" in
  200|201) log "data view created" ;;
  400)
    if grep -q "Duplicate" /tmp/dv.out 2>/dev/null; then
      log "data view already exists (ok)"
    else
      log "ERROR: Kibana 400 response: $(cat /tmp/dv.out)"
      exit 1
    fi
    ;;
  409) log "data view already exists (ok)" ;;
  *)
    log "ERROR: unexpected Kibana response ($http_code): $(cat /tmp/dv.out)"
    exit 1
    ;;
esac

log "done"
