// Copyright 2020 the Drone Authors. All rights reserved.
// Use of this source code is governed by the Blue Oak Model License
// that can be found in the LICENSE file.

package plugin

import (
	"bytes"
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
// If repoData is provided, the token will be scoped to those repositories
// If permissions is provided, the token will have those specific permissions
func installationToken(jwt, installation string, repoData map[string]interface{}, permissions map[string]string) (response TokenResponse, err error) {
	var reqBody *bytes.Buffer
	
	// Build request data with repositories and/or permissions if provided
	// Expected JSON structure:
	// {
	//   "repositories": ["repo1", "repo2"] OR "repository_ids": [1001, 1002],
	//   "permissions": {"contents": "read", "issues": "write"}
	// }
	reqData := make(map[string]interface{})
	
	if repoData != nil {
		for key, value := range repoData {
			reqData[key] = value
		}
	}
	
	if permissions != nil && len(permissions) > 0 {
		reqData["permissions"] = permissions
	}
	
	if len(reqData) > 0 {
		jsonData, err := json.Marshal(reqData)
		if err != nil {
			return response, err
		}
		reqBody = bytes.NewBuffer(jsonData)
	} else {
		reqBody = bytes.NewBuffer([]byte{})
	}
	
	req, err := http.NewRequest("POST", fmt.Sprintf("https://api.github.com/app/installations/%s/access_tokens", installation), reqBody)
	if err != nil {
		return
	}

	req.Header.Add("Accept", "application/vnd.github+json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", jwt))
	if len(reqData) > 0 {
		req.Header.Add("Content-Type", "application/json")
	}

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
