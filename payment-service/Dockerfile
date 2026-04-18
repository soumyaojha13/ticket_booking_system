FROM golang:1.25-alpine AS build
WORKDIR /src

RUN apk add --no-cache ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/payment ./payment-service/cmd

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /out/payment /app/payment

EXPOSE 50052
ENV PAYMENT_PORT=50052

ENTRYPOINT ["/app/payment"]

