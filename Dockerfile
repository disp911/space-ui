# ========================================================
# Stage: Frontend (built once on the build machine's platform)
# ========================================================
FROM --platform=$BUILDPLATFORM node:24-alpine AS frontend
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json frontend/
RUN npm ci --prefix frontend
COPY frontend frontend
COPY web/translation web/translation
RUN npm --prefix frontend run build

# ========================================================
# Stage: Builder
# ========================================================
FROM golang:1.26-alpine AS builder
WORKDIR /app
ARG TARGETARCH

RUN apk --no-cache --update add \
  build-base \
  gcc \
  curl \
  unzip

COPY . .
COPY --from=frontend /app/web/app web/app

ENV CGO_ENABLED=1
ENV CGO_CFLAGS="-D_LARGEFILE64_SOURCE"
RUN go build -ldflags "-w -s" -o build/space-ui main.go
RUN ./DockerInit.sh "$TARGETARCH"

# ========================================================
# Stage: Final Image of Space UI
# ========================================================
FROM alpine
ENV TZ=Asia/Tehran
WORKDIR /app

RUN apk add --no-cache --update \
  ca-certificates \
  tzdata \
  fail2ban \
  bash \
  curl \
  openssl

COPY --from=builder /app/build/ /app/
COPY --from=builder /app/DockerEntrypoint.sh /app/
COPY --from=builder /app/space-ui.sh /usr/bin/space-ui


# Configure fail2ban
RUN rm -f /etc/fail2ban/jail.d/alpine-ssh.conf \
  && cp /etc/fail2ban/jail.conf /etc/fail2ban/jail.local \
  && sed -i "s/^\[ssh\]$/&\nenabled = false/" /etc/fail2ban/jail.local \
  && sed -i "s/^\[sshd\]$/&\nenabled = false/" /etc/fail2ban/jail.local \
  && sed -i "s/#allowipv6 = auto/allowipv6 = auto/g" /etc/fail2ban/fail2ban.conf

RUN chmod +x \
  /app/DockerEntrypoint.sh \
  /app/space-ui \
  /usr/bin/space-ui

ENV SPACE_UI_ENABLE_FAIL2BAN="true"
EXPOSE 2053
VOLUME [ "/etc/space-ui" ]
CMD [ "./space-ui" ]
ENTRYPOINT [ "/app/DockerEntrypoint.sh" ]
