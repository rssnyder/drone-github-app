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
	"strconv"
	"strings"
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
	ClientId      string `envconfig:"PLUGIN_CLIENT_ID"` // Recommended alternative to APP_ID
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

	// Repository selection (mutually exclusive)
	RepoIDs     string `envconfig:"PLUGIN_REPO_IDS"`     // Comma-separated list of repository IDs
	RepoNames   string `envconfig:"PLUGIN_REPO_NAMES"`   // Comma-separated list of repository names
	RepoIDsFile string `envconfig:"PLUGIN_REPO_IDS_FILE"` // File containing repository IDs

	// Permissions for installation token
	Permissions string `envconfig:"PLUGIN_PERMISSIONS"` // Comma-separated list of permissions (e.g., "contents:read,issues:write")
}

// AppResponse is what github returns when querying yourself
type AppResponse struct {
	ID   int    `json:"id"`
	Slug string `json:"slug"`
}

// TokenResponse is what github returns when gettting an installation token
type TokenResponse struct {
	Token              string                    `json:"token"`
	ExpiresAt          string                    `json:"expires_at"`
	Permissions        map[string]string         `json:"permissions,omitempty"`
	RepositorySelection string                   `json:"repository_selection,omitempty"`
	Repositories       []TokenResponseRepository `json:"repositories,omitempty"`
}

// TokenResponseRepository represents a repository in the token response
type TokenResponseRepository struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// JsonOutput is custom output for json file
type JsonOutput struct {
	Token           TokenResponse `json:"token"`
	Jwt             string        `json:"jwt"`
	RepositoryCount int           `json:"repository_count,omitempty"`
	Permissions     map[string]string `json:"permissions,omitempty"`
}

// Exec executes the plugin.
func Exec(ctx context.Context, args Args) (err error) {

	if args.AppId == "" && args.ClientId == "" {
		return errors.New("either app_id or client_id needs to be set")
	}

	if args.AppId != "" && args.ClientId != "" {
		return errors.New("only one of app_id or client_id should be set, not both. Prefer client_id for future GHEC with Data Residency compatibility.")
	}

	// Validate repository selection parameters
	err = validateRepositoryArgs(args)
	if err != nil {
		return err
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

	// Determine the issuer - use ClientId if provided, otherwise AppId
	issuer := args.AppId
	if args.ClientId != "" {
		issuer = args.ClientId
	}

	builtToken := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Minute * time.Duration(10)).Unix(),
		"iss": issuer,
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
		// Parse repository data if any repository selection is specified
		var repoData map[string]interface{}
		if args.RepoIDs != "" || args.RepoNames != "" || args.RepoIDsFile != "" {
			repoData, err = parseRepositoryData(args)
			if err != nil {
				return err
			}
		}
		
		// Parse permissions if provided
		var permissions map[string]string
		if args.Permissions != "" {
			permissions, err = parsePermissions(args.Permissions)
			if err != nil {
				return err
			}
		}
		
		tokenData, err = installationToken(jwtSigned, args.Installation, repoData, permissions)
		if err != nil {
			return err
		}

		// Log token information including repository details
		logMsg := fmt.Sprintf("token received, expires %s", tokenData.ExpiresAt)
		if len(tokenData.Repositories) > 0 {
			logMsg += fmt.Sprintf(", repositories: %d", len(tokenData.Repositories))
			for _, repo := range tokenData.Repositories {
				log.Println(fmt.Sprintf("  - %s (ID: %d)", repo.Name, repo.ID))
			}
		}
		if len(tokenData.Permissions) > 0 {
			logMsg += ", permissions:"
			for resource, permission := range tokenData.Permissions {
				logMsg += fmt.Sprintf(" %s:%s", resource, permission)
			}
		}
		log.Println(logMsg)
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
			Token:           tokenData,
			Jwt:             jwtSigned,
			RepositoryCount: len(tokenData.Repositories),
			Permissions:     tokenData.Permissions,
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

// validateRepositoryArgs validates that repository selection arguments are mutually exclusive
// and that installation is required when repository selection is used
func validateRepositoryArgs(args Args) error {
	repoArgsCount := 0
	if args.RepoIDs != "" {
		repoArgsCount++
	}
	if args.RepoNames != "" {
		repoArgsCount++
	}
	if args.RepoIDsFile != "" {
		repoArgsCount++
	}

	if repoArgsCount > 1 {
		return errors.New("only one of repo_ids, repo_names, or repo_ids_file can be specified")
	}

	if repoArgsCount > 0 && args.Installation == "" {
		return errors.New("installation must be specified when using repository selection")
	}

	return nil
}

// parseRepositoryData parses repository data from the various input sources
// Returns a map with either "repository_ids" ([]int) or "repositories" ([]string)
func parseRepositoryData(args Args) (map[string]interface{}, error) {
	var items []string
	useNames := false

	if args.RepoIDs != "" {
		items = strings.Split(strings.TrimSpace(args.RepoIDs), ",")
	} else if args.RepoNames != "" {
		items = strings.Split(strings.TrimSpace(args.RepoNames), ",")
		useNames = true
	} else if args.RepoIDsFile != "" {
		content, err := os.ReadFile(args.RepoIDsFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read repo_ids_file: %v", err)
		}
		// Split by newlines and commas, filter empty strings
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				if strings.Contains(line, ",") {
					// Split comma-separated values on this line
					lineItems := strings.Split(line, ",")
					for _, item := range lineItems {
						item = strings.TrimSpace(item)
						if item != "" {
							items = append(items, item)
						}
					}
				} else {
					items = append(items, line)
				}
			}
		}
	}

	if len(items) == 0 {
		return nil, nil
	}

	if len(items) > 500 {
		return nil, errors.New("repository list cannot contain more than 500 entries")
	}

	if useNames {
		// Return repository names as strings (repo name only, not owner/repo)
		var repoNames []string
		for _, name := range items {
			name = strings.TrimSpace(name)
			if name != "" {
				// Validate that this is just a repo name, not owner/repo format
				if strings.Contains(name, "/") {
					return nil, fmt.Errorf("repository name '%s' should not include owner - use just the repository name (e.g., 'hello-world' not 'owner/hello-world')", name)
				}
				repoNames = append(repoNames, name)
			}
		}
		return map[string]interface{}{"repositories": repoNames}, nil
	} else {
		// Convert string IDs to integers
		var repoIDs []int
		for _, idStr := range items {
			idStr = strings.TrimSpace(idStr)
			if idStr == "" {
				continue
			}
			id, err := strconv.Atoi(idStr)
			if err != nil {
				return nil, fmt.Errorf("invalid repository ID '%s': %v", idStr, err)
			}
			repoIDs = append(repoIDs, id)
		}
		return map[string]interface{}{"repository_ids": repoIDs}, nil
	}
}

// parsePermissions parses permissions from comma-separated string in format "resource:permission"
func parsePermissions(permissionsStr string) (map[string]string, error) {
	if permissionsStr == "" {
		return nil, nil
	}

	permissions := make(map[string]string)
	items := strings.Split(strings.TrimSpace(permissionsStr), ",")

	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}

		parts := strings.Split(item, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid permission format '%s': expected 'resource:permission'", item)
		}

		resource := strings.TrimSpace(parts[0])
		permission := strings.TrimSpace(parts[1])

		if resource == "" || permission == "" {
			return nil, fmt.Errorf("invalid permission format '%s': resource and permission cannot be empty", item)
		}

		permissions[resource] = permission
	}

	return permissions, nil
}
