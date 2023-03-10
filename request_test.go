package saver_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	saver "github.com/takanoriyanagitani/go-simple-req-saver"

	"archive/tar"
	"bytes"
	"io"
)

func TestRequest(t *testing.T) {
	t.Parallel()

	t.Run("RequestSerializer", func(t *testing.T) {
		t.Parallel()

		t.Run("RequestSerializerNewGenericTar", func(t *testing.T) {
			t.Parallel()

			t.Run("empty", func(t *testing.T) {
				t.Parallel()
				var dummyHeader uint8 = 0
				var dummyBody uint8 = 0

				var ts saver.RequestSerializer[
					[]byte, uint8, uint8,
				] = saver.RequestSerializerNewGenericTar(
					func(_header uint8, user func(key, val []byte)) {},
					func(_key []byte) string { return "" },
					func(_body uint8) []byte { return nil },
					func(e error) {},
				)

				var q saver.Request[uint8, uint8] = saver.RequestNew(
					dummyHeader,
					dummyBody,
				)
				serialized, e := ts(q)

				t.Run("no error", assertNil(e))
				t.Run("non 0 bytes", assertTrue(0 < len(serialized)))

				var tr *tar.Reader = tar.NewReader(bytes.NewReader(serialized))
				hdr, e := tr.Next()
				t.Run("no read error", assertNil(e))
				t.Run("expected name", assertEq(hdr.Name, "body/body"))

				var buf bytes.Buffer
				n, e := io.Copy(&buf, tr)
				t.Run("no io read error", assertNil(e))
				t.Run("0 bytes", assertEq(n, 0))
			})

			t.Run("minimal", func(t *testing.T) {
				t.Parallel()
				var dummyHeader uint8 = 0
				var dummyBody uint8 = 0

				const reqBody string = `{
					"count200": 634,
					"count404": 42,
					"count500": 2,
					"unixtime": 123456789
				}`
				var ts saver.RequestSerializer[
					[]byte, uint8, uint8,
				] = saver.RequestSerializerNewGenericTar(
					func(_header uint8, user func(key, val []byte)) {
						user([]byte("Content-Type"), []byte("application/json"))
						user([]byte("Content-Encoding"), []byte("gzip"))
					},
					func(key []byte) string { return string(key) },
					func(_body uint8) []byte {
						return []byte(reqBody)
					},
					func(e error) { panic(e) },
				)

				var q saver.Request[uint8, uint8] = saver.RequestNew(
					dummyHeader,
					dummyBody,
				)
				serialized, e := ts(q)

				t.Run("no error", assertNil(e))
				t.Run("non 0 bytes", assertTrue(0 < len(serialized)))

				var buf bytes.Buffer
				checker := func(tr *tar.Reader, name string, content []byte) func(*testing.T) {
					return func(t *testing.T) {
						hdr, e := tr.Next()
						t.Run("no error", assertNil(e))
						t.Run("name check", assertEq(hdr.Name, name))
						buf.Reset()
						_, _ = io.Copy(&buf, tr)
						t.Run("content check", assertTrue(bytes.Equal(buf.Bytes(), content)))
					}
				}

				var tr *tar.Reader = tar.NewReader(bytes.NewReader(serialized))

				t.Run(
					"content type",
					checker(tr, "header/Content-Type", []byte("application/json")),
				)

				t.Run(
					"content encoding",
					checker(tr, "header/Content-Encoding", []byte("gzip")),
				)

				t.Run(
					"content encoding",
					checker(tr, "body/body", []byte(reqBody)),
				)

			})
		})
	})

	t.Run("RequestStdConv", func(t *testing.T) {
		t.Parallel()

		t.Run("RequestStdConvNew", func(t *testing.T) {
			t.Parallel()

			t.Run("limited request", func(t *testing.T) {
				t.Parallel()

				const limit int64 = 65536

				var rsc saver.RequestStdConv = saver.RequestStdConvNew(limit)
				var dummySerializedBody []byte = []byte(`{
					"status_200": 3776,
					"status_400": 634,
					"status_404": 42,
					"status_500": 2,
					"unixtime": 123456789
				}`)
				var q *http.Request = httptest.NewRequest(
					"POST",
					"/api/v1/write",
					bytes.NewReader(dummySerializedBody),
				)
				var h http.Header = q.Header
				h.Set("Content-Type", "application/json")

				converted, e := rsc(q)
				t.Run("no error", assertNil(e))

				var conv saver.Request[http.Header, []byte] = saver.Request[http.Header, []byte](
					converted,
				)

				t.Run("type check", assertEq(
					conv.Header().Get("Content-Type"),
					"application/json",
				))

				t.Run("body check", assertTrue(bytes.Equal(
					conv.Body(),
					dummySerializedBody,
				)))

			})
		})
	})

	t.Run("DupStdRequestSerializerNew", func(t *testing.T) {
		t.Parallel()

		t.Run("empty", func(t *testing.T) {
			t.Parallel()
			var dupSerializer saver.RequestStd2bytes = saver.DupStdRequestSerializerNew()

			var emptyBody bytes.Buffer
			var postRequest *http.Request = httptest.NewRequest(
				"POST",
				"/",
				&emptyBody,
			)

			serialized, e := dupSerializer(postRequest)
			t.Run("no error", assertNil(e))
			t.Run("empty body", assertEq(len(serialized), 0))
		})

		t.Run("non-empty", func(t *testing.T) {
			t.Parallel()
			var dupSerializer saver.RequestStd2bytes = saver.DupStdRequestSerializerNew()

			var body *bytes.Reader = bytes.NewReader([]byte("hw"))
			var postRequest *http.Request = httptest.NewRequest(
				"POST",
				"/",
				body,
			)

			serialized, e := dupSerializer(postRequest)
			t.Run("no error", assertNil(e))
			t.Run("same body", assertTrue(bytes.Equal(serialized, []byte("hw"))))
		})

	})
}
