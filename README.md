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

### 3. run reverse proxy server
```sh
$ go run main.go -log_level=0`
```
