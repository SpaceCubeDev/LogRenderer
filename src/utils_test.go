package main

import (
	"context"
	"net/http"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMaxLineCountExtraction(t *testing.T) {
	maxLinesCountChan := make(chan int, 1)

	muxServer := http.NewServeMux()
	muxServer.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		maxLinesCountChan <- extractMaxLinesCount(r)
	})
	httpServer := http.Server{
		Addr:    testAddr,
		Handler: muxServer,
	}
	go func() {
		err := httpServer.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			t.Error("Failed to start http server:", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	testMaxLineCountWith(t, 0, maxLinesCountChan)
	testMaxLineCountWith(t, 1, maxLinesCountChan)
	testMaxLineCountWith(t, defaultMaxLinesCount, maxLinesCountChan)
	testMaxLineCountWith(t, 1234567, maxLinesCountChan)
	testMaxLineCountWith(t, -42, maxLinesCountChan)

	// testing with no cookie provided
	req, err := http.NewRequest(http.MethodGet, "http://localhost"+testAddr, nil)
	if err != nil {
		t.Fatal("Failed to create request:", err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal("Failed to execute request:", err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("Invalid response status: %d / %q\n", res.StatusCode, res.Status)
	}
	receivedMaxLinesCount := <-maxLinesCountChan
	assert.Equal(t, defaultMaxLinesCount, receivedMaxLinesCount, "Invalid max lines count")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err = httpServer.Shutdown(ctx)
	if err != nil {
		t.Log("Failed to shutdown http server:", err)
	}
}

func testMaxLineCountWith(t *testing.T, maxLinesCount int, maxLinesCountChan chan int) {
	t.Logf("Testing with %d line(s) ...", maxLinesCount)
	req := newLineCountRequestWith(t, maxLinesCount)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal("Failed to execute request:", err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("Invalid response status: %d / %q\n", res.StatusCode, res.Status)
	}
	expected := maxLinesCount
	if maxLinesCount == 0 {
		expected = defaultMaxLinesCount
	}
	receivedMaxLinesCount := <-maxLinesCountChan
	assert.Equal(t, expected, receivedMaxLinesCount, "Invalid max lines count")
}

func newLineCountRequestWith(t *testing.T, maxLinesCount int) *http.Request {
	req, err := http.NewRequest(http.MethodGet, "http://localhost"+testAddr, nil)
	if err != nil {
		t.Fatal("Failed to create request:", err)
	}
	cookie := new(http.Cookie)
	cookie.Name = "max-lines-count"
	cookie.Value = strconv.Itoa(maxLinesCount)
	req.AddCookie(cookie)
	return req
}

func TestFindAllGroups(t *testing.T) {
	type m map[string]string
	expectFindAllGroups(t, dynamicServerPathRegexp, "/dynamic/srv/inst", m{"server": "srv", "instance": "inst"})
	expectFindAllGroups(t, dynamicServerPathRegexp, "/dynamic/srv/", m{"server": "srv", "instance": ""})
	expectFindAllGroups(t, dynamicServerPathRegexp, "/dynamic///inst", m{})
	expectFindAllGroups(t, dynamicServerPathRegexp, "", m{})
	expectFindAllGroups(t, regexp.MustCompile(`.*`), "any", m{})
	subGroupsRe := regexp.MustCompile(`(?P<a>\w{2}(?P<b>\w{3}(?P<c>\w{4})))`)
	expectFindAllGroups(t, subGroupsRe, "aabbbcccc", m{"a": "aabbbcccc", "b": "bbbcccc", "c": "cccc"})
}

func expectFindAllGroups(t *testing.T, re *regexp.Regexp, str string, expected map[string]string) {
	result := findAllGroups(re, str)
	assert.Equal(t, expected, result)
}

func TestFilePathEscape(t *testing.T) {
	assert.Equal(t, "", filePathEscape(""))
	assert.Equal(t, "Lw%3D%3D", filePathEscape("/"))
	assert.Equal(t,
		"L3BhdGgvdG8vRHluYW1pY1NlcnZlcnMvUGFwZXJfKi9sb2dzL2xhdGVzdC5sb2c%3D",
		filePathEscape("/path/to/DynamicServers/Paper_*/logs/latest.log"),
	)
}

func TestFilePathUnescape(t *testing.T) {
	assert.Equal(t, "", filePathUnescape(""))
	assert.Equal(t, "/", filePathUnescape("Lw%3D%3D"))
	assert.Equal(t,
		"/path/to/DynamicServers/Paper_*/logs/latest.log",
		filePathUnescape("L3BhdGgvdG8vRHluYW1pY1NlcnZlcnMvUGFwZXJfKi9sb2dzL2xhdGVzdC5sb2c%3D"),
	)
}
