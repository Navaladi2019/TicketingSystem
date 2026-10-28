package api_test

import "os"

func readTestdata() ([]byte, error) { return os.ReadFile("testdata/openapi.presplit.yaml") }
