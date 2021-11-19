# tgbot-lambda

# Todo

[x] Support searching for groups by keyword
[x] group groups by tags and categories
[ ] rate the group periodically, then sort them during search recalling

# Set up webhook

1. Set up the github workflow of building & deploying to lambda according to [this tutorial](https://blog.jakoblind.no/aws-lambda-github-actions/)
2. The Go lambda program should consist a `main` function and the handler name in aws lambda web console should be the executable name of your program. On invoking, lambda runtime will invoke the `main` function of your program.
   checkout this demo to learn how do we write a main.go for lambda: https://docs.aws.amazon.com/code-samples/latest/catalog/lambda_functions-blank-go-function-main.go.html
3. Proxy your lambda function through API Gateway.
4. Set the API gateway endpoint as the webhook of your telegram bot.

   easy way:
   ```
   curl -X "POST" "https://api.telegram.org/bot<token>/setWebhook"  -d '{"url": "https://91wg5oku56.execute-api.ap-east-1.amazonaws.com/default/bot<token>"}'  -H 'Content-Type: application/json; charset=utf-8'
   ```

# Issues during developing

1. My bot works on webhook mode and once a time it keeps receiving the update message enormous times!

   This is caused of the failed responding of the webhook event, telegram bot API keeps updating us the event if it doesn't receive a "ok"(HTTP 200 OK) response.

    My initial buggy code shows as following where, the `update.Message.Text` is empty for an non-Message Update, the `bot.Send(msg)` will then failed because it constructs an empty message.

    ```
	if update.Message != nil { // ignore any non-Message Updates
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
		msg.ReplyToMessageID = update.Message.MessageID
		_, err := bot.Send(msg)
		if err != nil {
			log.Fatalln(err)
		}
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
	}, nil
    ```

    The error message show as:

    ```
    sendMessage resp: {"ok":false,"error_code":400,"description":"Bad Request: message text is empty"}
    ```
    And the `log.Fatalln(err)` causes the handling process exits before sending 200 OK to telegram bot API. The fix for it is fairly simple, just substitute the `log.Fatalln()` with `log.Println()`.

2. Lambda is unable to call dynamoDB due to the lack of permissions

   Follow [this](https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies_examples_lambda-access-dynamodb.html) to grant permissions to lambda.
