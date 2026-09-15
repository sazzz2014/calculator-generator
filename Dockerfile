FROM rust:1.90.0-bookworm AS rust-build
WORKDIR /src/native/rust
COPY native/rust/Cargo.toml native/rust/Cargo.lock ./
COPY native/rust/src ./src
RUN cargo test --locked && cargo build --locked --release

FROM golang:1.25.4-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=rust-build /src/native/rust/target/release/libcalculator_rust.a ./native/rust/target/release/
ENV CGO_ENABLED=1

FROM build AS verify
RUN test -z "$(gofmt -l cmd internal scripts)" && go test ./... && go test -race ./... && go vet ./...

FROM build AS binaries
RUN go build -trimpath -o /out/calculator ./cmd/calculator && CGO_ENABLED=0 go build -trimpath -o /out/generator ./cmd/generator && CGO_ENABLED=0 go build -trimpath -o /out/healthcheck ./cmd/healthcheck

FROM debian:bookworm-slim AS runtime
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*
USER 65532:65532
STOPSIGNAL SIGINT

FROM runtime AS calculator
COPY --from=binaries /out/calculator /usr/local/bin/calculator
COPY --from=binaries /out/healthcheck /usr/local/bin/healthcheck
EXPOSE 8080
ENTRYPOINT ["calculator"]

FROM runtime AS generator
COPY --from=binaries /out/generator /usr/local/bin/generator
ENTRYPOINT ["generator"]
