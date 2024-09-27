# Stage 1: Build stage
FROM golang:1.22-alpine AS build-stage

# Create a working directory inside the image
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy everything from root directory into WORKDIR
COPY . .

# Build application
RUN go build -o /main

# Stage 2: Final stage
FROM alpine:edge AS final-stage

WORKDIR /

# Copy the conf directory
COPY /conf/config.yaml /conf/config.yaml

# Copy the binary from the build stage
COPY --from=build-stage /main .

# Copy the startup script
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 8080

CMD ["/entrypoint.sh"]
