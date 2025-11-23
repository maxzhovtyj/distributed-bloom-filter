FROM golang:1.25-alpine AS build

WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o server ./cmd/bloomnode/...

FROM alpine:3.20

WORKDIR /app
COPY --from=build /app/server .

EXPOSE 8000 9000

CMD ["./server"]