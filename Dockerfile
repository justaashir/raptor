FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN go build -ldflags "-X raptor/cmd.Version=${VERSION}" -o /raptor .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=build /raptor /raptor
EXPOSE 8080
CMD ["/raptor", "serve"]
