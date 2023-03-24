package saver

import (
	"bytes"
	"io"
	"net/http"

	"archive/tar"
	"strings"
)

// A Http Request.
type Request[H, B any] struct {
	header H
	body   B
}

// Header gets a http request header.
func (q Request[H, B]) Header() H { return q.header }

// Body gets a http request body.
func (q Request[H, B]) Body() B { return q.body }

// RequestNew creates a request.
func RequestNew[H, B any](header H, body B) Request[H, B] {
	return Request[H, B]{
		header,
		body,
	}
}

type RequestSerializer[S, H, B any] func(Request[H, B]) (serialized S, e error)

func RequestSerializerNewGeneric[P, S, H, B any](
	getHeaders func(header H, user func(key, val []byte)),
	headerKey2string func(headerKey []byte) string,
	getBodyBytes func(body B) []byte,
	initialize func() (partial P),
	generic func(partial P, namespace, name string, content []byte),
	finalize func(partial P) (serialized S, e error),
) RequestSerializer[S, H, B] {
	const nsHeader = "header"
	const nsBody = "body"
	return func(q Request[H, B]) (serialized S, e error) {
		var partial P = initialize()
		getHeaders(
			q.header,
			func(key, val []byte) {
				var keyString string = headerKey2string(key)
				generic(partial, nsHeader, keyString, val)
			},
		)
		var body []byte = getBodyBytes(q.body)
		generic(partial, nsBody, "body", body)
		return finalize(partial)
	}
}

func RequestSerializerNewGenericTar[H, B any](
	getHeaders func(header H, user func(key, val []byte)),
	headerKey2string func(headerKey []byte) string,
	getBodyBytes func(body B) []byte,
	errorHandler func(error),
) RequestSerializer[[]byte, H, B] {
	var buf bytes.Buffer
	var bs strings.Builder
	return RequestSerializerNewGeneric(
		getHeaders,
		headerKey2string,
		getBodyBytes,
		func() (partial *tar.Writer) {
			buf.Reset()
			return tar.NewWriter(&buf)
		},
		func(partial *tar.Writer, namespace, name string, content []byte) {
			bs.Reset()
			_, _ = bs.WriteString(namespace) // always nil error
			_, _ = bs.WriteString("/")       // always nil error
			_, _ = bs.WriteString(name)      // always nil error

			_, e := Compose(
				func(h *tar.Header) ([]byte, error) { return content, partial.WriteHeader(h) },
				func(body []byte) (int, error) { return partial.Write(body) },
			)(&tar.Header{
				Name: bs.String(),
				Mode: 0400,
				Size: int64(len(content)),
			})
			if nil != e {
				errorHandler(e)
				return
			}
		},
		func(partial *tar.Writer) (serialized []byte, e error) {
			e = partial.Close()
			return buf.Bytes(), e
		},
	)
}

type RequestStd Request[http.Header, []byte]

func (q RequestStd) Serialize2bytes(
	ser RequestSerializer[[]byte, http.Header, []byte],
) (serialized []byte, e error) {
	return ser(Request[http.Header, []byte](q))
}

type RequestStdConv func(*http.Request) (RequestStd, error)
type RequestStd2bytes func(*http.Request) (serialized []byte, e error)

var NopStdRequestSerializer RequestStd2bytes = func(_ *http.Request) ([]byte, error) {
	return nil, nil
}

func DupStdRequestSerializerNew() RequestStd2bytes {
	var buf bytes.Buffer
	return func(q *http.Request) (serialized []byte, e error) {
		buf.Reset()
		_, e = io.Copy(&buf, q.Body)
		return buf.Bytes(), e
	}
}

func (c RequestStdConv) NewRequestStd2bytes(
	ser RequestSerializer[[]byte, http.Header, []byte],
) RequestStd2bytes {
	return Compose(
		c,
		func(s RequestStd) ([]byte, error) { return s.Serialize2bytes(ser) },
	)
}

func RequestStdConvNew(limit int64) RequestStdConv {
	var buf bytes.Buffer
	return func(r *http.Request) (q RequestStd, e error) {
		return Compose(
			Compose(
				func(body io.ReadCloser) (int64, error) {
					buf.Reset()
					return io.Copy(&buf, io.LimitReader(body, limit))
				},
				func(_n int64) ([]byte, error) {
					return buf.Bytes(), nil
				},
			),
			func(body []byte) (q RequestStd, e error) {
				_q := Request[http.Header, []byte]{
					header: r.Header,
					body:   body,
				}
				return RequestStd(_q), nil
			},
		)(r.Body)
	}
}
