package saver

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
)

// RequestSaver saves a request.
type RequestSaver[Q, R any] func(request Q) (result R, e error)

// RequestSaverNewKV creates a request saver which saves a key/value pair.
//
// # Arguments
//   - request2kvpair: Gets a key/value pair from a request.
//   - saver: Saves a key/value pair.
func RequestSaverNewKV[Q, R, P any](
	request2kvpair func(request Q) (kvpair P, e error),
	saver func(kvpair P) (result R, e error),
) RequestSaver[Q, R] {
	return Compose(request2kvpair, saver)
}

var RequestLimiterErrTooMany error = errors.New("too many requests")

type RequestLimiter[L any] func(limit L) (tooMany bool)

// A RequestSaverLimitedBuilder creates a request saver which may reject saves.
type RequestSaverLimitedBuilder[Q, R, L any] func(lmt L) func(RequestSaver[Q, R]) RequestSaver[Q, R]

func RequestSaverLimitedNew[Q, R, L any](l RequestLimiter[L]) RequestSaverLimitedBuilder[Q, R, L] {
	return func(limit L) func(RequestSaver[Q, R]) RequestSaver[Q, R] {
		return func(original RequestSaver[Q, R]) RequestSaver[Q, R] {
			return func(request Q) (result R, e error) {
				var tooMany bool = l(limit)
				if tooMany {
					e = RequestLimiterErrTooMany
					return
				}
				return original(request)
			}
		}
	}
}

// RequestSaverStd saves a standard(net/http) request.
type RequestSaverStd[R any] RequestSaver[*http.Request, R]

// ToHandlerFunc converts a RequestSaverStd to a HandlerFunc.
//
// # Arguments
//   - result2writer: Writes a save result.
func (s RequestSaverStd[R]) ToHandlerFunc(
	result2writer func(result R, e error, writer http.ResponseWriter),
) http.HandlerFunc {
	return func(w http.ResponseWriter, q *http.Request) {
		result, e := s(q)
		result2writer(result, e, w)
	}
}

// RequestSaverNew creates a RequestSaver which saves a serialized request.
//
// # Arguments
//   - serializer: Serializes a request.
//   - saver: Saves a serialized request.
func RequestSaverNew[Q, S, R any](
	serializer func(request Q) (serialized S, e error),
	saver func(serialized S) (result R, e error),
) RequestSaver[Q, R] {
	return Compose(
		serializer,
		saver,
	)
}

func RequestSaverNewWriter[Q any](
	serializer func(request Q) (serialized []byte, e error),
	writer io.Writer,
) RequestSaver[Q, int64] {
	return RequestSaverNew(
		serializer,
		func(serialized []byte) (written int64, e error) {
			var rdr *bytes.Reader = bytes.NewReader(serialized)
			return io.Copy(writer, rdr)
		},
	)
}

func RequestSaverNewFsSelfChecked[Q any](
	serializer func(request Q) (selfCheckedBytes []byte, e error),
	nameGen func() (fullpath string),
	bytes2file func(fullpath string, selfCheckedBytes []byte) (written int64, e error),
) RequestSaver[Q, int64] {
	var b2f func(fullpath string) func([]byte) (int64, error) = Curry(bytes2file)
	return RequestSaverNew(
		serializer,
		func(selfCheckedBytes []byte) (written int64, e error) {
			return b2f(nameGen())(selfCheckedBytes)
		},
	)
}

func RequestSaverNewFsSelfCheckedWithFileMode[Q any](
	serializer func(request Q) (selfCheckedBytes []byte, e error),
	nameGen func() (fullpath string),
	filemode os.FileMode,
) RequestSaver[Q, int64] {
	return RequestSaverNewFsSelfChecked(
		serializer,
		nameGen,
		func(fullpath string, selfCheckedBytes []byte) (written int64, e error) {
			return Compose(
				func(writer func(string, []byte, os.FileMode) error) (int, error) {
					return len(selfCheckedBytes), writer(fullpath, selfCheckedBytes, filemode)
				},
				func(written int) (int64, error) { return int64(written), nil },
			)(os.WriteFile)
		},
	)
}

func RequestSaverNewFsNoFsync[Q any](
	serializer func(request Q) (selfCheckedBytes []byte, e error),
	nameGen func() (fullpath string),
	createFile func(fullpath string) (*os.File, error),
) RequestSaver[Q, int64] {
	var writer *bufio.Writer = bufio.NewWriter(nil)
	return RequestSaverNewFsSelfChecked(
		serializer,
		nameGen,
		func(fullpath string, selfCheckedBytes []byte) (written int64, e error) {
			return Compose(
				createFile,
				func(f *os.File) (int64, error) {
					writer.Reset(f)

					written, e = Compose(
						Curry(io.Copy)(writer),
						func(written int64) (int64, error) { return written, writer.Flush() },
					)(bytes.NewReader(selfCheckedBytes))

					return written, errors.Join(e, f.Close())
				},
			)(fullpath)
		},
	)
}

func RequestSaverNewStd[S, R any](
	serializer func(request *http.Request) (serialized S, e error),
	saver func(serialized S) (result R, e error),
) RequestSaverStd[R] {
	var s RequestSaver[*http.Request, R] = RequestSaverNew(serializer, saver)
	var t RequestSaverStd[R] = RequestSaverStd[R](s)
	return t
}

func RequestSaverNewStdBytes[R any](
	serializer RequestStd2bytes,
	saver func(serialized []byte) (result R, e error),
) RequestSaverStd[R] {
	return RequestSaverNewStd(serializer, saver)
}

type BytesSaver func(serialized []byte) (bytesCount int64, e error)

func (b BytesSaver) NewRequestSaverStd(serializer RequestStd2bytes) RequestSaverStd[int64] {
	return RequestSaverNewStdBytes(serializer, b)
}
