[
  {
    name: "create-foo-network",
    command: "docker",
    args: ["create", "network", "foo"],
  },
  {
    name: "echo",
    command: "docker",
    args: ["run", "--network", "foo", "hashicorp/http-echo", '-text="hello world"'],
  },
  {
    name: "t1",
    command: "docker",
    args: ["run", "--network", "foo", "curlimages/curl", "sh -c 'while true; do curl -s http://echo:5678; sleep 1; done'"],
  },
  // image: curlimages/curl
  // command: sh -c 'while true; do curl -s http://echo:5678; sleep 1; done'
  // networks:
  //   - foo
  // depends_on:
  //   echo:
  //     condition: service_healthy
]
