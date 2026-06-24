FROM golang:1.22 AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download || true
COPY . .
RUN go mod tidy && CGO_ENABLED=0 go build -o /voice .

FROM gcr.io/distroless/static-debian12
COPY --from=build /voice /voice
WORKDIR /
EXPOSE 8080
ENTRYPOINT ["/voice"]
