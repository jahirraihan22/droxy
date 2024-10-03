FROM golang:alpine3.19 as builder
LABEL authors="jahir"
WORKDIR /app
COPY . ./
RUN go mod download && go build -o /droxy

FROM alpine:3.19
RUN apk update &&  \
    apk --no-cache add ca-certificates

WORKDIR /
COPY --from=builder /droxy /droxy
EXPOSE 8000
EXPOSE 8080
ENTRYPOINT ["/droxy","top","b"]