FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod main.go ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /hello-api .

FROM scratch
COPY --from=build /hello-api /hello-api
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/hello-api"]