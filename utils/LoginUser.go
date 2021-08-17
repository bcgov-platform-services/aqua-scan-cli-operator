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
	Id       string `json:"id"`
	Password string `json:"password"`
}

type LoginRes struct {
	Token string `json:"token"`
}

type JwtPayload struct {
	Exp int64 `json:"exp"`
}

func (aa *AquaAuth) GetJWT() string {
	now := time.Now().Unix()

	if now > aa.exp {
		aa.Login()
	}

	return aa.jwt
}

func (aa *AquaAuth) Login() {
	fmt.Println("Logging into Aqua")

	aquaUrl := os.Getenv("AQUA_URL")
	aquaUsername := os.Getenv("AQUA_USER")
	aquaPassword := os.Getenv("AQUA_PASSWORD")
	fmt.Println(aquaPassword, aquaUsername, aquaUrl)
	reqBody := LoginReqBody{Id: aquaUsername, Password: aquaPassword}
	buffer, _ := json.Marshal(reqBody)
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
		aa.jwt = jsonData.Token

		exp := JwtPayload{}
		token, _ := jwt.Decode([]byte(jsonData.Token))

		json.Unmarshal(token.Payload, &exp)

		aa.exp = exp.Exp
	} else {
		// failure operator needs to quit
	}
}
