![Docker Pulls](https://img.shields.io/docker/pulls/rssnyder/drone-github-app)

A plugin to get a jwt or installation token for a github app.

# Usage

The following settings changes this plugin's behavior.

* APP_ID (required) github app id.
* PEM (optional) rsa private key.
* PEM_FILE (optional) local file path of rsa private key.
* PEM_B64 (optional) local file path of base64 encoded rsa private key.
* INSTALLATION (optional) installation id. required if wanting a token.
* JWT_FILE (optional) output file for jwt.
* TOKEN_FILE (optional) output file for token.
* JSON_FILE (optional) output file for both jwt and token in json.

**one of PEM, PEM_FILE, PEM_B64 is required**

Below is an example `.drone.yml` that uses this plugin.

```yaml
kind: pipeline
name: default

steps:
- name: run rssnyder/drone-github-app plugin
  image: rssnyder/drone-github-app
  pull: if-not-exists
  settings:
    APP_ID: "264043"
    INSTALLATION: "31437931"
    PEM_B64:
      from_secret: github_app_b64
    JSON_FILE: output.json
```

Below is an example harness step that uses this plugin.

```yaml
- step:
    type: Plugin
    name: get token
    identifier: get_token
    spec:
    connectorRef: dockerhub
    image: rssnyder/drone-github-app
    settings:
        APP_ID: "264043"
        INSTALLATION: "31437931"
        PEM_B64: <+secrets.getValue("github_app_b64")>
        JSON_FILE: output.json
```

# Building

Build the plugin binary:

```text
scripts/build.sh
```

Build the plugin image:

```text
docker build -t rssnyder/drone-github-app -f docker/Dockerfile .
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

If you need to view the intallations for your app, use the plugin to get a JWT and make the following HTTP call:

```shell
curl \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer $JWT"\
  https://api.github.com/app/installations
```

test commit
