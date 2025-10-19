#!/bin/bash

echo "Package Coverage Report"
echo "======================"

./go/bin/go list ./... | while read pkg; do
  result=$(./go/bin/go test -cover $pkg 2>&1 | grep "coverage:")
  if [ -n "$result" ]; then
    coverage=$(echo $result | grep -oP 'coverage: \K[0-9.]+')
    echo "$pkg ${coverage}%"
  fi
done | sort -t' ' -k2 -n
