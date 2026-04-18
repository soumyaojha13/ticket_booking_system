FROM golang:1.25-alpine AS build
WORKDIR /src

RUN apk add --no-cache ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/inventory ./inventory-service/cmd

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /out/inventory /app/inventory

EXPOSE 50051
ENV INVENTORY_PORT=50051

ENTRYPOINT ["/app/inventory"]

