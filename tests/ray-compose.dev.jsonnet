[{
  name: "db-dev-service",
  command: "sh",
  args: ["-c", |||
    docker network create rainbow-dev-network --driver bridge || true
    docker volume create db-dev || true
    docker run \
      --name %(container_name)s \
      -p %(ports)s \
      --restart-policy %(restart)s \
      -e POSTGRES_PASSWORD=docker \
      -e POSTGRES_USER=docker \
      --network %(networks)s \
      --volume %(volumes)s \
      $(image)s
  ||| % {
    image: "postgres",
    container_name: "rainbow-db-dev",
    volumes: std.join("-v ", ["db-dev:/var/lib/postgresql/data/"]),
    ports: std.join("--ports ", ["5400:5432"]),
    environment: {
      POSTGRES_USER: "docker",
      POSTGRES_PASSWORD: "docker",
    },
    restart: "always",
    networks: std.join("--network ", ["rainbow-dev-network"]),
  }],
}]
