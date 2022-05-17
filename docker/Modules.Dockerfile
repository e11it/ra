ARG golang_version=latest
FROM golang:${golang_version} as origin
WORKDIR /app
COPY Makefile go.mod ./
RUN make go-download && make go-lint-install