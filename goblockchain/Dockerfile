FROM golang:lastest

WORKDIR /go/src/goblockchain

COPY ../goblockchain /go/src/goblockchain/

RUN go mod download

EXPOSE 5000
EXPOSE 8080

COPY entrypoint.sh /go/src/goblockchain/entrypoint.sh
RUN chmod +x /go/src/goblockchain/entrypoint.sh

CMD [ "/go/src/goblockchain/entrypoint.sh" ]