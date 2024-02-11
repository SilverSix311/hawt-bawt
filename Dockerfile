# Stage 0 - Build
FROM golang:1.21.3-alpine

WORKDIR $GOPATH/src/github.com/thegrandpackard/palworld-discord-bot
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /palworld-discord-bot

# Stage 1 - Run
FROM alpine:3.19.0
COPY --from=0 /palworld-discord-bot /palworld-discord-bot

EXPOSE 2112
CMD ["/palworld-discord-bot"]