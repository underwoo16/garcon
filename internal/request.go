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
	if len(requestBytes) < 2 {
		return nil
	}

	statusAndHeaders := requestBytes[0]
	statusHeaderParts := bytes.Split(statusAndHeaders, []byte(crlf))
	if len(statusHeaderParts) < 2 {
		return nil
	}

	statusLine := statusHeaderParts[0]
	if len(statusLine) == 0 {
		return nil
	}

	statusLineParts := bytes.Split(statusLine, []byte(" "))
	if len(statusLineParts) != 3 {
		return nil
	}

	method := string(statusLineParts[0])
	path := string(statusLineParts[1])
	httpVersion := string(statusLineParts[2])
	headers := statusHeaderParts[1:]
	body := bytes.Trim(requestBytes[1], "\x00")

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
