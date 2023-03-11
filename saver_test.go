package saver_test

import (
	"testing"

	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	saver "github.com/takanoriyanagitani/go-simple-req-saver"
)

type testSaverKvPair struct {
	bucket string
	key    string
	val    []byte
}

func (p *testSaverKvPair) updateBucket(bucket map[string][]byte) { bucket[p.key] = p.val }
func (p *testSaverKvPair) upsert(buckets map[string]map[string][]byte) {
	bucket, found := buckets[p.bucket]
	if !found {
		bucket = make(map[string][]byte)
	}
	p.updateBucket(bucket)
}

func assertEqNew[T any](comp func(a, b T) (same bool)) func(a, b T) func(*testing.T) {
	return func(a, b T) func(*testing.T) {
		return func(t *testing.T) {
			var same bool = comp(a, b)
			if !same {
				t.Errorf("Unexpected value got\n")
				t.Errorf("Expected: %v\n", b)
				t.Fatalf("Got:      %v\n", a)
			}
		}
	}
}

func assertEq[T comparable](a, b T) func(*testing.T) {
	var comp func(a, b T) (same bool) = func(a, b T) (same bool) { return a == b }
	return assertEqNew(comp)(a, b)
}

func assertTrue(a bool) func(*testing.T) { return assertEq(a, true) }

func assertNil(e error) func(*testing.T) { return assertEq(nil == e, true) }

func TestSaver(t *testing.T) {
	t.Parallel()

	t.Run("RequestSaver", func(t *testing.T) {
		t.Parallel()

		t.Run("RequestSaverNewFsSelfChecked", func(t *testing.T) {
			t.Parallel()

			t.Run("short test", func(t *testing.T) {
				t.Parallel()

				var dummyRequest uint8 = 0
				var dummySerialized []byte = []byte(`{
					"Content-Type": "application/json",
					"Content-Encoding": "gzip",
				}`)
				var dummyFs map[string][]byte = make(map[string][]byte)
				var rs saver.RequestSaver[uint8, int64] = saver.RequestSaverNewFsSelfChecked(
					func(_request uint8) (selfCheckedBytes []byte, e error) {
						return dummySerialized, nil
					},
					func() (fullpath string) { return "./00.request.json" },
					func(fullpath string, selfCheckedBytes []byte) (written int64, e error) {
						var buf []byte = make([]byte, len(selfCheckedBytes))
						copy(buf, selfCheckedBytes)
						dummyFs[fullpath] = buf
						return int64(len(buf)), nil
					},
				)

				written, e := rs(dummyRequest)

				t.Run("no error", assertNil(e))
				t.Run("len check", assertEq(written, int64(len(dummySerialized))))
				saved, found := dummyFs["./00.request.json"]
				t.Run("dummy file check", assertTrue(found))
				t.Run("dummy content check", assertTrue(bytes.Equal(saved, dummySerialized)))
			})
		})

		t.Run("RequestSaverLimitedNew", func(t *testing.T) {
			t.Parallel()

			t.Run("array limiter", func(t *testing.T) {
				t.Parallel()

				var ser saver.RequestStd2bytes = saver.DupStdRequestSerializerNew()
				const limit int = 3
				var buf [][256]byte = make([][256]byte, 0, limit)
				var saved [][256]byte = buf[:0]
				var temp [256]byte

				var sav saver.BytesSaver = func(serialized []byte) (bytesCount int64, e error) {
					var i int = copy(temp[:], serialized)
					saved = append(saved, temp)
					bytesCount = int64(i)
					return
				}
				var req2saver saver.RequestSaverStd[int64] = sav.NewRequestSaverStd(ser)

				var limiter saver.RequestLimiter[int] = func(ixLimit int) (tooMany bool) {
					return limit <= len(saved)
				}

				var limitedBuilder saver.RequestSaverLimitedBuilder[
					*http.Request,
					int64,
					int,
				] = saver.RequestSaverLimitedNew[*http.Request, int64, int](limiter)

				var saverBuilder func(
					saver.RequestSaver[*http.Request, int64],
				) saver.RequestSaver[*http.Request, int64] = limitedBuilder(limit)

				var limitedSaver saver.RequestSaver[*http.Request, int64] = saverBuilder(
					saver.RequestSaver[*http.Request, int64](req2saver),
				)

				var body *bytes.Reader = bytes.NewReader([]byte("hw"))
				_, e := limitedSaver(httptest.NewRequest(
					"POST",
					"/",
					body,
				))
				t.Run("no error 1", assertNil(e))

				body.Reset([]byte("hh"))
				_, e = limitedSaver(httptest.NewRequest(
					"POST",
					"/",
					body,
				))
				t.Run("no error 2", assertNil(e))

				body.Reset([]byte("iii"))
				_, e = limitedSaver(httptest.NewRequest(
					"POST",
					"/",
					body,
				))
				t.Run("no error 3", assertNil(e))

				body.Reset([]byte("iv"))
				_, e = limitedSaver(httptest.NewRequest(
					"POST",
					"/",
					body,
				))
				t.Run("must fail", assertTrue(nil != e))

			})
		})

		t.Run("RequestSaverNewKV", func(t *testing.T) {
			t.Parallel()

			var buckets map[string]map[string][]byte = make(map[string]map[string][]byte)
			var sav func(*testSaverKvPair) (result int, e error) = func(p *testSaverKvPair) (int, error) {
				p.upsert(buckets)
				return 1, nil
			}
			var dummyRequest uint8 = 0
			var dummyRequest2pair func(_ uint8) (*testSaverKvPair, error) = func(_ uint8) (*testSaverKvPair, error) {
				return &testSaverKvPair{
					bucket: "2023_03_12",
					key:    "12:34:56.789Z",
					val:    []byte("hw"),
				}, nil
			}
			var rs saver.RequestSaver[uint8, int] = saver.RequestSaverNewKV(
				dummyRequest2pair,
				sav,
			)
			cnt, e := rs(dummyRequest)
			t.Run("no error", assertNil(e))
			t.Run("single item", assertEq(cnt, 1))
		})
	})
}

func TestSaverIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping non-short tests...")
	}

	t.Parallel()

	t.Run("RequestSaver", func(t *testing.T) {
		t.Parallel()

		var dirReqSaver string = "test.d/RequestSaver"

		t.Run("RequestSaverNewFsSelfChecked", func(t *testing.T) {
			t.Parallel()

			var dirNewFsSelfChecked string = filepath.Join(
				dirReqSaver, "RequestSaverNewFsSelfChecked",
			)

			t.Run("non-short test", func(t *testing.T) {
				t.Parallel()

				var nonShortTest string = filepath.Join(dirNewFsSelfChecked, "non-short-test")

				var dummyRequest uint8 = 0
				var dummySerialized []byte = []byte(`{
					"Content-Type": "application/json",
					"Content-Encoding": "gzip"
				}`)

				mkdirErr := os.MkdirAll(nonShortTest, 0755)
				t.Run("test dir", assertNil(mkdirErr))

				var rs saver.RequestSaver[
					uint8, int64,
				] = saver.RequestSaverNewFsSelfCheckedWithFileMode(
					func(_request uint8) (selfCheckedBytes []byte, e error) {
						return dummySerialized, nil
					},
					func() (fullpath string) {
						return filepath.Join(nonShortTest, "42-req.json")
					},
					0644,
				)

				written, e := rs(dummyRequest)
				t.Run("no error", assertNil(e))
				t.Run("len check", assertEq(written, int64(len(dummySerialized))))
				saved, e := os.ReadFile(filepath.Join(nonShortTest, "42-req.json"))
				t.Run("no read error", assertNil(e))
				t.Run("content check", assertTrue(bytes.Equal(saved, dummySerialized)))
			})
		})

		t.Run("RequestSaverNewFsNoFsync", func(t *testing.T) {
			t.Parallel()

			var dirNewFsSelfChecked string = filepath.Join(
				dirReqSaver, "RequestSaverNewFsNoFsync",
			)

			t.Run("non-short test", func(t *testing.T) {
				t.Parallel()

				var nonShortTest string = filepath.Join(dirNewFsSelfChecked, "non-short-test")

				var dummyRequest uint8 = 0
				var dummySerialized []byte = []byte(`{
					"Content-Type": "application/json",
					"Content-Encoding": "gzip"
				}`)

				mkdirErr := os.MkdirAll(nonShortTest, 0755)
				t.Run("test dir", assertNil(mkdirErr))

				var rs saver.RequestSaver[
					uint8, int64,
				] = saver.RequestSaverNewFsNoFsync(
					func(_request uint8) (selfCheckedBytes []byte, e error) {
						return dummySerialized, nil
					},
					func() (fullpath string) {
						return filepath.Join(nonShortTest, "634-req.json")
					},
					os.Create,
				)

				written, e := rs(dummyRequest)
				t.Run("no error", assertNil(e))
				t.Run("len check", assertEq(written, int64(len(dummySerialized))))
				saved, e := os.ReadFile(filepath.Join(nonShortTest, "634-req.json"))
				t.Run("no read error", assertNil(e))
				t.Run("content check", assertTrue(bytes.Equal(saved, dummySerialized)))
			})
		})
	})
}
