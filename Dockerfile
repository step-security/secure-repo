FROM golang:1.27.3@sha256:2920bb124f1a2e12a1d39f0cab756ae0068441ee1da7655dec7f5e374944095b AS build

WORKDIR /app

# cache dependencies
ADD go.mod go.sum ./
RUN go mod download

# build
ADD . .
# Static binary: the runtime image (Amazon Linux 2) has an older glibc than the
# build image, so a binary linked against the build image's glibc would not start.
RUN CGO_ENABLED=0 go build -o /main

COPY knowledge-base /knowledge-base

# copy artifacts to a clean image
FROM public.ecr.aws/lambda/provided:al2@sha256:474828b1e530e5a34b2c093cddc3919fac9a9cee47ac6fb9453604da54d470e8
COPY --from=build /main /main
COPY --from=build /knowledge-base /knowledge-base
ENTRYPOINT [ "/main" ]     


