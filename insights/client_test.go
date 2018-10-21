package insights_test

// RoundTripFunc .
import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/go-franky/newrelic/insights"
	"github.com/pkg/errors"
)

type RoundTripFunc func(req *http.Request) *http.Response

// RoundTrip .
func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

// NewTestClient returns *http.Client with Transport replaced to avoid making real calls
func NewTestClient(fn RoundTripFunc) (*insights.Client, error) {
	cl := insights.NewClient(
		&http.Client{
			Transport: RoundTripFunc(fn),
		},
		insights.AccountID("1"),
		insights.InsertKey("insert-abc"),
		insights.QueryKey("query-abc"),
	)
	return cl, nil
}

func TestPublish(t *testing.T) {
	client, err := NewTestClient(func(req *http.Request) *http.Response {
		expectedURL := "https://insights-collector.newrelic.com/v1/accounts/1/events"
		if req.URL.String() != expectedURL {
			t.Errorf("expected %v, got %v", expectedURL, req.URL.String())
		}
		if req.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected %v, got %v", "application/json", req.Header.Get("Content-Type"))
		}
		if req.Header.Get("X-Insert-Key") != "insert-abc" {
			t.Errorf("expected %v, got %v", "abc", req.Header.Get("X-Insert-Key"))
		}
		b, err := ioutil.ReadAll(req.Body)
		if err != nil {
			t.Errorf("could not read body: %v", err)
		}
		expectedBody := `{"bool":true,"duration":3,"eventType":"hello","float":3.2,"int":3,"string":"foo","time":1136239445}`
		if string(b) != expectedBody {
			t.Errorf("\nexpeted: %s\n    got: %v", expectedBody, string(b))
		}
		return &http.Response{
			StatusCode: 200,
			// Send response to be tested
			Body: ioutil.NopCloser(bytes.NewBufferString(`OK`)),
			// Must be set to non-nil value or it panics
			Header: make(http.Header),
		}
	})

	if err != nil {
		t.Errorf("could not instantiate the client: %v", err)
		return
	}
	ts, err := time.Parse(time.RFC1123Z, "Mon, 02 Jan 2006 15:04:05 -0700")
	if err != nil {
		t.Errorf(err.Error())
	}

	data := map[string]interface{}{
		"bool":     true,
		"duration": time.Duration(3123456),
		"float":    3.2,
		"int":      3,
		"string":   "foo",
		"time":     ts,
	}
	if err := client.Publish("hello", data); err != nil {
		t.Errorf("expected no errors, got %v", err)
	}
}
func TestPublishErrors(t *testing.T) {
	tables := []struct {
		name   string
		given  map[string]interface{}
		expect error
	}{
		{
			name:   "Nested maps",
			given:  map[string]interface{}{"foo": map[string]interface{}{}},
			expect: errors.New(fmt.Sprintf("could not cast %v of type %T to valid attributes", map[string]interface{}{}, map[string]interface{}{})),
		},
		{
			name:   "Too many attributes",
			given:  longMap(256),
			expect: errors.New("too many attributes"),
		},
	}

	//tRun(name string, f func(t *T)) bool {

	for _, eg := range tables {
		t.Run(eg.name, func(t *testing.T) {
			client, err := NewTestClient(func(req *http.Request) *http.Response {
				expectedURL := "https://insights-collector.newrelic.com/v1/accounts/1/events"
				if req.URL.String() != expectedURL {
					t.Errorf("expected %v, got %v", expectedURL, req.URL.String())
				}
				if req.Header.Get("Content-Type") != "application/json" {
					t.Errorf("expected %v, got %v", "application/json", req.Header.Get("Content-Type"))
				}
				if req.Header.Get("X-Insert-Key") != "insert-abc" {
					t.Errorf("expected %v, got %v", "insert-abc", req.Header.Get("X-Insert-Key"))
				}
				b, err := ioutil.ReadAll(req.Body)
				if err != nil {
					t.Errorf("could not read body: %v", err)
				}
				expectedBody := `{"eventType":"hello","foo":"bar"}`
				if string(b) != expectedBody {
					t.Errorf("\nexpeted body: %s\n     got:%v", expectedBody, string(b))
				}
				return &http.Response{
					StatusCode: 200,
					// Send response to be tested
					Body: ioutil.NopCloser(bytes.NewBufferString(`OK`)),
					// Must be set to non-nil value or it panics
					Header: make(http.Header),
				}
			})

			if err != nil {
				t.Errorf("could not instantiate the client: %v", err)
				return
			}

			if err := client.Publish("hello", eg.given).Error(); err != eg.expect.Error() {
				t.Errorf("\nexpected %v\n     got %v", eg.expect, err)
			}
		})
	}
}

func longMap(length int) map[string]interface{} {
	m := map[string]interface{}{}
	for i := 0; i < length; i++ {
		m[fmt.Sprintf("%d", i)] = fmt.Sprintf("%d", i)
	}
	return m
}

type fakeDoer struct {
	data *bytes.Buffer
	r    *http.Request
}

type result struct {
	Average float32 `json:"average"`
}
type queryResponse struct {
	Results []result `json:"results"`
}

func (c *fakeDoer) Do(r *http.Request) (*http.Response, error) {
	c.r = r
	return &http.Response{
		StatusCode: 200,
		Body:       ioutil.NopCloser(bytes.NewBufferString(`{"results":[]}`)),
	}, nil
}

func TestQuery(t *testing.T) {
	client, err := NewTestClient(func(req *http.Request) *http.Response {
		expectedURL := "https://insights-api.newrelic.com/v1/accounts/1/query?nrql=SELECT+%2A+FROM+1"
		if req.URL.String() != expectedURL {
			t.Errorf("expected %v, got %v", expectedURL, req.URL.String())
		}
		if req.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected %v, got %v", "application/json", req.Header.Get("Content-Type"))
		}
		if req.Header.Get("X-Query-Key") != "query-abc" {
			t.Errorf("expected %v, got %v", "abc", req.Header.Get("X-Insert-Key"))
		}
		return &http.Response{
			StatusCode: 200,
			// Send response to be tested
			Body: ioutil.NopCloser(bytes.NewBufferString(`{"results":[{"average":2.3}]}`)),
			// Must be set to non-nil value or it panics
			Header: make(http.Header),
		}
	})
	if err != nil {
		t.Errorf("could not instantiate the client: %v", err)
		return
	}

	r := &queryResponse{}
	if err := client.Query("SELECT * FROM 1", r); err != nil {
		t.Errorf(err.Error())
	}

	expectedResult := &queryResponse{
		Results: []result{
			{
				Average: 2.3,
			},
		},
	}
	if !reflect.DeepEqual(expectedResult, r) {
		t.Errorf("\nexpected: %+v\n     got: %+v", expectedResult, r)
	}
}
