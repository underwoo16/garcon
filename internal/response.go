package internal

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"net"
	"strings"
)

type Response struct {
	HttpVersion string
	Status      string
	Headers     map[string]string
	Body        []byte
}

// set status code and message
func (r *Response) SetStatus(status string) {
	r.Status = status
}

// set a header
func (r *Response) SetHeader(key string, value string) {
	r.Headers[key] = value
}

// set the body
func (r *Response) SetBody(body []byte) {
	r.Body = body
}

func (r *Response) WriteTo(conn net.Conn, request *Request) error {
	_, err := conn.Write([]byte(r.HttpVersion + " " + r.Status + "\r\n"))
	if err != nil {
		return err
	}

	if strings.Contains(request.Headers["Accept-Encoding"], "gzip") {
		r.SetHeader("Content-Encoding", "gzip")

		var b bytes.Buffer
		gz := gzip.NewWriter(&b)
		_, err := gz.Write(r.Body)
		if err != nil {
			return err
		}
		err = gz.Close()
		if err != nil {
			return err
		}
		r.SetHeader("Content-Length", fmt.Sprintf("%d", len(b.Bytes())))
		r.SetBody(b.Bytes())
	} else {
		r.SetHeader("Content-Length", fmt.Sprintf("%d", len(r.Body)))
	}

	for key, value := range r.Headers {
		_, err := conn.Write([]byte(key + ": " + value + "\r\n"))
		if err != nil {
			return err
		}
	}

	_, err = conn.Write([]byte("\r\n"))
	if err != nil {
		return err
	}

	_, err = conn.Write(r.Body)
	if err != nil {
		return err
	}

	return nil
}
