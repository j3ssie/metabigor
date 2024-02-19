FROM golang:latest-bullseye AS builder

# Build Args
ARG BUILD_COMMAND="go build -o metabigor ."
ARG BINARY_NAME="metabigor"
ARG CGO_ENABLED="0"
# Env setup
ENV CGO_ENABLED=${CGO_ENABLED}

# Setup workdir
WORKDIR /build

# Copy source code
COPY . .

# Fetch dependencies
RUN apt install build-essential
RUN go mod download

RUN go build -o ${BINARY_NAME} .

# Runner stage
FROM golang:latest-alpine AS runner

# Build Args
ARG BINARY_NAME="metabigor"
ARG START_COMMAND="./metabigor"

# Setup workdir
WORKDIR /app
RUN useradd -D metabigor
RUN chown -R metabigor:metabigor /app

# Copy binary
COPY --from=builder /build/${BINARY_NAME} .

# Create entrypoint
RUN echo ${START_COMMAND} > /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh
USER metabigor

# Setup Entrypoint
ENTRYPOINT ["sh", "-c", "/user/entrypoint.sh"]
