FROM golang:latest as builder
RUN mkdir -p /service
WORKDIR /service
ADD ./ /service
RUN GIT_COMMIT=$(git rev-list -1 HEAD) && GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD) && \
  CGO_ENABLED=0 go install -ldflags "-X main.gitCommit=$GIT_COMMIT -X main.gitBranch=$GIT_BRANCH" ./...

FROM golang:latest
COPY --from=builder /go/bin/nanoreddit /app/nanoreddit
WORKDIR /app
ENTRYPOINT [ "/app/nanoreddit" ]
