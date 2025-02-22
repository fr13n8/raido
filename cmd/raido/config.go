package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	proxyAddr     string
	proxyProtocol string
	serviceAddr   string
	agentId       string
	routes        []string
	proxyDomain   string
	logFile       string
)

var (
	defaultLogFile string
	dirPermMode    = os.FileMode(0744) // rwxr--r--
	filePermMode   = os.FileMode(0644) // rw-r--r--
)

func init() {
	serviceAddr = "unix:///var/run/raido.sock"

	defaultLogFileDir := "/var/log/raido/"
	switch runtime.GOOS {
	case "windows":
		defaultLogFileDir = os.Getenv("PROGRAMDATA") + "\\Raido\\"
		serviceAddr = "tcp://127.0.0.1:11051"
	}

	defaultLogFile = defaultLogFileDir + "raido.log"
}

func createFileWriter(fullPath string) (io.Writer, error) {
	_, err := os.Stat(fullPath)
	if err != nil {
		if err := createDirFile(fullPath); err != nil {
			return nil, fmt.Errorf("failed to create directory: %w", err)
		}

		return os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY, filePermMode)
	}

	return os.OpenFile(fullPath, os.O_APPEND|os.O_WRONLY, filePermMode)
}

func createDirFile(fullPath string) error {
	dir := filepath.Dir(fullPath)
	_, err := os.Stat(dir)
	if err != nil {
		err = os.MkdirAll(dir, dirPermMode)
		if err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	return nil
}

func initLogger(logFile string) error {
	if logFile != "console" {
		logFileWriter, err := createFileWriter(logFile)
		if err != nil {
			return fmt.Errorf("failed to create log file writer: %w", err)
		}

		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        logFileWriter,
			TimeFormat: time.DateTime,
		})
	}
	if logFile == "console" {
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out: os.Stdout,
			FormatTimestamp: func(i interface{}) string {
				return ""
			},
		})
	}

	return nil
}
