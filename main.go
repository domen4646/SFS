package main

import (
	"os"
	"fmt"
	"errors"
)

const VERSION = "1.0"

// Default index.html
const DEFAULT_INDEX_HTML = `<html>
<head>
	<title>Simple file server ` + VERSION + `</title>
</head>
<body>
	<h1>About</h1>
	<p>This is simple file server version ` + VERSION + `</p>
	<hr/>
	<a href="/upload">Upload file</a> | 
	<a href="/uploads">See uploaded files</a>
</body>
</html>`

// Default upload.html
const DEFAULT_UPLOAD_HTML = `<html>
<head>
	<title>Upload file</title>
</head>
<body>
	<form method="POST" action="/upload" enctype="multipart/form-data">
		<input type="file" name="fileUpload"><br/>
		<input type="submit" value="Upload File">
	</form>
</body>
</html>`

// Entry point
func main() {

	os.Mkdir("./www", os.ModePerm)
	if _, err := os.Stat("./www/index.html"); errors.Is(err, os.ErrNotExist) {
		fmt.Println("www/index.html does not exist, creating it ...");
		err = os.WriteFile("./www/index.html", []byte(DEFAULT_INDEX_HTML), os.ModePerm)
		if err != nil {
			fmt.Println("Failed to write to www/index.html!");
			return
		}
	}
	if _, err := os.Stat("./www/upload.html"); errors.Is(err, os.ErrNotExist) {
		fmt.Println("www/upload.html does not exist, creating it ...");
		err = os.WriteFile("./www/upload.html", []byte(DEFAULT_UPLOAD_HTML), os.ModePerm)
		if err != nil {
			fmt.Println("Failed to write to www/upload.html!");
			return
		}
	}

	fmt.Println("Loading settings ...")
	settings := LoadSettingsOrPanic("settings.json")

	var server SimpleFileServer
	fmt.Println("Initializing server ...")
	err := server.Init(settings, true)
	if err != nil {
		fmt.Printf("Error initializing server: %s", err)
		return
	}
	fmt.Println("Starting server ...")
	server.Start()
}
