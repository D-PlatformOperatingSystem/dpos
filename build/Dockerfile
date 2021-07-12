FROM ubuntu:16.04

WORKDIR /root
COPY dplatformos dplatformos
COPY dplatformos-cli dplatformos-cli
COPY dplatformos.toml dplatformos-solo.toml ./

RUN ./dplatformos-cli cert --host=127.0.0.1

CMD ["/root/dplatformos", "-f", "/root/dplatformos.toml"]
