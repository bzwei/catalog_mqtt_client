# Catalog Worker

The Catalog Worker runs under the aegis of Red Hat Connect (rhc). It can be run in a vm or container and talks to an on-prem Ansible Tower. It can
* Collect inventory objects from Ansible Tower
* Launch and monitor jobs on Ansible Tower.

The Red Hat Connect (rhc) subscribes to a specific topic on the Cloud Connector based on its
unqiue guid. When a task needs to be done on the client the Cloud Connector sends a small message
packet to the RHC which transfers the message to the Catalog Worker via GRPC. The message includes the url to get the task details, a date time stamp and the kind of task.
```json
{
    "url": "http://cloud.redhat.com/api/catalog-inventory/v3.0/tasks/xxxx",
    "kind": "catalog",
    "sent": "2020-10-03T12:34:56Z"
} 
```

Once the catalog worker gets this message it looks at the URL and fetches the task details.

The client updates the task after it has finished processing.
The client can either send the response directly to the task#result or it can upload a 
compress tar file to the upload service. Since the inventory data tends to be big we usually upload
that via a compressed tar file. For other simple requests we directly update the task#results.

A task is a collection of jobs alongwith result format and upload url.

e.g.
```json
{
    "response_format": "tar|json",
    "upload_url": "https://cloud.redhat.com/api/v1/ingress/upload"
    "jobs": [{
        "href_slug": "/api/v2/job_templates",
        "method": "get",
        "fetch_all_pages": true,
        "apply_filter": "results[].{id:id, inventory:inventory, type:type, url:url}"
        "fetch_related": [{
            "href_slug": "survey_spec",
            "predicate": "survey_enabled"
        }]
    }]
}
```
# Parameters for Catalog Worker
 The Parameters for the catalog worker are stored in /etc/yggdrasil/workers/catalog.conf


# Task Parameters 
|Keyword| Description | Example
|--|--|--
|**response_format**| Compressed tar file or json| tar
|**upload_url**| The URL of the upload service| https://cloud.redhat.com/api/ingress/v1/upload
|**jobs**|An array of jobs for this task| See example below
# Job Parameters 
|Keyword| Description | Example
|--|--|--
|**href_slug**| The Partial URL (required) |/api/v2/job_templates
|**method**| One of get/post/monitor/launch (required) | get
|fetch_all_pages| Fetch all pages from Tower for a URL | true
|apply_filter|JMES Path filter to trim data | **results[].{id:id, type:type, created:created,name:name**
|params| Post Params or Query Params|
|fetch_related| Optionally fetch other related objects

The list of inventory objects to be collected from the tower is sent from the cloud.redhat.com.
The list of objects needed by catalog are
 1. Job Templates
 2. Inventories
 3. Credentials
 4. Credential Types
 5. Survey Spec
 6. Workflow Templates
 7. Workflow Template Nodes

General Workflow

![Alt UsingUploadService](./docs/catalog_worker.png?raw=true)
