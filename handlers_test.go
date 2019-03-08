package main

import (
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
