FROM golang:1.18-alpine AS builder
RUN apk update && apk add make
WORKDIR /build
ADD . .
RUN make build

FROM alpine

# Add ffmpeg
RUN apk update
RUN apk add ffmpeg

COPY --from=builder /build/telegram-bot-connector /bin/telegram-bot-connector
RUN chmod +x /bin/telegram-bot-connector

ENTRYPOINT ["/bin/telegram-bot-connector"]