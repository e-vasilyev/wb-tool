FROM golang:1.23.3-alpine3.20 AS build

WORKDIR /app

COPY . .

RUN go build -v -o wb-tool ./cmd/tool

FROM alpine:3.20.3

ARG USERNMAE=app
ARG USER_UID=1001
ARG USER_GID=${USER_UID}

RUN apk add --no-cache tzdata

RUN addgroup -g ${USER_GID} -S ${USERNMAE}\
    && adduser -u ${USER_UID} -G ${USERNMAE} -S ${USERNMAE}\
    -H

USER ${USERNMAE}

WORKDIR /app

COPY --from=build /app/wb-tool ./wb-tool

CMD ["/app/wb-tool"]