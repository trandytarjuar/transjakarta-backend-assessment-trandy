FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG TARGET=api
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /app ./cmd/${TARGET}

FROM alpine:3.22
RUN addgroup -S app && adduser -S app -G app
COPY --from=build /app /app
USER app
ENTRYPOINT ["/app"]
