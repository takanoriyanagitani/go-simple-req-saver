#!/bin/sh

url=http://localhost:8888/api/v1/write
key=test-list-key

test_curl(){
    curl \
        --fail \
        --show-error \
        --silent \
        --output /dev/null \
        --request POST \
        --data hw \
        $url \
        &
    
    curl \
        --fail \
        --show-error \
        --silent \
        --output /dev/null \
        --request POST \
        --data hw2 \
        $url
    
    wait
    
    redis-cli lrange $key 0 0 \
      | tar \
        --list \
        --verbose
    echo;
    
    echo -- body --
    redis-cli lrange $key 0 0 \
      | tar \
        --extract \
        --to-stdout \
        body/body
    echo;
    echo;
    
    echo -- content type --
    redis-cli lrange $key 0 0 \
      | tar \
        --extract \
        --to-stdout \
        header/Content-Type
    echo;
    echo;
    
    echo -- user agent --
    redis-cli lrange $key 0 0 \
      | tar \
        --extract \
        --to-stdout \
        header/User-Agent
    echo;
}

test_avalanche(){
    avalanche \
      --remote-batch-size=2000 \
      --remote-requests-count=100 \
      --remote-url=$url
    keycnt=$( redis-cli llen $key )
    test 300 -eq ${keycnt} && exit 0
    echo Unexpected key cnt: ${keycnt}
    exit 1
}

main(){
    ps h -C req2redis | fgrep --silent req2redis \
      || exec echo req2redis not running.

    which redis-cli | fgrep --silent redis-cli \
      || exec echo redis-cli missing.

    keylen=$( redis-cli llen $key )
    test 0 -lt ${keylen} && exec echo key $key not empty.

    which avalanche | fgrep --silent avalanche && test_avalanche

    which curl | fgrep --silent curl \
      || exec echo curl missing.
    test_curl
}

main
