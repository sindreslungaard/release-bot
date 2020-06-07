FROM golang:1.14-alpine

WORKDIR /go/src/release-bot
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...

EXPOSE 3000

CMD ["release-bot"]