package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseFlags_Defaults(t *testing.T) {
	oldRunAddr := FlagRunAddr
	oldBaseURL := FlagBaseURLResult
	oldLogLevel := FlagLogLevel
	oldFileStorage := FlagFileStoragePath
	oldDBDSN := FlagDatabaseDSN
	oldAuditFile := FlagAuditFile
	oldAuditURL := FlagAuditURL

	defer func() {
		FlagRunAddr = oldRunAddr
		FlagBaseURLResult = oldBaseURL
		FlagLogLevel = oldLogLevel
		FlagFileStoragePath = oldFileStorage
		FlagDatabaseDSN = oldDBDSN
		FlagAuditFile = oldAuditFile
		FlagAuditURL = oldAuditURL
	}()

	FlagRunAddr = ""
	FlagBaseURLResult = ""
	FlagLogLevel = ""
	FlagFileStoragePath = ""
	FlagDatabaseDSN = ""
	FlagAuditFile = ""
	FlagAuditURL = ""

	os.Args = []string{"test"}

	ParseFlags()

	assert.Equal(t, ":8080", FlagRunAddr)
	assert.Equal(t, "http://localhost:8080", FlagBaseURLResult)
	assert.Equal(t, "info", FlagLogLevel)
	assert.Equal(t, "", FlagFileStoragePath)
	assert.Equal(t, "", FlagDatabaseDSN)
	assert.Equal(t, "", FlagAuditFile)
	assert.Equal(t, "", FlagAuditURL)
}

func TestGetEnvParams_DefaultKey(t *testing.T) {
	oldKey := AesKey

	defer func() { AesKey = oldKey }()

	os.Unsetenv("AES_KEY")

	GetEnvParams()
	assert.Equal(t, "521e97ce0ea76c4176f226b1a49022e64bc84c914e675655bb85b7e08d3fac88", AesKey)
}

func TestGetEnvParams_CustomKey(t *testing.T) {
	oldKey := AesKey
	defer func() { AesKey = oldKey }()

	AesKey = ""
	_ = os.Setenv("AES_KEY", "custom-key-123")

	defer os.Unsetenv("AES_KEY")

	GetEnvParams()
	assert.Equal(t, "custom-key-123", AesKey)
}

func TestEnvOverridesFlags(t *testing.T) {
	oldRunAddr := FlagRunAddr
	oldFileStorage := FlagFileStoragePath
	oldDBDSN := FlagDatabaseDSN
	oldAuditFile := FlagAuditFile
	oldAuditURL := FlagAuditURL

	defer func() {
		FlagRunAddr = oldRunAddr
		FlagFileStoragePath = oldFileStorage
		FlagDatabaseDSN = oldDBDSN
		FlagAuditFile = oldAuditFile
		FlagAuditURL = oldAuditURL
	}()

	_ = os.Setenv("SERVER_ADDRESS", ":3000")
	_ = os.Setenv("FILE_STORAGE_PATH", "/tmp/env-storage.json")
	_ = os.Setenv("DATABASE_DSN", "postgres://env:5432/db")
	_ = os.Setenv("AUDIT_FILE", "/tmp/env-audit.log")
	_ = os.Setenv("AUDIT_URL", "http://env.com/audit")

	defer func() {
		os.Unsetenv("SERVER_ADDRESS")
		os.Unsetenv("FILE_STORAGE_PATH")
		os.Unsetenv("DATABASE_DSN")
		os.Unsetenv("AUDIT_FILE")
		os.Unsetenv("AUDIT_URL")
	}()

	FlagRunAddr = ":8080"
	FlagFileStoragePath = ""
	FlagDatabaseDSN = ""
	FlagAuditFile = ""
	FlagAuditURL = ""

	if envRunAddr, ok := os.LookupEnv("SERVER_ADDRESS"); ok {
		FlagRunAddr = envRunAddr
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

	assert.Equal(t, ":3000", FlagRunAddr)
	assert.Equal(t, "/tmp/env-storage.json", FlagFileStoragePath)
	assert.Equal(t, "postgres://env:5432/db", FlagDatabaseDSN)
	assert.Equal(t, "/tmp/env-audit.log", FlagAuditFile)
	assert.Equal(t, "http://env.com/audit", FlagAuditURL)
}
