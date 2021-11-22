package utils

import (
	"errors"
	"os"
	"regexp"
	"strconv"
	"testing"

	ctrl "sigs.k8s.io/controller-runtime"
)

func mockGetJwtFail() (string, error) {
	return "", errors.New("this func failed")
}

func mockGetJwtPass() (string, error) {
	return "", nil
}

func TestSetEnvForAsaLoginCheck(t *testing.T) {
	SetEnvForAsaLoginCheck(mockGetJwtFail, ctrl.Log)
	b, _ := strconv.ParseBool(os.Getenv("ASA_LOGIN_CHECK_DID_FAIL"))
	if !b {
		t.Errorf("SetEnvForAsaLoginCheck was supposed to return true but got %v", os.Getenv("ASA_LOGIN_CHECK_DID_FAIL"))
	}

	SetEnvForAsaLoginCheck(mockGetJwtPass, ctrl.Log)
	c, _ := strconv.ParseBool(os.Getenv("ASA_LOGIN_CHECK_DID_FAIL"))

	if c {
		t.Errorf("SetEnvForAsaLoginCheck was supposed to return false but got %v", os.Getenv("ASA_LOGIN_CHECK_DID_FAIL"))
	}
}

func TestUtilsGetTechnicalContact(t *testing.T) {
	technicalContactAnnotation := "- role: Product Owner\n  email: matt.damon@gov.bc.ca\n  rocketchat:\n- role: Technical Lead\n  email: patrick.simonian@gov.bc.ca\n  rocketchat:\n"
	tc := GetTechnicalContactFromAnnotation(technicalContactAnnotation)

	if tc != "patrick.simonian@gov.bc.ca" {
		t.Errorf("GetTechnicalContactFromAnnotation was supposed to return patrick.simonian@gov.bc.ca but got %v", tc)
	}
}

func TestUtilsGeneratePassword(t *testing.T) {
	pw1 := GeneratePassword(8, false, false, false)
	pw2 := GeneratePassword(8, true, true, true)
	if len(pw1) != 8 {
		t.Errorf("GeneratePassword was support to return a password with length 8 but got %v", len(pw1))
	}
	bPw2 := []byte(pw2)
	matchedPw1, _ := regexp.Match("[a-z]{8}", []byte(pw1))
	numCheck, _ := regexp.Compile("[0-9]")
	upperCheck, _ := regexp.Compile("[A-Z]")
	symbolCheck, _ := regexp.Compile("[!@#$]")

	if !matchedPw1 {
		t.Errorf("GeneratePassword was supposed to return a password with only lower case but got %v", pw1)
	}

	if !numCheck.Match(bPw2) {
		t.Errorf("GeneratePassword was supposed to return a password with at least 1 number but got %v", pw2)
	}

	if !upperCheck.Match(bPw2) {
		t.Errorf("GeneratePassword was supposed to return a password with at least 1 uppercase letter but got %v", pw2)
	}

	if !symbolCheck.Match(bPw2) {
		t.Errorf("GeneratePassword was supposed to return a password with at least 1 symbol but got %v", pw2)
	}
}
