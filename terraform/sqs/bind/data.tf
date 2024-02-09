data "aws_iam_policy_document" "user_policy" {
  statement {
    sid = "sqsAccess"
    actions = [
      "sqs:*", // TODO: we should minimise the permissions, but for now we are just trying to get things working
    ]
    resources = [
      var.arn
    ]
  }
}