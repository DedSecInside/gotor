package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DedSecInside/gotor/internal/crawl"
)

func requestJSON(t *testing.T, method, url string, body any) (*http.Response, []byte) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(encoded)
	}
	request, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	return response, payload
}

func waitForTerminalJob(t *testing.T, apiURL, id string) Job {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		response, payload := requestJSON(t, http.MethodGet, apiURL+APIBasePath+"/jobs/"+id, nil)
		if response.StatusCode != http.StatusOK {
			t.Fatalf("unexpected job status %d: %s", response.StatusCode, payload)
		}
		var job Job
		if err := json.Unmarshal(payload, &job); err != nil {
			t.Fatal(err)
		}
		switch job.Status {
		case JobCompleted, JobFailed, JobCancelled:
			return job
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for job")
	return Job{}
}

func TestJobControlAPICompletesCrawl(t *testing.T) {
	t.Parallel()
	target := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/html")
		fmt.Fprint(writer, `<html><title>API target</title><a href="mailto:api@example.com">Email</a></html>`)
	}))
	defer target.Close()

	apiServer := httptest.NewServer(NewServer("127.0.0.1", 0).Handler())
	defer apiServer.Close()

	healthResponse, healthPayload := requestJSON(t, http.MethodGet, apiServer.URL+"/_healthz", nil)
	if healthResponse.StatusCode != http.StatusOK {
		t.Fatalf("health check failed: %s", healthPayload)
	}

	response, payload := requestJSON(t, http.MethodPost, apiServer.URL+APIBasePath+"/jobs", crawl.Request{
		URL:               target.URL,
		Depth:             0,
		UseTor:            false,
		RequestsPerSecond: 100,
		Burst:             100,
	})
	if response.StatusCode != http.StatusAccepted {
		t.Fatalf("start job returned %d: %s", response.StatusCode, payload)
	}
	var started Job
	if err := json.Unmarshal(payload, &started); err != nil {
		t.Fatal(err)
	}

	completed := waitForTerminalJob(t, apiServer.URL, started.ID)
	if completed.Status != JobCompleted {
		t.Fatalf("expected completed job, got %#v", completed)
	}
	if completed.Report == nil || len(completed.Report.Pages) != 1 {
		t.Fatalf("missing crawl report: %#v", completed.Report)
	}
	emails := completed.Report.Pages[0].Metadata.Emails
	if len(emails) != 1 || emails[0] != "api@example.com" {
		t.Fatalf("unexpected emails: %#v", emails)
	}
}

func TestJobControlAPICancelsCrawl(t *testing.T) {
	t.Parallel()
	target := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		<-request.Context().Done()
	}))
	defer target.Close()

	apiServer := httptest.NewServer(NewServer("127.0.0.1", 0).Handler())
	defer apiServer.Close()

	response, payload := requestJSON(t, http.MethodPost, apiServer.URL+APIBasePath+"/jobs", crawl.Request{
		URL:    target.URL,
		Depth:  0,
		UseTor: false,
	})
	if response.StatusCode != http.StatusAccepted {
		t.Fatalf("start job returned %d: %s", response.StatusCode, payload)
	}
	var started Job
	if err := json.Unmarshal(payload, &started); err != nil {
		t.Fatal(err)
	}

	response, payload = requestJSON(
		t,
		http.MethodPost,
		apiServer.URL+APIBasePath+"/jobs/"+started.ID+"/cancel",
		nil,
	)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("cancel job returned %d: %s", response.StatusCode, payload)
	}
	cancelled := waitForTerminalJob(t, apiServer.URL, started.ID)
	if cancelled.Status != JobCancelled {
		t.Fatalf("expected cancelled job, got %#v", cancelled)
	}
}
