#!/bin/bash -e

VERSION="2.2.4"
BINARY_FILENAME="LogRenderer-$VERSION"
OUTPUT_DIR="compiled"

if [ $# -ge 1 ] && [ -n "$1" ] && [ "$1" == "-h" ] || [ "$1" == "--help" ]; then
    echo "$BINARY_FILENAME build help:"
    echo "  -h, --help:"
    echo "      Display this help"
    echo "  --dynamic:"
    echo "      Make the binary dynamically linked"
    exit 0
fi

STATIC=" -extldflags=-static"
TAGS=" -tags osusergo,netgo"
STATIC_MSG=" statically"

if [ $# -ge 1 ] && [ -n "$1" ] && [ "$1" == "--dynamic" ]; then
    # remove static 'modifiers'
    STATIC=""
    TAGS=""
    STATIC_MSG=""
fi

echo "Building the app (v$VERSION)$STATIC_MSG ..."

(cd src && go build -ldflags="-X 'main.version=$VERSION'$STATIC"$TAGS -o $BINARY_FILENAME && strip $BINARY_FILENAME && xz $BINARY_FILENAME && mv "$BINARY_FILENAME.xz" ../$OUTPUT_DIR)

echo "Successfully exported $BINARY_FILENAME to $OUTPUT_DIR/$BINARY_FILENAME.xz !"
