package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealthCheckHandler(t *testing.T) {
	req, _ := http.NewRequest("GET", "/healthz", nil)

	rec := httptest.NewRecorder()
	handler := http.HandlerFunc(healthCheckHandler)
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Still OK", rec.Body.String())
}

func TestIndexHandler(t *testing.T) {
	const HOST = "test-tool.example.com"

	// Render index template string
	var expectedBody bytes.Buffer
	err := indexTemplates.ExecuteTemplate(&expectedBody, "layout", HOST)
	assert.Nil(t, err)

	req, _ := http.NewRequest("GET", "/", nil)
	req.Host = HOST

	rec := httptest.NewRecorder()
	handler := http.HandlerFunc(indexHandler)
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, expectedBody.String(), rec.Body.String(), "Response body didn't match template: '%s'", expectedBody.String())
}
