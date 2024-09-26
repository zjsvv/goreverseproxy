# goreverseproxy

# run test for all unit tests and generate report
`$ go test -v -coverprofile=cover.out ./...`

# check coverage report
`$ go tool cover -html=cover.out`

# run reverse proxy server
`$ go run main.go -log_level=0`
