# Telegram-bot-connector

## Installation


### Docker build
```
docker build -t telegram-bot-connector .
```

### Docker run
```
docker run \
  -e TELEGRAM_BOT_TOKEN=[your token] \
  -e REDIS_ADDR [redis_addr] \
  -e REDIS_PASSWORD [redis_password_if_set] \
  -e INBOX tg_inbox_1 \
  -e OUTBOX tg_outbox_1 \
  telegram-bot-connector
```



## Telegram Stripe Payment

https://core.telegram.org/bots/payments#introducing-payments-2-0

### 1. setting account

Use the /mybots command in the chat with BotFather and choose the @merchantbot that will be offering goods or services.
Go to Bot Settings > Payments.
Choose a provider, and you will be redirected to the relevant bot.
Enter the required details so that the payments provider is connected successfully, go back to the chat with Botfather.
The message will now show available providers. Each will have a name, a token, and the date the provider was connected.
You will use the token when working with the Bot API.


### 2. receive PreCheckoutQuery and send PreCheckoutConfig

### 3. if user checkout success, receive as message contains SuccessfulPayment


refer: https://github.com/tingwei628/pgo/blob/5d8be8774c17fd6aee378cf670ebd79ddb2ca3a5/tgbotpay/tgbotpay.go