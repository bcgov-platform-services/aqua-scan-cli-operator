package aqua

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/bcgov-platform-services/aqua-scan-cli-operator/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type ApplicationScope struct {
	Name               string
	NamespacePrefix    string
	Description        string
	TechnicalLeadEmail string
}

func DeleteAquaApplicationScope(reqLogger *log.DelegatingLogger, applicationScope string) error {
	reqLogger.Info("Deleting applicationScope %v in aqua", applicationScope)

	aquaAuth := utils.GetAquaAuth()
	jwt, jwtErr := aquaAuth.GetJWT()
	if jwtErr != nil {
		reqLogger.Error(jwtErr, "Failed to login to Aqua")
		return jwtErr
	}

	reqPayload, jsonErr := json.Marshal([]string{applicationScope})

	if jsonErr != nil {
		reqLogger.Error(jsonErr, "Failed to marshal json %v", []string{applicationScope})
		return jsonErr
	}

	reqUrl := os.Getenv("AQUA_URL") + "/api/v2/access_management/scopes/delete"
	client := &http.Client{}

	req, _ := http.NewRequest("POST", reqUrl, bytes.NewBuffer(reqPayload))
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	res, err := client.Do(req)

	if err != nil {
		reqLogger.Error(err, "Failed request to POST to /api/v2/access_management/scopes/delete in aqua")
		return err
	}

	if res.StatusCode != 204 {
		e := errors.NewBadRequest(fmt.Sprintf("Error: Could not delete application scope, the response status from aqua was %v", res.StatusCode))
		return e
	}
	return nil
}

func CreateAquaApplicationScope(reqLogger *log.DelegatingLogger, appScope ApplicationScope) error {
	reqLogger.Info("Creating applicationScope %v-* in aqua", appScope.NamespacePrefix)
	aquaAuth := utils.GetAquaAuth()
	jwt, jwtErr := aquaAuth.GetJWT()
	if jwtErr != nil {
		reqLogger.Error(jwtErr, "Failed to login to Aqua")
		return jwtErr
	}

	b, fileErr := ioutil.ReadFile("../templates/ApplicationScope.json.tmpl")

	if fileErr != nil {
		reqLogger.Error(fileErr, "Failed to read template file ApplicationScope.json.tmpl")
		return fileErr
	}

	ut, templateErr := template.New("ApplicationScope").Parse(string(b))

	if templateErr != nil {
		reqLogger.Error(templateErr, "Failed to parse template file ApplicationScope.json.tmpl")
		return templateErr
	}

	var appScopeBuffer bytes.Buffer
	ut.Execute(&appScopeBuffer, appScope)

	reqUrl := os.Getenv("AQUA_URL") + " /api/v2/access_management/scopes"
	client := &http.Client{}
	req, _ := http.NewRequest("POST", reqUrl, &appScopeBuffer)
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	res, err := client.Do(req)

	if err != nil {
		reqLogger.Error(err, "Failed request to POST to  /api/v2/access_management/scopes in aqua")
		return err
	}
	defer res.Body.Close()

	var jsonData aquaResponseJson
	body, _ := ioutil.ReadAll(res.Body)

	json.Unmarshal(body, &jsonData)

	if res.StatusCode == 404 && strings.Contains(jsonData.Message, "application scope "+appScope.Name+" already exists") || res.StatusCode == 201 {
		return nil
	} else {
		e := errors.NewBadRequest(fmt.Sprintf("Error: Could not create ApplicationScope, the response status from aqua was %v", res.StatusCode))

		reqLogger.Error(e, "Unable to create ApplicationScope")
		return e
	}
}
