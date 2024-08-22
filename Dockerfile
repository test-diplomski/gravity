FROM golang:alpine as build_container
WORKDIR /app
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN go build -o agent_queue ./cmd/server/main.go

FROM alpine
COPY --from=build_container /app/agent_queue /usr/bin
ARG GRPC_PORT
EXPOSE $GRPC_PORT
ENTRYPOINT ["agent_queue"]