FROM golang:alpine AS builder

WORKDIR /build
ADD .env /build/
COPY . .
RUN go mod download
RUN apk add build-base
RUN go build -o bot cmd/main.go

FROM alpine

WORKDIR /

COPY --from=builder /build .
CMD [ "./bot" ]