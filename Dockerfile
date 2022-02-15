ARG GO_VERSION=1.17

FROM golang:${GO_VERSION}-alpine AS builder

RUN mkdir /user \
    && echo 'daemon:x:2:2:daemon:/:' > /user/passwd \
    && echo 'daemon:x:2:' > /user/group

WORKDIR ${GOPATH}/src

COPY . ./

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w "\
       -a -installsuffix 'static' -o /jobgraphs ./cmd/*

FROM scratch AS cronjobber

COPY --from=builder /user/group /user/passwd /etc/
COPY --from=builder /jobgraphs /jobgraphs

USER daemon:daemon

ENTRYPOINT ["/jobgraphs"]

