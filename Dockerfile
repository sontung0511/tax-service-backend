FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /tax-api ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /tax-api /tax-api
VOLUME ["/data"]
ENV HTTP_ADDR=:8080 TAX_DATA_PATH=/data/tax.json
EXPOSE 8080
ENTRYPOINT ["/tax-api"]
