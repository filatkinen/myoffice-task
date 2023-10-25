FROM golang:1.21

WORKDIR /usr/src/app/

COPY . /usr/src/app/
RUN go build -v -o ./build/cliurl  ./cmd

ENTRYPOINT [ "tail","-f","/dev/null"]
