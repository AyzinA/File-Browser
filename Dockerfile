# Build stage
FROM golang:1.23 AS build
WORKDIR /src

COPY go.mod ./
COPY . ./

# We embed templates/static into the binary
ENV CGO_ENABLED=0
RUN go build -o /out/file-browser ./main.go

# Minimal runtime
FROM gcr.io/distroless/static:nonroot
WORKDIR /app
COPY --from=build /out/file-browser /app/file-browser
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app/file-browser"]
