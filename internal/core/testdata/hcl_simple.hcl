process "simple-service" {
  command = "echo"
  args    = ["Hello", "HCL!"]
  cwd     = "/tmp"
  env = {
    SERVICE_NAME = "test"
    VERSION      = "1.0.0"
  }
  tags    = ["simple", "test"]
  startup = false
}