FROM scratch

WORKDIR /app

COPY compiled/LogRenderer-2.3.1 ./LogRenderer

ENTRYPOINT ["./LogRenderer", "--config", "config.yml"]
