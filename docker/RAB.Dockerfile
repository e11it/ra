FROM golang:modules as modules
FROM golang:1.14 as build
COPY --from=modules /go/pkg /go/pkg
WORKDIR /rab
COPY . .
COPY --from=modules /app/bin/golangci-lint /rab/bin/golangci-lint
RUN make go-build
FROM gcr.io/distroless/base
COPY --from=build /rab/rab /
ENTRYPOINT ["/rab"]