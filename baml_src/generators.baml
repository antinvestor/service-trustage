generator target {
  output_type "go"
  output_dir "../"
  version "0.217.0"
  client_package_name "github.com/antinvestor/service-trustage"
  on_generate "gofmt -w . && goimports -w . && go mod tidy"
}
