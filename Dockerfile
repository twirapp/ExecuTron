FROM golang:1.24.2-alpine as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -ldflags="-s -w" -o /executron ./cmd/main.go

FROM alpine:3.18
COPY --from=builder /executron /executron
ENTRYPOINT [ "/executron" ]
