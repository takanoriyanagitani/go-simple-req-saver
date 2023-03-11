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
    exit 0
}

test_avalanche(){
    avalanche \
      --remote-batch-size=2000 \
      --remote-requests-count=100 \
      --remote-url=$url
    keycnt=$( redis-cli llen $key )
    test 300 -eq ${keycnt} || exec sh -c "echo Unexpected key cnt: ${keycnt}; exit 1"
    exit 0
}

test_default(){
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

test_redis(){
    which redis-cli | fgrep --silent redis-cli || exec echo redis-cli missing.

    local typ=$(
        redis-cli lrange $key 0 0 |
            tar \
            --extract \
            --to-stdout \
            header/Content-Type
    )
    test "application/x-protobuf" = "$typ" || exec echo Unexpected content type

    local encoding=$(
        redis-cli lrange $key 0 0 |
            tar \
            --extract \
            --to-stdout \
            header/Content-Encoding
    )
    test "snappy" = "$encoding" || exec echo Unexpected content encoding

    which snappytool | fgrep --silent snappytool || exec echo snappytool missing.

    local size=$(
        redis-cli lrange $key 0 0 |
            tar \
            --extract \
            --to-stdout \
            body/body \
            | snappytool -d \
            | wc --bytes
    )

    test 0 -lt $size || exec echo empty data.
    echo decoded bytes: $size
    exit 0
}

main(){
    local typ="$1"
    local arg="$2"

    test 0 = ${#typ} && test 0 = ${#arg} && test_default

    case "$typ" in
        avalanche)
            test_avalanche
            ;;
        curl)
            test_curl
            ;;
        redis)
            test_redis
            ;;
        *)
            exec "$0"
            ;;
    esac
}

main "$1" "$2"
