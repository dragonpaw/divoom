FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/divoom ./cmd/divoom

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/divoom /usr/local/bin/divoom

# Default to the frame we discovered during dev. Override with `-e` or in
# docker-compose.yml if you ever swap devices or have more than one on the LAN.
ENV DIVOOM_FRAME_MAC=4c37deff59df

ENTRYPOINT ["/usr/local/bin/divoom"]
CMD ["help"]
