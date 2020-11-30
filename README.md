# Catalog MQTT Client

The Catalog MQTT Client runs in a vm or container and talks to an on-prem Ansible Tower. It can
* Collect inventory objects from Ansible Tower
* Launch and monitor jobs on Ansible Tower.

The Catalog MQTT Client subscribes to a specific topic on the cloud MQTT controller based on its
unqiue guid. When a task needs to be done on the client the cloud controller sends a small message
packet that includes the url to get the task details, a date time stamp and the kind of task.
```json
{
    "url": "http://cloud.redhat.com/api/catalog-inventory/v3.0/tasks/xxxx",
    "kind": "catalog",
    "sent": "2020-10-03T12:34:56Z"
} 
```

Once the client gets this message it looks at the URL and fetches the task details.

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
# Input Parameters for Catalog MQTT Client

 1. Debug
 2. Tower Token
 3. Tower URL
 4. MQTT_URL

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

For Inventory Collection
![Alt UsingUploadService](cat_mqtt1.png?raw=true)

For Single Job 

![Alt DirectTaskUpdate](cat_mqtt2.png?raw=true)
