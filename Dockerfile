from golang

RUN mkdir /plugin

WORKDIR /plugin

COPY . .

RUN go build -o plugin

ENTRYPOINT /plugin/plugin