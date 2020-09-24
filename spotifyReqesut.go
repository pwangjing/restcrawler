package restcrawler

import (
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gammazero/workerpool"
	"github.com/hashicorp/go-retryablehttp"
	log "github.com/sirupsen/logrus"
)

// RestCall is an interface implemented each type of Rest api call.
type RestCall interface {
	CreateRequest() (*http.Request, error)
	HandleResponse(*http.Response, chan<- RestCall)
}

// RestClient is a type manges a pull of Restcall workers.
type RestClient struct {
	name    string
	client  *http.Client
	chin    <-chan RestCall
	chout   chan<- RestCall
	nWorker int
	//TODO: max error count
}

// Option is used to support functional option.
type Option func(*RestClient)

// Workers is used to pass number of workers option.
func Workers(n int) Option {
	return func(o *RestClient) {
		o.nWorker = n
	}
}

// Client is used to pass custom http client to RestClient.
func Client(c *http.Client) Option {
	return func(o *RestClient) {
		o.client = c
	}
}

// ChannelOut can be set to chain together many rest calls.
func ChannelOut(chout chan<- RestCall) Option {
	return func(o *RestClient) {
		o.chout = chout
	}
}

// NewRestClient initialize a new Rest Cralwer client.
func NewRestClient(name string, chin <-chan RestCall, opts ...Option) *RestClient {

	rc := RestClient{
		name:    name,
		client:  DefautBackoffHTTPClient(),
		chin:    chin,
		nWorker: 2,
	}

	for _, opt := range opts {
		opt(&rc)
	}

	return &rc
}

// Run starts a pool of rest calls worker. Often run inside a goroute.
func (c *RestClient) Run(wg *sync.WaitGroup) {
	defer wg.Done()
	wp := workerpool.New(c.nWorker)
	for rc := range c.chin {
		rc := rc
		wp.Submit(
			func() {
				req, err := rc.CreateRequest()
				if err != nil {
					log.WithField("error", err).WithField("restcall", rc).Error("Error when creating http request")
					return
				}
				resp, err := c.client.Do(req)
				if err != nil {
					log.WithField("name", c.name).WithField("restcall", rc).WithField("err", err).Info("Http Request Failed")
					return
				}

				rc.HandleResponse(resp, c.chout)
			})
	}
	wp.StopWait()
	log.WithField("name", c.name).Info("Processed all jobs for RestClient")

	// close out channel after we arecertain no more restcall will be write to it anymore.
	if c.chout != nil {
		close(c.chout)
	}

}

// DefautBackoffHTTPClient function returns an Jitter backoff http client for rest crawler
func DefautBackoffHTTPClient() *http.Client {
	retryClient := retryablehttp.NewClient()
	retryClient.Backoff = retryablehttp.LinearJitterBackoff
	retryClient.RetryMax = 4
	standardClient := retryClient.StandardClient() // *http.Client
	return standardClient
}

// ExponetialBackoffWithJetter implments an exponential Jitter backoff policy
func ExponetialBackoffWithJetter(min, max time.Duration, attemptNum int, resp *http.Response) time.Duration {
	if resp != nil {
		if resp.StatusCode == http.StatusTooManyRequests {
			if s, ok := resp.Header["Retry-After"]; ok {
				if sleep, err := strconv.ParseInt(s[0], 10, 64); err == nil {
					return time.Second * time.Duration(sleep)
				}
			}
		}
	}

	// Seed rand; doing this every time is fine
	rand := rand.New(rand.NewSource(int64(time.Now().Nanosecond())))

	// Pick a random number that lies somewhere between the min and max and
	// multiply by the attemptNum. attemptNum starts at zero so we always
	// increment here. We first get a random percentage, then apply that to the
	// difference between min and max, and add to min.
	jitter := rand.Float64() * float64(max-min)

	mult := math.Pow(2, float64(attemptNum))*float64(min) + jitter
	sleep := time.Duration(mult)
	if float64(sleep) != mult || sleep > max {
		sleep = max
	}
	return sleep
}
