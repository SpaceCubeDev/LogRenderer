#!/bin/bash

echo "Updating project dependencies ..."
go get -u ./... && go mod tidy
echo "Update successfully completed !"