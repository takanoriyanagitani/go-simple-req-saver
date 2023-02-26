package saver_test

import (
	"testing"

	"bytes"
	"os"
	"path/filepath"

	saver "github.com/takanoriyanagitani/go-simple-req-saver"
)

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
