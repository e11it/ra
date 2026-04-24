ARG GOLANG_VERSION=1.26.2
ARG APP_VERSION=latest
ARG APP_IMAGE_DATE_CREATED
ARG APP_COMMIT_SHA
ARG GO_TAGS=nomsgpack

FROM golang:${GOLANG_VERSION} AS modules
WORKDIR /app
COPY Makefile go.mod go.sum ./
RUN make go-download && make go-lint-install

FROM golang:${GOLANG_VERSION} AS build
ARG GO_TAGS
LABEL autodelete="true"
# Статическая сборка: без glibc в рантайме — меньше CVE (libc6 в base-debian12), проще trivy.
# RA — чистый Go, CGO не нужен (net, DNS — в netgo).
ENV CGO_ENABLED=0
COPY --from=modules /go/pkg /go/pkg
COPY --from=modules /app/bin/golangci-lint /app/bin/golangci-lint
WORKDIR /app
COPY . .
RUN PROJECTNAME=ra GO_TAGS=${GO_TAGS} make go-build
# static-debian12: нет пакетного glibc; :nonroot — не root в контейнере.
FROM gcr.io/distroless/static-debian12:nonroot
ARG APP_VERSION
ARG APP_IMAGE_DATE_CREATED
ARG APP_COMMIT_SHA

LABEL org.opencontainers.image.authors="ilya makarov" \
      org.opencontainers.image.created=${APP_IMAGE_DATE_CREATED} \
      org.opencontainers.image.version="${APP_VERSION}" \
      org.opencontainers.image.revision="${APP_COMMIT_SHA}" \
      org.opencontainers.image.title="RA" \
      org.opencontainers.image.description="Nginx auth module to validate req"
WORKDIR /app

COPY --from=build /app/ra /app/bin/ra
COPY --from=build /app/api/openapi /app/api/openapi
CMD ["/app/bin/ra"]