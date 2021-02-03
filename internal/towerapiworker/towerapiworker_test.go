package towerapiworker

import (
	"testing"

	"github.com/RedHatInsights/rhc_catalog_worker/internal/common"
)

func TestGet(t *testing.T) {
	t.Parallel()
	responseBody := []string{`{"count": 200, "previous": null, "next": "/page/2", "results": [ {"name": "jt1", "id": 1, "url": "url1"},{"name": "jt2", "id": 2, "url":"url2"}]}`,
		`{"count": 450, "previous": "/page/1", "next": null, "results": [ {"name": "jt3", "id": 3, "url": "url3"},{"name": "jt4", "id": 4, "url": "url4"}]}`}

	results1 := []interface{}{
		map[string]interface{}{"id": float64(1), "url": "url1"},
		map[string]interface{}{"id": float64(2), "url": "url2"},
	}

	results2 := []interface{}{
		map[string]interface{}{"id": float64(3), "url": "url3"},
		map[string]interface{}{"id": float64(4), "url": "url4"},
	}

	responses := []map[string]interface{}{
		{
			"count":    float64(200),
			"previous": nil,
			"next":     "/page/2",
			"results":  results1,
		},
		{
			"count":    float64(450),
			"previous": "/page/1",
			"next":     nil,
			"results":  results2,
		},
	}
	jp := common.JobParam{
		Method:        "get",
		HrefSlug:      "/api/v2/job_templates?page_size=15&name=Fred",
		FetchAllPages: true,
		ApplyFilter:   "results[].{id:id, url:url}",
	}

	ts := &testScaffold{}
	ts.runSuccess(t, jp, 200, responseBody, responses)
}

func TestMonitor(t *testing.T) {
	t.Parallel()
	responseBody := []string{`{"name": "job15", "id": 15, "url": "url15","status":"waiting"}`,
		`{"name": "job15", "id": 15, "url": "url15", "status":"successful"}`}

	responses := []map[string]interface{}{
		{
			"id":     float64(15),
			"name":   "job15",
			"status": "successful",
			"url":    "url15",
		},
	}
	jp := common.JobParam{
		Method:                 "monitor",
		HrefSlug:               "/api/v2/jobs/15",
		RefreshIntervalSeconds: 1,
	}
	ts := &testScaffold{}
	ts.runSuccess(t, jp, 200, responseBody, responses)
}

func TestMonitorMissing(t *testing.T) {
	t.Parallel()
	responseBody := []string{"Job Missing"}
	errors := []string{"URL: /api/v2/jobs/15 Status: 404 Message: Job Missing"}
	jp := common.JobParam{
		Method:   "monitor",
		HrefSlug: "/api/v2/jobs/15",
	}
	ts := &testScaffold{}
	ts.runFail(t, jp, 404, responseBody, errors)
}

func TestMonitorStatusMissing(t *testing.T) {
	t.Parallel()
	responseBody := []string{`{"name": "job15", "id": 15, "url": "url15"}`}
	errors := []string{"URL: /api/v2/jobs/15 Status: 0 Message: Object does not contain a status attribute"}
	jp := common.JobParam{
		Method:   "monitor",
		HrefSlug: "/api/v2/jobs/15",
	}
	ts := &testScaffold{}
	ts.runFail(t, jp, 200, responseBody, errors)
}

func TestMonitorStatusInvalid(t *testing.T) {
	responseBody := []string{`{"name": "job15", "id": 15, "url": "url15", "status":"Charkie"}`}
	errors := []string{"URL: /api/v2/jobs/15 Status: 0 Message: Status Charkie is not one of the known status"}
	jp := common.JobParam{
		Method:   "monitor",
		HrefSlug: "/api/v2/jobs/15",
	}
	ts := &testScaffold{}
	ts.runFail(t, jp, 200, responseBody, errors)
}

func TestPost(t *testing.T) {
	t.Parallel()
	responseBody := []string{`{"name": "job1", "id": 1, "artifacts":{"expose_to_redhat_com_name": "Fred"}}`}
	responses := []map[string]interface{}{
		{
			"name":      "job1",
			"id":        float64(1),
			"artifacts": map[string]interface{}{},
		},
	}

	jp := common.JobParam{
		Method:   "post",
		HrefSlug: "/api/v2/job_templates/5/launch",
	}
	ts := &testScaffold{}
	ts.runSuccess(t, jp, 200, responseBody, responses)
}

func TestUnknownMethod(t *testing.T) {
	t.Parallel()
	jp := common.JobParam{
		Method:   "unknown",
		HrefSlug: "/api/v2/job_templates/5/launch",
	}
	responseBody := []string{"Fail"}
	errors := []string{"URL: /api/v2/job_templates/5/launch Status: 0 Message: Invalid method received unknown"}
	ts := &testScaffold{}
	ts.runFail(t, jp, 200, responseBody, errors)
}
