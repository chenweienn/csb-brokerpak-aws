module "sqs" {
  source = "../community"

  create              = true
  create_queue_policy = false

  create_dlq              = false
  create_dlq_queue_policy = false
  redrive_policy          = ""
  redrive_allow_policy    = ""

  fifo_queue = false
}
