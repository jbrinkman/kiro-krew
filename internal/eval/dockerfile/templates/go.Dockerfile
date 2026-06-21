# Install Go
FROM golang:1.21-alpine AS go-builder
RUN apk add --no-cache git