#!/bin/bash

echo "=== Checking Hot Articles ==="
docker exec redis redis-cli -p 6379 ZREVRANGE hot:articles 0 9 WITHSCORES

echo -e "\n=== Checking Partition Keys ==="
for i in 0 1 2; do
    count=$(docker exec redis redis-cli -p 6379 ZCARD hot:articles:p$i)
    echo "Partition $i: $count articles"
done

echo -e "\n=== Checking Merge Lock ==="
docker exec redis redis-cli -p 6379 GET hot:merge:lock
