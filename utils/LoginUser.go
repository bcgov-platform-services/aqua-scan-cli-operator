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
	"k8s.io/apimachinery/pkg/api/errors"
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
		err := aa.Login()

		if err != nil {
			panic(err)
		}
	}

	return aa.jwt
}

func (aa *AquaAuth) Login() error {
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
		return errors.NewInternalError(err)
	}

	defer res.Body.Close()

	var jsonData LoginRes
	body, jsonErr := ioutil.ReadAll(res.Body)

	if jsonErr != nil {
		return jsonErr
	}

	json.Unmarshal(body, &jsonData)

	if res.StatusCode == 200 {
		aa.jwt = jsonData.Token

		exp := JwtPayload{}
		token, decodeErr := jwt.Decode([]byte(jsonData.Token))

		if decodeErr != nil {
			return decodeErr
		}

		json.Unmarshal(token.Payload, &exp)

		aa.exp = exp.Exp
		return nil
	} else {
		// failure operator needs to quit
		e := fmt.Errorf("failed to login to Aqua, returned status code was %v", res.StatusCode)
		return e
	}
}
