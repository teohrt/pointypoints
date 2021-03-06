module "newSession_lambda" {
  source = "./endpoint"

  aws_region = var.aws_region

  api_id = aws_apigatewayv2_api.pointing.id
  name   = "newSession"
  route  = "newSession"

  policy = data.aws_iam_policy_document.session_modifying_lambda_policy.json

  lambda_env = local.session_modifying_lambda_env
}
