package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/kataras/jwt"
)

type AquaAuth struct {
	jwt string
	exp int64
}

type LoginReqBody struct {
	id       string
	password string
}

type LoginRes struct {
	token string
}

func (aa *AquaAuth) GetJWT() string {
	now := time.Now().Unix()

	if now > aa.exp {
		// login again
	}

	return aa.jwt
}

func (aa *AquaAuth) Login() {
	fmt.Println("Logging into Aqua")

	aquaUrl := os.Getenv("AQUA_URL")
	aquaUsername := os.Getenv("AQUA_USER")
	aquaPassword := os.Getenv("AQUA_PASSWORD")

	reqBody := LoginReqBody{id: aquaUsername, password: aquaPassword}
	buffer, err := json.Marshal(reqBody)
	reqUrl := aquaUrl + "/api/v1/login"
	client := &http.Client{}
	req, _ := http.NewRequest("POST", reqUrl, bytes.NewBuffer(buffer))

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	res, err := client.Do(req)

	if err != nil {
		// reqLogger.Error(err, "Failed request to POST to /api/v1/login in aqua")
		// return err
	}
	defer res.Body.Close()

	var jsonData LoginRes
	body, _ := ioutil.ReadAll(res.Body)

	json.Unmarshal(body, &jsonData)

	if res.StatusCode == 200 {
		aa.jwt = jsonData.token
		token, err := jwt.Decode([]byte(jsonData.token))
	} else {

	}
}
