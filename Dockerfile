FROM golang:1.26.2-alpine AS build

WORKDIR /src

COPY go.mod ./
COPY *.go ./

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/hashctl \
    .

FROM scratch

COPY --from=build /out/hashctl /hashctl

USER 65532:65532
EXPOSE 8080

ENTRYPOINT ["/hashctl"]
CMD ["serve", "--listen", ":8080"]
