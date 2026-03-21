package cmd

import (
	"net/http"
	"time"
)

var httpClient = &http.Client{Timeout: 60 * time.Second}
