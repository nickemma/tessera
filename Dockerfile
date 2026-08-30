FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/gateway ./cmd/gateway
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/mock-model ./cmd/mock-model

FROM scratch AS gateway
COPY --from=build /out/gateway /gateway
USER 65532:65532
ENTRYPOINT ["/gateway"]

FROM scratch AS mock-model
COPY --from=build /out/mock-model /mock-model
USER 65532:65532
ENTRYPOINT ["/mock-model"]
