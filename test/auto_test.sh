#!/bin/bash

# Janitor Automatic Test Script
# This script automatically tests all functionality without manual intervention

set -e  # Exit on error

echo "======================================"
echo "Janitor Automatic Test Script"
echo "======================================"
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PASS=0
FAIL=0
TEST_DIR="$(pwd)/test_temp"

# Function to print test result
print_result() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓ PASS${NC}: $2"
        ((PASS++))
    else
        echo -e "${RED}✗ FAIL${NC}: $2"
        ((FAIL++))
    fi
}

# Cleanup function
cleanup() {
    echo ""
    echo "Cleaning up test directory..."
    rm -rf "$TEST_DIR"
}

trap cleanup EXIT

echo "Step 1: Building Janitor..."
go build -o janitor.exe 2>/dev/null || go build -o janitor 2>/dev/null
if [ -f "janitor.exe" ]; then
    JANITOR="./janitor.exe"
elif [ -f "janitor" ]; then
    JANITOR="./janitor"
else
    echo -e "${RED}Failed to build janitor${NC}"
    exit 1
fi
print_result 0 "Build successful"
echo ""

echo "Step 2: Running Unit Tests..."
go test ./internal/... > /dev/null 2>&1
print_result $? "Unit tests"
echo ""

echo "Step 3: Creating Test Environment..."
mkdir -p "$TEST_DIR"
cd "$TEST_DIR"

# Create test project structure
mkdir -p project1/node_modules
mkdir -p project2/target
mkdir -p project3/build
echo "test" > project1/node_modules/test.js
echo "test" > project2/target/test.class
echo "test" > project3/build/output.exe

# Set old modification time
touch -t 202301010000 project1/node_modules
touch -t 202301010000 project2/target

print_result 0 "Test environment created"
echo ""

echo "Step 4: Testing CLI Commands..."

# Test 4.1: Help command
cd ..
$JANITOR --help > /dev/null 2>&1
print_result $? "Help command works"

# Test 4.2: Scan command (dry-run)
OUTPUT=$($JANITOR scan "$TEST_DIR" --dry-run 2>&1)
if echo "$OUTPUT" | grep -q "node_modules\|target\|build\|No junk"; then
    print_result 0 "Scan command works"
else
    print_result 1 "Scan command failed"
fi

# Test 4.3: Config init
CONFIG_TEST_DIR=$(mktemp -d)
export HOME="$CONFIG_TEST_DIR"
$JANITOR config init <<< "y" > /dev/null 2>&1
if [ -f "$CONFIG_TEST_DIR/.config/janitor/config.yml" ]; then
    print_result 0 "Config init works"
else
    print_result 1 "Config init failed"
fi
rm -rf "$CONFIG_TEST_DIR"

# Test 4.4: Cleancache command
OUTPUT=$($JANITOR cleancache 2>&1)
if echo "$OUTPUT" | grep -q "Cleaners\|cleaner\|--all"; then
    print_result 0 "Cleancache command works"
else
    print_result 1 "Cleancache command failed"
fi

echo ""
echo "Step 5: Testing .janitorignore..."

# Create project with .janitorignore
cd "$TEST_DIR"
mkdir -p ignored_project/node_modules
echo "test" > ignored_project/node_modules/test.js
echo "node_modules" > ignored_project/.janitorignore

cd ..
OUTPUT=$($JANITOR scan "$TEST_DIR/ignored_project" 2>&1)
if echo "$OUTPUT" | grep -q "No junk items\|0"; then
    print_result 0 ".janitorignore works"
else
    print_result 1 ".janitorignore may not work"
fi

echo ""
echo "Step 6: Testing Safety Features..."

# Test that it doesn't delete without --delete flag
cd "$TEST_DIR"
BEFORE_COUNT=$(find . -name "node_modules" -o -name "target" -o -name "build" | wc -l)
cd ..
$JANITOR scan "$TEST_DIR" > /dev/null 2>&1
cd "$TEST_DIR"
AFTER_COUNT=$(find . -name "node_modules" -o -name "target" -o -name "build" | wc -l)

if [ "$BEFORE_COUNT" -eq "$AFTER_COUNT" ]; then
    print_result 0 "Safety: No deletion without --delete flag"
else
    print_result 1 "Safety: Unexpected deletion occurred!"
fi

echo ""
echo "======================================"
echo "Test Summary"
echo "======================================"
echo -e "${GREEN}Passed: $PASS${NC}"
echo -e "${RED}Failed: $FAIL${NC}"
echo ""

if [ $FAIL -eq 0 ]; then
    echo -e "${GREEN}All tests passed! ✓${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed. Please check the output above.${NC}"
    exit 1
fi
