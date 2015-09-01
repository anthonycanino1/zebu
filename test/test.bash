#!/usr/bin/env bash

# Copyright 2015 The Zebu Authors. All rights reserved.

if [ ! -f check.rb ]; then
  echo "test.bash must be run from $ZEBUROOT/test" 1>&2
  exit 1
fi

if ! hash zebu 2>/dev/null; then
  echo "zebu not found in path"
  exit 1
fi

fails=0
passed=0
for f in *.zb; do
  result=$(./check.rb $f)
  status=$?
  if [[ $status != 0 ]]; then
    printf "FAIL %10s\n" $f
    printf "exit status %d\n" $status
    printf "%s\n" "$result"
    fails=$(($fails+1))
  else
    printf "OK %10s\n" $f
    passed=$(($passed+1))
  fi
done

echo "testing completed with $passed passes and $fails failures"
if [[ $fails != 0 ]]; then
  exit 1
else
  exit 0
fi
