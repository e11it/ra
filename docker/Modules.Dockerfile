FROM golang:1.14 as origin
WORKDIR /app
COPY Makefile go.mod ./
RUN make go-download && make go-lint-install