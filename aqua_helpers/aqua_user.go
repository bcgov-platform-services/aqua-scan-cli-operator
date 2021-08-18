package aqua

import (
	"bytes"
	"encoding/json"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/bcgov-platform-services/aqua-scan-cli-operator/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type User struct {
	Name string
	Role
	Password string
}

type aquaResponseJson struct {
	Message string `json:"message"`
}

func DeleteAquaAccount(reqLogger *log.DelegatingLogger, accountName string) error {
	reqLogger.Info("Deleting user %v in aqua", accountName)

	aquaAuth := utils.GetAquaAuth()
	jwt, jwtErr := aquaAuth.GetJWT()
	if jwtErr != nil {
		reqLogger.Error(jwtErr, "Failed to login to Aqua")
		return jwtErr
	}

	reqUrl := os.Getenv("AQUA_URL") + "/api/v1/users/" + accountName
	client := &http.Client{}
	req, _ := http.NewRequest("DELETE", reqUrl, nil)
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("Accept", "application/json")

	res, err := client.Do(req)

	if err != nil {
		reqLogger.Error(err, "Failed request to DELETE /api/v1/users %v from aqua", accountName)
		return err
	}

	if res.StatusCode != 204 {
		reqLogger.Error(err, "Failed to DELETE /api/v1/users %v from aqua", accountName)
		return errors.NewBadRequest("Failed to DELETE user from aqua")
	}
	reqLogger.Info("User %v deleted", accountName)
	return nil
}

func CreateAquaAccount(reqLogger *log.DelegatingLogger, user User) error {
	reqLogger.Info("Creating user %v in aqua", user.Name)

	aquaAuth := utils.GetAquaAuth()
	jwt, jwtErr := aquaAuth.GetJWT()
	if jwtErr != nil {
		reqLogger.Error(jwtErr, "Failed to login to Aqua")
		return jwtErr
	}

	b, fileErr := ioutil.ReadFile("../templates/User.json.tmpl")

	if fileErr != nil {
		reqLogger.Error(fileErr, "Failed to read template file User.json.tmpl")
		return fileErr
	}

	ut, templateErr := template.New("User").Parse(string(b))

	if templateErr != nil {
		reqLogger.Error(templateErr, "Failed to parse template file User.json.tmpl")
		return templateErr
	}

	var userBuffer bytes.Buffer
	ut.Execute(&userBuffer, user)

	reqUrl := os.Getenv("AQUA_URL") + "/api/v1/users/"
	client := &http.Client{}
	req, _ := http.NewRequest("POST", reqUrl, &userBuffer)
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("Content-Type", "application/json")

	res, err := client.Do(req)

	if err != nil {
		reqLogger.Error(err, "Failed request to POST /api/v1/ %v from aqua", user.Name)
		return err
	}

	var jsonData aquaResponseJson
	body, _ := ioutil.ReadAll(res.Body)

	json.Unmarshal(body, &jsonData)

	if res.StatusCode == 204 {
		reqLogger.Info("User %v created in aqua", user.Name)
		return nil
	}

	if res.StatusCode == 400 && strings.Contains(jsonData.Message, "User with username "+user.Name+" already exists") {
		reqLogger.Info("User %v already exists in aqua", user.Name)
		return nil
	}

	reqLogger.Error(err, "Failed to POST %v from aqua. Status code is %v", user.Name, res.StatusCode)
	return errors.NewBadRequest("Failed to POST user from aqua")
}
