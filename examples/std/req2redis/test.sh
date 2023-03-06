#!/bin/sh

curl \
	--fail \
	--show-error \
	--silent \
	--output /dev/null \
	--request POST \
	--data hw \
	http://localhost:8888 \
	&

curl \
	--fail \
	--show-error \
	--silent \
	--output /dev/null \
	--request POST \
	--data hw2 \
	http://localhost:8888

wait

redis-cli lrange test-list-key 0 0 \
  | tar \
    --list \
	--verbose
echo;

echo -- body --
redis-cli lrange test-list-key 0 0 \
  | tar \
    --extract \
	--to-stdout \
	body/body
echo;
echo;

echo -- content type --
redis-cli lrange test-list-key 0 0 \
  | tar \
    --extract \
	--to-stdout \
	header/Content-Type
echo;
echo;

echo -- user agent --
redis-cli lrange test-list-key 0 0 \
  | tar \
    --extract \
	--to-stdout \
	header/User-Agent
echo;
