FROM golang:1.25.5 AS build

WORKDIR /go/src/app
COPY . .

RUN go mod download

RUN CGO_ENABLED=0 go build -o /go/bin/app ./cmd/opengymserver

# Frontend build stage
FROM node:20-slim AS frontend-build

ENV PNPM_HOME="/pnpm"
ENV PATH="$PNPM_HOME:$PATH"
RUN corepack enable && corepack prepare pnpm@9 --activate

WORKDIR /app

# Copy package files for dependency installation
COPY frontend/package.json frontend/pnpm-lock.yaml ./

# Install dependencies with BuildKit cache mount
RUN --mount=type=cache,id=pnpm,target=/pnpm/store pnpm install --frozen-lockfile

# Copy frontend source files
COPY frontend/ ./

# Build frontend
RUN pnpm build

FROM gcr.io/distroless/static:nonroot

COPY --from=build --chmod=755 /go/bin/app /app
COPY --from=frontend-build /app/dist /frontend/dist
CMD ["/app"]
