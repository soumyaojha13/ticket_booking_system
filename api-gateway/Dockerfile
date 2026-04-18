FROM golang:1.25-alpine AS build
WORKDIR /src

RUN apk add --no-cache ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api-gateway ./api-gateway/cmd

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /out/api-gateway /app/api-gateway

EXPOSE 8080
ENV API_GATEWAY_PORT=8080

ENTRYPOINT ["/app/api-gateway"]

