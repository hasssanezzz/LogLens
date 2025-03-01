
$target = "POST http://127.0.0.1:3000"
$body = './body.txt'

echo $target | vegeta attack `
  -rate=3000 `
  -duration=10s `
  -header "Content-Type: application/json" `
  -header "KV-environment: production" `
  -header "KV-level: info" `
  -body "$body" | vegeta report