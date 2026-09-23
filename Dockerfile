FROM --platform=${BUILDPLATFORM} golang:alpine AS quartzbuilder
WORKDIR /usr/src/app

COPY . .

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -v -a \
    -ldflags="-X 'github.com/pgvillage-tools/pgquartz/cmd.Version=$VERSION'" -o ./bin/pgquartz ./cmd/pgquartz

FROM alpine/git

COPY --from=quartzbuilder /usr/src/app/bin/pgquartz /usr/local/bin/
COPY jobs /etc/pgquatz/jobs
ENTRYPOINT [ "/usr/local/bin/pgquartz" ]
CMD [ "-c", "/etc/pgquatz/jobs/jobspec1/job.yml" ]
