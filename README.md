# goreverseproxy

This reverse proxy forwards all incoming requests to their intended destinations while offering additional functionality, such as blocking requests based on predefined rules (limited to GET requests). It also provides logging capabilities by recording all incoming requests, including both headers and body content.

## Useful Commands for Development
### 1. run test for all unit tests and generate report
```sh
$ go test -v -coverprofile=cover.out ./...`
```

### 2. check coverage report
```sh
$ go tool cover -html=cover.out`
```

### 3. run reverse proxy server in local environment
```sh
$ go run main.go -port=8080 -log_level=-4`
```

### 4. build docker image
```sh
$ docker build -t goreverseproxy:latest  .
```

### 5. run docker image for debugging
```sh
$ docker run -it --rm -e PORT=8080 -e LOG_LEVEL=-4 -p 8080:8080 goreverseproxy:latest
```
