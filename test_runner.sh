#!/bin/bash

# Test runner script for the failsafe package

echo "Running failsafe package tests..."
echo "=================================="

# Run tests with coverage
echo "Running tests with coverage..."
go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

# Display coverage report
if [ -f coverage.out ]; then
    echo ""
    echo "Coverage Report:"
    echo "================"
    go tool cover -func=coverage.out
    
    # Generate HTML coverage report
    echo ""
    echo "Generating HTML coverage report..."
    go tool cover -html=coverage.out -o coverage.html
    echo "HTML coverage report generated: coverage.html"
fi

# Run benchmarks
echo ""
echo "Running benchmarks..."
echo "===================="
go test -bench=. -benchmem ./...

echo ""
echo "Test run complete!"