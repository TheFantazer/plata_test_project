FROM golang:1.26-alpine AS build

WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app ./cmd/app

FROM alpine:3.23
RUN apk add --no-cache ca-certificates

COPY --from=build /app /app
EXPOSE 8080
CMD ["/app"]