package config

import (
	"flag"
	"os"
)

var (
	FlagRunAddr         string
	FlagBaseURLResult   string
	FlagLogLevel        string
	FlagFileStoragePath string
)

func ParseFlags() {
	flag.StringVar(&FlagRunAddr, "a", ":8080", "address and port to run server")
	flag.StringVar(
		&FlagBaseURLResult,
		"b",
		"http://localhost:8080",
		"base address for shortened url",
	)
	flag.StringVar(&FlagLogLevel, "l", "info", "log level")
	flag.StringVar(&FlagFileStoragePath, "f", "data/data.json", "file storage path")
	flag.Parse()

	if envRunAddr := os.Getenv("SERVER_ADDRESS"); envRunAddr != "" {
		FlagRunAddr = envRunAddr
	}

	if envBaseURLResult := os.Getenv("BASE_URL"); envBaseURLResult != "" {
		FlagBaseURLResult = envBaseURLResult
	}

	if envLogLevel := os.Getenv("LOG_LEVEL"); envLogLevel != "" {
		FlagLogLevel = envLogLevel
	}

	if envFileStoragePath := os.Getenv("FILE_STORAGE_PATH"); envFileStoragePath != "" {
		FlagFileStoragePath = envFileStoragePath
	}
}
