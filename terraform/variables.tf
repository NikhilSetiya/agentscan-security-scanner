variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "us-west-2"
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "production"
  
  validation {
    condition     = contains(["development", "staging", "production"], var.environment)
    error_message = "Environment must be one of: development, staging, production."
  }
}

variable "owner" {
  description = "Owner of the infrastructure"
  type        = string
  default     = "platform-team"
}

variable "cluster_version" {
  description = "Kubernetes cluster version"
  type        = string
  default     = "1.28"
}

variable "node_groups" {
  description = "EKS node group configurations"
  type = map(object({
    instance_types = list(string)
    capacity_type  = string
    min_size      = number
    max_size      = number
    desired_size  = number
    disk_size     = number
    labels        = map(string)
    taints        = list(object({
      key    = string
      value  = string
      effect = string
    }))
  }))
  
  default = {
    general = {
      instance_types = ["m5.large", "m5.xlarge"]
      capacity_type  = "ON_DEMAND"
      min_size      = 2
      max_size      = 10
      desired_size  = 3
      disk_size     = 50
      labels = {
        role = "general"
      }
      taints = []
    }
    
    compute = {
      instance_types = ["c5.xlarge", "c5.2xlarge"]
      capacity_type  = "SPOT"
      min_size      = 0
      max_size      = 20
      desired_size  = 2
      disk_size     = 100
      labels = {
        role = "compute"
        workload = "scanning"
      }
      taints = [
        {
          key    = "workload"
          value  = "scanning"
          effect = "NO_SCHEDULE"
        }
      ]
    }
  }
}

variable "database_config" {
  description = "RDS PostgreSQL configuration"
  type = object({
    instance_class    = string
    allocated_storage = number
    max_allocated_storage = number
    backup_retention_period = number
    backup_window     = string
    maintenance_window = string
    multi_az          = bool
    deletion_protection = bool
  })
  
  default = {
    instance_class    = "db.r6g.large"
    allocated_storage = 100
    max_allocated_storage = 1000
    backup_retention_period = 7
    backup_window     = "03:00-04:00"
    maintenance_window = "sun:04:00-sun:05:00"
    multi_az          = true
    deletion_protection = true
  }
}

variable "redis_config" {
  description = "ElastiCache Redis configuration"
  type = object({
    node_type           = string
    num_cache_nodes     = number
    parameter_group_name = string
    port               = number
    maintenance_window = string
    snapshot_retention_limit = number
    snapshot_window    = string
  })
  
  default = {
    node_type           = "cache.r6g.large"
    num_cache_nodes     = 2
    parameter_group_name = "default.redis7"
    port               = 6379
    maintenance_window = "sun:05:00-sun:06:00"
    snapshot_retention_limit = 5
    snapshot_window    = "03:00-05:00"
  }
}

variable "monitoring_config" {
  description = "Monitoring and observability configuration"
  type = object({
    enable_prometheus     = bool
    enable_grafana       = bool
    enable_alertmanager  = bool
    enable_jaeger        = bool
    retention_days       = number
  })
  
  default = {
    enable_prometheus     = true
    enable_grafana       = true
    enable_alertmanager  = true
    enable_jaeger        = true
    retention_days       = 30
  }
}

variable "backup_config" {
  description = "Backup and disaster recovery configuration"
  type = object({
    enable_automated_backups = bool
    backup_schedule         = string
    retention_days          = number
    cross_region_backup     = bool
    backup_regions          = list(string)
  })
  
  default = {
    enable_automated_backups = true
    backup_schedule         = "0 2 * * *"  # Daily at 2 AM
    retention_days          = 30
    cross_region_backup     = true
    backup_regions          = ["us-east-1"]
  }
}

variable "security_config" {
  description = "Security configuration"
  type = object({
    enable_waf              = bool
    enable_shield           = bool
    enable_guardduty        = bool
    enable_security_hub     = bool
    enable_config           = bool
    enable_cloudtrail       = bool
  })
  
  default = {
    enable_waf              = true
    enable_shield           = true
    enable_guardduty        = true
    enable_security_hub     = true
    enable_config           = true
    enable_cloudtrail       = true
  }
}

variable "domain_name" {
  description = "Domain name for the application"
  type        = string
  default     = "agentscan.dev"
}

variable "certificate_arn" {
  description = "ACM certificate ARN for HTTPS"
  type        = string
  default     = ""
}