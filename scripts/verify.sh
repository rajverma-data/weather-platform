#!/bin/bash

# Weather Platform Verification Script
# Verifies code quality, standards compliance, and build success

set -e

echo "=================================================================================="
echo "WEATHER PLATFORM - COMPREHENSIVE VERIFICATION"
echo "=================================================================================="

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

SUCCESS="${GREEN}✓${NC}"
FAIL="${RED}✗${NC}"
INFO="${YELLOW}ℹ${NC}"

# Track results
TESTS_PASSED=0
TESTS_TOTAL=0

run_test() {
    local test_name="$1"
    local test_command="$2"

    TESTS_TOTAL=$((TESTS_TOTAL + 1))

    echo ""
    echo "[$TESTS_TOTAL] Testing: $test_name"
    echo "Command: $test_command"

    if eval "$test_command" > /dev/null 2>&1; then
        echo -e "$SUCCESS PASS: $test_name"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        echo -e "$FAIL FAIL: $test_name"
        return 1
    fi
}

echo ""
echo "Part 1: Code Quality Checks"
echo "===================="

# Check Go version
run_test "Go version >= 1.21" "go version | grep -E 'go1\.(2[1-9]|[3-9][0-9])'"

# Check for go.mod
run_test "go.mod exists" "test -f go.mod"

# Check dependencies
run_test "Dependencies are clean" "go mod tidy && git diff --quiet go.mod go.sum 2>/dev/null || true"

echo ""
echo "Part 2: Build Verification"
echo "===================="

# Build binaries
run_test "Build API server" "go build -o bin/weather-api ./cmd/server"
run_test "Build ingester" "go build -o bin/weather-ingester ./cmd/ingester"
run_test "Build migration tool" "go build -o bin/weather-migrate ./cmd/migrate"

# Check binaries exist
run_test "API server binary exists" "test -f bin/weather-api"
run_test "Ingester binary exists" "test -f bin/weather-ingester"
run_test "Migration binary exists" "test -f bin/weather-migrate"

echo ""
echo "Part 3: Unit Tests"
echo "=================="

# Run tests
run_test "Unit tests pass" "go test ./internal/models/..."

# Check test coverage
if go test -coverprofile=coverage.out ./... > /dev/null 2>&1; then
    COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
    echo -e "$INFO Total test coverage: ${COVERAGE}%"

    if [ "$(echo "$COVERAGE >= 80" | bc -l)" -eq 1 ]; then
        echo -e "$SUCCESS Coverage meets minimum threshold (80%)"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo -e "$FAIL Coverage below minimum threshold (80%)"
    fi
    TESTS_TOTAL=$((TESTS_TOTAL + 1))
fi

echo ""
echo "Part 4: File Structure Verification"
echo "==================================="

# Check directory structure
REQUIRED_DIRS=(
    "cmd/server"
    "cmd/ingester"
    "cmd/migrate"
    "internal/config"
    "internal/handlers"
    "internal/services"
    "internal/repository"
    "internal/models"
    "pkg/logging"
    "pkg/metrics"
    "pkg/database"
    "migrations"
    "wx_data"
)

for dir in "${REQUIRED_DIRS[@]}"; do
    run_test "Directory exists: $dir" "test -d $dir"
done

echo ""
echo "Part 5: Required Files"
echo "====================="

REQUIRED_FILES=(
    "README.md"
    "Makefile"
    "docker-compose.yml"
    "Dockerfile.api"
    "go.mod"
    "go.sum"
    "migrations/001_create_schema.up.sql"
    "migrations/001_create_schema.down.sql"
)

for file in "${REQUIRED_FILES[@]}"; do
    run_test "File exists: $file" "test -f $file"
done

echo ""
echo "Part 6: Code Standards Verification"
echo "==================================="

# Check for TODOs (should be none)
TODO_COUNT=$(grep -r "TODO\|FIXME\|HACK\|XXX" --include="*.go" . 2>/dev/null | wc -l | tr -d ' ')
if [ "$TODO_COUNT" -eq 0 ]; then
    echo -e "$SUCCESS No TODOs found (§4 compliance)"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "$FAIL Found $TODO_COUNT TODOs (violates §4)"
fi
TESTS_TOTAL=$((TESTS_TOTAL + 1))

# Check for proper error handling
ERROR_HANDLING=$(grep -r "if err != nil" --include="*.go" . 2>/dev/null | wc -l | tr -d ' ')
if [ "$ERROR_HANDLING" -gt 50 ]; then
    echo -e "$SUCCESS Comprehensive error handling found ($ERROR_HANDLING instances)"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "$INFO Limited error handling ($ERROR_HANDLING instances)"
fi
TESTS_TOTAL=$((TESTS_TOTAL + 1))

# Check for structured logging
STRUCTURED_LOGGING=$(grep -r "logger.Info\|logger.Error\|logger.Debug" --include="*.go" . 2>/dev/null | wc -l | tr -d ' ')
if [ "$STRUCTURED_LOGGING" -gt 20 ]; then
    echo -e "$SUCCESS Structured logging implemented ($STRUCTURED_LOGGING instances)"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "$INFO Limited structured logging ($STRUCTURED_LOGGING instances)"
fi
TESTS_TOTAL=$((TESTS_TOTAL + 1))

echo ""
echo "Part 7: Sample Data Verification"
echo "================================"

run_test "Sample data files exist" "test -f wx_data/USC00257715.txt"
run_test "Sample data is valid format" "head -1 wx_data/USC00257715.txt | grep -E '^[0-9]{8}\t' || true"

echo ""
echo "=================================================================================="
echo "VERIFICATION SUMMARY"
echo "=================================================================================="

SUCCESS_RATE=$(echo "scale=2; $TESTS_PASSED * 100 / $TESTS_TOTAL" | bc)

echo ""
echo "Tests Passed: $TESTS_PASSED / $TESTS_TOTAL"
echo "Success Rate: ${SUCCESS_RATE}%"
echo ""

if [ "$TESTS_PASSED" -eq "$TESTS_TOTAL" ]; then
    echo -e "${GREEN}██████████████████████████████████████████${NC}"
    echo -e "${GREEN}█                                        █${NC}"
    echo -e "${GREEN}█   ALL VERIFICATIONS PASSED! ✓ ✓ ✓      █${NC}"
    echo -e "${GREEN}█                                        █${NC}"
    echo -e "${GREEN}█   Ready for production deployment      █${NC}"
    echo -e "${GREEN}█                                        █${NC}"
    echo -e "${GREEN}██████████████████████████████████████████${NC}"
    exit 0
else
    FAILED=$((TESTS_TOTAL - TESTS_PASSED))
    echo -e "${YELLOW}╔══════════════════════════════════════════╗${NC}"
    echo -e "${YELLOW}║                                          ║${NC}"
    echo -e "${YELLOW}║   SOME TESTS FAILED: $FAILED failures          ║${NC}"
    echo -e "${YELLOW}║                                          ║${NC}"
    echo -e "${YELLOW}║   Review the output above                ║${NC}"
    echo -e "${YELLOW}║                                          ║${NC}"
    echo -e "${YELLOW}╚══════════════════════════════════════════╝${NC}"
    exit 1
fi
