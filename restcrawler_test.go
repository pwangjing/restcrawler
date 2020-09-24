package restcrawler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestCreateClient(t *testing.T) {
	ch := make(chan RestCall)
	chout := make(chan RestCall)

	c := NewRestClient("test abc", ch)
	if c.nWorker != 2 {
		t.Fatalf("err %v", "incorrect default workers")
	}
	if c.name != "test abc" {
		t.Fatalf("err %v", "incorrect default name")
	}

	if c.chin != ch {
		t.Fatalf("err %v", "incorrect in channel")
	}

	c = NewRestClient("test abc", ch, Workers(10), ChannelOut(chout))
	if c.chout != chout {
		t.Fatalf("err %v", "incorrect out channel")
	}
	if c.nWorker != 10 {
		t.Fatalf("err %v", "incorrect number worker assigned")
	}

}

type testCall struct {
	url string
}

func (tc *testCall) CreateRequest() (*http.Request, error) {
	getReq, _ := http.NewRequest("GET", tc.url, nil)
	return getReq, nil
}

func (tc *testCall) HandleResponse(resp *http.Response, chout chan<- RestCall) {
	fmt.Println(resp)
	chout <- tc
}

func TestRun(t *testing.T) {
	count := 0
	// Mock server which always responds 500.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		w.WriteHeader(200)
	}))
	defer ts.Close()

	defer ts.Close()
	var wg sync.WaitGroup
	chin := make(chan RestCall)
	chout := make(chan RestCall)
	c := NewRestClient("test abc", chin, Workers(10), ChannelOut(chout))
	wg.Add(1)
	go c.Run(&wg)
	wg.Add(1)
	go func() {
		defer wg.Done()
		chin <- &testCall{ts.URL}
		chin <- &testCall{ts.URL}
		chin <- &testCall{ts.URL}
		chin <- &testCall{ts.URL}

		close(chin)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for c := range chout {
			fmt.Print(c)
		}
		<-chout
	}()
	wg.Wait()
	if count != 4 {
		t.Fatalf("err %v", "incorrect number events received")
	}
	fmt.Printf("%d\n", count)

}
