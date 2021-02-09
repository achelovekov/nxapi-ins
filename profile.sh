#!/bin/bash

for i in {1..10}; do
    go run po-new.go >> po-new.go.out.$i
done