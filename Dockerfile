# STEP 1 build executable binary

FROM golang:alpine as builder

# Install git
RUN apk update && apk add git

COPY . $GOPATH/src/github.com/AlbinoDrought/creamy-slanger
WORKDIR $GOPATH/src/github.com/AlbinoDrought/creamy-slanger

#get dependancies
#you can also use dep
RUN go get -d -v

#build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o /go/bin/creamy-slanger

# STEP 2 build a small image

# start from scratch
FROM scratch

# Copy our static executable
COPY --from=builder /go/bin/creamy-slanger /go/bin/creamy-slanger
ENTRYPOINT ["/go/bin/creamy-slanger"]
