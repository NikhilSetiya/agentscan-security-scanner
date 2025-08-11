# AWS Backup Vault
resource "aws_backup_vault" "main" {
  name        = "${local.name}-backup-vault"
  kms_key_arn = aws_kms_key.backup.arn

  tags = local.tags
}

# KMS Key for AWS Backup
resource "aws_kms_key" "backup" {
  description             = "KMS key for AWS Backup"
  deletion_window_in_days = 7

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "Enable IAM User Permissions"
        Effect = "Allow"
        Principal = {
          AWS = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:root"
        }
        Action   = "kms:*"
        Resource = "*"
      },
      {
        Sid    = "Allow AWS Backup to use the key"
        Effect = "Allow"
        Principal = {
          Service = "backup.amazonaws.com"
        }
        Action = [
          "kms:Decrypt",
          "kms:GenerateDataKey",
          "kms:ReEncrypt*",
          "kms:CreateGrant",
          "kms:DescribeKey"
        ]
        Resource = "*"
      }
    ]
  })

  tags = merge(local.tags, {
    Name = "${local.name}-backup-kms"
  })
}

resource "aws_kms_alias" "backup" {
  name          = "alias/${local.name}-backup"
  target_key_id = aws_kms_key.backup.key_id
}

# IAM Role for AWS Backup
resource "aws_iam_role" "backup" {
  name = "${local.name}-backup-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "backup.amazonaws.com"
        }
      }
    ]
  })

  tags = local.tags
}

resource "aws_iam_role_policy_attachment" "backup" {
  role       = aws_iam_role.backup.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSBackupServiceRolePolicyForBackup"
}

resource "aws_iam_role_policy_attachment" "backup_restore" {
  role       = aws_iam_role.backup.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSBackupServiceRolePolicyForRestores"
}

# Backup Plan
resource "aws_backup_plan" "main" {
  name = "${local.name}-backup-plan"

  rule {
    rule_name         = "daily_backup"
    target_vault_name = aws_backup_vault.main.name
    schedule          = var.backup_config.backup_schedule

    start_window      = 60  # 1 hour
    completion_window = 300 # 5 hours

    lifecycle {
      cold_storage_after = 30
      delete_after       = var.backup_config.retention_days
    }

    recovery_point_tags = merge(local.tags, {
      BackupType = "daily"
    })

    dynamic "copy_action" {
      for_each = var.backup_config.cross_region_backup ? var.backup_config.backup_regions : []
      content {
        destination_vault_arn = "arn:aws:backup:${copy_action.value}:${data.aws_caller_identity.current.account_id}:backup-vault:${local.name}-backup-vault-${copy_action.value}"
        
        lifecycle {
          cold_storage_after = 30
          delete_after       = var.backup_config.retention_days
        }
      }
    }
  }

  rule {
    rule_name         = "weekly_backup"
    target_vault_name = aws_backup_vault.main.name
    schedule          = "cron(0 3 ? * SUN *)"  # Weekly on Sunday at 3 AM

    start_window      = 60
    completion_window = 300

    lifecycle {
      cold_storage_after = 7
      delete_after       = 90  # Keep weekly backups for 90 days
    }

    recovery_point_tags = merge(local.tags, {
      BackupType = "weekly"
    })
  }

  tags = local.tags
}

# Backup Selection for RDS
resource "aws_backup_selection" "rds" {
  iam_role_arn = aws_iam_role.backup.arn
  name         = "${local.name}-rds-backup-selection"
  plan_id      = aws_backup_plan.main.id

  resources = [
    aws_db_instance.main.arn
  ]

  condition {
    string_equals {
      key   = "aws:ResourceTag/Environment"
      value = var.environment
    }
  }

  tags = local.tags
}

# Backup Selection for EBS Volumes
resource "aws_backup_selection" "ebs" {
  iam_role_arn = aws_iam_role.backup.arn
  name         = "${local.name}-ebs-backup-selection"
  plan_id      = aws_backup_plan.main.id

  resources = ["arn:aws:ec2:*:*:volume/*"]

  condition {
    string_equals {
      key   = "aws:ResourceTag/kubernetes.io/cluster/${local.name}"
      value = "owned"
    }
  }

  tags = local.tags
}

# Cross-region backup vaults (if enabled)
resource "aws_backup_vault" "cross_region" {
  for_each = var.backup_config.cross_region_backup ? toset(var.backup_config.backup_regions) : []
  
  provider = aws.backup_region
  
  name        = "${local.name}-backup-vault-${each.value}"
  kms_key_arn = aws_kms_key.backup_cross_region[each.value].arn

  tags = local.tags
}

resource "aws_kms_key" "backup_cross_region" {
  for_each = var.backup_config.cross_region_backup ? toset(var.backup_config.backup_regions) : {}
  
  provider = aws.backup_region
  
  description             = "KMS key for cross-region backup in ${each.value}"
  deletion_window_in_days = 7

  tags = merge(local.tags, {
    Name = "${local.name}-backup-kms-${each.value}"
  })
}

# S3 Bucket for application data backups
resource "aws_s3_bucket" "backup" {
  bucket = "${local.name}-backup-${random_id.bucket_suffix.hex}"

  tags = local.tags
}

resource "random_id" "bucket_suffix" {
  byte_length = 4
}

resource "aws_s3_bucket_versioning" "backup" {
  bucket = aws_s3_bucket.backup.id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_encryption" "backup" {
  bucket = aws_s3_bucket.backup.id

  server_side_encryption_configuration {
    rule {
      apply_server_side_encryption_by_default {
        kms_master_key_id = aws_kms_key.backup.arn
        sse_algorithm     = "aws:kms"
      }
    }
  }
}

resource "aws_s3_bucket_lifecycle_configuration" "backup" {
  bucket = aws_s3_bucket.backup.id

  rule {
    id     = "backup_lifecycle"
    status = "Enabled"

    transition {
      days          = 30
      storage_class = "STANDARD_IA"
    }

    transition {
      days          = 90
      storage_class = "GLACIER"
    }

    transition {
      days          = 365
      storage_class = "DEEP_ARCHIVE"
    }

    expiration {
      days = var.backup_config.retention_days * 2  # Keep S3 backups longer
    }

    noncurrent_version_expiration {
      noncurrent_days = 30
    }
  }
}

resource "aws_s3_bucket_public_access_block" "backup" {
  bucket = aws_s3_bucket.backup.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# Lambda function for custom backup operations
resource "aws_lambda_function" "backup_orchestrator" {
  filename         = "backup_orchestrator.zip"
  function_name    = "${local.name}-backup-orchestrator"
  role            = aws_iam_role.backup_lambda.arn
  handler         = "index.handler"
  runtime         = "python3.11"
  timeout         = 900  # 15 minutes

  environment {
    variables = {
      CLUSTER_NAME = module.eks.cluster_name
      S3_BUCKET    = aws_s3_bucket.backup.bucket
      ENVIRONMENT  = var.environment
    }
  }

  tags = local.tags

  depends_on = [data.archive_file.backup_orchestrator]
}

# Lambda deployment package
data "archive_file" "backup_orchestrator" {
  type        = "zip"
  output_path = "backup_orchestrator.zip"
  
  source {
    content = templatefile("${path.module}/lambda/backup_orchestrator.py", {
      cluster_name = module.eks.cluster_name
      s3_bucket    = aws_s3_bucket.backup.bucket
    })
    filename = "index.py"
  }
}

# IAM Role for Lambda
resource "aws_iam_role" "backup_lambda" {
  name = "${local.name}-backup-lambda-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "lambda.amazonaws.com"
        }
      }
    ]
  })

  tags = local.tags
}

resource "aws_iam_role_policy" "backup_lambda" {
  name = "${local.name}-backup-lambda-policy"
  role = aws_iam_role.backup_lambda.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents"
        ]
        Resource = "arn:aws:logs:*:*:*"
      },
      {
        Effect = "Allow"
        Action = [
          "eks:DescribeCluster",
          "eks:ListClusters"
        ]
        Resource = "*"
      },
      {
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:PutObject",
          "s3:DeleteObject",
          "s3:ListBucket"
        ]
        Resource = [
          aws_s3_bucket.backup.arn,
          "${aws_s3_bucket.backup.arn}/*"
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "kms:Decrypt",
          "kms:GenerateDataKey"
        ]
        Resource = aws_kms_key.backup.arn
      }
    ]
  })
}

# EventBridge rule for scheduled backups
resource "aws_cloudwatch_event_rule" "backup_schedule" {
  name                = "${local.name}-backup-schedule"
  description         = "Trigger backup orchestrator"
  schedule_expression = var.backup_config.backup_schedule

  tags = local.tags
}

resource "aws_cloudwatch_event_target" "backup_lambda" {
  rule      = aws_cloudwatch_event_rule.backup_schedule.name
  target_id = "BackupOrchestratorTarget"
  arn       = aws_lambda_function.backup_orchestrator.arn
}

resource "aws_lambda_permission" "allow_eventbridge" {
  statement_id  = "AllowExecutionFromEventBridge"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.backup_orchestrator.function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.backup_schedule.arn
}