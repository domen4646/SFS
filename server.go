package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
	"path/filepath"
	"strings"
	"io"
)

type SimpleFileServer struct {
	settings Settings
	doLogToFile bool
	latestLogFile *os.File
	currentLogFile *os.File
	spaceUsed uint64
	sizeLimitBytes uint64
	singleFileSizeLimitBytes uint64
	maxMultipartBytes uint64
	_initialized bool // Boolean defaults to false
}

func (s *SimpleFileServer) initializeLogs() error {
	if !s.doLogToFile {
		return nil
	}
	os.Mkdir("./logs", os.ModePerm)
	logfile, err := os.Create("./logs/latest.txt")
	if err != nil {
		return err
	}
	s.latestLogFile = logfile
	currentTime := time.Now()
	currentLogName := fmt.Sprintf("./logs/%s.txt", currentTime.Format("2006_01_02_at_15_04_05"))
	s.currentLogFile, err = os.Create(currentLogName)
	if err != nil {
		return err
	}
	return nil
}

func (s *SimpleFileServer) log(logText string) {
	fmt.Println(logText)

	if s.doLogToFile {
		_, err := fmt.Fprintln(s.currentLogFile, logText)
		if err != nil {
			fmt.Println("Failed to write to current log file!")
		}
		_, err = fmt.Fprintln(s.latestLogFile, logText)
		if err != nil {
			fmt.Println("Failed to write to latest log file!")
		}
	}
}

// Generator for log middleware
func (s *SimpleFileServer) generateLogHandler() (func (http.Handler) http.Handler) {
	return func (next http.Handler) http.Handler {
		return http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
			logText := fmt.Sprintf("%s %s %s", r.RemoteAddr, r.Method, r.URL)
			s.log(logText)
			next.ServeHTTP(w, r)
		})
	}
}

func (s *SimpleFileServer) updateUsedSpace() error {
	files, err := os.ReadDir(s.settings.FolderPath)
	if err != nil {
		return err
	}

	var used uint64 = 0
	for _, file := range files {
		fileinfo, err := file.Info()
		if err != nil {
			return err
		}
		used += uint64(fileinfo.Size())
	}
	s.spaceUsed = used
	return nil
}

func (s *SimpleFileServer) Init(_settings Settings, _doLogToFile bool) error {
	s.settings = _settings
	// Size limit in settings is in MB
	s.sizeLimitBytes = s.settings.SizeLimit * (1 << 20) 
	s.singleFileSizeLimitBytes = s.settings.SingleFileSizeLimit * (1 << 20)
	// Additional space for other fields in data
	s.maxMultipartBytes = s.singleFileSizeLimitBytes + (10 << 10) 
	s.doLogToFile = _doLogToFile
	s.initializeLogs()
	//fmt.Printf("[DEBUG] Creating dir %s\n", s.settings.FolderPath)
	err := os.Mkdir(s.settings.FolderPath, os.ModePerm)
	if err == nil {
		s.spaceUsed = 0
	} else {
		err = s.updateUsedSpace()
		if err != nil {
			return err
		}
	}
	s._initialized = true
	return nil
}

func (s *SimpleFileServer) generateFileUploadHandler() func(http.ResponseWriter, *http.Request) {

	return func (w http.ResponseWriter, r *http.Request) {

		if r.Method == "GET" {
			s.log("Serving the HTML form file ...")
			http.ServeFile(w, r, "./www/upload.html")
			return
		}

		if r.Method != "POST" {
			s.log(fmt.Sprintf("Unexpected request method: %s", r.Method))
			return
		}
		if s.settings.ReadOnly {
			s.log("Cannot upload file: Directory is marked as readonly in the settings.")
			fmt.Fprintf(w, "Cannot upload file: Directory is marked as readonly in the settings.")
		}

		r.ParseMultipartForm(int64(s.maxMultipartBytes))
		file, handler, err := r.FormFile("fileUpload")
		if err != nil {
			full := fmt.Sprintf("Error when parsing file upload: %s", err)
			s.log(full)
			fmt.Fprintf(w, "File upload failed! See log for details.")
			return
		}

		defer file.Close()
		full := fmt.Sprintf("%s: Uploading %s of size %d ...", r.RemoteAddr, handler.Filename, handler.Size)
		s.log(full)

		if uint64(handler.Size) > s.singleFileSizeLimitBytes {
			s.log("File too large!")
			fmt.Fprintf(w, "File too large!")
			return
		}
		if s.spaceUsed + uint64(handler.Size) > s.sizeLimitBytes {
			s.log("Not enough space for the file!")
			fmt.Fprintf(w, "Not enough space to upload this file.")
			return
		}

		filename := fmt.Sprintf("./uploads/%s", handler.Filename)
		ext := filepath.Ext(filename)
		for _, e := range s.settings.ForbiddenExtensions {
			if strings.EqualFold(e, ext) {
				filename += ".txt"
				break
			}
		}

		tmp, err := os.Create(filename)
		if err != nil {
			s.log(fmt.Sprintf("Failed to create file in the uploads: %s", err))
			fmt.Fprintf(w, "File upload failed! See log for details.")
			return
		}
		defer tmp.Close()

		_, err = io.Copy(tmp, file)
		if err != nil {
			s.log(fmt.Sprintf("Failed to copy file: %s", err))
			fmt.Fprintf(w, "File upload failed! See log for details.")
			return
		}

		err = s.updateUsedSpace()
		if err != nil {
			s.log(fmt.Sprintf("Failed to update used space: %s", err))
		}

		s.log("File upload successful.")
		fmt.Fprintf(w, "File upload successful.")
	}
	
}

func (s *SimpleFileServer) Start() error {
	if !s._initialized {
		fmt.Println("Error: Attempting to start uninitialized server.")
		return nil
	}

	logMiddleware := s.generateLogHandler()
	fmt.Println("Config done")


	mux := http.NewServeMux()
	uploadHandler := http.HandlerFunc(s.generateFileUploadHandler())
	mux.Handle("/upload", logMiddleware(uploadHandler))
	indexHandler := http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./www/index.html")
	})
	mux.Handle("/", logMiddleware(indexHandler))

	// File servers
	staticFileServer := http.FileServer(http.Dir("./static"))
	uploadFileServer := http.FileServer(http.Dir(s.settings.FolderPath))
	mux.Handle("/static/", logMiddleware(http.StripPrefix("/static/", staticFileServer)))
	mux.Handle("/uploads/", logMiddleware(http.StripPrefix("/uploads/", uploadFileServer)))
	s.log("Starting server ...")

	http.ListenAndServe(":8080", mux)
	return nil
}
