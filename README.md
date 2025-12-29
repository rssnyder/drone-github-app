![Docker Pulls](https://img.shields.io/docker/pulls/rssnyder/drone-github-app)

A plugin to get a jwt or installation token for a github app.

# Usage

The following settings changes this plugin's behavior.

## Authentication Parameters
* APP_ID (optional) github app id (legacy, use CLIENT_ID instead).
* CLIENT_ID (optional, recommended) github app client id string.
* PEM (optional) rsa private key.
* PEM_FILE (optional) local file path of rsa private key.
* PEM_B64 (optional) base64 encoded rsa private key.

## Installation & Repository Scoping
* INSTALLATION (optional) installation id. required if wanting a token.
* REPO_IDS (optional) comma-separated list of repository IDs to scope token to.
* REPO_NAMES (optional) comma-separated list of repository names to scope token to.
* REPO_IDS_FILE (optional) file containing repository IDs (newline or comma separated).
* PERMISSIONS (optional) comma-separated permissions in format "resource:permission" (e.g., "contents:read,issues:write").

## Output Options
* JWT_FILE (optional) output file for jwt.
* TOKEN_FILE (optional) output file for token.
* JSON_FILE (optional) output file for both jwt and token with metadata.
* JWT_SECRET (optional) harness secret id for setting jwt as a secret
* TOKEN_SECRET (optional) harness secret id for setting token as a secret
* JSON_SECRET (optional) harness secret id for setting json as a secret
* SECRET_MANAGER (optional, defaults to harness secrets manager) harness secret manager to use

If setting harness secrets, you also need to set the follow in the environment for the step:

- HARNESS_PLATFORM_API_KEY: harness nextgen api key
- HARNESS_ACCOUNT_ID: harness account id
- HARNESS_PLATFORM_ORGANIZATION: organization id
- HARNESS_PLATFORM_PROJECT: project id

## Requirements

**Authentication**: Either `APP_ID` or `CLIENT_ID` is required (prefer `CLIENT_ID`).
**Private Key**: One of `PEM`, `PEM_FILE`, or `PEM_B64` is required.
**Repository Scoping**: Only one of `REPO_IDS`, `REPO_NAMES`, or `REPO_IDS_FILE` can be used to limit repo access.
**Permission Scoping**: `PERMISSIONS` can be used to scope down token permissions.
**Installation Token**: `INSTALLATION` is required when requesting tokens.

## Examples

### Basic JWT Generation

```yaml
kind: pipeline
name: default

steps:
- name: generate github app jwt
  image: rssnyder/drone-github-app
  pull: if-not-exists
  settings:
    CLIENT_ID: "Iv1.a629723bfa6c7c08"
    PEM_B64:
      from_secret: github_app_b64
    JWT_FILE: app.jwt
```

### Installation Token with Repository Scoping

```yaml
kind: pipeline
name: default

steps:
- name: run rssnyder/drone-github-app plugin
  image: rssnyder/drone-github-app
  pull: if-not-exists
  settings:
    CLIENT_ID: "Iv1.a629723bfa6c7c08"
    INSTALLATION: "31437931"
    REPO_IDS: "1001,1002,1003"
    PERMISSIONS: "contents:read,issues:write,pull_requests:read"
    PEM_B64:
      from_secret: github_app_b64
    JSON_FILE: output.json
```

### Repository Names with Custom Permissions

```yaml
kind: pipeline
name: default

steps:
- name: get token for specific repos
  image: rssnyder/drone-github-app
  pull: if-not-exists
  settings:
    CLIENT_ID: "Iv1.a629723bfa6c7c08"
    INSTALLATION: "31437931"
    REPO_NAMES: "hello-world,my-awesome-repo"
    PERMISSIONS: "contents:write,actions:read"
    PEM_FILE: /secrets/github-app.pem
    TOKEN_FILE: github_token.txt
```

### Harness CI Example

```yaml
- step:
    type: Plugin
    name: get token
    identifier: get_token
    spec:
      connectorRef: dockerhub
      image: rssnyder/drone-github-app
      settings:
        CLIENT_ID: "Iv1.a629723bfa6c7c08"
        INSTALLATION: "31437931"
        REPO_IDS: "1001,1002"
        PERMISSIONS: "contents:read,metadata:read"
        PEM_B64: <+secrets.getValue("github_app_b64")>
        JSON_FILE: output.json
```

### Using Repository IDs from File

```yaml
kind: pipeline
name: default

steps:
- name: get token from repo file
  image: rssnyder/drone-github-app
  pull: if-not-exists
  settings:
    CLIENT_ID: "Iv1.a629723bfa6c7c08"
    INSTALLATION: "31437931"
    REPO_IDS_FILE: ./repo_list.txt
    PERMISSIONS: "issues:write,pull_requests:write"
    PEM_B64:
      from_secret: github_app_b64
    TOKEN_SECRET: github_installation_token
```

## JSON Output Format

When using `JSON_FILE` or `JSON_SECRET`, the output includes token information:

```json
{
  "token": {
    "token": "ghs_12345ABCDE98765",
    "expires_at": "2016-07-11T22:14:10Z",
    "permissions": {
      "contents": "read",
      "issues": "write"
    },
    "repository_selection": "selected",
    "repositories": [
      {
        "id": 1296269,
        "name": "Hello-World"
      }
    ]
  },
  "jwt": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "repository_count": 1,
  "permissions": {
    "contents": "read",
    "issues": "write"
  }
}
```

## Repository ID File Format

When using `REPO_IDS_FILE`, the file can contain repository IDs in various formats:

```text
# One per line
1001
1002
1003

# Comma-separated on same line
1004,1005,1006

# Mixed format
1007
1008,1009
1010
```

# Building

Build the plugin binary:

```text
scripts/build.sh
```

Build the plugin image:

```text
docker build -t rssnyder/drone-github-app -f docker/Dockerfile.linux.amd64 .
```

# Testing

Execute the plugin from your current working directory:

```text
docker run --rm -e PLUGIN_PARAM1=foo -e PLUGIN_PARAM2=bar \
  -e DRONE_COMMIT_SHA=8f51ad7884c5eb69c11d260a31da7a745e6b78e2 \
  -e DRONE_COMMIT_BRANCH=master \
  -e DRONE_BUILD_NUMBER=43 \
  -e DRONE_BUILD_STATUS=success \
  -w /drone/src \
  -v $(pwd):/drone/src \
  rssnyder/drone-github-app
```

## Installations

If you need to view the installations for your app, use the plugin to get a JWT and make the following HTTP call:

```shell
curl \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer $JWT"\
  https://api.github.com/app/installations
```
