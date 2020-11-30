package artifacts

import (
	"strings"
	"testing"
)

func TestSuccess(t *testing.T) {
	data := map[string]interface{}{"abc": "123",
		"expose_to_cloud_redhat_com_name": "Fred",
		"expose_to_cloud_redhat_com_age":  45}

	result, err := Sanctify(data)
	if err != nil {
		t.Errorf("Artifact test failed %v", err)
	}

	value := result["expose_to_cloud_redhat_com_age"]
	if value.(int) != 45 {
		t.Errorf("Artifact age didn't match")
	}

	value = result["expose_to_cloud_redhat_com_name"]
	if value.(string) != "Fred" {
		t.Errorf("Artifact name didn't match")
	}

	_, ok := result["abc"]
	if ok {
		t.Errorf("abc key should not be included in artifact")
	}
}

func TestHugeArtifact(t *testing.T) {
	longString := strings.Repeat("na", 512)
	data := map[string]interface{}{
		"expose_to_cloud_redhat_com_name": longString,
		"expose_to_cloud_redhat_com_age":  45}

	_, err := Sanctify(data)
	if !strings.Contains(err.Error(), "Artifacts is greater than 1024 bytes") {
		t.Error("Failed message does not match")
	}
}
func TestNoArtifact(t *testing.T) {
	longString := strings.Repeat("na", 512)
	data := map[string]interface{}{
		"name": longString,
		"age":  45}

	result, err := Sanctify(data)
	if err != nil {
		t.Error("Encountered error")
	}
	_, ok := result["name"]
	if ok {
		t.Errorf("name key should not be included in artifact")
	}
}
