# ElastiCache Subnet Group
resource "aws_elasticache_subnet_group" "main" {
  name       = "${local.name}-redis-subnet-group"
  subnet_ids = module.vpc.private_subnets

  tags = local.tags
}

# ElastiCache Parameter Group
resource "aws_elasticache_parameter_group" "main" {
  family = "redis7.x"
  name   = "${local.name}-redis-params"

  parameter {
    name  = "maxmemory-policy"
    value = "allkeys-lru"
  }

  parameter {
    name  = "timeout"
    value = "300"
  }

  parameter {
    name  = "tcp-keepalive"
    value = "300"
  }

  tags = local.tags
}

# Random auth token for Redis
resource "random_password" "redis_auth_token" {
  length  = 32
  special = false
}

# ElastiCache Replication Group (Redis Cluster)
resource "aws_elasticache_replication_group" "main" {
  replication_group_id       = "${local.name}-redis"
  description                = "Redis cluster for AgentScan"

  # Node configuration
  node_type               = var.redis_config.node_type
  port                    = var.redis_config.port
  parameter_group_name    = aws_elasticache_parameter_group.main.name

  # Cluster configuration
  num_cache_clusters      = var.redis_config.num_cache_nodes
  
  # Network
  subnet_group_name       = aws_elasticache_subnet_group.main.name
  security_group_ids      = [aws_security_group.redis.id]

  # Security
  at_rest_encryption_enabled = true
  transit_encryption_enabled = true
  auth_token                 = random_password.redis_auth_token.result
  kms_key_id                = aws_kms_key.redis.arn

  # Backup
  snapshot_retention_limit = var.redis_config.snapshot_retention_limit
  snapshot_window         = var.redis_config.snapshot_window
  
  # Maintenance
  maintenance_window = var.redis_config.maintenance_window
  auto_minor_version_upgrade = true

  # Logging
  log_delivery_configuration {
    destination      = aws_cloudwatch_log_group.redis_slow.name
    destination_type = "cloudwatch-logs"
    log_format       = "text"
    log_type         = "slow-log"
  }

  tags = merge(local.tags, {
    Name = "${local.name}-redis"
  })

  lifecycle {
    ignore_changes = [auth_token]
  }
}

# KMS Key for Redis encryption
resource "aws_kms_key" "redis" {
  description             = "KMS key for Redis encryption"
  deletion_window_in_days = 7

  tags = merge(local.tags, {
    Name = "${local.name}-redis-kms"
  })
}

resource "aws_kms_alias" "redis" {
  name          = "alias/${local.name}-redis"
  target_key_id = aws_kms_key.redis.key_id
}

# CloudWatch Log Group for Redis slow logs
resource "aws_cloudwatch_log_group" "redis_slow" {
  name              = "/aws/elasticache/${local.name}-redis/slow-log"
  retention_in_days = 7

  tags = local.tags
}

# Store Redis credentials in AWS Secrets Manager
resource "aws_secretsmanager_secret" "redis_credentials" {
  name        = "${local.name}-redis-credentials"
  description = "Redis credentials for AgentScan"

  tags = local.tags
}

resource "aws_secretsmanager_secret_version" "redis_credentials" {
  secret_id = aws_secretsmanager_secret.redis_credentials.id
  secret_string = jsonencode({
    auth_token = random_password.redis_auth_token.result
    host       = aws_elasticache_replication_group.main.primary_endpoint_address
    port       = aws_elasticache_replication_group.main.port
  })

  lifecycle {
    ignore_changes = [secret_string]
  }
}