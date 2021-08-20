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

type Role struct {
	Name        string
	Description string
	ApplicationScope
}

func DeleteAquaRole(reqLogger *log.DelegatingLogger, role string) error {
	reqLogger.Info("Deleting role %v in aqua", "role", role)

	aquaAuth := GetAquaAuth()
	jwt, jwtErr := aquaAuth.GetJWT()
	if jwtErr != nil {
		reqLogger.Error(jwtErr, "Failed to login to Aqua")
		return jwtErr
	}

	reqUrl := os.Getenv("AQUA_URL") + "/api/v2/access_management/roles/" + role
	client := &http.Client{}

	req, _ := http.NewRequest("DELETE", reqUrl, nil)
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	res, err := client.Do(req)

	if err != nil {
		reqLogger.Error(err, "Failed request to DELETE to /api/v2/access_management/roles/ in aqua")
		return err
	}

	if res.StatusCode != 204 {
		e := errors.NewBadRequest(fmt.Sprintf("Error: Could not delete role, the response status from aqua was %v", res.StatusCode))
		return e
	}
	return nil
}

func CreateAquaRole(reqLogger *log.DelegatingLogger, role Role) error {
	reqLogger.Info("Creating Role %v in aqua", "role", role.Name)

	aquaAuth := GetAquaAuth()
	jwt, jwtErr := aquaAuth.GetJWT()
	if jwtErr != nil {
		reqLogger.Error(jwtErr, "Failed to login to Aqua")
		return jwtErr
	}
	wd, _ := os.Getwd()
	path := filepath.Join(wd, "./templates/Role.json.tmpl")

	b, fileErr := ioutil.ReadFile(path)

	if fileErr != nil {
		reqLogger.Error(fileErr, "Failed to read template file Role.json.tmpl")
		return fileErr
	}

	ut, templateErr := template.New("Role").Parse(string(b))

	if templateErr != nil {
		reqLogger.Error(templateErr, "Failed to parse template file Role.json.tmpl")
		return templateErr
	}

	var roleBuffer bytes.Buffer
	ut.Execute(&roleBuffer, role)

	reqUrl := os.Getenv("AQUA_URL") + "/api/v2/access_management/roles"
	client := &http.Client{}
	req, clientErr := http.NewRequest("POST", reqUrl, &roleBuffer)

	if clientErr != nil {
		reqLogger.Error(clientErr, "unable to create client")
		return clientErr
	}

	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	res, err := client.Do(req)

	if err != nil {
		reqLogger.Error(err, "Failed request to POST to  /api/v2/access_management/roles in aqua")
		return err
	}
	defer res.Body.Close()

	var jsonData aquaResponseJson
	body, _ := ioutil.ReadAll(res.Body)

	json.Unmarshal(body, &jsonData)

	if res.StatusCode == 404 && strings.Contains(jsonData.Message, "role "+role.Name+" already exists") || res.StatusCode == 201 {
		return nil
	} else {
		e := errors.NewBadRequest(fmt.Sprintf("Error: Could not create role, the response status from aqua was %v", res.StatusCode))

		reqLogger.Error(e, "Unable to create Role")
		return e
	}
}
