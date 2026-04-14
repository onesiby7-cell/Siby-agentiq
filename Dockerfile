FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /siby ./cmd/siby-agentiq

FROM alpine:3.19

RUN apk add --no-cache curl bash openssh-client git

COPY --from=builder /siby /usr/local/bin/siby
COPY --from=builder /app/install.sh /install.sh

RUN chmod +x /usr/local/bin/siby && \
    chmod +x /install.sh && \
    mkdir -p /root/.local/bin && \
    ln -s /usr/local/bin/siby /root/.local/bin/siby

ENV PATH="/root/.local/bin:${PATH}"

ENTRYPOINT ["siby"]
CMD ["--help"]
