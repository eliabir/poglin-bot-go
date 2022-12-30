FROM golang:1.19.4-alpine AS build
WORKDIR /app
COPY . .
RUN apk update && \
    apk add --no-cache git && \
    git clone -b 2022.10.21 --single-branch https://github.com/philiptn/bash-toolbox.git && \
    go get ./internal && \
    go build -o /bin/pogbot internal/main.go

FROM ubuntu:22.04 
WORKDIR /usr/src/bot
COPY --from=build /app/bin/pogbot /app/bash-toolbox/yt-dlp_discord ./
RUN mkdir -p /usr/src/bot/videos && \
    apt update -y && \
    apt install -y \
        curl \
        python3-pip \
        ffmpeg && \
    pip3 install yt-dlp

CMD [ "/bin/bash", "/usr/src/bot/pogbot"]
