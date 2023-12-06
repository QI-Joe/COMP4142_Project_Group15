FROM golang:latest

WORKDIR /goblockchain

COPY ./goblockchain /goblockchain/

RUN go mod download

EXPOSE 5000
EXPOSE 8080

# COPY entrypoint.sh /go/src/goblockchain/entrypoint.sh
# RUN chmod +x /go/src/goblockchain/entrypoint.sh

CMD [ "go", "run", "./blockchain_server" ]