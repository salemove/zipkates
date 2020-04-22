FROM golang:1.14-alpine AS build

WORKDIR /src/

COPY go.* /src/
RUN GO111MODULE=on go mod download

COPY main.go /src/
RUN CGO_ENABLED=0 go build -o /bin/proxy

FROM scratch
COPY --from=build /bin/proxy /bin/proxy
ENTRYPOINT ["/bin/proxy"]
