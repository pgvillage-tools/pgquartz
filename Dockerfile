FROM --platform=${BUILDPLATFORM} golang:alpine AS quartzbuilder
WORKDIR /usr/src/app

COPY . .

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=v0.5.1-devel

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -v -a \
    -ldflags="-X 'github.com/pgvillage-tools/PgQuartz/internal/appVersion=$VERSION'" -o ./bin/pgquartz ./cmd/pgquartz

FROM alpine/git

COPY --from=quartzbuilder /usr/src/app/bin/pgquartz /usr/local/bin/
COPY jobs /etc/pgquatz/jobs
ENTRYPOINT [ "/usr/local/bin/pgquartz" ]
CMD [ "-c", "/etc/pgquatz/jobs/jobspec1/job.yml" ]
