#!/bin/bash

# PNJ Anonymous Bot - Quality Gate Script
# This script runs various checks to ensure code quality before pushing.

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}🚀 Starting Quality Gate...${NC}"

# 1. Formatting check
echo -e "\n${YELLOW}1. checking formatting (go fmt)...${NC}"
GOFMT_FILES=$(gofmt -l .)
if [ -n "$GOFMT_FILES" ]; then
    echo -e "${RED}❌ The following files are not formatted correctly:${NC}"
    echo "$GOFMT_FILES"
    echo -e "${YELLOW}💡 Run 'go fmt ./...' to fix formatting.${NC}"
    exit 1
else
    echo -e "${GREEN}✅ Formatting is perfect.${NC}"
fi

# 2. Dependency check
echo -e "\n${YELLOW}2. Checking dependencies (go mod tidy)...${NC}"
go mod tidy
if [[ -n $(git status --porcelain go.mod go.sum) ]]; then
    echo -e "${RED}❌ go.mod or go.sum are not tidy.${NC}"
    echo -e "${YELLOW}💡 Run 'go mod tidy' and commit the changes before pushing.${NC}"
    exit 1
else
    echo -e "${GREEN}✅ Dependencies are tidy.${NC}"
fi

# 3. Static Analysis (go vet)
echo -e "\n${YELLOW}3. Running go vet...${NC}"
go vet ./...
echo -e "${GREEN}✅ go vet passed.${NC}"

# 4. Linting (golangci-lint)
if command -v golangci-lint &> /dev/null; then
    echo -e "\n${YELLOW}4. Running golangci-lint...${NC}"
    golangci-lint run ./...
    echo -e "${GREEN}✅ golangci-lint passed.${NC}"
else
    echo -e "\n${YELLOW}4. golangci-lint not found, skipping...${NC}"
fi

# 5. Unit Tests
echo -e "\n${YELLOW}5. Running unit tests...${NC}"
go test -short ./...
echo -e "${GREEN}✅ All tests passed.${NC}"

# 6. Build Check
echo -e "\n${YELLOW}6. Verifying build...${NC}"
make build
echo -e "${GREEN}✅ Build successful.${NC}"

echo -e "\n${GREEN}✨ Quality Gate Passed! You are ready to push. ✨${NC}"
