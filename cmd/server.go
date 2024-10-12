package main

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/underwoo16/garcon/internal"
)

var directory = "."

func main() {
	// TODO: parse command line arguments robustly
	if len(os.Args) > 1 {
		if os.Args[1] == "--directory" {
			directory = os.Args[2]
		}
	}

	// TODO: set port from command line arguments
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}
	defer l.Close()

	for {
		// TODO: pool connections
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

	// TODO: handle arbitary length requests
	b := make([]byte, 1024)
	_, err := conn.Read(b)
	if err != nil {
		fmt.Println("Error reading: ", err.Error())
		os.Exit(1)
	}

	// TODO: handle request parsing errors
	request := internal.ParseRequest(b)
	fmt.Printf("Method: %s\n", request.Method)
	fmt.Printf("Path: %s\n", request.Path)
	fmt.Printf("HttpVersion: %s\n", request.HttpVersion)
	fmt.Printf("Headers: %s\n", request.Headers)
	fmt.Printf("Body: %s\n", request.Body)

	switch request.Method {
	case "GET":
		handleGetRequest(conn, request)
	case "POST":
		handlePostRequest(conn, request)
	default:
		response := internal.Response{
			HttpVersion: request.HttpVersion,
			Status:      "405 Method Not Allowed",
			Headers:     make(map[string]string),
			Body:        []byte{},
		}
		err := response.WriteTo(conn, request)
		if err != nil {
			fmt.Println("Error writing: ", err.Error())
		}
	}
}

func handleGetRequest(conn net.Conn, request *internal.Request) {
	defer conn.Close()
	response := internal.Response{
		HttpVersion: request.HttpVersion,
		Status:      "200 OK",
		Headers:     make(map[string]string),
		Body:        []byte{},
	}

	// TODO: add _ping route
	// TODO: add _status route
	// TODO: serve index.html for /
	switch {
	case request.Path == "/_ping":
		response.SetStatus("200 OK")
		response.SetBody([]byte("pong"))
	default:
		response.SetStatus("404 Not Found")
	}

	err := response.WriteTo(conn, request)
	if err != nil {
		fmt.Println("Error writing: ", err.Error())
	}
}

// TODO: we probably don't want this to arbitrarily write files to disk
func handlePostRequest(conn net.Conn, request *internal.Request) {
	defer conn.Close()
	response := internal.Response{
		HttpVersion: request.HttpVersion,
		Status:      "200 OK",
		Headers:     make(map[string]string),
		Body:        []byte{},
	}

	err := response.WriteTo(conn, request)
	if err != nil {
		fmt.Println("Error writing response: ", err.Error())
	}
}

// TODO: handle file paths robustly
// TODO: handle content types
func serveFile(path string, response internal.Response) internal.Response {
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
