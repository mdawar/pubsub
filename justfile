_default:
  @just -l -u

# Run the tests.
test:
  go test -race -cover -count 1
