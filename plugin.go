package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type AppResponse struct {
	ID   int    `json:"id"`
	Slug string `json:"slug"`
}

type TokenResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
}

type JsonOutput struct {
	Token TokenResponse `json:"token"`
	Jwt   string        `json:"jwt"`
}

func main() {
	var err error

	appId := os.Getenv("PLUGIN_APP_ID")
	pem := os.Getenv("PLUGIN_PEM")
	pemFile := os.Getenv("PLUGIN_PEM_FILE")
	installation := os.Getenv("PLUGIN_INSTALLATION")
	jwtFile := os.Getenv("PLUGIN_JWT_FILE")
	tokenFile := os.Getenv("PLUGIN_TOKEN_FILE")
	jsonFile := os.Getenv("PLUGIN_JSON_FILE")

	if appId == "" {
		log.Fatal("app id needs to be set")
	}

	var bPem []byte
	if pem == "" {
		if pemFile == "" {
			log.Fatal("one of pem or pam_file must be set")
		}
		bPem, err = os.ReadFile(pemFile)
		if err != nil {
			fmt.Print(err)
		}
	} else {
		bPem = []byte(pem)
	}

	if len(bPem) == 0 {
		log.Fatal("unable to parse pem")
	}

	signKey, err := jwt.ParseRSAPrivateKeyFromPEM(bPem)
	if err != nil {
		log.Fatal(err)
	}

	builtToken := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Minute * time.Duration(10)).Unix(),
		"iss": appId,
	})

	jwtSigned, err := builtToken.SignedString(signKey)
	if err != nil {
		log.Fatal(err)
	}

	appData, err := validateJWT(jwtSigned)
	if err != nil {
		log.Fatal(err)
	}

	log.Println(fmt.Sprintf("authenticated as %s", appData.Slug))

	var tokenData TokenResponse
	if installation != "" {
		tokenData, err = installationToken(jwtSigned, installation)
		if err != nil {
			log.Fatal(err)
		}

		log.Println(fmt.Sprintf("token recived, expires %s", tokenData.ExpiresAt))
	}

	if jwtFile != "" {
		err = os.WriteFile(jwtFile, []byte(jwtSigned), 0600)
		if err != nil {
			log.Fatal(err)
		}
	}

	if tokenFile != "" {
		err = os.WriteFile(tokenFile, []byte(tokenData.Token), 0600)
		if err != nil {
			log.Fatal(err)
		}
	}

	if jsonFile != "" {
		jsonData := JsonOutput{
			Token: tokenData,
			Jwt:   jwtSigned,
		}
		file, err := json.MarshalIndent(jsonData, "", " ")
		if err != nil {
			log.Fatal(err)
		}

		err = os.WriteFile(jsonFile, file, 0600)
		if err != nil {
			log.Fatal(err)
		}
	}
}

// validateJWT retives information on the github app to verify the jwt is valid
func validateJWT(jwt string) (response AppResponse, err error) {
	req, err := http.NewRequest("GET", "https://api.github.com/app", nil)
	if err != nil {
		return
	}

	req.Header.Add("Accept", "application/vnd.github+json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", jwt))

	client := &http.Client{}
	resp, err := client.Do(req)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return
	}

	return
}

// installationToken returns a github pat for the given installation scope
func installationToken(jwt, installation string) (response TokenResponse, err error) {
	req, err := http.NewRequest("POST", fmt.Sprintf("https://api.github.com/app/installations/%s/access_tokens", installation), nil)
	if err != nil {
		return
	}

	req.Header.Add("Accept", "application/vnd.github+json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", jwt))

	client := &http.Client{}
	resp, err := client.Do(req)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return
	}

	return
}
