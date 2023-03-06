package main

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gomodule/redigo/redis"

	saver "github.com/takanoriyanagitani/go-simple-req-saver"
)

func req2tar2redisNew(listKey []byte) func(*redis.Pool) saver.RequestSaverStd[int64] {
	var saverBuilder func(*redis.Pool) saver.BytesSaver = byteSaverListNew(listKey)
	return func(p *redis.Pool) saver.RequestSaverStd[int64] {
		var ser saver.RequestStd2bytes = reqStd2bytesTarNew()
		var sav saver.BytesSaver = saverBuilder(p)
		return sav.NewRequestSaverStd(ser)
	}
}

var req2tar2redisBuilderDefault func(*redis.Pool) saver.RequestSaverStd[int64] = req2tar2redisNew(
	[]byte("test-list-key"),
)

func reqSaverLockedNew[R any](original saver.RequestSaverStd[R]) saver.RequestSaverStd[R] {
	var l sync.Mutex
	return func(request *http.Request) (result R, e error) {
		l.Lock()
		defer l.Unlock()
		return original(request)
	}
}

func reqHandlerNew(sav saver.RequestSaverStd[int64]) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, q *http.Request) {
		_, e := sav(q)
		if nil != e {
			w.WriteHeader(500)
			_, _ = w.Write([]byte("Unexpected Error"))
			log.Printf("error: %v\n", e)
			return
		}

		_, _ = w.Write([]byte("saved"))
	}
}

func main() {
	var addr string = "localhost:6379"
	var pool *redis.Pool = &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial:        func() (redis.Conn, error) { return redis.Dial("tcp", addr) },
	}

	var c redis.Conn = pool.Get()
	defer c.Close()
	pong, e := c.Do("PING")
	if nil != e {
		panic(e)
	}
	if "PONG" != pong {
		panic(pong)
	}

	var req2tar2redis saver.RequestSaverStd[int64] = req2tar2redisBuilderDefault(pool)
	var locked saver.RequestSaverStd[int64] = reqSaverLockedNew(req2tar2redis)

	http.HandleFunc("/", reqHandlerNew(locked))
	e = http.ListenAndServe(":8888", nil)
	if nil != e {
		panic(e)
	}
}
