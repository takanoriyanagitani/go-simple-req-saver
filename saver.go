package saver

import (
	"net/http"
)

type RequestSaver[Q, R any] func(request Q) (result R, e error)

type RequestSaverStd[R any] RequestSaver[*http.Request, R]

func (s RequestSaverStd[R]) ToHandlerFunc(
	result2writer func(result R, e error, writer http.ResponseWriter),
) http.HandlerFunc {
	return func(w http.ResponseWriter, q *http.Request) {
		result, e := s(q)
		result2writer(result, e, w)
	}
}

func RequestSaverNew[Q, S, R any](
	serializer func(request Q) (serialized S, e error),
	saver func(serialized S) (result R, e error),
) RequestSaver[Q, R] {
	return Compose(
		serializer,
		saver,
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
	serializer func(request *http.Request) (serialized []byte, e error),
	saver func(serialized []byte) (result R, e error),
) RequestSaverStd[R] {
	return RequestSaverNewStd(serializer, saver)
}
