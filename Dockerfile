FROM golang:1.21.6 AS build

WORKDIR /src
COPY ./go.mod ./go.sum ./
RUN go mod download

COPY ./ ./
ENV CGO_ENABLED=0
RUN go mod download
RUN CGO_ENABLED=0 go build -o /vicinitas .


FROM gcr.io/distroless/static AS final

LABEL maintainer="soerenschneider"
USER nonroot:nonroot
COPY --from=build /vicinitas /vicinitas

ENTRYPOINT ["/vicinitas"]
