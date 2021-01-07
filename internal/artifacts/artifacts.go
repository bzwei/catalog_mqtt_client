package artifacts

import (
	"encoding/json"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
)

// ExposePrefix is the prefix of all extra_vars that will be collected
const ExposePrefix = "expose_to_cloud_redhat_com_"

// MaxArtifactsBytes is the maximum size of an artifact that can be collected
const MaxArtifactsBytes = 1024

// Sanctify the JSON payload for artifacts. The attribute key in the artifacts
// map should start with expose_to_cloud_redhat_com_ else they are excluded
func Sanctify(data map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	for k, v := range data {
		if strings.HasPrefix(k, ExposePrefix) {
			result[k] = v
		}
	}

	b, err := json.Marshal(result)
	if err != nil {
		log.Println("Error marshaling to json error:", err)
		return nil, err
	}

	if len(b) > MaxArtifactsBytes {
		err = fmt.Errorf("Artifacts is greater than %d bytes", MaxArtifactsBytes)
		return nil, err
	}
	return result, nil
}
