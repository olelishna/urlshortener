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
	FlagDatabaseDSN     string
	FlagAuditFile       string
	FlagAuditURL        string
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
	flag.StringVar(&FlagFileStoragePath, "f", "", "file storage path")
	flag.StringVar(&FlagDatabaseDSN, "d", "", "database DSN")
	flag.StringVar(&FlagAuditFile, "audit-file", "", "audit log file path")
	flag.StringVar(&FlagAuditURL, "audit-url", "", "audit remote server URL")

	flag.Parse()

	if envRunAddr, ok := os.LookupEnv("SERVER_ADDRESS"); ok {
		FlagRunAddr = envRunAddr
	}

	if envBaseURLResult, ok := os.LookupEnv("BASE_URL"); ok {
		FlagBaseURLResult = envBaseURLResult
	}

	if envLogLevel, ok := os.LookupEnv("LOG_LEVEL"); ok {
		FlagLogLevel = envLogLevel
	}

	if envFileStoragePath, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok {
		FlagFileStoragePath = envFileStoragePath
	}

	if envDatabaseDSN, ok := os.LookupEnv("DATABASE_DSN"); ok {
		FlagDatabaseDSN = envDatabaseDSN
	}

	if envAuditFile, ok := os.LookupEnv("AUDIT_FILE"); ok {
		FlagAuditFile = envAuditFile
	}

	if envAuditURL, ok := os.LookupEnv("AUDIT_URL"); ok {
		FlagAuditURL = envAuditURL
	}
}
