package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/underwoo16/garcon/internal"
)

var directory = ""

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
			// TODO: just log the error and continue
			os.Exit(1)
		}

		go handleRequest(conn)
	}
}

func handleRequest(conn net.Conn) {
	defer conn.Close()

	// TODO: handle arbitary length requests
	b := make([]byte, 2048)
	_, err := conn.Read(b)
	if err != nil {
		fmt.Println("Error reading: ", err.Error())
		// TODO: return 500 server error?
		os.Exit(1)
	}

	request := internal.ParseRequest(b)
	if request == nil {
		fmt.Println("Error parsing request")
		// TODO: return request malformed response
		return
	}

	fmt.Printf("Method: %s\n", request.Method)
	fmt.Printf("Path: %s\n", request.Path)
	fmt.Printf("HttpVersion: %s\n", request.HttpVersion)
	fmt.Printf("Headers: %s\n", request.Headers)
	fmt.Printf("Body: %s\n\n", request.Body)

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

	switch {
	case request.Path == "/_ping":
		response.SetStatus("200 OK")
		response.SetBody([]byte("pong"))
	case directory != "":
		response = serveFile(request, response)
	default:
		response.SetStatus("404 Not Found")
	}

	err := response.WriteTo(conn, request)
	if err != nil {
		fmt.Println("Error writing: ", err.Error())
	}
}

func handlePostRequest(conn net.Conn, request *internal.Request) {
	defer conn.Close()
	response := internal.Response{
		HttpVersion: request.HttpVersion,
		Status:      "404 Not Found",
		Headers:     make(map[string]string),
		Body:        []byte{},
	}

	err := response.WriteTo(conn, request)
	if err != nil {
		fmt.Println("Error writing response: ", err.Error())
	}
}

func serveFile(request *internal.Request, response internal.Response) internal.Response {
	path := request.Path
	if path == "/" {
		path = "/index.html"
	}

	filePath := strings.TrimPrefix(path, "/")
	filePath = filepath.Join(directory, filePath)

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

	contentLength := fileInfo.Size()
	fileContents := make([]byte, contentLength)

	_, err = file.Read(fileContents)
	if err != nil {
		fmt.Println("Error reading file: ", err.Error())
		response.SetStatus("404 Not Found")
		return response
	}
	file.Close()

	response.SetStatus("200 OK")
	contentType := "text/html"
	response.SetHeader("Content-Type", contentType)
	response.SetHeader("Content-Length", fmt.Sprintf("%d", contentLength))
	response.SetBody(fileContents)

	return response
}
