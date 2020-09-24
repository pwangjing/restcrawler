# Rest Crawler
Rest Crawler is a Go library that can be used to quickly extract data from various restful endpoint. It abstract away the implementation of rate limit retries, backoff and concurrency from developer. Developer could focus on implementing request and handling response. 

## Quick Start

#### Create a RestCall struct. 

```go
type DetailMusicCall struct {
	token string
	id    string
}
```
#### Implement RestCall Interface.

```go
func (d *DetailMusicCall) CreateRequest() (*http.Request, error) {
	getReq, _ := http.NewRequest("GET", fmt.Sprintf("https://api.spotify.com/v1/tracks/%v", d.id), nil)
	getReq.Header.Add("Accept", "application/json")
	getReq.Header.Add("Authorization", fmt.Sprintf("Bearer %v", d.token))
	return getReq, nil

}

func (d *DetailMusicCall) HandleResponse(resp *http.Response, chout chan<- restcrawler.RestCall) {
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	var payload interface{}
	err = json.Unmarshal(body, &payload)
	sid, err1 := jsonpath.Read(payload, "$.uri")
	if err1 != nil {
		fmt.Println("failed")
	}
	fmt.Printf("uri: %v\n", sid)

}
```

#### Send the Rescall Struct to a channel, and start the rest crawler to process it. 
```go
	searchCh := make(chan restcrawler.RestCall)

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
    wg.Wait()
```
## Sample Application 
```bash
go run ./example/spotify/main.go <spotify_oauth_token>
```

## Features
1. Configurable Pool size for every type of rest endpoint.
2. Configurable retry numbers, backoff policy. 
3. Fully customizable Http client. 
4. Chain multiple rest endpoint calls together. 

