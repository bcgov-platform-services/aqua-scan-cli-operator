package utils

import (
	"testing"
)

func TestUtilsGetTechnicalContact(t *testing.T) {
	technicalContactAnnotation := "- role: Product Owner\n  email: matt.damon@gov.bc.ca\n  rocketchat:\n- role: Technical Lead\n  email: patrick.simonian@gov.bc.ca\n  rocketchat:\n"
	tc := GetTechnicalContactFromAnnotation(technicalContactAnnotation)

	if tc != "patrick.simonian@gov.bc.ca" {
		t.Errorf("GetTechnicalContactFromAnnotation was supposed to return patrick.simonian@gov.bc.ca but got %v", tc)
	}
}
