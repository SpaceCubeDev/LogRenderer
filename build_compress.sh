#!/bin/bash -eu

VERSION="2.1.2"
OUTPUT_FILENAME="LogRenderer-$VERSION"
OUTPUT_DIR="compiled"

echo "Building $OUTPUT_FILENAME ..."
go build -o $OUTPUT_DIR/$OUTPUT_FILENAME ./src && strip $OUTPUT_DIR/$OUTPUT_FILENAME && xz $OUTPUT_DIR/$OUTPUT_FILENAME
echo "Done !"
