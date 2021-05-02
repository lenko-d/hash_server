#!/bin/bash

NUM_ITERATIONS=100000

ab -T 'application/x-www-form-urlencoded'  -n $NUM_ITERATIONS -p post.data http://localhost:8080/hash/  &
ab -n $NUM_ITERATIONS  http://localhost:8080/hash/1  &
ab -n $NUM_ITERATIONS  http://localhost:8080/stats  &