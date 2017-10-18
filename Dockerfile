FROM golang:1.9

RUN go get github.com/whyrusleeping/gx
RUN go get github.com/whyrusleeping/gx-go

ADD package.json package.json
RUN gx install

ADD . .

RUN go get -d .
RUN go build -o /bin/ipfs-watcher

CMD "/bin/ipfs-watcher"
