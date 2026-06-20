env "local" {
  url = "postgres://postgres:postgres@localhost:5432/bitly?sslmode=disable"
  migration {
    dir = "file://db/migrations"
  }
}
