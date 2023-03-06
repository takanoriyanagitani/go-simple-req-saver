package main

import (
	"github.com/gomodule/redigo/redis"

	saver "github.com/takanoriyanagitani/go-simple-req-saver"
)

func byteSaverListNew(listKey []byte) func(*redis.Pool) saver.BytesSaver {
	return func(p *redis.Pool) saver.BytesSaver {
		return func(serialized []byte) (bytesCount int64, e error) {
			var c redis.Conn = p.Get()
			defer c.Close()
			_, e = c.Do("LPUSH", listKey, serialized)
			return int64(len(serialized)), e
		}
	}
}
