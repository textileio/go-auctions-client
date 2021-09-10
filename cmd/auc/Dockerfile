# syntax = docker/dockerfile:1-experimental

FROM golang:1.17-buster as builder

RUN mkdir /app 
WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download -x
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
  BIN_BUILD_FLAGS="CGO_ENABLED=0 GOOS=linux" make build

FROM alpine
COPY --from=builder /app/auc /app/auc
WORKDIR /app

ENV AUC_PATH /data/auc
RUN mkdir -p $AUC_PATH \
  && adduser -D -h $AUC_PATH -u 1000 -G users auc \
  && chown auc:users $AUC_PATH
USER auc
VOLUME $AUC_PATH

ENTRYPOINT ["./auc"]
CMD ["wallet daemon"]
