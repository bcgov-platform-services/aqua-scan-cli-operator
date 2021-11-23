package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type PermissionSet struct {
	Name               string
	Description        string
	TechnicalLeadEmail string
}

func DeleteAquaPermissionSet(reqLogger *log.DelegatingLogger, permissionSet string) error {
	reqLogger.Info("Deleting permissionSet %v in aqua", "permissionSet", permissionSet)

	aquaAuth := GetAquaAuth()
	jwt, jwtErr := aquaAuth.GetJWT()
	if jwtErr != nil {
		reqLogger.Error(jwtErr, "Failed to login to Aqua")
		return jwtErr
	}

	reqPayload, jsonErr := json.Marshal([]string{permissionSet})

	if jsonErr != nil {
		reqLogger.Error(jsonErr, "Failed to marshal json %v", []string{permissionSet})
		return jsonErr
	}

	reqUrl := os.Getenv("AQUA_URL") + "/api/v2/access_management/permissions/" + permissionSet
	client := &http.Client{}

	req, _ := http.NewRequest("POST", reqUrl, bytes.NewBuffer(reqPayload))
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	res, err := client.Do(req)

	if err != nil {
		reqLogger.Error(err, "Failed request to DELETE to /api/v2/access_management/permissions/"+permissionSet+" in aqua")
		return err
	}

	if res.StatusCode != 204 && res.StatusCode != 404 {
		e := errors.NewBadRequest(fmt.Sprintf("Error: Could not delete Permission Set, the response status from aqua was %v", res.StatusCode))
		return e
	}
	return nil
}

func CreateAquaPermissionSet(reqLogger *log.DelegatingLogger, permissionSet PermissionSet) error {
	wd, _ := os.Getwd()

	reqLogger.Info("Creating permissionSet %v in aqua", "Name", permissionSet.Name)
	aquaAuth := GetAquaAuth()
	jwt, jwtErr := aquaAuth.GetJWT()
	if jwtErr != nil {
		reqLogger.Error(jwtErr, "Failed to login to Aqua")
		return jwtErr
	}

	path := filepath.Join(wd, "./templates/PermissionSet.json.tmpl")

	b, fileErr := ioutil.ReadFile(path)

	if fileErr != nil {
		reqLogger.Error(fileErr, "Failed to read template file PermissionSet.json.tmpl")
		return fileErr
	}

	ut, templateErr := template.New("PermissionSet").Parse(string(b))

	if templateErr != nil {
		reqLogger.Error(templateErr, "Failed to parse template file PermissionSet.json.tmpl")
		return templateErr
	}

	var permissionSetBuffer bytes.Buffer
	ut.Execute(&permissionSetBuffer, permissionSet)

	reqUrl := os.Getenv("AQUA_URL") + "api/v2/access_management/permissions"
	client := &http.Client{}
	req, clientErr := http.NewRequest("POST", reqUrl, bytes.NewBuffer(permissionSetBuffer.Bytes()))

	if clientErr != nil {
		reqLogger.Error(clientErr, "unable to create client")
		return clientErr
	}

	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	res, err := client.Do(req)

	if err != nil {
		reqLogger.Error(err, "Failed request to POST to  api/v2/access_management/permissions in aqua")
		return err
	}
	defer res.Body.Close()

	var jsonData AquaResponseJson
	body, _ := ioutil.ReadAll(res.Body)

	json.Unmarshal(body, &jsonData)

	// idempotency check
	if res.StatusCode == 404 && strings.Contains(jsonData.Message, "permission "+permissionSet.Name+" already exists") || res.StatusCode == 201 {
		return nil
	} else {
		e := errors.NewBadRequest(fmt.Sprintf("Error: Could not create PermissionSet, the response status from aqua was %v", res.StatusCode))

		reqLogger.Error(e, "Unable to create PermissionSet")
		return e
	}
}
