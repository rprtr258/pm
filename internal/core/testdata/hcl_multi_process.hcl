process "database" {
  command     = "postgres"
  args        = ["-D", "/var/lib/postgresql/data"]
  cwd         = "/var/lib/postgresql"
  env = {
    DB_HOST = "localhost"
    DB_PORT = "5432"
  }
  tags       = ["database", "storage"]
  startup    = true
  depends_on = ["network"]
}

process "web-server" {
  command = "nginx"
  args    = ["-c", "/etc/nginx/nginx.conf"]
  env = {
    PORT = "80"
  }
  tags = ["web"]
}