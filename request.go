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

// RequestSerializer must serialize a request.
type RequestSerializer[S, H, B any] func(Request[H, B]) (serialized S, e error)

// RequestSerializerNewGeneric creates a request serializer.
//
// # Arguments
//   - getHeaders: Gets header items(key/value pairs).
//   - headerKey2string: Gets a header key string.
//   - getBodyBytes: Gets a request body(a slice of bytes).
//   - initialize: Initializes a serializer.
//   - generic: Writes an item(namespace/name/content).
//   - finalize: Finalizes a serializer.
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
	return func(req Request[H, B]) (serialized S, e error) {
		var partial P = initialize()
		getHeaders(
			req.header,
			func(key, val []byte) {
				var keyString string = headerKey2string(key)
				generic(partial, nsHeader, keyString, val)
			},
		)
		var body []byte = getBodyBytes(req.body)
		generic(partial, nsBody, "body", body)
		return finalize(partial)
	}
}

// RequestSerializerNewGenericTar creates a request serializer which creates a tar archive(a slice of bytes).
//
// # Arguments
//   - getHeaders: Gets header items(key/value pairs).
//   - headerKey2string: Gets a header key string.
//   - getBodyBytes: Gets a request body(a slice of bytes).
//   - errorHandler: Handles errors.
func RequestSerializerNewGenericTar[H, B any](
	getHeaders func(header H, user func(key, val []byte)),
	headerKey2string func(headerKey []byte) string,
	getBodyBytes func(body B) []byte,
	errorHandler func(error),
) RequestSerializer[[]byte, H, B] {
	var buf bytes.Buffer
	var strBuf strings.Builder
	return RequestSerializerNewGeneric(
		getHeaders,
		headerKey2string,
		getBodyBytes,
		func() (partial *tar.Writer) {
			buf.Reset()
			return tar.NewWriter(&buf)
		},
		func(partial *tar.Writer, namespace, name string, content []byte) {
			strBuf.Reset()
			_, _ = strBuf.WriteString(namespace) // always nil error
			_, _ = strBuf.WriteString("/")       // always nil error
			_, _ = strBuf.WriteString(name)      // always nil error

			_, err := Compose(
				func(h *tar.Header) ([]byte, error) { return content, partial.WriteHeader(h) },
				func(body []byte) (int, error) { return partial.Write(body) },
			)(&tar.Header{
				Name: strBuf.String(),
				Mode: 0400,
				Size: int64(len(content)),
			})
			if nil != err {
				errorHandler(err)
				return
			}
		},
		func(partial *tar.Writer) (serialized []byte, e error) {
			e = partial.Close()
			return buf.Bytes(), e
		},
	)
}

// RequestStd is a standard(net/http) request.
type RequestStd Request[http.Header, []byte]

func (q RequestStd) Serialize2bytes(
	ser RequestSerializer[[]byte, http.Header, []byte],
) (serialized []byte, e error) {
	return ser(Request[http.Header, []byte](q))
}

// RequestStdConv must get a slice of bytes from a standard(net/http) request.
type RequestStdConv func(*http.Request) (RequestStd, error)

// RequestStd2bytes must serialize a standard(net/http) request as a slice of bytes.
type RequestStd2bytes func(*http.Request) (serialized []byte, e error)

var NopStdRequestSerializer RequestStd2bytes = func(_ *http.Request) ([]byte, error) {
	return nil, nil
}

// DupStdRequestSerializerNew creates a request serializer which copies a request body.
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

// RequestStdConvNew creates a standard(net/http) request converter.
//
// # Arguments
//   - limit: Number of bytes to read(resource limit).
func RequestStdConvNew(limit int64) RequestStdConv {
	var buf bytes.Buffer
	return func(req *http.Request) (q RequestStd, e error) {
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
					header: req.Header,
					body:   body,
				}
				return RequestStd(_q), nil
			},
		)(req.Body)
	}
}
