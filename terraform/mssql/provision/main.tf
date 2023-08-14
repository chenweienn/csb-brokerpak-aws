resource "aws_security_group" "rds-sg" {
  count  = length(var.rds_vpc_security_group_ids) == 0 ? 1 : 0
  name   = format("%s-sg", var.instance_name)
  vpc_id = local.vpc_id
}

resource "aws_db_subnet_group" "rds-private-subnet" {
  count      = length(var.rds_subnet_group) == 0 ? 1 : 0
  name       = format("%s-p-sn", var.instance_name)
  subnet_ids = local.subnet_ids
}

resource "aws_security_group_rule" "rds_inbound_access" {
  count             = length(var.rds_vpc_security_group_ids) == 0 ? 1 : 0
  from_port         = local.port
  protocol          = "tcp"
  security_group_id = aws_security_group.rds-sg[0].id
  to_port           = local.port
  type              = "ingress"
  cidr_blocks       = ["0.0.0.0/0"]
}

resource "random_string" "username" {
  length  = 16
  special = false
  numeric = false
}

resource "random_password" "password" {
  length  = 32
  special = false
  // https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/CHAP_Limits.html#RDS_Limits.Constraints
  override_special = "~_-."
}

resource "aws_db_instance" "db_instance" {
  license_model          = "license-included"
  allocated_storage      = var.storage_gb
  max_allocated_storage  = var.max_allocated_storage
  engine                 = var.engine
  engine_version         = var.mssql_version
  instance_class         = var.instance_class
  identifier             = var.instance_name
  db_name                = null # Otherwise: Error: InvalidParameterValue: DBName must be null for engine: sqlserver-xx
  username               = random_string.username.result
  password               = random_password.password.result
  tags                   = var.labels
  vpc_security_group_ids = local.security_group_ids
  db_subnet_group_name   = local.subnet_group_name
  apply_immediately      = true
  storage_encrypted      = var.storage_encrypted
  kms_key_id             = var.kms_key_id == "" ? null : var.kms_key_id
  skip_final_snapshot    = true
  storage_type           = var.storage_type
  iops                   = contains(local.valid_storage_types_for_iops, var.storage_type) ? var.iops : null

  lifecycle {
    prevent_destroy = true
  }

  timeouts {
    create = "60m"
  }
}