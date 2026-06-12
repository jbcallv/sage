FROM golang:1.25 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /proxy ./cmd/proxy

FROM alpine:3.20
COPY --from=build /proxy /proxy
COPY policies/agents.cedar /policies/agents.cedar
ENV POLICY_FILE=/policies/agents.cedar
EXPOSE 8080
ENTRYPOINT ["/proxy"]
