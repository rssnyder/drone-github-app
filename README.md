# drone-github-app

![Docker Pulls](https://img.shields.io/docker/pulls/rssnyder/drone-github-app)
![Docker Image Version (latest by date)](https://img.shields.io/docker/v/rssnyder/drone-github-app?sort=date)

drone/harness plugin to get a jwt or installation token for a github app

## inputs

- APP_ID: (required) github app id
- PEM: rsa private key
- PEM_FILE: local file path of rsa private key
- PEM_B64: base64 encoded rsa private key
- INSTALLATION: installation id
- JWT_FILE: output file for jwt
- TOKEN_FILE: output file for token
- JSON_FILE: output file for both jwt and token in json

## useage

### drone

```yaml
- name: run
  image: rssnyder/drone-github-app
  settings:
    APP_ID: "264043"
    INSTALLATION: "31437931"
    PEM_B64:
      from_secret: github_app_b64
    JSON_FILE: output.json
```

### harness

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

## build

`docker build -t drone-github-app .`

`go build -o drone-github-app`
