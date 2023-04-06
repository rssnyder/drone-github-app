// Copyright 2020 the Drone Authors. All rights reserved.
// Use of this source code is governed by the Blue Oak Model License
// that can be found in the LICENSE file.

package plugin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/rssnyder/harness-go-utils/config"
	"github.com/rssnyder/harness-go-utils/secrets"
)

// Args provides plugin execution arguments.
type Args struct {
	Pipeline

	// Level defines the plugin log level.
	Level string `envconfig:"PLUGIN_LOG_LEVEL"`

	// TODO replace or remove
	AppId         string `envconfig:"PLUGIN_APP_ID"`
	Pem           string `envconfig:"PLUGIN_PEM"`
	PemFile       string `envconfig:"PLUGIN_PEM_FILE"`
	PemB64        string `envconfig:"PLUGIN_PEM_B64"`
	Installation  string `envconfig:"PLUGIN_INSTALLATION"`
	JwtFile       string `envconfig:"PLUGIN_JWT_FILE"`
	TokenFile     string `envconfig:"PLUGIN_TOKEN_FILE"`
	JsonFile      string `envconfig:"PLUGIN_JSON_FILE"`
	JwtSecret     string `envconfig:"PLUGIN_JWT_SECRET"`
	TokenSecret   string `envconfig:"PLUGIN_TOKEN_SECRET"`
	JsonSecret    string `envconfig:"PLUGIN_JSON_SECRET"`
	SecretManager string `envconfig:"PLUGIN_SECRET_MANAGER"`
}

// AppResponse is what github returns when querying yourself
type AppResponse struct {
	ID   int    `json:"id"`
	Slug string `json:"slug"`
}

// TokenResponse is what github returns when gettting an installation token
type TokenResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
}

// JsonOutput is custom output for json file
type JsonOutput struct {
	Token TokenResponse `json:"token"`
	Jwt   string        `json:"jwt"`
}

// Exec executes the plugin.
func Exec(ctx context.Context, args Args) (err error) {

	if args.AppId == "" {
		return errors.New("app id needs to be set")
	}

	var bPem []byte
	if args.Pem != "" {
		bPem = []byte(args.Pem)
	} else if args.PemFile != "" {
		bPem, err = os.ReadFile(args.PemFile)
		if err != nil {
			fmt.Print(err)
		}
	} else if args.PemB64 != "" {
		bPem, err = base64.StdEncoding.DecodeString(args.PemB64)
		if err != nil {
			fmt.Print(err)
		}
	} else {
		return errors.New("one of pem, pam_file, or pem_b64 must be set")
	}

	if len(bPem) == 0 {
		return errors.New("unable to parse pem")
	}

	signKey, err := jwt.ParseRSAPrivateKeyFromPEM(bPem)
	if err != nil {
		return err
	}

	builtToken := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Minute * time.Duration(10)).Unix(),
		"iss": args.AppId,
	})

	jwtSigned, err := builtToken.SignedString(signKey)
	if err != nil {
		return err
	}

	appData, err := validateJWT(jwtSigned)
	if err != nil {
		return err
	}

	log.Println(fmt.Sprintf("authenticated as %s", appData.Slug))

	var tokenData TokenResponse
	if args.Installation != "" {
		tokenData, err = installationToken(jwtSigned, args.Installation)
		if err != nil {
			return err
		}

		log.Println(fmt.Sprintf("token recived, expires %s", tokenData.ExpiresAt))
	}

	if args.JwtFile != "" {
		err = os.WriteFile(args.JwtFile, []byte(jwtSigned), 0600)
		if err != nil {
			return err
		}
	}

	if args.TokenFile != "" {
		if args.Installation == "" {
			log.Println("requested TOKEN_FILE but no INSTALLATION specified, skipping")
		} else {
			err = os.WriteFile(args.TokenFile, []byte(tokenData.Token), 0600)
			if err != nil {
				return err
			}
		}
	}

	if args.JsonFile != "" {
		jsonData := JsonOutput{
			Token: tokenData,
			Jwt:   jwtSigned,
		}
		file, err := json.MarshalIndent(jsonData, "", " ")
		if err != nil {
			return err
		}

		err = os.WriteFile(args.JsonFile, file, 0600)
		if err != nil {
			return err
		}
	}

	client, hCtx := config.GetNextgenClient()
	if args.JwtSecret != "" {
		err = secrets.SetSecretText(hCtx, client, args.JwtSecret, args.JwtSecret, jwtSigned, args.SecretManager)
		if err != nil {
			return err
		}
		log.Println(fmt.Sprintf("jwt saved in %s", args.JwtSecret))
	}
	if args.TokenSecret != "" {
		err = secrets.SetSecretText(hCtx, client, args.TokenSecret, args.TokenSecret, tokenData.Token, args.SecretManager)
		if err != nil {
			return err
		}
		log.Println(fmt.Sprintf("token saved in %s", args.TokenSecret))
	}
	if args.JsonSecret != "" {
		jsonData := JsonOutput{
			Token: tokenData,
			Jwt:   jwtSigned,
		}
		file, err := json.MarshalIndent(jsonData, "", " ")
		if err != nil {
			return err
		}

		err = secrets.SetSecretText(hCtx, client, args.JsonSecret, args.JsonSecret, string(file), args.SecretManager)
		if err != nil {
			return err
		}
		log.Println(fmt.Sprintf("json saved in %s", args.JsonSecret))
	}
	return
}
