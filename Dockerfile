FROM golang AS builder

WORKDIR /builder

ENV CGO_ENABLED=0

COPY go.mod go.sum /builder/
RUN go mod download

COPY * /builder/
RUN go build -v -o /etcd-postgresql-sync

FROM alpine

COPY --from=builder /etcd-postgresql-sync /bin/etcd-postgresql-sync

ENTRYPOINT ["/bin/etcd-postgresql-sync"]
