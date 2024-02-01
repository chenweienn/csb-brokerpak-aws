module "sqs" {
  source = "../community"

  name = var.instance_name
  fifo_queue = var.fifo_queue

  # Properties below don't need to be configurable yet.
  # Strictly speaking, they aren't required to work
  # since the community module uses these same defaults.
  #
  # However, they are important enough to be explicit
  # about them to ensure that we won't be affected if
  # the default values change in new versions.
  create              = true
  create_queue_policy = false

  create_dlq              = false
  create_dlq_queue_policy = false
  redrive_policy          = {}
  redrive_allow_policy    = {}
}
