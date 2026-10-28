FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -tags nomsgpack -trimpath -ldflags="-s -w" -o /out/server .

FROM scratch
ENV GIN_MODE=release GOMEMLIMIT=20MiB GOGC=50 GOMAXPROCS=1 PGSSLMODE=disable
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /out/server /server
USER 10001:10001
EXPOSE 8080
ENTRYPOINT ["/server"]
