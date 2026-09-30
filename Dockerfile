FROM golang:1.27.1-bookworm AS build

WORKDIR /src

ENV CGO_ENABLED=0

COPY go.mod go.sum ./
RUN go mod download
RUN go mod verify

COPY . .

RUN go test -mod=readonly -count=1 ./...
RUN go vet -mod=readonly ./...

RUN go build \
    -mod=readonly \
    -trimpath \
    -o /out/cinema \
    ./cmd/cinema

FROM scratch

COPY --from=build \
    /etc/ssl/certs/ca-certificates.crt \
    /etc/ssl/certs/ca-certificates.crt

COPY --from=build /out/cinema /cinema

USER 65532:65532

EXPOSE 8080

ENTRYPOINT ["/cinema"]
CMD ["serve"]
