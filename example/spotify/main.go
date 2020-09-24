/*
A Sample application queries spotify api. it first calls the search api and get a list of tracks uri.
then use the tracks uri to query for additional track data.
*/
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"restcrawler"
	"strings"
	"sync"

	"github.com/yalp/jsonpath"
)

// DetailMusicCall implement RestCall to retrieve tracks data via the track endpoint
type DetailMusicCall struct {
	token string
	id    string
}

// CreateRequest creates Detail api request
func (d *DetailMusicCall) CreateRequest() (*http.Request, error) {
	getReq, _ := http.NewRequest("GET", fmt.Sprintf("https://api.spotify.com/v1/tracks/%v", d.id), nil)
	getReq.Header.Add("Accept", "application/json")
	getReq.Header.Add("Authorization", fmt.Sprintf("Bearer %v", d.token))
	return getReq, nil

}

// HandleResponse handles Detail api response and prints its uri
func (d *DetailMusicCall) HandleResponse(resp *http.Response, chout chan<- restcrawler.RestCall) {
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	fmt.Println(resp.Status)

	var payload interface{}
	err = json.Unmarshal(body, &payload)
	sid, err1 := jsonpath.Read(payload, "$.uri")
	if err1 != nil {
		fmt.Println("failed")
	}
	fmt.Printf("detail uri: %v\n", sid)

}

// SearchMusicCall implement RestCall to retrieve tracks uri from spotify serach api, and create new Track api calls.
type SearchMusicCall struct {
	token  string
	offset int
	queue  chan<- restcrawler.RestCall
}

// CreateRequest creates the track search api
func (s *SearchMusicCall) CreateRequest() (*http.Request, error) {
	getReq, _ := http.NewRequest("GET", fmt.Sprintf("https://api.spotify.com/v1/search?q=p&type=track&market=US&limit=50&offset=%v", s.offset*50), nil)
	getReq.Header.Add("Accept", "application/json")
	getReq.Header.Add("Authorization", fmt.Sprintf("Bearer %v", s.token))
	return getReq, nil
}

// HandleResponse handles response from search api call and create tracks api call.
func (s *SearchMusicCall) HandleResponse(resp *http.Response, chout chan<- restcrawler.RestCall) {

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("failed")
	}
	if resp.StatusCode == 401 {
		panic("Authentication Failure")
	}
	var payload interface{}
	err = json.Unmarshal(body, &payload)
	sid, err1 := jsonpath.Read(payload, "$.tracks.items[*].uri")
	if err1 != nil {
		fmt.Println("failed")
	}
	for _, uri := range sid.([]interface{}) {
		c := &DetailMusicCall{
			token: s.token,
			id:    strings.Split(fmt.Sprintf("%v", uri), ":")[2],
		}
		chout <- c

	}

	fmt.Println("handle response finished")
}

func main() {
	token := os.Args[1]

	var wg sync.WaitGroup

	// search rest calls
	searchCh := make(chan restcrawler.RestCall)
	// track detail rest calls
	detailCh := make(chan restcrawler.RestCall)

	// Create intial list of Api calls
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			c := &SearchMusicCall{
				token:  token,
				offset: i,
				queue:  searchCh,
			}
			fmt.Println(i)
			searchCh <- c
		}
		close(searchCh)
	}()

	// Invoke Inital list of api calls
	fmt.Print("Tracks search calls")
	searchRc := restcrawler.NewRestClient("search", searchCh, restcrawler.Workers(2), restcrawler.ChannelOut(detailCh))
	wg.Add(1)
	go searchRc.Run(&wg)

	//Additional API call based on response from last call.
	fmt.Println("Tracks Detail calls on track id returned by search call ")
	wg.Add(1)
	detailRc := restcrawler.NewRestClient("detail", detailCh, restcrawler.Workers(2))
	go detailRc.Run(&wg)

	wg.Wait()
	fmt.Println("All calls are completed")

}
