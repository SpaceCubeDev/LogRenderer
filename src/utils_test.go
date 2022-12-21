package main

import (
	"net/http"
	"strconv"
	"testing"
	"time"
)

const addr = ":8181"

func TestMaxLineCountExtraction(t *testing.T) {
	maxLinesCountChan := make(chan int, 1)

	go func() {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			maxLinesCountChan <- extractMaxLinesCount(r)
		})
		t.Error(http.ListenAndServe(addr, nil))
	}()
	time.Sleep(100 * time.Millisecond)

	testWith(t, 0, maxLinesCountChan)
	testWith(t, 1, maxLinesCountChan)
	testWith(t, defaultMaxLinesCount, maxLinesCountChan)
	testWith(t, 1234567, maxLinesCountChan)
	testWith(t, -42, maxLinesCountChan)
}

func testWith(t *testing.T, maxLinesCount int, maxLinesCountChan chan int) {
	t.Logf("Testing with %d line(s) ...", maxLinesCount)
	req := newRequestWith(t, maxLinesCount)
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
	if receivedMaxLinesCount != expected {
		t.Errorf("Expected %d, got %d", expected, receivedMaxLinesCount)
	}
}

func newRequestWith(t *testing.T, maxLinesCount int) *http.Request {
	req, err := http.NewRequest(http.MethodGet, "http://localhost"+addr, nil)
	if err != nil {
		t.Fatal("Failed to create request:", err)
	}
	cookie := new(http.Cookie)
	cookie.Name = "max-lines-count"
	cookie.Value = strconv.Itoa(maxLinesCount)
	req.AddCookie(cookie)
	return req
}
