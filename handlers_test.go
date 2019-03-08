package main

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealthCheckHandler(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(healthCheckHandler))
	defer ts.Close()

	resp, err := http.Get(ts.URL)

	assert.Nil(t, err)

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	assert.Equal(t, "Still OK", string(body))
}

func TestIndexHandler(t *testing.T) {
	const HOST = "test-tool.example.com"

	ts := httptest.NewServer(http.HandlerFunc(indexHandler))
	defer ts.Close()

	req, _ := http.NewRequest("GET", ts.URL, nil)
	req.Host = HOST

	resp, err := http.DefaultClient.Do(req)

	assert.Nil(t, err)

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	// Render index template string
	var expectedBody bytes.Buffer
	err = indexTemplates.ExecuteTemplate(&expectedBody, "layout", HOST)
	assert.Nil(t, err)

	assert.Equal(t, expectedBody.String(), string(body), "Response body didn't match template: '%s'", expectedBody.String())
}
