# shokai

Landing page for our servers

## Setup
* Docker Compose
    * `docker-compose.yaml`
    ```yaml
    services:
    app:
        container_name: shokai
        image: ghcr.io/yude/shokai:main
        volumes:
        - type: bind
            source: ./config.toml
            target: /app/config.toml
        - type: bind
            source: /etc/hostname
            target: /etc/host_hostname
            read_only: true
        restart: always
    ```
    * `config.toml`
        * Please refer to `config.sample.toml` for format.

## License
MIT
