FROM golang:1.25-trixie

# Development image
WORKDIR /app
ENV CGO_ENABLED=0
RUN go install github.com/air-verse/air@latest
RUN go install github.com/go-delve/delve/cmd/dlv@latest
COPY go.mod go.sum ./

RUN go mod download
CMD ["air"]
