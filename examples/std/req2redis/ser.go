package main

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	saver "github.com/takanoriyanagitani/go-simple-req-saver"
)

func reqSerStdTarNew() saver.RequestSerializer[[]byte, http.Header, io.ReadCloser] {
	var headerKeyBuf strings.Builder
	var bodyBuf bytes.Buffer

	var hkBuf bytes.Buffer
	var hvBuf bytes.Buffer

	return saver.RequestSerializerNewGenericTar(
		func(h http.Header, user func(key, val []byte)) {
			for key, values := range h {
				for _, val := range values {
					hkBuf.Reset()
					hvBuf.Reset()
					_, _ = hkBuf.WriteString(key)
					_, _ = hvBuf.WriteString(val)
					user(hkBuf.Bytes(), hvBuf.Bytes())
				}
			}
		},
		func(headerKey []byte) (s string) {
			headerKeyBuf.Reset()
			_, _ = headerKeyBuf.Write(headerKey)
			return headerKeyBuf.String()
		},
		func(body io.ReadCloser) []byte {
			bodyBuf.Reset()
			_, _ = io.Copy(&bodyBuf, body)
			return bodyBuf.Bytes()
		},
		func(e error) {
			panic(e)
		},
	)
}

var ReqSerStdTarDefault saver.RequestSerializer[
	[]byte, http.Header, io.ReadCloser,
] = reqSerStdTarNew()

func reqStd2bytesTarNew() saver.RequestStd2bytes {
	var ser saver.RequestSerializer[[]byte, http.Header, io.ReadCloser] = reqSerStdTarNew()
	return func(q *http.Request) (serialized []byte, e error) {
		var req saver.Request[http.Header, io.ReadCloser] = saver.RequestNew(q.Header, q.Body)
		return ser(req)
	}
}
