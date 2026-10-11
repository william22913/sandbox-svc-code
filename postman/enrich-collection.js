#!/usr/bin/env node
// Re-applies the structural test script to the generated Postman collection.
//
// The collection is a build artifact of `openapi2postmanv2` (see the command in
// docs/FEAT-001/dev-test-results.md) — regenerating it overwrites anything hand
// added, so the additions live here instead of in the JSON. Run after every
// regeneration:
//
//   npx openapi-to-postmanv2 -s api/openapi.yaml -o postman/collection.json -p -O folderStrategy=Tags
//   node postman/enrich-collection.js
//
// The tests are deliberately structural only — a JSON body and a sane latency.
// Per-route status codes are asserted by the Newman run's own expectations in
// docs/FEAT-001/dev-test-results.md, not baked in here as business values.

const fs = require('fs')
const path = require('path')

const file = path.join(__dirname, 'collection.json')
const collection = JSON.parse(fs.readFileSync(file, 'utf8'))

const source = [
  "pm.test('responds with a JSON body', () => {",
  '  pm.response.json()',
  '})',
  "pm.test('responds within 2s', () => {",
  '  pm.expect(pm.response.responseTime).to.be.below(2000)',
  '})',
].join('\n')

collection.event = [
  {
    listen: 'test',
    script: { type: 'text/javascript', exec: source.split('\n') },
  },
]

// The converter fills a parameter's value with a type hint (`<long>` here)
// rather than a usable value — in a path variable, and in this API's one
// numeric JSON body field. Neither is something Newman can send, so both are
// pointed at the environment variable the run is otherwise driven by.
//
// The body placeholder is written *without* surrounding quotes so the
// substituted value stays a JSON number rather than becoming a string.
const PLACEHOLDERS = ['<long>', '<integer>', '<int32>', '<int64>', '<number>']

function detype(text) {
  let out = text
  for (const placeholder of PLACEHOLDERS) {
    out = out.split(`"${placeholder}"`).join('{{item_id}}')
    out = out.split(placeholder).join('{{item_id}}')
  }
  return out
}

let rewritten = 0
for (const folder of collection.item || []) {
  for (const request of folder.item || []) {
    for (const variable of request.request?.url?.variable || []) {
      const next = detype(String(variable.value))
      if (next !== variable.value) {
        variable.value = next
        rewritten++
      }
    }

    const body = request.request?.body
    if (body && typeof body.raw === 'string') {
      const next = detype(body.raw)
      if (next !== body.raw) {
        body.raw = next
        rewritten++
      }
    }
  }
}

fs.writeFileSync(file, JSON.stringify(collection, null, 2) + '\n')
console.log(
  `added collection-level structural tests; rewrote ${rewritten} path-parameter placeholder(s)`
)
