#!/bin/sh

# Use the PORT environment variable (default is 8080)
PORT=${PORT:-8080}

# Use the LOG_LEVEL environment variable (default is 0)
LOG_LEVEL=${LOG_LEVEL:-0}

# Run the application with the specified port
echo "Starting app on PORT($PORT) with LOG_LEVEL($LOG_LEVEL)"
./main
