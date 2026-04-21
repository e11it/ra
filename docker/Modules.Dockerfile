ARG golang_version=1.26.2
FROM golang:${golang_version} as origin
WORKDIR /app
COPY Makefile go.mod ./
RUN make go-download && make go-lint-install