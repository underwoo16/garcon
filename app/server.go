package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"net"
	"os"
	"strings"
)

var directory = "."
var crlf = []byte("\r\n")

func main() {
	// parse --directory flag
	if len(os.Args) > 1 {
		if os.Args[1] == "--directory" {
			directory = os.Args[2]
		}
	}

	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		go handleRequest(conn)
	}
}

func handleRequest(conn net.Conn) {
	defer conn.Close()

	b := make([]byte, 1024)
	_, err := conn.Read(b)
	if err != nil {
		fmt.Println("Error reading: ", err.Error())
		os.Exit(1)
	}

	request := parseRequest(b)
	fmt.Printf("Method: %s", request.Method)
	fmt.Printf("Path: %s", request.Path)
	fmt.Printf("HttpVersion: %s", request.HttpVersion)
	fmt.Printf("Headers: %s", request.Headers)
	fmt.Printf("Body: %s", request.Body)

	switch request.Method {
	case "GET":
		handleGetRequest(conn, request)
	case "POST":
		handlePostRequest(conn, request)
	default:
		response := "HTTP/1.1 405 Method Not Allowed\r\n\r\n"
		_, err = conn.Write([]byte(response))
		if err != nil {
			fmt.Println("Error writing: ", err.Error())
		}
	}
}

func handleGetRequest(conn net.Conn, request *Request) {
	defer conn.Close()
	response := Response{
		HttpVersion: request.HttpVersion,
		Status:      "200 OK",
		Headers:     make(map[string]string),
		Body:        []byte{},
	}

	if strings.HasPrefix(request.Path, "/echo/") {
		echo := strings.TrimPrefix(request.Path, "/echo/")
		length := len(echo)
		response.SetHeader("Content-Type", "text/plain")
		response.SetHeader("Content-Length", fmt.Sprintf("%d", length))
		response.SetBody([]byte(echo))
	} else if request.Path == "/user-agent" {
		length := len(request.Headers["User-Agent"])
		response.SetHeader("Content-Type", "text/plain")
		response.SetHeader("Content-Length", fmt.Sprintf("%d", length))
		response.SetBody([]byte(request.Headers["User-Agent"]))
	} else if strings.HasPrefix(request.Path, "/files/") {
		response = serveFile(request.Path, response)
	} else if request.Path != "/" {
		response.SetStatus("404 Not Found")
	}

	err := response.WriteTo(conn, request)
	if err != nil {
		fmt.Println("Error writing: ", err.Error())
	}
}

func handlePostRequest(conn net.Conn, request *Request) {
	defer conn.Close()
	response := Response{
		HttpVersion: request.HttpVersion,
	}

	if strings.HasPrefix(request.Path, "/files/") {
		filePath := strings.TrimPrefix(request.Path, "/files/")
		filePath = fmt.Sprintf("%s/%s", directory, filePath)

		file, err := os.Create(filePath)

		if err != nil {
			fmt.Println("Error creating file: ", err.Error())
			response.SetStatus("500 Internal Server Error")
		} else {
			_, err = file.Write(request.Body)
			if err != nil {
				fmt.Println("Error writing to file: ", err.Error())
				response.SetStatus("500 Internal Server Error")
			} else {
				response.SetStatus("201 Created")
			}
			file.Sync()
			file.Close()
		}
	} else {
		response.SetStatus("404 Not Found")
	}

	err := response.WriteTo(conn, request)
	if err != nil {
		fmt.Println("Error writing response: ", err.Error())
	}
}

type Request struct {
	Method      string
	Path        string
	HttpVersion string
	Headers     map[string]string
	Body        []byte
}

func parseRequest(b []byte) *Request {
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

func serveFile(path string, response Response) Response {
	filePath := strings.TrimPrefix(path, "/files/")
	filePath = fmt.Sprintf("%s/%s", directory, filePath)

	_, err := os.Stat(filePath)
	if err != nil {
		fmt.Println("Error reading file: ", err.Error())
		response.SetStatus("404 Not Found")
		return response
	}

	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Error opening file: ", err.Error())
		response.SetStatus("404 Not Found")
		return response
	}

	fileInfo, err := file.Stat()
	if err != nil {
		fmt.Println("Error opening file: ", err.Error())
		response.SetStatus("404 Not Found")
		return response
	}

	contentType := "application/octet-stream"
	contentLength := fileInfo.Size()

	fileContents := make([]byte, contentLength)
	_, err = file.Read(fileContents)
	if err != nil {
		fmt.Println("Error reading file: ", err.Error())
		// return "HTTP/1.1 404 Not Found\r\n\r\n"
		response.SetStatus("404 Not Found")
		return response
	}
	file.Close()

	response.SetStatus("200 OK")
	response.SetHeader("Content-Type", contentType)
	response.SetHeader("Content-Length", fmt.Sprintf("%d", contentLength))
	response.SetBody(fileContents)

	return response
}

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
