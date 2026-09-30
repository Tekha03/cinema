FROM golang:1.27.1-bookworm AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build \
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