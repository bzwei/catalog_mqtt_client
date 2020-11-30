package main

import (
	"testing"
)

func TestGet(t *testing.T) {
	responseBody := []string{`{"count": 200, "previous": null, "next": "/page/2", "results": [ {"name": "jt1", "id": 1, "url": "url1"},{"name": "jt2", "id": 2, "url":"url2"}]}`,
		`{"count": 450, "previous": "/page/1", "next": null, "results": [ {"name": "jt3", "id": 3, "url": "url3"},{"name": "jt4", "id": 4, "url": "url4"}]}`}

	responses := []map[string]interface{}{
		{
			"count":    200,
			"previous": nil,
		},
		{
			"count": 450,
			"next":  nil,
		},
	}
	jp := JobParam{
		Method:         "get",
		HrefSlug:       "/api/v2/job_templates?page_size=15&name=Fred",
		FetchAllPages:  true,
		AcceptEncoding: "gzip",
		ApplyFilter:    "results[].{id:id, url:url}",
	}

	ts := &testScaffold{}
	ts.runSuccess(t, jp, 200, responseBody, responses)
}

func TestMonitor(t *testing.T) {
	responseBody := []string{`{"name": "job15", "id": 15, "url": "url15","status":"waiting"}`,
		`{"name": "job15", "id": 15, "url": "url15", "status":"successful"}`}

	responses := []map[string]interface{}{
		{
			"name":   "job15",
			"status": "successful",
		},
	}
	jp := JobParam{
		Method:                 "monitor",
		HrefSlug:               "/api/v2/jobs/15",
		RefreshIntervalSeconds: 1,
		AcceptEncoding:         "gzip",
	}
	ts := &testScaffold{}
	ts.runSuccess(t, jp, 200, responseBody, responses)
}

func TestMonitorMissing(t *testing.T) {
	responseBody := []string{"Job Missing"}
	jp := JobParam{
		Method:         "monitor",
		HrefSlug:       "/api/v2/jobs/15",
		AcceptEncoding: "gzip",
	}
	ts := &testScaffold{}
	ts.runFail(t, jp, 404, responseBody, "Job Missing")
}

func TestMonitorStatusMissing(t *testing.T) {
	responseBody := []string{`{"name": "job15", "id": 15, "url": "url15"}`}
	jp := JobParam{
		Method:         "monitor",
		HrefSlug:       "/api/v2/jobs/15",
		AcceptEncoding: "gzip",
	}
	ts := &testScaffold{}
	ts.runFail(t, jp, 200, responseBody, "Object does not contain a status attribute")
}

func TestMonitorStatusInvalid(t *testing.T) {
	responseBody := []string{`{"name": "job15", "id": 15, "url": "url15", "status":"Charkie"}`}
	jp := JobParam{
		Method:         "monitor",
		HrefSlug:       "/api/v2/jobs/15",
		AcceptEncoding: "gzip",
	}
	ts := &testScaffold{}
	ts.runFail(t, jp, 200, responseBody, "Status: Charkie is not one of the known status")
}

func TestPost(t *testing.T) {
	responseBody := []string{`{"name": "job1", "id": 1, "artifacts":{"expose_to_redhat_com_name": "Fred"}}`}
	responses := []map[string]interface{}{
		{
			"name": "job1",
			"id":   1,
		},
	}

	jp := JobParam{
		Method:   "post",
		HrefSlug: "/api/v2/job_templates/5/launch",
	}
	ts := &testScaffold{}
	ts.runSuccess(t, jp, 200, responseBody, responses)
}

func TestUnknownMethod(t *testing.T) {
	jp := JobParam{
		Method:   "unknown",
		HrefSlug: "/api/v2/job_templates/5/launch",
	}
	responseBody := []string{"Fail"}
	ts := &testScaffold{}
	ts.runFail(t, jp, 200, responseBody, "Invalid method received unknown")
}
