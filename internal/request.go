package internal

import "bytes"

type Request struct {
	Method      string
	Path        string
	HttpVersion string
	Headers     map[string]string
	Body        []byte
}

var crlf = []byte("\r\n")

func ParseRequest(b []byte) *Request {
	requestBytes := bytes.Split(b, []byte("\r\n\r\n"))
	statusAndHeaders := requestBytes[0]
	statusLine := bytes.Split(statusAndHeaders, []byte(crlf))[0]
	statusLineParts := bytes.Split(statusLine, []byte(" "))
	method := string(statusLineParts[0])
	path := string(statusLineParts[1])
	httpVersion := string(statusLineParts[2])
	headers := bytes.Split(statusAndHeaders, []byte(crlf))[1:]
	body := requestBytes[1]
	body = bytes.Trim(body, "\x00")

	headersMap := make(map[string]string)
	for _, header := range headers {
		headerParts := bytes.Split(header, []byte(": "))
		headersMap[string(headerParts[0])] = string(headerParts[1])
	}
	request := Request{
		Method:      method,
		Path:        path,
		HttpVersion: httpVersion,
		Headers:     headersMap,
		Body:        body,
	}

	return &request
}
