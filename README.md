# shokai
A landing page for Linux servers

## Setup
* Docker Compose
    * `docker-compose.yaml`
    ```yaml
    version: '3.9'
    services:
    app:
        container_name: shokai
        image: ghcr.io/yude/shokai:main
        volumes:
        - type: bind
            source: ./config.toml
            target: /app/config.toml
        restart: always
    ```
    * `config.toml`
        * Change the values as you like.
    ```toml
    [general]
        location_id = "KIX"
        location_pretty = "Osaka, Japan"
        domain = "kix.example.com"

    [http]
        destinations = [
            "https://www.google.co.jp"
        ]
    ```

## License
MIT