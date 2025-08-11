# Prometheus and Grafana monitoring stack
resource "helm_release" "prometheus_stack" {
  count = var.monitoring_config.enable_prometheus ? 1 : 0
  
  name       = "prometheus-stack"
  repository = "https://prometheus-community.github.io/helm-charts"
  chart      = "kube-prometheus-stack"
  version    = "51.2.0"
  namespace  = "monitoring"
  
  create_namespace = true
  
  values = [
    templatefile("${path.module}/helm-values/prometheus-stack.yaml", {
      cluster_name = module.eks.cluster_name
      environment  = var.environment
      grafana_admin_password = random_password.grafana_admin.result
      retention_days = var.monitoring_config.retention_days
    })
  ]
  
  depends_on = [module.eks]
}

# Random password for Grafana admin
resource "random_password" "grafana_admin" {
  length  = 16
  special = true
}

# Store Grafana admin password in Secrets Manager
resource "aws_secretsmanager_secret" "grafana_admin" {
  count = var.monitoring_config.enable_grafana ? 1 : 0
  
  name        = "${local.name}-grafana-admin"
  description = "Grafana admin password for AgentScan monitoring"
  
  tags = local.tags
}

resource "aws_secretsmanager_secret_version" "grafana_admin" {
  count = var.monitoring_config.enable_grafana ? 1 : 0
  
  secret_id = aws_secretsmanager_secret.grafana_admin[0].id
  secret_string = jsonencode({
    username = "admin"
    password = random_password.grafana_admin.result
  })
}

# Jaeger for distributed tracing
resource "helm_release" "jaeger" {
  count = var.monitoring_config.enable_jaeger ? 1 : 0
  
  name       = "jaeger"
  repository = "https://jaegertracing.github.io/helm-charts"
  chart      = "jaeger"
  version    = "0.71.2"
  namespace  = "monitoring"
  
  create_namespace = true
  
  values = [
    templatefile("${path.module}/helm-values/jaeger.yaml", {
      cluster_name = module.eks.cluster_name
      environment  = var.environment
    })
  ]
  
  depends_on = [module.eks]
}

# CloudWatch Container Insights
resource "aws_cloudwatch_log_group" "container_insights" {
  name              = "/aws/containerinsights/${module.eks.cluster_name}/application"
  retention_in_days = var.monitoring_config.retention_days
  
  tags = local.tags
}

resource "aws_cloudwatch_log_group" "container_insights_host" {
  name              = "/aws/containerinsights/${module.eks.cluster_name}/host"
  retention_in_days = var.monitoring_config.retention_days
  
  tags = local.tags
}

resource "aws_cloudwatch_log_group" "container_insights_dataplane" {
  name              = "/aws/containerinsights/${module.eks.cluster_name}/dataplane"
  retention_in_days = var.monitoring_config.retention_days
  
  tags = local.tags
}

# CloudWatch agent for Container Insights
resource "helm_release" "cloudwatch_agent" {
  name       = "cloudwatch-agent"
  repository = "https://aws.github.io/eks-charts"
  chart      = "aws-cloudwatch-metrics"
  version    = "0.0.7"
  namespace  = "amazon-cloudwatch"
  
  create_namespace = true
  
  set {
    name  = "clusterName"
    value = module.eks.cluster_name
  }
  
  depends_on = [module.eks]
}

# AWS Load Balancer Controller
resource "helm_release" "aws_load_balancer_controller" {
  name       = "aws-load-balancer-controller"
  repository = "https://aws.github.io/eks-charts"
  chart      = "aws-load-balancer-controller"
  version    = "1.6.2"
  namespace  = "kube-system"
  
  set {
    name  = "clusterName"
    value = module.eks.cluster_name
  }
  
  set {
    name  = "serviceAccount.create"
    value = "true"
  }
  
  set {
    name  = "serviceAccount.name"
    value = "aws-load-balancer-controller"
  }
  
  set {
    name  = "serviceAccount.annotations.eks\\.amazonaws\\.com/role-arn"
    value = module.load_balancer_controller_irsa_role.iam_role_arn
  }
  
  depends_on = [module.eks]
}

# Cluster Autoscaler
resource "helm_release" "cluster_autoscaler" {
  name       = "cluster-autoscaler"
  repository = "https://kubernetes.github.io/autoscaler"
  chart      = "cluster-autoscaler"
  version    = "9.29.0"
  namespace  = "kube-system"
  
  set {
    name  = "autoDiscovery.clusterName"
    value = module.eks.cluster_name
  }
  
  set {
    name  = "awsRegion"
    value = var.aws_region
  }
  
  set {
    name  = "serviceAccount.create"
    value = "true"
  }
  
  set {
    name  = "serviceAccount.name"
    value = "cluster-autoscaler"
  }
  
  set {
    name  = "serviceAccount.annotations.eks\\.amazonaws\\.com/role-arn"
    value = module.cluster_autoscaler_irsa_role.iam_role_arn
  }
  
  depends_on = [module.eks]
}

# CloudWatch Alarms for critical metrics
resource "aws_cloudwatch_metric_alarm" "high_cpu_utilization" {
  alarm_name          = "${local.name}-high-cpu-utilization"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "CPUUtilization"
  namespace           = "AWS/EKS"
  period              = "300"
  statistic           = "Average"
  threshold           = "80"
  alarm_description   = "This metric monitors EKS cluster CPU utilization"
  alarm_actions       = [aws_sns_topic.alerts.arn]
  
  dimensions = {
    ClusterName = module.eks.cluster_name
  }
  
  tags = local.tags
}

resource "aws_cloudwatch_metric_alarm" "high_memory_utilization" {
  alarm_name          = "${local.name}-high-memory-utilization"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "MemoryUtilization"
  namespace           = "ContainerInsights"
  period              = "300"
  statistic           = "Average"
  threshold           = "85"
  alarm_description   = "This metric monitors EKS cluster memory utilization"
  alarm_actions       = [aws_sns_topic.alerts.arn]
  
  dimensions = {
    ClusterName = module.eks.cluster_name
  }
  
  tags = local.tags
}

resource "aws_cloudwatch_metric_alarm" "pod_restart_rate" {
  alarm_name          = "${local.name}-high-pod-restart-rate"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "pod_restart_total"
  namespace           = "ContainerInsights"
  period              = "300"
  statistic           = "Sum"
  threshold           = "10"
  alarm_description   = "This metric monitors pod restart rate"
  alarm_actions       = [aws_sns_topic.alerts.arn]
  
  dimensions = {
    ClusterName = module.eks.cluster_name
    Namespace   = "agentscan"
  }
  
  tags = local.tags
}

resource "aws_cloudwatch_metric_alarm" "rds_cpu_utilization" {
  alarm_name          = "${local.name}-rds-high-cpu"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "CPUUtilization"
  namespace           = "AWS/RDS"
  period              = "300"
  statistic           = "Average"
  threshold           = "80"
  alarm_description   = "This metric monitors RDS CPU utilization"
  alarm_actions       = [aws_sns_topic.alerts.arn]
  
  dimensions = {
    DBInstanceIdentifier = aws_db_instance.main.id
  }
  
  tags = local.tags
}

resource "aws_cloudwatch_metric_alarm" "rds_connection_count" {
  alarm_name          = "${local.name}-rds-high-connections"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "DatabaseConnections"
  namespace           = "AWS/RDS"
  period              = "300"
  statistic           = "Average"
  threshold           = "80"
  alarm_description   = "This metric monitors RDS connection count"
  alarm_actions       = [aws_sns_topic.alerts.arn]
  
  dimensions = {
    DBInstanceIdentifier = aws_db_instance.main.id
  }
  
  tags = local.tags
}

resource "aws_cloudwatch_metric_alarm" "redis_cpu_utilization" {
  alarm_name          = "${local.name}-redis-high-cpu"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "CPUUtilization"
  namespace           = "AWS/ElastiCache"
  period              = "300"
  statistic           = "Average"
  threshold           = "80"
  alarm_description   = "This metric monitors Redis CPU utilization"
  alarm_actions       = [aws_sns_topic.alerts.arn]
  
  dimensions = {
    CacheClusterId = aws_elasticache_replication_group.main.id
  }
  
  tags = local.tags
}

# SNS Topic for alerts
resource "aws_sns_topic" "alerts" {
  name = "${local.name}-alerts"
  
  tags = local.tags
}

resource "aws_sns_topic_policy" "alerts" {
  arn = aws_sns_topic.alerts.arn
  
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Principal = {
          Service = "cloudwatch.amazonaws.com"
        }
        Action = "SNS:Publish"
        Resource = aws_sns_topic.alerts.arn
      }
    ]
  })
}

# SNS Topic Subscription for email alerts
resource "aws_sns_topic_subscription" "email_alerts" {
  count = var.environment == "production" ? 1 : 0
  
  topic_arn = aws_sns_topic.alerts.arn
  protocol  = "email"
  endpoint  = "alerts@agentscan.dev"  # Replace with actual email
}

# CloudWatch Dashboard
resource "aws_cloudwatch_dashboard" "main" {
  dashboard_name = "${local.name}-dashboard"
  
  dashboard_body = jsonencode({
    widgets = [
      {
        type   = "metric"
        x      = 0
        y      = 0
        width  = 12
        height = 6
        
        properties = {
          metrics = [
            ["AWS/EKS", "cluster_failed_request_count", "ClusterName", module.eks.cluster_name],
            [".", "cluster_request_total", ".", "."]
          ]
          view    = "timeSeries"
          stacked = false
          region  = var.aws_region
          title   = "EKS API Server Requests"
          period  = 300
        }
      },
      {
        type   = "metric"
        x      = 0
        y      = 6
        width  = 12
        height = 6
        
        properties = {
          metrics = [
            ["AWS/RDS", "CPUUtilization", "DBInstanceIdentifier", aws_db_instance.main.id],
            [".", "DatabaseConnections", ".", "."],
            [".", "ReadLatency", ".", "."],
            [".", "WriteLatency", ".", "."]
          ]
          view    = "timeSeries"
          stacked = false
          region  = var.aws_region
          title   = "RDS Performance Metrics"
          period  = 300
        }
      },
      {
        type   = "metric"
        x      = 0
        y      = 12
        width  = 12
        height = 6
        
        properties = {
          metrics = [
            ["AWS/ElastiCache", "CPUUtilization", "CacheClusterId", aws_elasticache_replication_group.main.id],
            [".", "NetworkBytesIn", ".", "."],
            [".", "NetworkBytesOut", ".", "."],
            [".", "CurrConnections", ".", "."]
          ]
          view    = "timeSeries"
          stacked = false
          region  = var.aws_region
          title   = "Redis Performance Metrics"
          period  = 300
        }
      }
    ]
  })
}

# Application-specific custom metrics
resource "aws_cloudwatch_log_metric_filter" "scan_errors" {
  name           = "${local.name}-scan-errors"
  log_group_name = aws_cloudwatch_log_group.container_insights.name
  pattern        = "[timestamp, request_id, level=\"ERROR\", message=\"*scan*failed*\"]"
  
  metric_transformation {
    name      = "ScanErrors"
    namespace = "AgentScan/Application"
    value     = "1"
  }
}

resource "aws_cloudwatch_metric_alarm" "scan_error_rate" {
  alarm_name          = "${local.name}-high-scan-error-rate"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "ScanErrors"
  namespace           = "AgentScan/Application"
  period              = "300"
  statistic           = "Sum"
  threshold           = "10"
  alarm_description   = "This metric monitors scan error rate"
  alarm_actions       = [aws_sns_topic.alerts.arn]
  treat_missing_data  = "notBreaching"
  
  tags = local.tags
}

resource "aws_cloudwatch_log_metric_filter" "scan_duration" {
  name           = "scan-duration"
  log_group_name = aws_cloudwatch_log_group.container_insights.name
  pattern        = "[timestamp, request_id, level=\"INFO\", message=\"scan completed\", duration]"
  
  metric_transformation {
    name      = "ScanDuration"
    namespace = "AgentScan/Application"
    value     = "$duration"
    unit      = "Seconds"
  }
}

resource "aws_cloudwatch_metric_alarm" "scan_duration_high" {
  alarm_name          = "${local.name}-high-scan-duration"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "ScanDuration"
  namespace           = "AgentScan/Application"
  period              = "300"
  statistic           = "Average"
  threshold           = "1800"  # 30 minutes
  alarm_description   = "This metric monitors average scan duration"
  alarm_actions       = [aws_sns_topic.alerts.arn]
  treat_missing_data  = "notBreaching"
  
  tags = local.tags
}