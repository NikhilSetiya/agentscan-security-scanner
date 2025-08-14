#!/bin/bash

# AgentScan Monitoring and Alerting Setup Script
# This script sets up comprehensive monitoring for the AgentScan production deployment

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
GRAFANA_VERSION="10.2.0"
PROMETHEUS_VERSION="2.47.0"
ALERTMANAGER_VERSION="0.26.0"
NODE_EXPORTER_VERSION="1.6.1"

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

usage() {
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  --domain DOMAIN        Domain for monitoring (e.g., monitoring.agentscan.dev)"
    echo "  --slack-webhook URL    Slack webhook URL for alerts"
    echo "  --email EMAIL          Email for alerts"
    echo "  --help                 Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  SLACK_WEBHOOK_URL      Slack webhook URL"
    echo "  ALERT_EMAIL           Email for alerts"
}

parse_arguments() {
    # Default values
    DOMAIN="${DOMAIN:-monitoring.agentscan.dev}"
    SLACK_WEBHOOK_URL="${SLACK_WEBHOOK_URL:-}"
    ALERT_EMAIL="${ALERT_EMAIL:-}"
    
    # Parse options
    while [[ $# -gt 0 ]]; do
        case $1 in
            --domain)
                DOMAIN="$2"
                shift 2
                ;;
            --slack-webhook)
                SLACK_WEBHOOK_URL="$2"
                shift 2
                ;;
            --email)
                ALERT_EMAIL="$2"
                shift 2
                ;;
            --help)
                usage
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                usage
                exit 1
                ;;
        esac
    done
}

check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if Docker is installed
    if ! command -v docker &> /dev/null; then
        log_error "Docker is required but not installed."
        exit 1
    fi
    
    # Check if Docker Compose is installed
    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose is required but not installed."
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

create_monitoring_stack() {
    log_info "Creating monitoring stack configuration..."
    
    # Create monitoring directory
    mkdir -p monitoring/{prometheus,grafana,alertmanager}/{config,data}
    
    # Create Prometheus configuration
    cat > monitoring/prometheus/config/prometheus.yml << 'EOF'
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - "alert_rules.yml"

alerting:
  alertmanagers:
    - static_configs:
        - targets:
          - alertmanager:9093

scrape_configs:
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']

  - job_name: 'agentscan-api'
    static_configs:
      - targets: ['agentscan-production-api-server.ondigitalocean.app:443']
    scheme: https
    metrics_path: /metrics
    scrape_interval: 30s

  - job_name: 'agentscan-orchestrator'
    static_configs:
      - targets: ['agentscan-production-orchestrator.ondigitalocean.app:443']
    scheme: https
    metrics_path: /metrics
    scrape_interval: 30s

  - job_name: 'agentscan-web'
    static_configs:
      - targets: ['agentscan-production-web-frontend.ondigitalocean.app:443']
    scheme: https
    metrics_path: /metrics
    scrape_interval: 30s

  - job_name: 'node-exporter'
    static_configs:
      - targets: ['node-exporter:9100']

  - job_name: 'blackbox'
    static_configs:
      - targets:
        - https://agentscan.dev
        - https://api.agentscan.dev/health
        - https://app.agentscan.dev
    metrics_path: /probe
    params:
      module: [http_2xx]
    relabel_configs:
      - source_labels: [__address__]
        target_label: __param_target
      - source_labels: [__param_target]
        target_label: instance
      - target_label: __address__
        replacement: blackbox-exporter:9115
EOF

    # Create alert rules
    cat > monitoring/prometheus/config/alert_rules.yml << 'EOF'
groups:
  - name: agentscan_alerts
    rules:
      - alert: ServiceDown
        expr: up == 0
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Service {{ $labels.instance }} is down"
          description: "{{ $labels.instance }} has been down for more than 5 minutes."

      - alert: HighErrorRate
        expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High error rate on {{ $labels.instance }}"
          description: "Error rate is {{ $value }} errors per second."

      - alert: HighResponseTime
        expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 2
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "High response time on {{ $labels.instance }}"
          description: "95th percentile response time is {{ $value }} seconds."

      - alert: DatabaseConnectionsHigh
        expr: pg_stat_activity_count > 80
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High database connections"
          description: "Database has {{ $value }} active connections."

      - alert: RedisMemoryHigh
        expr: redis_memory_used_bytes / redis_memory_max_bytes > 0.9
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Redis memory usage high"
          description: "Redis memory usage is {{ $value | humanizePercentage }}."

      - alert: ScanQueueBacklog
        expr: agentscan_scan_queue_size > 100
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Scan queue backlog"
          description: "Scan queue has {{ $value }} pending jobs."

      - alert: AgentFailureRate
        expr: rate(agentscan_agent_failures_total[5m]) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High agent failure rate"
          description: "Agent failure rate is {{ $value }} failures per second."

      - alert: DiskSpaceHigh
        expr: (node_filesystem_size_bytes - node_filesystem_free_bytes) / node_filesystem_size_bytes > 0.85
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Disk space usage high"
          description: "Disk usage is {{ $value | humanizePercentage }} on {{ $labels.instance }}."

      - alert: CPUUsageHigh
        expr: 100 - (avg by(instance) (rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100) > 80
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "High CPU usage"
          description: "CPU usage is {{ $value }}% on {{ $labels.instance }}."

      - alert: MemoryUsageHigh
        expr: (node_memory_MemTotal_bytes - node_memory_MemAvailable_bytes) / node_memory_MemTotal_bytes > 0.9
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High memory usage"
          description: "Memory usage is {{ $value | humanizePercentage }} on {{ $labels.instance }}."
EOF

    # Create Alertmanager configuration
    cat > monitoring/alertmanager/config/alertmanager.yml << EOF
global:
  smtp_smarthost: 'localhost:587'
  smtp_from: 'alerts@agentscan.dev'

route:
  group_by: ['alertname']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 1h
  receiver: 'web.hook'

receivers:
  - name: 'web.hook'
    slack_configs:
      - api_url: '${SLACK_WEBHOOK_URL}'
        channel: '#alerts'
        title: 'AgentScan Alert'
        text: '{{ range .Alerts }}{{ .Annotations.summary }}{{ end }}'
        send_resolved: true
    email_configs:
      - to: '${ALERT_EMAIL}'
        subject: 'AgentScan Alert: {{ .GroupLabels.alertname }}'
        body: |
          {{ range .Alerts }}
          Alert: {{ .Annotations.summary }}
          Description: {{ .Annotations.description }}
          {{ end }}

inhibit_rules:
  - source_match:
      severity: 'critical'
    target_match:
      severity: 'warning'
    equal: ['alertname', 'dev', 'instance']
EOF

    # Create Grafana provisioning
    mkdir -p monitoring/grafana/config/{provisioning/{dashboards,datasources},dashboards}
    
    cat > monitoring/grafana/config/provisioning/datasources/prometheus.yml << 'EOF'
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
EOF

    cat > monitoring/grafana/config/provisioning/dashboards/agentscan.yml << 'EOF'
apiVersion: 1

providers:
  - name: 'AgentScan Dashboards'
    orgId: 1
    folder: ''
    type: file
    disableDeletion: false
    updateIntervalSeconds: 10
    allowUiUpdates: true
    options:
      path: /etc/grafana/dashboards
EOF

    log_success "Monitoring stack configuration created"
}

create_docker_compose() {
    log_info "Creating Docker Compose configuration..."
    
    cat > monitoring/docker-compose.yml << EOF
version: '3.8'

services:
  prometheus:
    image: prom/prometheus:v${PROMETHEUS_VERSION}
    container_name: prometheus
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus/config:/etc/prometheus
      - ./prometheus/data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'
      - '--storage.tsdb.retention.time=200h'
      - '--web.enable-lifecycle'
    restart: unless-stopped
    networks:
      - monitoring

  alertmanager:
    image: prom/alertmanager:v${ALERTMANAGER_VERSION}
    container_name: alertmanager
    ports:
      - "9093:9093"
    volumes:
      - ./alertmanager/config:/etc/alertmanager
    command:
      - '--config.file=/etc/alertmanager/alertmanager.yml'
      - '--storage.path=/alertmanager'
      - '--web.external-url=http://localhost:9093'
    restart: unless-stopped
    networks:
      - monitoring

  grafana:
    image: grafana/grafana:${GRAFANA_VERSION}
    container_name: grafana
    ports:
      - "3000:3000"
    volumes:
      - ./grafana/data:/var/lib/grafana
      - ./grafana/config/provisioning:/etc/grafana/provisioning
      - ./grafana/config/dashboards:/etc/grafana/dashboards
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin123
      - GF_USERS_ALLOW_SIGN_UP=false
      - GF_SERVER_DOMAIN=${DOMAIN}
      - GF_SERVER_ROOT_URL=https://${DOMAIN}
    restart: unless-stopped
    networks:
      - monitoring

  node-exporter:
    image: prom/node-exporter:v${NODE_EXPORTER_VERSION}
    container_name: node-exporter
    ports:
      - "9100:9100"
    volumes:
      - /proc:/host/proc:ro
      - /sys:/host/sys:ro
      - /:/rootfs:ro
    command:
      - '--path.procfs=/host/proc'
      - '--path.rootfs=/rootfs'
      - '--path.sysfs=/host/sys'
      - '--collector.filesystem.mount-points-exclude=^/(sys|proc|dev|host|etc)($$|/)'
    restart: unless-stopped
    networks:
      - monitoring

  blackbox-exporter:
    image: prom/blackbox-exporter:latest
    container_name: blackbox-exporter
    ports:
      - "9115:9115"
    volumes:
      - ./blackbox/config:/etc/blackbox_exporter
    restart: unless-stopped
    networks:
      - monitoring

  cadvisor:
    image: gcr.io/cadvisor/cadvisor:latest
    container_name: cadvisor
    ports:
      - "8080:8080"
    volumes:
      - /:/rootfs:ro
      - /var/run:/var/run:rw
      - /sys:/sys:ro
      - /var/lib/docker/:/var/lib/docker:ro
    restart: unless-stopped
    networks:
      - monitoring

networks:
  monitoring:
    driver: bridge

volumes:
  prometheus_data:
  grafana_data:
EOF

    # Create blackbox exporter config
    mkdir -p monitoring/blackbox/config
    cat > monitoring/blackbox/config/blackbox.yml << 'EOF'
modules:
  http_2xx:
    prober: http
    timeout: 5s
    http:
      valid_http_versions: ["HTTP/1.1", "HTTP/2.0"]
      valid_status_codes: []
      method: GET
      follow_redirects: true
      preferred_ip_protocol: "ip4"
EOF

    log_success "Docker Compose configuration created"
}

create_grafana_dashboards() {
    log_info "Creating Grafana dashboards..."
    
    # Create AgentScan overview dashboard
    cat > monitoring/grafana/config/dashboards/agentscan-overview.json << 'EOF'
{
  "dashboard": {
    "id": null,
    "title": "AgentScan Overview",
    "tags": ["agentscan"],
    "style": "dark",
    "timezone": "browser",
    "panels": [
      {
        "id": 1,
        "title": "Service Status",
        "type": "stat",
        "targets": [
          {
            "expr": "up{job=~\"agentscan.*\"}",
            "legendFormat": "{{instance}}"
          }
        ],
        "fieldConfig": {
          "defaults": {
            "color": {
              "mode": "thresholds"
            },
            "thresholds": {
              "steps": [
                {"color": "red", "value": 0},
                {"color": "green", "value": 1}
              ]
            }
          }
        },
        "gridPos": {"h": 8, "w": 12, "x": 0, "y": 0}
      },
      {
        "id": 2,
        "title": "Request Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(http_requests_total[5m])",
            "legendFormat": "{{instance}}"
          }
        ],
        "gridPos": {"h": 8, "w": 12, "x": 12, "y": 0}
      },
      {
        "id": 3,
        "title": "Response Time",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))",
            "legendFormat": "95th percentile"
          }
        ],
        "gridPos": {"h": 8, "w": 12, "x": 0, "y": 8}
      },
      {
        "id": 4,
        "title": "Error Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(http_requests_total{status=~\"5..\"}[5m])",
            "legendFormat": "5xx errors"
          }
        ],
        "gridPos": {"h": 8, "w": 12, "x": 12, "y": 8}
      }
    ],
    "time": {
      "from": "now-1h",
      "to": "now"
    },
    "refresh": "30s"
  }
}
EOF

    log_success "Grafana dashboards created"
}

setup_nginx_proxy() {
    log_info "Setting up Nginx reverse proxy..."
    
    # Create Nginx configuration for monitoring
    cat > monitoring/nginx.conf << EOF
server {
    listen 80;
    server_name ${DOMAIN};
    return 301 https://\$server_name\$request_uri;
}

server {
    listen 443 ssl http2;
    server_name ${DOMAIN};

    ssl_certificate /etc/ssl/certs/monitoring.crt;
    ssl_certificate_key /etc/ssl/private/monitoring.key;

    location / {
        proxy_pass http://grafana:3000;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }

    location /prometheus/ {
        proxy_pass http://prometheus:9090/;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }

    location /alertmanager/ {
        proxy_pass http://alertmanager:9093/;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF

    log_success "Nginx configuration created"
}

create_startup_script() {
    log_info "Creating startup script..."
    
    cat > monitoring/start-monitoring.sh << 'EOF'
#!/bin/bash

# Start AgentScan monitoring stack

set -euo pipefail

echo "Starting AgentScan monitoring stack..."

# Create data directories with proper permissions
sudo mkdir -p {prometheus,grafana,alertmanager}/data
sudo chown -R 472:472 grafana/data  # Grafana user
sudo chown -R 65534:65534 prometheus/data  # Nobody user
sudo chown -R 65534:65534 alertmanager/data

# Start the stack
docker-compose up -d

echo "Monitoring stack started successfully!"
echo ""
echo "Access URLs:"
echo "  Grafana: http://localhost:3000 (admin/admin123)"
echo "  Prometheus: http://localhost:9090"
echo "  Alertmanager: http://localhost:9093"
echo ""
echo "To stop the stack: docker-compose down"
EOF

    chmod +x monitoring/start-monitoring.sh
    
    log_success "Startup script created"
}

print_setup_summary() {
    log_success "Monitoring setup completed!"
    echo ""
    echo "=== Monitoring Stack Information ==="
    echo "Location: ./monitoring/"
    echo "Domain: $DOMAIN"
    echo ""
    echo "=== Services ==="
    echo "- Prometheus: Metrics collection and alerting"
    echo "- Grafana: Visualization and dashboards"
    echo "- Alertmanager: Alert routing and notifications"
    echo "- Node Exporter: System metrics"
    echo "- Blackbox Exporter: Endpoint monitoring"
    echo "- cAdvisor: Container metrics"
    echo ""
    echo "=== Getting Started ==="
    echo "1. Start the monitoring stack:"
    echo "   cd monitoring && ./start-monitoring.sh"
    echo ""
    echo "2. Access Grafana:"
    echo "   URL: http://localhost:3000"
    echo "   Username: admin"
    echo "   Password: admin123"
    echo ""
    echo "3. Configure alerts:"
    echo "   - Slack webhook: $SLACK_WEBHOOK_URL"
    echo "   - Email alerts: $ALERT_EMAIL"
    echo ""
    echo "=== Next Steps ==="
    echo "1. Set up SSL certificates for production"
    echo "2. Configure DNS for $DOMAIN"
    echo "3. Customize alert rules in prometheus/config/alert_rules.yml"
    echo "4. Import additional Grafana dashboards"
    echo "5. Set up log aggregation (ELK stack or similar)"
    echo ""
}

# Main execution
main() {
    log_info "Setting up AgentScan monitoring and alerting..."
    
    check_prerequisites
    create_monitoring_stack
    create_docker_compose
    create_grafana_dashboards
    setup_nginx_proxy
    create_startup_script
    print_setup_summary
    
    log_success "Monitoring setup completed successfully!"
}

# Parse arguments and run
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    parse_arguments "$@"
    main
fi