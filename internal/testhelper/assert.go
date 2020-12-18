package testhelper

import (
	"strings"
	"testing"
)

// Assert compares whether actual value equals expected.
func Assert(t *testing.T, description string, expected interface{}, actual interface{}) {
	if actual != expected {
		t.Errorf("%s not equal. Expected %v but actual %v", description, expected, actual)
	}
}

// AssertNoError checks whether no error is returned.
func AssertNoError(t *testing.T, description string, actual error) {
	if actual != nil {
		t.Errorf("%s expects no error but actual %v", description, actual)
	}
}

// AssertErrorExists checks whether an error is returned from the test.
func AssertErrorExists(t *testing.T, description string, actual error) {
	if actual == nil {
		t.Errorf("%s expects an error but not received", description)
	}
}

// AssertErrorMessage checks whether an expected message is included in error.
func AssertErrorMessage(t *testing.T, description string, expectedMessage string, actual error) {
	if actual == nil {
		t.Errorf("%s expects an error but not received", description)
	} else if !strings.Contains(actual.Error(), expectedMessage) {
		t.Errorf("%s expects the error contains %s, but actual %v", description, expectedMessage, actual)
	}
}
