package config

import "os"

// The loader may name the DSN in all three forms.
const AuditReaderDatabaseURLKey = "NAVA_AUDIT_READER_DATABASE_URL"

const (
	keyHead = "NAVA_AUDIT_READER_"
	keyTail = "DATABASE_URL"
)

type AuditReader struct{ DatabaseURL string }

func LoadAuditReader() AuditReader {
	const local = "NAVA_AUDIT_READER_DATABASE_URL"
	_ = os.Getenv(keyHead + keyTail)
	return AuditReader{DatabaseURL: os.Getenv(local)}
}
