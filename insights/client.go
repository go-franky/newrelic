package insights

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const (
	defaultInsertURL = "https://insights-collector.newrelic.com"
	insertPath       = "v1/accounts/%s/events"

	defaultQueryURL = "https://insights-api.newrelic.com"
	queryPath       = "v1/accounts/%s/query"
)

type doer interface {
	Do(*http.Request) (*http.Response, error)
}

// Client manages communication with the Insights API.
type Client struct {
	client           doer
	insertKey        string
	queryKey         string
	accountID        string
	defaultInsertURL *url.URL
	defaultQueryURL  *url.URL
}

// NewClient returns a new Insights API client
func NewClient(httpClient doer, options ...func(*Client)) *Client {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: time.Duration(10 * time.Second),
		}
	}
	diu, _ := url.Parse(defaultInsertURL)
	dqu, _ := url.Parse(defaultQueryURL)

	client := &Client{client: httpClient, defaultInsertURL: diu, defaultQueryURL: dqu}

	for _, option := range options {
		option(client)
	}
	return client
}

// AccountID sets the account id for Insights
func AccountID(accountID string) func(c *Client) {
	return func(c *Client) {
		c.accountID = accountID
	}
}

// InsertKey set the API Key for inserting events
func InsertKey(insertKey string) func(c *Client) {
	return func(c *Client) {
		c.insertKey = insertKey
	}
}

// QueryKey sets the query key for querying Insights
func QueryKey(queryKey string) func(c *Client) {
	return func(c *Client) {
		c.queryKey = queryKey
	}
}

// Query makes an insights NRQL and passes the response
// back into the interface.
func (c *Client) Query(query string, i interface{}) error {
	req, err := http.NewRequest("GET", c.fullurl(query), nil)
	if err != nil {
		return errors.Wrapf(err, "could not create request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Query-Key", c.queryKey)
	resp, err := c.client.Do(req)
	if err != nil {
		return errors.Wrap(err, "could not make the request")
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "could not read body")
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request unsuccessfull: %d - %s", resp.StatusCode, body)
	}

	if err := json.Unmarshal(body, i); err != nil {
		return errors.Wrapf(err, "could not unmarshal %s", body)
	}

	return nil
}

func (c *Client) fullurl(query string) string {
	values := url.Values{}
	values.Add("nrql", query)
	path := fmt.Sprintf(c.defaultQueryURL.String()+"/"+queryPath+"?", c.accountID)
	return path + values.Encode()
}

// Publish puts an event into Insights
func (c *Client) Publish(eventType string, e map[string]interface{}) error {
	newMap := map[string]interface{}{"eventType": eventType}
	for k, v := range e {
		switch v.(type) {
		case int, bool, string, float32, float64:
			newMap[k] = v
		case time.Time:
			newMap[k] = v.(time.Time).Unix()
		case time.Duration:
			newMap[k] = (v.(time.Duration)).Round(time.Duration(time.Millisecond)).Nanoseconds() / time.Duration(time.Millisecond).Nanoseconds()
		default:
			return fmt.Errorf("could not cast %v of type %T to valid attributes", v, v)
		}
	}
	if len(newMap) > 255 {
		return errors.New("too many attributes")
	}
	contentBody, err := json.Marshal(newMap)
	if err != nil {
		return errors.Wrap(err, "could not marshal the body")
	}

	path := fmt.Sprintf(insertPath, c.accountID)
	path = fmt.Sprintf("%s/%s", c.defaultInsertURL.String(), path)

	req, err := http.NewRequest("POST", path, strings.NewReader(string(contentBody)))
	if err != nil {
		return errors.Wrap(err, "could not create request")
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Insert-Key", c.insertKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return errors.Wrap(err, "could not post the request")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "could not read body")
	}

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("%d: %s", resp.StatusCode, string(body))
	}

	return nil
}
