package main

import (
	"faas-engine-go/internal/buildcontext"
	"flag"
	"log/slog"
	"path/filepath"
)

func main() {
	file_path := flag.String("file", "", "Path to the function code directory")
	function_name := flag.String("function-name", "", "Name of the function")
	flag.Parse()

	if *file_path == "" {
		panic("file path is required")
	}

	if *function_name == "" {
		panic("function name is required")
	}

	abspath, err := filepath.Abs(*file_path)
	if err != nil {
		panic("failed to get absolute path: " + err.Error())
	}
	//create a tar stream of the function directory
	tarstream, err := buildcontext.CreateTarStream(abspath)
	if err != nil {
		panic("failed to create tar stream: " + err.Error())
	}

	//send the tarstream to the server
	Response, err := buildcontext.SendTarStream(tarstream, "http://localhost:8080/functions", *function_name)
	if err != nil {
		slog.Info("failed to send tar stream", "error", err)
		return
	}

	slog.Info("response from server:", "Message", Response)
}
