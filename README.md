# tgbot-lambda

1. Set up the github workflow of building & deploying to lambda according to [this tutorial](https://blog.jakoblind.no/aws-lambda-github-actions/)
2. The Go lambda program should consist a `main` function and the handler name in aws lambda web console should be the executable name of your program. On invoking, lambda runtime will invoke the `main` function of your program.
   checkout this demo to learn how do we write a main.go for lambda: https://docs.aws.amazon.com/code-samples/latest/catalog/lambda_functions-blank-go-function-main.go.html
3. Proxy your lambda function through API Gateway: https://docs.aws.amazon.com/lambda/latest/dg/services-apigateway-blueprint.html
4. Set the API gateway endpoint as the webhook of your telegram bot.
