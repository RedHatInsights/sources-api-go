#!/bin/bash
set -e

# Re-encryption runner script for Sources authentication credentials
# This script helps run the re-encryption process safely

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BINARY="${SCRIPT_DIR}/reencrypt"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Sources Credential Re-encryption Tool${NC}"
echo "========================================="

# Check if both keys are set
if [ -z "$ENCRYPTION_KEY" ]; then
    echo -e "${RED}ERROR: ENCRYPTION_KEY is not set${NC}"
    exit 1
fi

if [ -z "$ENCRYPTION_KEY_OLD" ]; then
    echo -e "${YELLOW}WARNING: ENCRYPTION_KEY_OLD is not set${NC}"
    echo "Re-encryption requires the old key to decrypt existing data."
    echo "If you're sure all data is already encrypted with the new key, you can proceed."
    read -p "Continue anyway? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Build the binary
echo -e "${GREEN}Building re-encryption tool...${NC}"
go build -o "$BINARY" "$SCRIPT_DIR/main.go"

if [ ! -f "$BINARY" ]; then
    echo -e "${RED}ERROR: Failed to build re-encryption tool${NC}"
    exit 1
fi

echo -e "${GREEN}Build successful${NC}"
echo ""

# Check command line arguments
DRY_RUN_FLAG=""
BATCH_SIZE="100"
WORKERS="5"

while [[ $# -gt 0 ]]; do
    case $1 in
        --dry-run)
            DRY_RUN_FLAG="--dry-run"
            shift
            ;;
        --batch-size)
            BATCH_SIZE="$2"
            shift 2
            ;;
        --workers)
            WORKERS="$2"
            shift 2
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            echo "Usage: $0 [--dry-run] [--batch-size N] [--workers N]"
            exit 1
            ;;
    esac
done

if [ -n "$DRY_RUN_FLAG" ]; then
    echo -e "${YELLOW}Running in DRY-RUN mode (no changes will be made)${NC}"
else
    echo -e "${RED}WARNING: This will re-encrypt all authentication passwords in the database${NC}"
    echo "Make sure you have:"
    echo "  1. A recent database backup"
    echo "  2. Both ENCRYPTION_KEY and ENCRYPTION_KEY_OLD set correctly"
    echo "  3. Tested this in a lower environment first"
    echo ""
    read -p "Are you sure you want to proceed? (yes/no) " -r
    echo
    if [[ ! $REPLY =~ ^yes$ ]]; then
        echo "Aborted."
        exit 1
    fi
fi

echo ""
echo -e "${GREEN}Starting re-encryption...${NC}"
echo "Batch size: $BATCH_SIZE"
echo "Workers: $WORKERS"
echo ""

# Run the re-encryption
"$BINARY" $DRY_RUN_FLAG --batch-size="$BATCH_SIZE" --workers="$WORKERS"

EXIT_CODE=$?

if [ $EXIT_CODE -eq 0 ]; then
    echo ""
    echo -e "${GREEN}Re-encryption completed successfully!${NC}"

    if [ -z "$DRY_RUN_FLAG" ]; then
        echo ""
        echo "Next steps:"
        echo "  1. Verify no fallback key usage in logs:"
        echo "     grep 'fallback key' /var/log/sources-api.log"
        echo "  2. Check re-encryption status in database:"
        echo "     SELECT COUNT(*) FROM reencryption_completed;"
        echo "  3. After 24-48 hours of monitoring, proceed with Phase 3"
    fi
else
    echo ""
    echo -e "${RED}Re-encryption failed with exit code $EXIT_CODE${NC}"
    echo "Check the logs above for details"
    exit $EXIT_CODE
fi
