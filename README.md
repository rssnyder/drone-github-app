# drone-github-app

drone/harness plugin to get a jwt or installation token for a github app

## inputs

- APP_ID: (required) github app id
- PEM: rsa private key
- PEM_FILE: local file path of rsa private key
- INSTALLATION: installation id
- JWT_FILE: output file for jwt
- TOKEN_FILE: output file for token
- JSON_FILE: output file for both jwt and token in json

## build

`docker build -t drone-github-app .`

`go build -o drone-github-app`
