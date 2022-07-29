FROM golang:1.17@sha256:bfb57478eb0b381f242b3ab27b373bca5516eb9d35eef98a41a0ba2742ab517d as build

WORKDIR /app

# cache dependencies
ADD go.mod go.sum ./
RUN go mod download

# build
ADD . .
RUN go build -o /main

COPY knowledge-base /knowledge-base

# copy artifacts to a clean image
FROM public.ecr.aws/lambda/provided:al2@sha256:474828b1e530e5a34b2c093cddc3919fac9a9cee47ac6fb9453604da54d470e8
COPY --from=build /main /main
COPY --from=build /knowledge-base /knowledge-base
ENTRYPOINT [ "/main" ]     


