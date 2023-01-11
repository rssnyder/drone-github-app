// Copyright 2020 the Drone Authors. All rights reserved.
// Use of this source code is governed by the Blue Oak Model License
// that can be found in the LICENSE file.

package plugin

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
)

func writeCard(path, schema string, card interface{}) {
	data, _ := json.Marshal(map[string]interface{}{
		"schema": schema,
		"data":   card,
	})
	switch {
	case path == "/dev/stdout":
		writeCardTo(os.Stdout, data)
	case path == "/dev/stderr":
		writeCardTo(os.Stderr, data)
	case path != "":
		ioutil.WriteFile(path, data, 0644)
	}
}

func writeCardTo(out io.Writer, data []byte) {
	encoded := base64.StdEncoding.EncodeToString(data)
	io.WriteString(out, "\u001B]1338;")
	io.WriteString(out, encoded)
	io.WriteString(out, "\u001B]0m")
	io.WriteString(out, "\n")
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
