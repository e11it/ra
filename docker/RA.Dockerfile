ARG GOLANG_VERSION=1.21.6
ARG APP_VERSION=latest
ARG APP_IMAGE_DATE_CREATED
ARG APP_COMMIT_SHA

FROM golang:modules as modules
FROM golang:${GOLANG_VERSION} as build
LABEL autodelete="true"
COPY --from=modules /go/pkg /go/pkg
WORKDIR /app
COPY . .
COPY --from=modules /app/bin/golangci-lint /app/bin/golangci-lint
RUN PROJECTNAME=ra make go-build




FROM gcr.io/distroless/base-debian12

LABEL org.opencontainers.image.authors="ilya makarov" \
      org.opencontainers.image.created=${APP_IMAGE_DATE_CREATED} \
      org.opencontainers.image.version="${APP_VERSION}" \
      org.opencontainers.image.revision="${APP_COMMIT_SHA}" \
      org.opencontainers.image.title="RA" \
      org.opencontainers.image.description="Nginx auth module to validate req"
WORKDIR /app

COPY --from=build /app/ra /app/bin/ra
CMD ["/app/bin/ra"]