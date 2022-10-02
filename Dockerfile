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
FROM public.ecr.aws/lambda/provided:al2@sha256:d1a684c6effdc97b155ff28b9c779263ef45f25e167a103946d29ab1129f210d
COPY --from=build /main /main
COPY --from=build /knowledge-base /knowledge-base
ENTRYPOINT [ "/main" ]     


